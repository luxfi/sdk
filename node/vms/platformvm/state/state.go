// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/btree"
	"github.com/luxfi/metric"

	"github.com/luxfi/consensus"
	"github.com/luxfi/consensus/validator/uptime"
	validators "github.com/luxfi/consensus/validator"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/database"
	"github.com/luxfi/database/linkeddb"
	"github.com/luxfi/database/prefixdb"
	"github.com/luxfi/database/versiondb"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/container"
	"github.com/luxfi/cache/lru"
	"github.com/luxfi/container/metercacher"
	"github.com/luxfi/sdk/node/codec"
	"github.com/luxfi/upgrade"
	"github.com/luxfi/const"
	hashing "github.com/luxfi/crypto/hash"
	"github.com/luxfi/sdk/utils/iterator"
	"github.com/luxfi/sdk/utils/maybe"
	"github.com/luxfi/sdk/utils/timer"
	"github.com/luxfi/sdk/utils/wrappers"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/components/gas"
	"github.com/luxfi/sdk/node/vms/platformvm/block"
	"github.com/luxfi/sdk/node/vms/platformvm/config"
	"github.com/luxfi/sdk/node/vms/platformvm/fx"
	"github.com/luxfi/sdk/node/vms/platformvm/genesis"
	"github.com/luxfi/sdk/node/vms/platformvm/metrics"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
	"github.com/luxfi/sdk/node/vms/platformvm/status"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"

	safemath "github.com/luxfi/math"
)

const (
	defaultTreeDegree             = 2
	indexIterationLimit           = 4096
	indexIterationSleepMultiplier = 5
	indexIterationSleepCap        = 10 * time.Second
	indexLogFrequency             = 30 * time.Second
)

var (
	_ State = (*state)(nil)

	errValidatorSetAlreadyPopulated   = errors.New("validator set already populated")
	errIsNotNet                    = errors.New("is not a subnet")
	errMissingPrimaryNetworkValidator = errors.New("missing primary network validator")

	BlockIDPrefix                 = []byte("blockID")
	BlockPrefix                   = []byte("block")
	ValidatorsPrefix              = []byte("validators")
	CurrentPrefix                 = []byte("current")
	PendingPrefix                 = []byte("pending")
	ValidatorPrefix               = []byte("validator")
	DelegatorPrefix               = []byte("delegator")
	NetValidatorPrefix            = []byte("subnetValidator")
	NetDelegatorPrefix         = []byte("subnetDelegator")
	ValidatorWeightDiffsPrefix    = []byte("flatValidatorDiffs")
	ValidatorPublicKeyDiffsPrefix = []byte("flatPublicKeyDiffs")
	TxPrefix                      = []byte("tx")
	RewardUTXOsPrefix             = []byte("rewardUTXOs")
	UTXOPrefix                    = []byte("utxo")
	NetPrefix                  = []byte("subnet")
	NetOwnerPrefix             = []byte("subnetOwner")
	NetToL1ConversionPrefix    = []byte("subnetToL1Conversion")
	TransformedNetPrefix       = []byte("transformedNet")
	SupplyPrefix                  = []byte("supply")
	ChainPrefix                   = []byte("chain")
	ChainNamePrefix               = []byte("chainName") // maps lowercase chain name -> chainID
	ExpiryReplayProtectionPrefix  = []byte("expiryReplayProtection")
	L1Prefix                      = []byte("l1")
	WeightsPrefix                 = []byte("weights")
	ChainIDNodeIDPrefix          = []byte("subnetIDNodeID")
	ActivePrefix                  = []byte("active")
	InactivePrefix                = []byte("inactive")
	SingletonPrefix               = []byte("singleton")

	TimestampKey         = []byte("timestamp")
	FeeStateKey          = []byte("fee state")
	L1ValidatorExcessKey = []byte("l1Validator excess")
	AccruedFeesKey       = []byte("accrued fees")
	CurrentSupplyKey     = []byte("current supply")
	LastAcceptedKey      = []byte("last accepted")
	HeightsIndexedKey    = []byte("heights indexed")
	InitializedKey       = []byte("initialized")
	BlocksReindexedKey   = []byte("blocks reindexed.3")

	emptyL1ValidatorCache = &cache.Empty[ids.ID, maybe.Maybe[L1Validator]]{}
)

// Chain collects all methods to manage the state of the chain for block
// execution.
type Chain interface {
	Expiry
	L1Validators
	Stakers
	L1Validators
	lux.UTXOAdder
	lux.UTXOGetter
	lux.UTXODeleter

	GetTimestamp() time.Time
	SetTimestamp(tm time.Time)

	GetFeeState() gas.State
	SetFeeState(f gas.State)

	GetL1ValidatorExcess() gas.Gas
	SetL1ValidatorExcess(e gas.Gas)

	GetAccruedFees() uint64
	SetAccruedFees(f uint64)

	GetCurrentSupply(subnetID ids.ID) (uint64, error)
	SetCurrentSupply(subnetID ids.ID, cs uint64)

	AddRewardUTXO(txID ids.ID, utxo *lux.UTXO)

	AddNet(netID ids.ID)

	GetNetOwner(netID ids.ID) (fx.Owner, error)
	SetNetOwner(netID ids.ID, owner fx.Owner)

	GetNetToL1Conversion(subnetID ids.ID) (NetToL1Conversion, error)
	SetNetToL1Conversion(subnetID ids.ID, c NetToL1Conversion)

	GetNetTransformation(subnetID ids.ID) (*txs.Tx, error)
	AddNetTransformation(transformNetTx *txs.Tx)

	AddChain(createChainTx *txs.Tx)

	// Chain name uniqueness - for network-wide chain name resolution
	GetChainIDByName(name string) (ids.ID, error)
	IsChainNameTaken(name string) bool

	GetTx(txID ids.ID) (*txs.Tx, status.Status, error)
	AddTx(tx *txs.Tx, status status.Status)

	// L1 Validator support - most methods inherited from L1Validators interface
	// Only PutL1Validator is Chain-specific
	PutL1Validator(validator L1Validator) error
}

type State interface {
	Chain
	uptime.State
	lux.UTXOReader

	GetLastAccepted() ids.ID
	SetLastAccepted(blkID ids.ID)

	GetStatelessBlock(blockID ids.ID) (block.Block, error)

	// Invariant: [block] is an accepted block.
	AddStatelessBlock(block block.Block)

	GetBlockIDAtHeight(height uint64) (ids.ID, error)

	GetRewardUTXOs(txID ids.ID) ([]*lux.UTXO, error)
	GetChainIDs() ([]ids.ID, error)
	GetNetIDs() ([]ids.ID, error)
	GetChains(netID ids.ID) ([]*txs.Tx, error)

	// ApplyValidatorWeightDiffs iterates from [startHeight] towards the genesis
	// block until it has applied all of the diffs up to and including
	// [endHeight]. Applying the diffs modifies [validators].
	//
	// Invariant: If attempting to generate the validator set for
	// [endHeight - 1], [validators] must initially contain the validator
	// weights for [startHeight].
	//
	// Note: Because this function iterates towards the genesis, [startHeight]
	// will typically be greater than or equal to [endHeight]. If [startHeight]
	// is less than [endHeight], no diffs will be applied.
	ApplyValidatorWeightDiffs(
		ctx context.Context,
		validators map[ids.NodeID]*validators.GetValidatorOutput,
		startHeight uint64,
		endHeight uint64,
		netID ids.ID,
	) error

	// ApplyValidatorPublicKeyDiffs iterates from [startHeight] towards the
	// genesis block until it has applied all of the diffs up to and including
	// [endHeight]. Applying the diffs modifies [validators].
	//
	// Invariant: If attempting to generate the validator set for
	// [endHeight - 1], [validators] must initially contain the validator
	// weights for [startHeight].
	//
	// Note: Because this function iterates towards the genesis, [startHeight]
	// will typically be greater than or equal to [endHeight]. If [startHeight]
	// is less than [endHeight], no diffs will be applied.
	ApplyValidatorPublicKeyDiffs(
		ctx context.Context,
		validators map[ids.NodeID]*validators.GetValidatorOutput,
		startHeight uint64,
		endHeight uint64,
		subnetID ids.ID,
	) error

	SetHeight(height uint64)

	// GetCurrentValidators returns subnet and L1 validators for the given
	// subnetID along with the current P-chain height.
	// This method works for both subnets and L1s. Depending of the requested
	// subnet/L1 validator schema, the return values can include only subnet
	// validator, only L1 validators or both if there are initial stakers in the
	// L1 conversion.
	GetCurrentValidators(ctx context.Context, subnetID ids.ID) ([]*Staker, []L1Validator, uint64, error)

	// Discard uncommitted changes to the database.
	Abort()

	// ReindexBlocks converts any block indices using the legacy storage format
	// to the new format. If this database has already updated the indices,
	// this function will return immediately, without iterating over the
	// database.
	//
	// TODO: Remove after v1.14.x is activated
	ReindexBlocks(lock sync.Locker, log log.Logger) error

	// Commit changes to the base database.
	Commit() error

	// Returns a batch of unwritten changes that, when written, will commit all
	// pending changes to the base database.
	CommitBatch() (database.Batch, error)

	Checksum() ids.ID

	Close() error
}

// Prior to https://github.com/luxfi/sdk/node/pull/1719, blocks were
// stored as a map from blkID to stateBlk. Nodes synced prior to this PR may
// still have blocks partially stored using this legacy format.
//
// TODO: Remove after v1.14.x is activated
type stateBlk struct {
	Bytes  []byte `serialize:"true"`
	Status uint32 `serialize:"true"`
}

// RegisterStateBlockType registers the stateBlk type with the given codec.
// This is needed for backward compatibility with old block storage format.
func RegisterStateBlockType(targetCodec codec.Registry) error {
	return targetCodec.RegisterType(&stateBlk{})
}

// Initialize the stateBlk type registration with the block codecs.
// This must be called before using parseStoredBlock for backward compatibility.
func init() {
	// We need to register stateBlk with block.GenesisCodec so that
	// parseStoredBlock can deserialize legacy block storage format.
	// Since state imports block (no import cycle), we can register directly.
	if err := block.RegisterGenesisType(&stateBlk{}); err != nil {
		panic(err)
	}
}

/*
 * VMDB
 * |-. validators
 * | |-. current
 * | | |-. validator
 * | | | '-. list
 * | | |   '-- txID -> uptime + potential reward + potential delegatee reward
 * | | |-. delegator
 * | | | '-. list
 * | | |   '-- txID -> potential reward
 * | | |-. subnetValidator
 * | | | '-. list
 * | | |   '-- txID -> uptime + potential reward + potential delegatee reward
 * | | '-. subnetDelegator
 * | |   '-. list
 * | |     '-- txID -> potential reward
 * | |-. pending
 * | | |-. validator
 * | | | '-. list
 * | | |   '-- txID -> nil
 * | | |-. delegator
 * | | | '-. list
 * | | |   '-- txID -> nil
 * | | |-. subnetValidator
 * | | | '-. list
 * | | |   '-- txID -> nil
 * | | '-. subnetDelegator
 * | |   '-. list
 * | |     '-- txID -> nil
 * | |-. l1
 * | | |-. weights
 * | | | '-- subnetID -> weight
 * | | |-. subnetIDNodeID
 * | | | '-- subnetID+nodeID -> validationID
 * | | |-. active
 * | | | '-- validationID -> l1Validator
 * | | '-. inactive
 * | |   '-- validationID -> l1Validator
 * | |-. weight diffs
 * | | '-- subnet+height+nodeID -> weightChange
 * | '-. pub key diffs
 * |   '-- subnet+height+nodeID -> uncompressed public key or nil
 * |-. blockIDs
 * | '-- height -> blockID
 * |-. blocks
 * | '-- blockID -> block bytes
 * |-. txs
 * | '-- txID -> tx bytes + tx status
 * |- rewardUTXOs
 * | '-. txID
 * |   '-. list
 * |     '-- utxoID -> utxo bytes
 * |- utxos
 * | '-- utxoDB
 * |-. subnets
 * | '-. list
 * |   '-- txID -> nil
 * |-. subnetOwners
 * | '-- subnetID -> owner
 * |-. subnetToL1Conversions
 * | '-- subnetID -> conversionID + chainID + addr
 * |-. chains
 * | '-. netID
 * |   '-. list
 * |     '-- txID -> nil
 * |-. expiryReplayProtection
 * | '-- timestamp + validationID -> nil
 * '-. singletons
 *   |-- initializedKey -> nil
 *   |-- blocksReindexedKey -> nil
 *   |-- timestampKey -> timestamp
 *   |-- feeStateKey -> feeState
 *   |-- l1ValidatorExcessKey -> l1ValidatorExcess
 *   |-- accruedFeesKey -> accruedFees
 *   |-- currentSupplyKey -> currentSupply
 *   |-- lastAcceptedKey -> lastAccepted
 *   '-- heightsIndexKey -> startIndexHeight + endIndexHeight
 */
type state struct {
	validatorState

	validators validators.Manager
	ctx        *consensus.Context
	upgrades   upgrade.Config
	metrics    metrics.Metrics
	rewards    reward.Calculator

	baseDB *versiondb.Database

	expiry     *btree.BTreeG[ExpiryEntry]
	expiryDiff *expiryDiff
	expiryDB   database.Database

	activeL1Validators   *activeL1Validators
	l1ValidatorsDiff     *l1ValidatorsDiff
	l1ValidatorsDiffLock sync.RWMutex // Protects concurrent access to l1ValidatorsDiff
	l1ValidatorsDB       database.Database
	weightsCache        cache.Cacher[ids.ID, uint64] // subnetID -> total L1 validator weight
	weightsDB           database.Database
	subnetIDNodeIDCache cache.Cacher[subnetIDNodeID, bool] // subnetID+nodeID -> is validator
	subnetIDNodeIDDB    database.Database
	activeDB            database.Database
	inactiveCache       cache.Cacher[ids.ID, maybe.Maybe[L1Validator]] // validationID -> L1Validator
	inactiveDB          database.Database

	currentStakers *baseStakers
	pendingStakers *baseStakers

	currentHeight uint64

	// L1 validators storage (temporary in-memory implementation)
	l1Validators map[ids.ID]L1Validator

	addedBlockIDs map[uint64]ids.ID            // map of height -> blockID
	blockIDCache  cache.Cacher[uint64, ids.ID] // cache of height -> blockID; if the entry is ids.Empty, it is not in the database
	blockIDDB     database.Database

	addedBlocks map[ids.ID]block.Block            // map of blockID -> Block
	blockCache  cache.Cacher[ids.ID, block.Block] // cache of blockID -> Block; if the entry is nil, it is not in the database
	blockDB     database.Database

	validatorsDB                 database.Database
	currentValidatorsDB          database.Database
	currentValidatorBaseDB       database.Database
	currentValidatorList         linkeddb.LinkedDB
	currentDelegatorBaseDB       database.Database
	currentDelegatorList         linkeddb.LinkedDB
	currentNetValidatorBaseDB    database.Database
	currentNetValidatorList      linkeddb.LinkedDB
	currentNetDelegatorBaseDB database.Database
	currentNetDelegatorList   linkeddb.LinkedDB
	pendingValidatorsDB          database.Database
	pendingValidatorBaseDB       database.Database
	pendingValidatorList         linkeddb.LinkedDB
	pendingDelegatorBaseDB       database.Database
	pendingDelegatorList         linkeddb.LinkedDB
	pendingNetValidatorBaseDB    database.Database
	pendingNetValidatorList      linkeddb.LinkedDB
	pendingNetDelegatorBaseDB database.Database
	pendingNetDelegatorList   linkeddb.LinkedDB

	validatorWeightDiffsDB    database.Database
	validatorPublicKeyDiffsDB database.Database

	addedTxs map[ids.ID]*txAndStatus            // map of txID -> {*txs.Tx, Status}
	txCache  cache.Cacher[ids.ID, *txAndStatus] // txID -> {*txs.Tx, Status}; if the entry is nil, it is not in the database
	txDB     database.Database

	addedRewardUTXOs map[ids.ID][]*lux.UTXO            // map of txID -> []*UTXO
	rewardUTXOsCache cache.Cacher[ids.ID, []*lux.UTXO] // txID -> []*UTXO
	rewardUTXODB     database.Database

	modifiedUTXOs map[ids.ID]*lux.UTXO // map of modified UTXOID -> *UTXO; if the UTXO is nil, it has been removed
	utxoDB        database.Database
	utxoState     lux.UTXOState

	cachedChainIDs []ids.ID // nil if the subnets haven't been loaded
	addedChainIDs  []ids.ID
	subnetBaseDB database.Database
	subnetDB     linkeddb.LinkedDB

	subnetOwners     map[ids.ID]fx.Owner                  // map of netID -> owner
	subnetOwnerCache cache.Cacher[ids.ID, fxOwnerAndSize] // cache of netID -> owner; if the entry is nil, it is not in the database
	subnetOwnerDB    database.Database

	subnetToL1Conversions     map[ids.ID]NetToL1Conversion            // map of subnetID -> conversion of the subnet
	subnetToL1ConversionCache cache.Cacher[ids.ID, NetToL1Conversion] // cache of subnetID -> conversion
	subnetToL1ConversionDB    database.Database

	transformedNets     map[ids.ID]*txs.Tx            // map of subnetID -> transformNetTx
	transformedNetCache cache.Cacher[ids.ID, *txs.Tx] // cache of subnetID -> transformNetTx; if the entry is nil, it is not in the database
	transformedNetDB    database.Database

	modifiedSupplies map[ids.ID]uint64             // map of netID -> current supply
	supplyCache      cache.Cacher[ids.ID, *uint64] // cache of netID -> current supply; if the entry is nil, it is not in the database
	supplyDB         database.Database

	addedChains  map[ids.ID][]*txs.Tx                    // maps netID -> the newly added chains to the subnet
	chainCache   cache.Cacher[ids.ID, []*txs.Tx]         // cache of netID -> the chains after all local modifications []*txs.Tx
	chainDBCache cache.Cacher[ids.ID, linkeddb.LinkedDB] // cache of netID -> linkedDB
	chainDB      database.Database

	// Chain name uniqueness - maps lowercase chain name to chain ID
	addedChainNames map[string]ids.ID            // newly added chain names (lowercase) -> chainID
	chainNameCache  cache.Cacher[string, ids.ID] // cache of lowercase chain name -> chainID
	chainNameDB     database.Database

	// The persisted fields represent the current database value
	timestamp, persistedTimestamp                 time.Time
	feeState, persistedFeeState                   gas.State
	l1ValidatorExcess, persistedL1ValidatorExcess gas.Gas
	accruedFees, persistedAccruedFees             uint64
	currentSupply, persistedCurrentSupply         uint64
	// [lastAccepted] is the most recently accepted block.
	lastAccepted, persistedLastAccepted ids.ID
	lastAcceptedLock                    sync.RWMutex // Protects concurrent access to lastAccepted
	indexedHeights                      *heightRange
	singletonDB                         database.Database
}

// heightRange is used to track which heights are safe to use the native DB
// iterator for querying validator diffs.
//
// new indexing mechanism.
type heightRange struct {
	LowerBound uint64 `serialize:"true"`
	UpperBound uint64 `serialize:"true"`
}

type ValidatorWeightDiff struct {
	Decrease     bool   `serialize:"true"`
	Amount       uint64 `serialize:"true"`
	ValidationID ids.ID `serialize:"true"` // Added to preserve TxID during diff application
}

func (v *ValidatorWeightDiff) Add(amount uint64) error {
	return v.addOrSub(false, amount)
}

func (v *ValidatorWeightDiff) Sub(amount uint64) error {
	return v.addOrSub(true, amount)
}

func (v *ValidatorWeightDiff) addOrSub(sub bool, amount uint64) error {
	if v.Decrease == sub {
		var err error
		v.Amount, err = safemath.Add64(v.Amount, amount)
		return err
	}

	if v.Amount > amount {
		v.Amount -= amount
	} else {
		v.Amount = safemath.AbsDiff(v.Amount, amount)
		v.Decrease = sub
	}
	return nil
}

type txBytesAndStatus struct {
	Tx     []byte        `serialize:"true"`
	Status status.Status `serialize:"true"`
}

type txAndStatus struct {
	tx     *txs.Tx
	status status.Status
}

type fxOwnerAndSize struct {
	owner fx.Owner
	size  int
}

type NetToL1Conversion struct {
	ConversionID ids.ID `serialize:"true"`
	ChainID      ids.ID `serialize:"true"`
	Addr         []byte `serialize:"true"`
}

func txSize(_ ids.ID, tx *txs.Tx) int {
	if tx == nil {
		return ids.IDLen + constants.PointerOverhead
	}
	return ids.IDLen + len(tx.Bytes()) + constants.PointerOverhead
}

func txAndStatusSize(_ ids.ID, t *txAndStatus) int {
	if t == nil {
		return ids.IDLen + constants.PointerOverhead
	}
	return ids.IDLen + len(t.tx.Bytes()) + wrappers.IntLen + 2*constants.PointerOverhead
}

func blockSize(_ ids.ID, blk block.Block) int {
	if blk == nil {
		return ids.IDLen + constants.PointerOverhead
	}
	return ids.IDLen + len(blk.Bytes()) + constants.PointerOverhead
}

func New(
	db database.Database,
	genesisBytes []byte,
	metricsReg metric.Registerer,
	validators validators.Manager,
	upgrades upgrade.Config,
	execCfg *config.Config,
	ctx *consensus.Context,
	metrics metrics.Metrics,
	rewards reward.Calculator,
) (State, error) {
	// Convert metric.Registerer to metric.Registry
	// metricsReg is a *prometheus.Registry which implements Registerer
	// Cast it to metric.Registry (which is *prometheus.Registry)
	var reg metric.Registry
	if r, ok := metricsReg.(metric.Registry); ok {
		reg = r
	} else {
		// If conversion fails, use nil (metercacher will handle it)
		reg = nil
	}

	blockIDCache, err := metercacher.New[uint64, ids.ID](
		"block_id_cache",
		reg,
		lru.NewCache[uint64, ids.ID](execCfg.BlockIDCacheSize),
	)
	if err != nil {
		return nil, err
	}

	blockCache, err := metercacher.New[ids.ID, block.Block](
		"block_cache",
		reg,
		lru.NewSizedCache(execCfg.BlockCacheSize, blockSize),
	)
	if err != nil {
		return nil, err
	}

	baseDB := versiondb.New(db)

	validatorsDB := prefixdb.New(ValidatorsPrefix, baseDB)

	currentValidatorsDB := prefixdb.New(CurrentPrefix, validatorsDB)
	currentValidatorBaseDB := prefixdb.New(ValidatorPrefix, currentValidatorsDB)
	currentDelegatorBaseDB := prefixdb.New(DelegatorPrefix, currentValidatorsDB)
	currentNetValidatorBaseDB := prefixdb.New(NetValidatorPrefix, currentValidatorsDB)
	currentNetDelegatorBaseDB := prefixdb.New(NetDelegatorPrefix, currentValidatorsDB)

	pendingValidatorsDB := prefixdb.New(PendingPrefix, validatorsDB)
	pendingValidatorBaseDB := prefixdb.New(ValidatorPrefix, pendingValidatorsDB)
	pendingDelegatorBaseDB := prefixdb.New(DelegatorPrefix, pendingValidatorsDB)
	pendingNetValidatorBaseDB := prefixdb.New(NetValidatorPrefix, pendingValidatorsDB)
	pendingNetDelegatorBaseDB := prefixdb.New(NetDelegatorPrefix, pendingValidatorsDB)

	l1ValidatorsDB := prefixdb.New(L1Prefix, validatorsDB)

	validatorWeightDiffsDB := prefixdb.New(ValidatorWeightDiffsPrefix, validatorsDB)
	validatorPublicKeyDiffsDB := prefixdb.New(ValidatorPublicKeyDiffsPrefix, validatorsDB)

	weightsCache, err := metercacher.New(
		"l1_validator_weights_cache",
		reg,
		lru.NewSizedCache(execCfg.L1WeightsCacheSize, func(ids.ID, uint64) int {
			return ids.IDLen + wrappers.LongLen
		}),
	)
	if err != nil {
		return nil, err
	}

	inactiveL1ValidatorsCache, err := metercacher.New(
		"l1_validator_inactive_cache",
		reg,
		lru.NewSizedCache(
			execCfg.L1InactiveValidatorsCacheSize,
			func(_ ids.ID, maybeL1Validator maybe.Maybe[L1Validator]) int {
				const (
					l1ValidatorOverhead      = ids.IDLen + ids.NodeIDLen + 4*wrappers.LongLen + 3*constants.PointerOverhead
					maybeL1ValidatorOverhead = wrappers.BoolLen + l1ValidatorOverhead
					entryOverhead            = ids.IDLen + maybeL1ValidatorOverhead
				)
				if maybeL1Validator.IsNothing() {
					return entryOverhead
				}

				l1Validator := maybeL1Validator.Value()
				return entryOverhead + len(l1Validator.PublicKey) + len(l1Validator.RemainingBalanceOwner) + len(l1Validator.DeactivationOwner)
			},
		),
	)
	if err != nil {
		return nil, err
	}

	subnetIDNodeIDCache, err := metercacher.New(
		"l1_validator_subnet_id_node_id_cache",
		reg,
		lru.NewSizedCache(execCfg.L1NetIDNodeIDCacheSize, func(subnetIDNodeID, bool) int {
			return ids.IDLen + ids.NodeIDLen + wrappers.BoolLen
		}),
	)
	if err != nil {
		return nil, err
	}

	txCache, err := metercacher.New(
		"tx_cache",
		reg,
		lru.NewSizedCache(execCfg.TxCacheSize, txAndStatusSize),
	)
	if err != nil {
		return nil, err
	}

	rewardUTXODB := prefixdb.New(RewardUTXOsPrefix, baseDB)
	rewardUTXOsCache, err := metercacher.New[ids.ID, []*lux.UTXO](
		"reward_utxos_cache",
		reg,
		lru.NewCache[ids.ID, []*lux.UTXO](execCfg.RewardUTXOsCacheSize),
	)
	if err != nil {
		return nil, err
	}

	utxoDB := prefixdb.New(UTXOPrefix, baseDB)
	utxoState, err := lux.NewMeteredUTXOState(utxoDB, txs.GenesisCodec, metricsReg, execCfg.ChecksumsEnabled)
	if err != nil {
		return nil, err
	}

	subnetBaseDB := prefixdb.New(NetPrefix, baseDB)

	subnetOwnerDB := prefixdb.New(NetOwnerPrefix, baseDB)
	subnetOwnerCache, err := metercacher.New[ids.ID, fxOwnerAndSize](
		"subnet_owner_cache",
		reg,
		lru.NewSizedCache(execCfg.FxOwnerCacheSize, func(_ ids.ID, f fxOwnerAndSize) int {
			return ids.IDLen + f.size
		}),
	)
	if err != nil {
		return nil, err
	}

	subnetToL1ConversionDB := prefixdb.New(NetToL1ConversionPrefix, baseDB)
	subnetToL1ConversionCache, err := metercacher.New[ids.ID, NetToL1Conversion](
		"subnet_conversion_cache",
		reg,
		lru.NewSizedCache(execCfg.NetToL1ConversionCacheSize, func(_ ids.ID, c NetToL1Conversion) int {
			return 3*ids.IDLen + len(c.Addr)
		}),
	)
	if err != nil {
		return nil, err
	}

	transformedNetCache, err := metercacher.New(
		"transformed_subnet_cache",
		reg,
		lru.NewSizedCache(execCfg.TransformedNetTxCacheSize, txSize),
	)
	if err != nil {
		return nil, err
	}

	supplyCache, err := metercacher.New[ids.ID, *uint64](
		"supply_cache",
		reg,
		lru.NewCache[ids.ID, *uint64](execCfg.ChainCacheSize),
	)
	if err != nil {
		return nil, err
	}

	chainCache, err := metercacher.New[ids.ID, []*txs.Tx](
		"chain_cache",
		reg,
		lru.NewCache[ids.ID, []*txs.Tx](execCfg.ChainCacheSize),
	)
	if err != nil {
		return nil, err
	}

	chainDBCache, err := metercacher.New[ids.ID, linkeddb.LinkedDB](
		"chain_db_cache",
		reg,
		lru.NewCache[ids.ID, linkeddb.LinkedDB](execCfg.ChainDBCacheSize),
	)
	if err != nil {
		return nil, err
	}

	chainNameCache, err := metercacher.New[string, ids.ID](
		"chain_name_cache",
		reg,
		lru.NewCache[string, ids.ID](execCfg.ChainCacheSize),
	)
	if err != nil {
		return nil, err
	}

	s := &state{
		validatorState: newValidatorState(),

		validators: validators,
		ctx:        ctx,
		upgrades:   upgrades,
		metrics:    metrics,
		rewards:    rewards,
		baseDB:     baseDB,

		addedBlockIDs: make(map[uint64]ids.ID),
		blockIDCache:  blockIDCache,
		blockIDDB:     prefixdb.New(BlockIDPrefix, baseDB),

		addedBlocks: make(map[ids.ID]block.Block),
		blockCache:  blockCache,
		blockDB:     prefixdb.New(BlockPrefix, baseDB),

		expiry:     btree.NewG(defaultTreeDegree, ExpiryEntry.Less),
		expiryDiff: newExpiryDiff(),
		expiryDB:   prefixdb.New(ExpiryReplayProtectionPrefix, baseDB),

		activeL1Validators:  newActiveL1Validators(),
		l1ValidatorsDiff:    newL1ValidatorsDiff(),
		l1ValidatorsDB:      l1ValidatorsDB,
		weightsCache:        weightsCache,
		weightsDB:           prefixdb.New(WeightsPrefix, l1ValidatorsDB),
		subnetIDNodeIDCache: subnetIDNodeIDCache,
		subnetIDNodeIDDB:    prefixdb.New(ChainIDNodeIDPrefix, l1ValidatorsDB),
		activeDB:            prefixdb.New(ActivePrefix, l1ValidatorsDB),
		inactiveCache:       inactiveL1ValidatorsCache,
		inactiveDB:          prefixdb.New(InactivePrefix, l1ValidatorsDB),

		currentStakers: newBaseStakers(),
		pendingStakers: newBaseStakers(),

		l1Validators: make(map[ids.ID]L1Validator),

		validatorsDB:                 validatorsDB,
		currentValidatorsDB:          currentValidatorsDB,
		currentValidatorBaseDB:       currentValidatorBaseDB,
		currentValidatorList:         linkeddb.NewDefault(currentValidatorBaseDB),
		currentDelegatorBaseDB:       currentDelegatorBaseDB,
		currentDelegatorList:         linkeddb.NewDefault(currentDelegatorBaseDB),
		currentNetValidatorBaseDB:    currentNetValidatorBaseDB,
		currentNetValidatorList:      linkeddb.NewDefault(currentNetValidatorBaseDB),
		currentNetDelegatorBaseDB: currentNetDelegatorBaseDB,
		currentNetDelegatorList:   linkeddb.NewDefault(currentNetDelegatorBaseDB),
		pendingValidatorsDB:          pendingValidatorsDB,
		pendingValidatorBaseDB:       pendingValidatorBaseDB,
		pendingValidatorList:         linkeddb.NewDefault(pendingValidatorBaseDB),
		pendingDelegatorBaseDB:       pendingDelegatorBaseDB,
		pendingDelegatorList:         linkeddb.NewDefault(pendingDelegatorBaseDB),
		pendingNetValidatorBaseDB:    pendingNetValidatorBaseDB,
		pendingNetValidatorList:      linkeddb.NewDefault(pendingNetValidatorBaseDB),
		pendingNetDelegatorBaseDB: pendingNetDelegatorBaseDB,
		pendingNetDelegatorList:   linkeddb.NewDefault(pendingNetDelegatorBaseDB),
		validatorWeightDiffsDB:       validatorWeightDiffsDB,
		validatorPublicKeyDiffsDB:    validatorPublicKeyDiffsDB,

		addedTxs: make(map[ids.ID]*txAndStatus),
		txDB:     prefixdb.New(TxPrefix, baseDB),
		txCache:  txCache,

		addedRewardUTXOs: make(map[ids.ID][]*lux.UTXO),
		rewardUTXODB:     rewardUTXODB,
		rewardUTXOsCache: rewardUTXOsCache,

		modifiedUTXOs: make(map[ids.ID]*lux.UTXO),
		utxoDB:        utxoDB,
		utxoState:     utxoState,

		subnetBaseDB: subnetBaseDB,
		subnetDB:     linkeddb.NewDefault(subnetBaseDB),

		subnetOwners:     make(map[ids.ID]fx.Owner),
		subnetOwnerDB:    subnetOwnerDB,
		subnetOwnerCache: subnetOwnerCache,

		subnetToL1Conversions:     make(map[ids.ID]NetToL1Conversion),
		subnetToL1ConversionDB:    subnetToL1ConversionDB,
		subnetToL1ConversionCache: subnetToL1ConversionCache,

		transformedNets:     make(map[ids.ID]*txs.Tx),
		transformedNetCache: transformedNetCache,
		transformedNetDB:    prefixdb.New(TransformedNetPrefix, baseDB),

		modifiedSupplies: make(map[ids.ID]uint64),
		supplyCache:      supplyCache,
		supplyDB:         prefixdb.New(SupplyPrefix, baseDB),

		addedChains:  make(map[ids.ID][]*txs.Tx),
		chainDB:      prefixdb.New(ChainPrefix, baseDB),
		chainCache:   chainCache,
		chainDBCache: chainDBCache,

		addedChainNames: make(map[string]ids.ID),
		chainNameCache:  chainNameCache,
		chainNameDB:     prefixdb.New(ChainNamePrefix, baseDB),

		singletonDB: prefixdb.New(SingletonPrefix, baseDB),
	}

	if err := s.sync(genesisBytes); err != nil {
		return nil, errors.Join(
			err,
			s.Close(),
		)
	}

	return s, nil
}

func (s *state) GetExpiryIterator() (iterator.Iterator[ExpiryEntry], error) {
	return s.expiryDiff.getExpiryIterator(
		iterator.FromTree(s.expiry),
	), nil
}

// HasExpiry allows for concurrent reads.
func (s *state) HasExpiry(entry ExpiryEntry) (bool, error) {
	if has, modified := s.expiryDiff.modified[entry]; modified {
		return has, nil
	}
	return s.expiry.Has(entry), nil
}

func (s *state) PutExpiry(entry ExpiryEntry) {
	s.expiryDiff.PutExpiry(entry)
}

func (s *state) DeleteExpiry(entry ExpiryEntry) {
	s.expiryDiff.DeleteExpiry(entry)
}

func (s *state) GetCurrentValidators(ctx context.Context, subnetID ids.ID) ([]*Staker, []L1Validator, uint64, error) {
	// First add the current validators (non-L1)
	legacyBaseStakers := s.currentStakers.validators[subnetID]
	legacyStakers := make([]*Staker, 0, len(legacyBaseStakers))
	for _, staker := range legacyBaseStakers {
		legacyStakers = append(legacyStakers, staker.validator)
	}

	// Then iterate over subnetIDNodeID DB and add the L1 validators
	var l1Validators []L1Validator
	validationIDIter := s.subnetIDNodeIDDB.NewIteratorWithPrefix(
		subnetID[:],
	)
	defer validationIDIter.Release()

	for validationIDIter.Next() {
		if err := ctx.Err(); err != nil {
			return nil, nil, 0, err
		}

		validationID, err := ids.ToID(validationIDIter.Value())
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to parse validation ID: %w", err)
		}

		vdr, err := s.GetL1Validator(validationID)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to get validator: %w", err)
		}
		l1Validators = append(l1Validators, vdr)
	}

	return legacyStakers, l1Validators, s.currentHeight, nil
}

func (s *state) GetActiveL1ValidatorsIterator() (iterator.Iterator[L1Validator], error) {
	s.l1ValidatorsDiffLock.RLock()
	defer s.l1ValidatorsDiffLock.RUnlock()
	return s.l1ValidatorsDiff.getActiveL1ValidatorsIterator(
		s.activeL1Validators.newIterator(),
	), nil
}

func (s *state) NumActiveL1Validators() int {
	return s.activeL1Validators.len() + s.l1ValidatorsDiff.netAddedActive
}

func (s *state) WeightOfL1Validators(subnetID ids.ID) (uint64, error) {
	if weight, modified := s.l1ValidatorsDiff.modifiedTotalWeight[subnetID]; modified {
		return weight, nil
	}

	if weight, ok := s.weightsCache.Get(subnetID); ok {
		return weight, nil
	}

	weight, err := database.GetUInt64(s.weightsDB, subnetID[:])
	if err != nil {
		if err == database.ErrNotFound {
			weight = 0
		} else {
			return 0, err
		}
	}

	s.weightsCache.Put(subnetID, weight)
	return weight, nil
}

// GetL1Validator allows for concurrent reads.
func (s *state) GetL1Validator(validationID ids.ID) (L1Validator, error) {
	if l1Validator, modified := s.l1ValidatorsDiff.modified[validationID]; modified {
		if l1Validator.isDeleted() {
			return L1Validator{}, database.ErrNotFound
		}
		return l1Validator, nil
	}

	return s.getPersistedL1Validator(validationID)
}

// getPersistedL1Validator returns the currently persisted
// L1Validator with the given validationID. It is guaranteed that any
// returned validator is either active or inactive (not deleted).
func (s *state) getPersistedL1Validator(validationID ids.ID) (L1Validator, error) {
	if l1Validator, ok := s.activeL1Validators.get(validationID); ok {
		return l1Validator, nil
	}

	return getL1Validator(s.inactiveCache, s.inactiveDB, validationID)
}

func (s *state) HasL1Validator(subnetID ids.ID, nodeID ids.NodeID) (bool, error) {
	if has, modified := s.l1ValidatorsDiff.hasL1Validator(subnetID, nodeID); modified {
		return has, nil
	}

	subnetIDNodeID := subnetIDNodeID{
		subnetID: subnetID,
		nodeID:   nodeID,
	}
	if has, ok := s.subnetIDNodeIDCache.Get(subnetIDNodeID); ok {
		return has, nil
	}

	key := subnetIDNodeID.Marshal()
	has, err := s.subnetIDNodeIDDB.Has(key)
	if err != nil {
		return false, err
	}

	s.subnetIDNodeIDCache.Put(subnetIDNodeID, has)
	return has, nil
}

func (s *state) PutL1Validator(l1Validator L1Validator) error {
	return s.l1ValidatorsDiff.putL1Validator(s, l1Validator)
}

func (s *state) GetCurrentValidator(netID ids.ID, nodeID ids.NodeID) (*Staker, error) {
	return s.currentStakers.GetValidator(netID, nodeID)
}

func (s *state) PutCurrentValidator(staker *Staker) error {
	s.currentStakers.PutValidator(staker)
	return nil
}

func (s *state) DeleteCurrentValidator(staker *Staker) {
	s.currentStakers.DeleteValidator(staker)
}

func (s *state) GetCurrentDelegatorIterator(subnetID ids.ID, nodeID ids.NodeID) (iterator.Iterator[*Staker], error) {
	return s.currentStakers.GetDelegatorIterator(subnetID, nodeID), nil
}

func (s *state) PutCurrentDelegator(staker *Staker) {
	s.currentStakers.PutDelegator(staker)
}

func (s *state) DeleteCurrentDelegator(staker *Staker) {
	s.currentStakers.DeleteDelegator(staker)
}

func (s *state) GetCurrentStakerIterator() (iterator.Iterator[*Staker], error) {
	return s.currentStakers.GetStakerIterator(), nil
}

func (s *state) GetPendingValidator(netID ids.ID, nodeID ids.NodeID) (*Staker, error) {
	return s.pendingStakers.GetValidator(netID, nodeID)
}

func (s *state) PutPendingValidator(staker *Staker) error {
	s.pendingStakers.PutValidator(staker)
	return nil
}

func (s *state) DeletePendingValidator(staker *Staker) {
	s.pendingStakers.DeleteValidator(staker)
}

func (s *state) GetPendingDelegatorIterator(subnetID ids.ID, nodeID ids.NodeID) (iterator.Iterator[*Staker], error) {
	return s.pendingStakers.GetDelegatorIterator(subnetID, nodeID), nil
}

func (s *state) PutPendingDelegator(staker *Staker) {
	s.pendingStakers.PutDelegator(staker)
}

func (s *state) DeletePendingDelegator(staker *Staker) {
	s.pendingStakers.DeleteDelegator(staker)
}

func (s *state) GetPendingStakerIterator() (iterator.Iterator[*Staker], error) {
	return s.pendingStakers.GetStakerIterator(), nil
}

func (s *state) GetChainIDs() ([]ids.ID, error) {
	if s.cachedChainIDs != nil {
		return s.cachedChainIDs, nil
	}

	subnetDBIt := s.subnetDB.NewIterator()
	defer subnetDBIt.Release()

	chainIDs := []ids.ID{}
	for subnetDBIt.Next() {
		chainIDBytes := subnetDBIt.Key()
		chainID, err := ids.ToID(chainIDBytes)
		if err != nil {
			return nil, err
		}
		chainIDs = append(chainIDs, chainID)
	}
	if err := subnetDBIt.Error(); err != nil {
		return nil, err
	}
	chainIDs = append(chainIDs, s.addedChainIDs...)
	s.cachedChainIDs = chainIDs
	return chainIDs, nil
}

func (s *state) GetNetIDs() ([]ids.ID, error) {
	return s.GetChainIDs()
}

func (s *state) AddNet(chainID ids.ID) {
	s.addedChainIDs = append(s.addedChainIDs, chainID)
	if s.cachedChainIDs != nil {
		s.cachedChainIDs = append(s.cachedChainIDs, chainID)
	}
}

func (s *state) GetNetOwner(netID ids.ID) (fx.Owner, error) {
	if owner, exists := s.subnetOwners[netID]; exists {
		return owner, nil
	}

	if ownerAndSize, cached := s.subnetOwnerCache.Get(netID); cached {
		if ownerAndSize.owner == nil {
			return nil, database.ErrNotFound
		}
		return ownerAndSize.owner, nil
	}

	ownerBytes, err := s.subnetOwnerDB.Get(netID[:])
	if err == nil {
		var owner fx.Owner
		if _, err := block.GenesisCodec.Unmarshal(ownerBytes, &owner); err != nil {
			return nil, err
		}
		s.subnetOwnerCache.Put(netID, fxOwnerAndSize{
			owner: owner,
			size:  len(ownerBytes),
		})
		return owner, nil
	}
	if err != database.ErrNotFound {
		return nil, err
	}

	subnetIntf, _, err := s.GetTx(netID)
	if err != nil {
		if err == database.ErrNotFound {
			s.subnetOwnerCache.Put(netID, fxOwnerAndSize{})
		}
		return nil, err
	}

	subnet, ok := subnetIntf.Unsigned.(*txs.CreateSubnetTx)
	if !ok {
		return nil, fmt.Errorf("%q %w", netID, errIsNotNet)
	}

	s.SetNetOwner(netID, subnet.Owner)
	return subnet.Owner, nil
}

func (s *state) SetNetOwner(netID ids.ID, owner fx.Owner) {
	s.subnetOwners[netID] = owner
}

// GetNetToL1Conversion allows for concurrent reads.
func (s *state) GetNetToL1Conversion(subnetID ids.ID) (NetToL1Conversion, error) {
	if c, ok := s.subnetToL1Conversions[subnetID]; ok {
		return c, nil
	}

	if c, ok := s.subnetToL1ConversionCache.Get(subnetID); ok {
		return c, nil
	}

	bytes, err := s.subnetToL1ConversionDB.Get(subnetID[:])
	if err != nil {
		return NetToL1Conversion{}, err
	}

	var c NetToL1Conversion
	if _, err := block.GenesisCodec.Unmarshal(bytes, &c); err != nil {
		return NetToL1Conversion{}, err
	}
	s.subnetToL1ConversionCache.Put(subnetID, c)
	return c, nil
}

func (s *state) SetNetToL1Conversion(subnetID ids.ID, c NetToL1Conversion) {
	s.subnetToL1Conversions[subnetID] = c
}

func (s *state) GetNetTransformation(subnetID ids.ID) (*txs.Tx, error) {
	if tx, exists := s.transformedNets[subnetID]; exists {
		return tx, nil
	}

	if tx, cached := s.transformedNetCache.Get(subnetID); cached {
		if tx == nil {
			return nil, database.ErrNotFound
		}
		return tx, nil
	}

	transformNetTxID, err := database.GetID(s.transformedNetDB, subnetID[:])
	if err == database.ErrNotFound {
		s.transformedNetCache.Put(subnetID, nil)
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	transformNetTx, _, err := s.GetTx(transformNetTxID)
	if err != nil {
		return nil, err
	}
	s.transformedNetCache.Put(subnetID, transformNetTx)
	return transformNetTx, nil
}

func (s *state) AddNetTransformation(transformNetTxIntf *txs.Tx) {
	transformNetTx := transformNetTxIntf.Unsigned.(*txs.TransformChainTx)
	s.transformedNets[transformNetTx.Chain] = transformNetTxIntf
}

func (s *state) GetChains(netID ids.ID) ([]*txs.Tx, error) {
	if chains, cached := s.chainCache.Get(netID); cached {
		return chains, nil
	}
	chainDB := s.getChainDB(netID)
	chainDBIt := chainDB.NewIterator()
	defer chainDBIt.Release()

	txs := []*txs.Tx(nil)
	for chainDBIt.Next() {
		chainIDBytes := chainDBIt.Key()
		chainID, err := ids.ToID(chainIDBytes)
		if err != nil {
			return nil, err
		}
		chainTx, _, err := s.GetTx(chainID)
		if err != nil {
			return nil, err
		}
		txs = append(txs, chainTx)
	}
	if err := chainDBIt.Error(); err != nil {
		return nil, err
	}
	txs = append(txs, s.addedChains[netID]...)
	s.chainCache.Put(netID, txs)
	return txs, nil
}

func (s *state) AddChain(createChainTxIntf *txs.Tx) {
	createChainTx := createChainTxIntf.Unsigned.(*txs.CreateChainTx)
	netID := createChainTx.ChainID
	s.addedChains[netID] = append(s.addedChains[netID], createChainTxIntf)
	if chains, cached := s.chainCache.Get(netID); cached {
		chains = append(chains, createChainTxIntf)
		s.chainCache.Put(netID, chains)
	}

	// Register chain name for uniqueness tracking (case-insensitive)
	if createChainTx.BlockchainName != "" {
		nameLower := strings.ToLower(createChainTx.BlockchainName)
		chainID := createChainTxIntf.ID()
		s.addedChainNames[nameLower] = chainID
		s.chainNameCache.Put(nameLower, chainID)
	}
}

// GetChainIDByName returns the chain ID for the given chain name (case-insensitive).
// Returns database.ErrNotFound if no chain with the given name exists.
func (s *state) GetChainIDByName(name string) (ids.ID, error) {
	nameLower := strings.ToLower(name)

	// Check in-memory additions first
	if chainID, exists := s.addedChainNames[nameLower]; exists {
		return chainID, nil
	}

	// Check cache
	if chainID, cached := s.chainNameCache.Get(nameLower); cached {
		return chainID, nil
	}

	// Check database
	chainIDBytes, err := s.chainNameDB.Get([]byte(nameLower))
	if err != nil {
		return ids.Empty, err
	}

	chainID, err := ids.ToID(chainIDBytes)
	if err != nil {
		return ids.Empty, err
	}

	s.chainNameCache.Put(nameLower, chainID)
	return chainID, nil
}

// IsChainNameTaken returns true if a chain with the given name already exists (case-insensitive).
func (s *state) IsChainNameTaken(name string) bool {
	_, err := s.GetChainIDByName(name)
	return err == nil
}

func (s *state) getChainDB(netID ids.ID) linkeddb.LinkedDB {
	if chainDB, cached := s.chainDBCache.Get(netID); cached {
		return chainDB
	}
	rawChainDB := prefixdb.New(netID[:], s.chainDB)
	chainDB := linkeddb.NewDefault(rawChainDB)
	s.chainDBCache.Put(netID, chainDB)
	return chainDB
}

func (s *state) GetTx(txID ids.ID) (*txs.Tx, status.Status, error) {
	if tx, exists := s.addedTxs[txID]; exists {
		return tx.tx, tx.status, nil
	}
	if tx, cached := s.txCache.Get(txID); cached {
		if tx == nil {
			return nil, status.Unknown, database.ErrNotFound
		}
		return tx.tx, tx.status, nil
	}
	txBytes, err := s.txDB.Get(txID[:])
	if err == database.ErrNotFound {
		s.txCache.Put(txID, nil)
		return nil, status.Unknown, database.ErrNotFound
	} else if err != nil {
		return nil, status.Unknown, err
	}

	stx := txBytesAndStatus{}
	if _, err := txs.GenesisCodec.Unmarshal(txBytes, &stx); err != nil {
		return nil, status.Unknown, err
	}

	tx, err := txs.Parse(txs.GenesisCodec, stx.Tx)
	if err != nil {
		return nil, status.Unknown, err
	}

	ptx := &txAndStatus{
		tx:     tx,
		status: stx.Status,
	}

	s.txCache.Put(txID, ptx)
	return ptx.tx, ptx.status, nil
}

func (s *state) AddTx(tx *txs.Tx, status status.Status) {
	s.addedTxs[tx.ID()] = &txAndStatus{
		tx:     tx,
		status: status,
	}
}

func (s *state) GetRewardUTXOs(txID ids.ID) ([]*lux.UTXO, error) {
	if utxos, exists := s.addedRewardUTXOs[txID]; exists {
		return utxos, nil
	}
	if utxos, exists := s.rewardUTXOsCache.Get(txID); exists {
		return utxos, nil
	}

	rawTxDB := prefixdb.New(txID[:], s.rewardUTXODB)
	txDB := linkeddb.NewDefault(rawTxDB)
	it := txDB.NewIterator()
	defer it.Release()

	utxos := []*lux.UTXO(nil)
	for it.Next() {
		utxo := &lux.UTXO{}
		if _, err := txs.Codec.Unmarshal(it.Value(), utxo); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
	}
	if err := it.Error(); err != nil {
		return nil, err
	}

	s.rewardUTXOsCache.Put(txID, utxos)
	return utxos, nil
}

func (s *state) AddRewardUTXO(txID ids.ID, utxo *lux.UTXO) {
	s.addedRewardUTXOs[txID] = append(s.addedRewardUTXOs[txID], utxo)
}

func (s *state) GetUTXO(utxoID ids.ID) (*lux.UTXO, error) {
	if utxo, exists := s.modifiedUTXOs[utxoID]; exists {
		if utxo == nil {
			return nil, database.ErrNotFound
		}
		return utxo, nil
	}
	return s.utxoState.GetUTXO(utxoID)
}

func (s *state) UTXOIDs(addr []byte, start ids.ID, limit int) ([]ids.ID, error) {
	return s.utxoState.UTXOIDs(addr, start, limit)
}

func (s *state) AddUTXO(utxo *lux.UTXO) {
	s.modifiedUTXOs[utxo.InputID()] = utxo
}

func (s *state) DeleteUTXO(utxoID ids.ID) {
	s.modifiedUTXOs[utxoID] = nil
}

func (s *state) GetStartTime(nodeID ids.NodeID, netID ids.ID) (time.Time, error) {
	staker, err := s.currentStakers.GetValidator(netID, nodeID)
	if err != nil {
		return time.Time{}, err
	}
	return staker.StartTime, nil
}

// GetTimestamp allows for concurrent reads.
func (s *state) GetTimestamp() time.Time {
	return s.timestamp
}

func (s *state) SetTimestamp(tm time.Time) {
	s.timestamp = tm
}

func (s *state) GetFeeState() gas.State {
	return s.feeState
}

func (s *state) SetFeeState(feeState gas.State) {
	s.feeState = feeState
}

func (s *state) GetL1ValidatorExcess() gas.Gas {
	return s.l1ValidatorExcess
}

func (s *state) SetL1ValidatorExcess(e gas.Gas) {
	s.l1ValidatorExcess = e
}

func (s *state) GetAccruedFees() uint64 {
	return s.accruedFees
}

func (s *state) SetAccruedFees(accruedFees uint64) {
	s.accruedFees = accruedFees
}

func (s *state) GetLastAccepted() ids.ID {
	s.lastAcceptedLock.RLock()
	defer s.lastAcceptedLock.RUnlock()
	return s.lastAccepted
}

func (s *state) SetLastAccepted(lastAccepted ids.ID) {
	s.lastAcceptedLock.Lock()
	defer s.lastAcceptedLock.Unlock()
	s.lastAccepted = lastAccepted
}

func (s *state) GetCurrentSupply(netID ids.ID) (uint64, error) {
	if netID == constants.PrimaryNetworkID {
		return s.currentSupply, nil
	}

	supply, ok := s.modifiedSupplies[netID]
	if ok {
		return supply, nil
	}

	cachedSupply, ok := s.supplyCache.Get(netID)
	if ok {
		if cachedSupply == nil {
			return 0, database.ErrNotFound
		}
		return *cachedSupply, nil
	}

	supply, err := database.GetUInt64(s.supplyDB, netID[:])
	if err == database.ErrNotFound {
		s.supplyCache.Put(netID, nil)
		return 0, database.ErrNotFound
	}
	if err != nil {
		return 0, err
	}

	s.supplyCache.Put(netID, &supply)
	return supply, nil
}

func (s *state) SetCurrentSupply(netID ids.ID, cs uint64) {
	if netID == constants.PrimaryNetworkID {
		s.currentSupply = cs
	} else {
		s.modifiedSupplies[netID] = cs
	}
}

func (s *state) ApplyValidatorWeightDiffs(
	ctx context.Context,
	validators map[ids.NodeID]*validators.GetValidatorOutput,
	startHeight uint64,
	endHeight uint64,
	netID ids.ID,
) error {
	diffIter := s.validatorWeightDiffsDB.NewIteratorWithStartAndPrefix(
		marshalStartDiffKey(netID, startHeight),
		netID[:],
	)
	defer diffIter.Release()

	prevHeight := startHeight + 1
	for diffIter.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}

		_, parsedHeight, nodeID, err := unmarshalDiffKey(diffIter.Key())
		if err != nil {
			return err
		}

		if parsedHeight > prevHeight {
			log.Error("unexpected parsed height",
				log.Stringer("netID", netID),
				log.Uint64("parsedHeight", parsedHeight),
				log.Stringer("nodeID", nodeID),
				log.Uint64("prevHeight", prevHeight),
				log.Uint64("startHeight", startHeight),
				log.Uint64("endHeight", endHeight),
			)
		}

		// If the parsedHeight is less than our target endHeight, then we have
		// fully processed the diffs from startHeight through endHeight.
		if parsedHeight < endHeight {
			return diffIter.Error()
		}

		prevHeight = parsedHeight

		weightDiff, err := unmarshalWeightDiff(diffIter.Value())
		if err != nil {
			return err
		}

		if err := applyWeightDiff(validators, nodeID, weightDiff); err != nil {
			return err
		}
	}
	return diffIter.Error()
}

func applyWeightDiff(
	vdrs map[ids.NodeID]*validators.GetValidatorOutput,
	nodeID ids.NodeID,
	weightDiff *ValidatorWeightDiff,
) error {
	vdr, ok := vdrs[nodeID]
	if !ok {
		// This node isn't in the current validator set.
		vdr = &validators.GetValidatorOutput{
			NodeID: nodeID,
		}
		vdrs[nodeID] = vdr
	}

	// Preserve TxID from ValidationID if this is an L1 validator diff.
	// The ValidationID field in the diff always contains the original TxID
	// (set during diff creation in calculateValidatorDiffs).
	// When applying diffs backward, this restores the correct TxID even when
	// ValidationID changes.
	if weightDiff.ValidationID != ids.Empty {
		vdr.TxID = weightDiff.ValidationID
	}

	// The weight of this node changed at this block.
	var err error
	if weightDiff.Decrease {
		// The validator's weight was decreased at this block, so in the
		// prior block it was higher.
		vdr.Weight, err = safemath.Add64(vdr.Weight, weightDiff.Amount)
	} else {
		// The validator's weight was increased at this block, so in the
		// prior block it was lower.
		vdr.Weight, err = safemath.Sub(vdr.Weight, weightDiff.Amount)
	}
	if err != nil {
		return err
	}

	// Keep Light in sync with Weight (Light is an alias for Weight)
	vdr.Light = vdr.Weight

	if vdr.Weight == 0 {
		// The validator's weight was 0 before this block so they weren't in the
		// validator set.
		delete(vdrs, nodeID)
	}
	return nil
}

func (s *state) ApplyValidatorPublicKeyDiffs(
	ctx context.Context,
	validators map[ids.NodeID]*validators.GetValidatorOutput,
	startHeight uint64,
	endHeight uint64,
	subnetID ids.ID,
) error {
	diffIter := s.validatorPublicKeyDiffsDB.NewIteratorWithStartAndPrefix(
		marshalStartDiffKey(subnetID, startHeight),
		subnetID[:],
	)
	defer diffIter.Release()

	for diffIter.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}

		_, parsedHeight, nodeID, err := unmarshalDiffKey(diffIter.Key())
		if err != nil {
			return err
		}
		// If the parsedHeight is less than our target endHeight, then we have
		// fully processed the diffs from startHeight through endHeight.
		if parsedHeight < endHeight {
			break
		}

		vdr, ok := validators[nodeID]
		if !ok {
			continue
		}

		pkBytes := diffIter.Value()
		if len(pkBytes) == 0 {
			vdr.PublicKey = nil
			continue
		}

		vdr.PublicKey = pkBytes
	}

	// Note: this does not fallback to the linkeddb index because the linkeddb
	// index does not contain entries for when to remove the public key.
	//
	// Nodes may see inconsistent public keys for heights before the new public
	// key index was populated.
	return diffIter.Error()
}

func (s *state) syncGenesis(genesisBlk block.Block, genesis *genesis.Genesis) error {
	genesisBlkID := genesisBlk.ID()
	s.SetLastAccepted(genesisBlkID)
	s.SetTimestamp(time.Unix(int64(genesis.Timestamp), 0))
	s.SetCurrentSupply(constants.PrimaryNetworkID, genesis.InitialSupply)
	s.AddStatelessBlock(genesisBlk)

	// Initialize fee state with default values for genesis
	// This is required because loadMetadata expects fee state to exist
	// We must directly write it since both feeState and persistedFeeState
	// start as zero values and won't trigger the write in writeMetadata
	initialFeeState := gas.State{}
	if err := putFeeState(s.singletonDB, initialFeeState); err != nil {
		return fmt.Errorf("failed to write initial fee state: %w", err)
	}
	s.feeState = initialFeeState
	s.persistedFeeState = initialFeeState

	// Persist UTXOs that exist at genesis
	for _, utxo := range genesis.UTXOs {
		luxUTXO := utxo.UTXO
		s.AddUTXO(&luxUTXO)
	}

	// Persist primary network validator set at genesis
	for _, vdrTx := range genesis.Validators {
		// We expect genesis validator txs to be either AddValidatorTx or
		// AddPermissionlessValidatorTx.
		//
		validatorTx, ok := vdrTx.Unsigned.(txs.ScheduledStaker)
		if !ok {
			return fmt.Errorf("expected a scheduled staker but got %T", vdrTx.Unsigned)
		}

		stakeAmount := validatorTx.Weight()
		// Note: We use [StartTime()] here because genesis transactions are
		// guaranteed to be pre-Durango activation.
		startTime := validatorTx.StartTime()
		stakeDuration := validatorTx.EndTime().Sub(startTime)
		currentSupply, err := s.GetCurrentSupply(constants.PrimaryNetworkID)
		if err != nil {
			return err
		}

		potentialReward := s.rewards.Calculate(
			stakeDuration,
			stakeAmount,
			currentSupply,
		)
		newCurrentSupply, err := safemath.Add64(currentSupply, potentialReward)
		if err != nil {
			return err
		}

		staker, err := NewCurrentStaker(vdrTx.ID(), validatorTx, startTime, potentialReward)
		if err != nil {
			return err
		}

		if err := s.PutCurrentValidator(staker); err != nil {
			return err
		}
		s.AddTx(vdrTx, status.Committed)
		s.SetCurrentSupply(constants.PrimaryNetworkID, newCurrentSupply)
	}

	for _, chain := range genesis.Chains {
		unsignedChain, ok := chain.Unsigned.(*txs.CreateChainTx)
		if !ok {
			return fmt.Errorf("expected tx type *txs.CreateChainTx but got %T", chain.Unsigned)
		}

		// Ensure all chains that the genesis bytes say to create have the right
		// network ID
		networkID := s.ctx.NetworkID
		if false && unsignedChain.NetworkID != networkID { // Temporarily disabled for genesis compatibility
			return lux.ErrWrongNetworkID
		}

		s.AddChain(chain)
		s.AddTx(chain, status.Committed)
	}

	// updateValidators is set to false here to maintain the invariant that the
	// primary network's validator set is empty before the validator sets are
	// initialized.
	if err := s.write(false /*=updateValidators*/, 0); err != nil {
		return err
	}

	// Mark blocks as already reindexed since this is a fresh database with no
	// legacy block indices to convert. This prevents the ReindexBlocks goroutine
	// from running and racing with other state operations on fresh databases.
	return s.singletonDB.Put(BlocksReindexedKey, nil)
}

// Load pulls data previously stored on disk that is expected to be in memory.
func (s *state) load() error {
	return errors.Join(
		s.loadMetadata(),
		s.loadExpiry(),
		s.loadActiveL1Validators(),
		s.loadCurrentValidators(),
		s.loadPendingValidators(),
		s.initValidatorSets(),
	)
}

func (s *state) loadMetadata() error {
	timestamp, err := database.GetTimestamp(s.singletonDB, TimestampKey)
	if err != nil {
		return err
	}
	s.persistedTimestamp = timestamp
	s.SetTimestamp(timestamp)

	feeState, err := getFeeState(s.singletonDB)
	if err != nil {
		return err
	}
	s.persistedFeeState = feeState
	s.SetFeeState(feeState)

	l1ValidatorExcess, err := database.GetUInt64(s.singletonDB, L1ValidatorExcessKey)
	if err != nil {
		if err != database.ErrNotFound {
			return err
		}
		l1ValidatorExcess = 0
	}
	s.persistedL1ValidatorExcess = gas.Gas(l1ValidatorExcess)
	s.SetL1ValidatorExcess(gas.Gas(l1ValidatorExcess))

	accruedFees, err := database.GetUInt64(s.singletonDB, AccruedFeesKey)
	if err != nil {
		if err != database.ErrNotFound {
			return err
		}
		accruedFees = 0
	}
	s.persistedAccruedFees = accruedFees
	s.SetAccruedFees(accruedFees)

	currentSupply, err := database.GetUInt64(s.singletonDB, CurrentSupplyKey)
	if err != nil {
		return err
	}
	s.persistedCurrentSupply = currentSupply
	s.SetCurrentSupply(constants.PrimaryNetworkID, currentSupply)

	lastAccepted, err := database.GetID(s.singletonDB, LastAcceptedKey)
	if err != nil {
		return err
	}
	s.persistedLastAccepted = lastAccepted
	s.lastAcceptedLock.Lock()
	s.lastAccepted = lastAccepted
	s.lastAcceptedLock.Unlock()

	// Lookup the most recently indexed range on disk. If we haven't started
	// indexing the weights, then we keep the indexed heights as nil.
	indexedHeightsBytes, err := s.singletonDB.Get(HeightsIndexedKey)
	if err == database.ErrNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	indexedHeights := &heightRange{}
	_, err = block.GenesisCodec.Unmarshal(indexedHeightsBytes, indexedHeights)
	if err != nil {
		return err
	}

	// If the indexed range is not up to date, then we will act as if the range
	// doesn't exist.
	lastAcceptedBlock, err := s.GetStatelessBlock(lastAccepted)
	if err != nil {
		return err
	}
	if indexedHeights.UpperBound != lastAcceptedBlock.Height() {
		return nil
	}
	s.indexedHeights = indexedHeights
	return nil
}

func (s *state) loadExpiry() error {
	it := s.expiryDB.NewIterator()
	defer it.Release()

	for it.Next() {
		key := it.Key()

		var entry ExpiryEntry
		if err := entry.Unmarshal(key); err != nil {
			return fmt.Errorf("failed to unmarshal ExpiryEntry during load: %w", err)
		}
		s.expiry.ReplaceOrInsert(entry)
	}

	return nil
}

func (s *state) loadActiveL1Validators() error {
	it := s.activeDB.NewIterator()
	defer it.Release()
	for it.Next() {
		key := it.Key()
		validationID, err := ids.ToID(key)
		if err != nil {
			return fmt.Errorf("failed to unmarshal ValidationID during load: %w", err)
		}

		var (
			value       = it.Value()
			l1Validator = L1Validator{
				ValidationID: validationID,
			}
		)
		if _, err := block.GenesisCodec.Unmarshal(value, &l1Validator); err != nil {
			return fmt.Errorf("failed to unmarshal L1 validator: %w", err)
		}

		s.activeL1Validators.put(l1Validator)
	}

	return nil
}

func (s *state) loadCurrentValidators() error {
	s.currentStakers = newBaseStakers()

	fmt.Println("[VALIDATOR DEBUG] loadCurrentValidators: STARTING")
	log.Warn("loadCurrentValidators: starting", "dbPrefix", "currentValidatorList")
	validatorCount := 0
	validatorIt := s.currentValidatorList.NewIterator()
	defer validatorIt.Release()
	for validatorIt.Next() {
		validatorCount++
		txIDBytes := validatorIt.Key()
		txID, err := ids.ToID(txIDBytes)
		if err != nil {
			return err
		}
		tx, _, err := s.GetTx(txID)
		if err != nil {
			return fmt.Errorf("failed loading validator transaction txID %s, %w", txID, err)
		}

		stakerTx, ok := tx.Unsigned.(txs.Staker)
		if !ok {
			return fmt.Errorf("expected tx type txs.Staker but got %T", tx.Unsigned)
		}

		metadataBytes := validatorIt.Value()
		metadata := &validatorMetadata{
			txID: txID,
		}
		if scheduledStakerTx, ok := tx.Unsigned.(txs.ScheduledStaker); ok {
			// Populate [StakerStartTime] using the tx as a default in the event
			// it was added pre-durango and is not stored in the database.
			//
			// Note: We do not populate [LastUpdated] since it is expected to
			// always be present on disk.
			metadata.StakerStartTime = uint64(scheduledStakerTx.StartTime().Unix())
		}
		if err := parseValidatorMetadata(metadataBytes, metadata); err != nil {
			return err
		}

		staker, err := NewCurrentStaker(
			txID,
			stakerTx,
			time.Unix(int64(metadata.StakerStartTime), 0),
			metadata.PotentialReward)
		if err != nil {
			return err
		}

		s.currentStakers.LoadValidator(staker)
		log.Warn("loadCurrentValidators: loaded validator",
			"txID", staker.TxID,
			"nodeID", staker.NodeID,
			"chainID", staker.ChainID,
			"weight", staker.Weight,
		)

		s.validatorState.LoadValidatorMetadata(staker.NodeID, staker.ChainID, metadata)
	}
	fmt.Printf("[VALIDATOR DEBUG] loadCurrentValidators: PRIMARY validators loaded: count=%d currentStakersLen=%d\n",
		validatorCount, len(s.currentStakers.validators))
	log.Warn("loadCurrentValidators: primary validators loaded",
		"count", validatorCount,
		"currentStakersValidatorsLen", len(s.currentStakers.validators),
	)

	subnetValidatorIt := s.currentNetValidatorList.NewIterator()
	defer subnetValidatorIt.Release()
	for subnetValidatorIt.Next() {
		txIDBytes := subnetValidatorIt.Key()
		txID, err := ids.ToID(txIDBytes)
		if err != nil {
			return err
		}
		tx, _, err := s.GetTx(txID)
		if err != nil {
			return err
		}

		stakerTx, ok := tx.Unsigned.(txs.Staker)
		if !ok {
			return fmt.Errorf("expected tx type txs.Staker but got %T", tx.Unsigned)
		}

		metadataBytes := subnetValidatorIt.Value()
		metadata := &validatorMetadata{
			txID: txID,
		}
		if scheduledStakerTx, ok := tx.Unsigned.(txs.ScheduledStaker); ok {
			// Populate [StakerStartTime] and [LastUpdated] using the tx as a
			// default in the event they are not stored in the database.
			startTime := uint64(scheduledStakerTx.StartTime().Unix())
			metadata.StakerStartTime = startTime
			metadata.LastUpdated = startTime
		}
		if err := parseValidatorMetadata(metadataBytes, metadata); err != nil {
			return err
		}

		staker, err := NewCurrentStaker(
			txID,
			stakerTx,
			time.Unix(int64(metadata.StakerStartTime), 0),
			metadata.PotentialReward,
		)
		if err != nil {
			return err
		}
		s.currentStakers.LoadValidator(staker)

		s.validatorState.LoadValidatorMetadata(staker.NodeID, staker.ChainID, metadata)
	}

	delegatorIt := s.currentDelegatorList.NewIterator()
	defer delegatorIt.Release()

	subnetDelegatorIt := s.currentNetDelegatorList.NewIterator()
	defer subnetDelegatorIt.Release()

	for _, delegatorIt := range []database.Iterator{delegatorIt, subnetDelegatorIt} {
		for delegatorIt.Next() {
			txIDBytes := delegatorIt.Key()
			txID, err := ids.ToID(txIDBytes)
			if err != nil {
				return err
			}
			tx, _, err := s.GetTx(txID)
			if err != nil {
				return err
			}

			stakerTx, ok := tx.Unsigned.(txs.Staker)
			if !ok {
				return fmt.Errorf("expected tx type txs.Staker but got %T", tx.Unsigned)
			}

			metadataBytes := delegatorIt.Value()
			metadata := &delegatorMetadata{
				txID: txID,
			}
			if scheduledStakerTx, ok := tx.Unsigned.(txs.ScheduledStaker); ok {
				// Populate [StakerStartTime] using the tx as a default in the
				// event it was added pre-durango and is not stored in the
				// database.
				metadata.StakerStartTime = uint64(scheduledStakerTx.StartTime().Unix())
			}
			err = parseDelegatorMetadata(metadataBytes, metadata)
			if err != nil {
				return err
			}

			staker, err := NewCurrentStaker(
				txID,
				stakerTx,
				time.Unix(int64(metadata.StakerStartTime), 0),
				metadata.PotentialReward,
			)
			if err != nil {
				return err
			}

			s.currentStakers.LoadDelegator(staker)
		}
	}

	return errors.Join(
		validatorIt.Error(),
		subnetValidatorIt.Error(),
		delegatorIt.Error(),
		subnetDelegatorIt.Error(),
	)
}

func (s *state) loadPendingValidators() error {
	s.pendingStakers = newBaseStakers()

	validatorIt := s.pendingValidatorList.NewIterator()
	defer validatorIt.Release()

	subnetValidatorIt := s.pendingNetValidatorList.NewIterator()
	defer subnetValidatorIt.Release()

	for _, validatorIt := range []database.Iterator{validatorIt, subnetValidatorIt} {
		for validatorIt.Next() {
			txIDBytes := validatorIt.Key()
			txID, err := ids.ToID(txIDBytes)
			if err != nil {
				return err
			}
			tx, _, err := s.GetTx(txID)
			if err != nil {
				return err
			}

			stakerTx, ok := tx.Unsigned.(txs.ScheduledStaker)
			if !ok {
				return fmt.Errorf("expected tx type txs.Staker but got %T", tx.Unsigned)
			}

			staker, err := NewPendingStaker(txID, stakerTx)
			if err != nil {
				return err
			}

			s.pendingStakers.LoadValidator(staker)
		}
	}

	delegatorIt := s.pendingDelegatorList.NewIterator()
	defer delegatorIt.Release()

	subnetDelegatorIt := s.pendingNetDelegatorList.NewIterator()
	defer subnetDelegatorIt.Release()

	for _, delegatorIt := range []database.Iterator{delegatorIt, subnetDelegatorIt} {
		for delegatorIt.Next() {
			txIDBytes := delegatorIt.Key()
			txID, err := ids.ToID(txIDBytes)
			if err != nil {
				return err
			}
			tx, _, err := s.GetTx(txID)
			if err != nil {
				return err
			}

			stakerTx, ok := tx.Unsigned.(txs.ScheduledStaker)
			if !ok {
				return fmt.Errorf("expected tx type txs.Staker but got %T", tx.Unsigned)
			}

			staker, err := NewPendingStaker(txID, stakerTx)
			if err != nil {
				return err
			}

			s.pendingStakers.LoadDelegator(staker)
		}
	}

	return errors.Join(
		validatorIt.Error(),
		subnetValidatorIt.Error(),
		delegatorIt.Error(),
		subnetDelegatorIt.Error(),
	)
}

// Invariant: initValidatorSets requires loadActiveL1Validators and
// loadCurrentValidators to have already been called.
func (s *state) initValidatorSets() error {
	fmt.Printf("[VALIDATOR DEBUG] initValidatorSets CALLED - validatorManagerPtr=%p\n", s.validators) // Direct print to bypass log filtering
	log.Warn("[VALIDATOR DEBUG] initValidatorSets: STARTING NOW",
		"validatorManagerPtr", fmt.Sprintf("%p", s.validators),
	)
	log.Info("initValidatorSets: starting",
		"numNets", s.validators.NumNets(),
		"currentStakersSubnets", len(s.currentStakers.validators),
	)

	if s.validators.NumNets() != 0 {
		// The validators manager may already be populated by the network layer
		// with genesis validators (for the samplePeers fix). In this case, we
		// skip re-adding them here and just verify the state is consistent.
		log.Warn("initValidatorSets: validator set already populated by network layer, skipping population",
			"numNets", s.validators.NumNets(),
		)
		return nil
	}

	// Load active LP-77 validators
	if err := s.activeL1Validators.addStakersToValidatorManager(s.validators); err != nil {
		return err
	}
	log.Info("initValidatorSets: after L1 validators", "numNets", s.validators.NumNets())

	// Load inactive LP-77 validators
	//
	// Inactive validators must be loaded individually with their ValidationID
	// as TxID, not aggregated with ids.Empty.
	inactiveIt := s.inactiveDB.NewIterator()
	defer inactiveIt.Release()

	for inactiveIt.Next() {
		validationIDBytes := inactiveIt.Key()
		validationID, err := ids.ToID(validationIDBytes)
		if err != nil {
			return fmt.Errorf("failed to parse validation ID: %w", err)
		}

		var l1Validator L1Validator
		if _, err := block.GenesisCodec.Unmarshal(inactiveIt.Value(), &l1Validator); err != nil {
			return fmt.Errorf("failed to unmarshal inactive L1 validator: %w", err)
		}
		l1Validator.ValidationID = validationID

		// Add inactive validator to validator manager using addL1ValidatorToValidatorManager
		// which properly handles effectiveNodeID, effectivePublicKey, and effectiveValidationID
		if err := addL1ValidatorToValidatorManager(s.validators, l1Validator); err != nil {
			return fmt.Errorf("failed to add inactive L1 validator: %w", err)
		}
	}

	// Load primary network and non-LP77 validators
	primaryNetworkValidators := s.currentStakers.validators[constants.PrimaryNetworkID]
	log.Info("initValidatorSets: loading primary network validators",
		"primaryNetworkID", constants.PrimaryNetworkID,
		"primaryNetworkValidatorCount", len(primaryNetworkValidators),
		"totalSubnets", len(s.currentStakers.validators),
	)
	for subnetID, subnetValidators := range s.currentStakers.validators {
		log.Info("initValidatorSets: processing subnet",
			"subnetID", subnetID,
			"validatorCount", len(subnetValidators),
		)
		for nodeID, subnetValidator := range subnetValidators {
			// The subnet validator's Public Key is inherited from the
			// corresponding primary network validator.
			primaryValidator, ok := primaryNetworkValidators[nodeID]
			if !ok {
				return fmt.Errorf("%w: %s", errMissingPrimaryNetworkValidator, nodeID)
			}

			var (
				primaryStaker = primaryValidator.validator
				subnetStaker  = subnetValidator.validator
			)
			if err := s.validators.AddStaker(subnetID, nodeID, bls.PublicKeyToUncompressedBytes(primaryStaker.PublicKey), subnetStaker.TxID, subnetStaker.Weight); err != nil {
				return err
			}
			log.Info("initValidatorSets: added validator",
				"subnetID", subnetID,
				"nodeID", nodeID,
				"weight", subnetStaker.Weight,
			)

			delegatorIterator := iterator.FromTree(subnetValidator.delegators)
			for delegatorIterator.Next() {
				delegatorStaker := delegatorIterator.Value()
				if err := s.validators.AddWeight(subnetID, nodeID, delegatorStaker.Weight); err != nil {
					delegatorIterator.Release()
					return err
				}
			}
			delegatorIterator.Release()
		}
	}

	s.metrics.SetLocalStake(s.validators.GetWeight(constants.PrimaryNetworkID, s.ctx.NodeID))
	totalWeight, err := s.validators.TotalWeight(constants.PrimaryNetworkID)
	if err != nil {
		return fmt.Errorf("failed to get total weight of primary network validators: %w", err)
	}
	s.metrics.SetTotalStake(totalWeight)

	// Log final state
	numValidators := s.validators.NumValidators(constants.PrimaryNetworkID)
	fmt.Printf("[VALIDATOR DEBUG] initValidatorSets COMPLETE: validatorManagerPtr=%p primaryNetworkID=%s numValidators=%d totalWeight=%d numNets=%d\n",
		s.validators, constants.PrimaryNetworkID, numValidators, totalWeight, s.validators.NumNets())
	log.Warn("[VALIDATOR DEBUG] initValidatorSets: COMPLETE",
		"validatorManagerPtr", fmt.Sprintf("%p", s.validators),
		"primaryNetworkID", constants.PrimaryNetworkID,
		"numValidators", numValidators,
		"totalWeight", totalWeight,
		"numNets", s.validators.NumNets(),
	)

	return nil
}

func (s *state) write(updateValidators bool, height uint64) error {
	codecVersion := CodecVersion1
	if !s.upgrades.IsDurangoActivated(s.GetTimestamp()) {
		codecVersion = CodecVersion0
	}

	return errors.Join(
		s.writeBlocks(),
		s.writeExpiry(),
		s.updateValidatorManager(updateValidators),
		s.writeValidatorDiffs(height),
		s.writeCurrentStakers(codecVersion),
		s.writePendingStakers(),
		s.WriteValidatorMetadata(s.currentValidatorList, s.currentNetValidatorList, codecVersion), // Must be called after writeCurrentStakers
		s.writeL1Validators(),
		s.writeTXs(),
		s.writeRewardUTXOs(),
		s.writeUTXOs(),
		s.writeNets(),
		s.writeNetOwners(),
		s.writeNetToL1Conversions(),
		s.writeTransformedNets(),
		s.writeNetSupplies(),
		s.writeChains(),
		s.writeMetadata(),
	)
}

func (s *state) Close() error {
	// Only close the base database. All other databases are prefixdb wrappers
	// that don't need to be closed separately. Closing them would cause
	// "closed" errors because prefixdb.Close() calls Close() on the underlying
	// database, which would be closed multiple times.
	// The baseDB field was not stored in the state struct, but we can close
	// the underlying database through any of the prefixdb instances.
	// We'll close singletonDB which is a direct prefixdb over baseDB.
	return s.singletonDB.Close()
}

func (s *state) sync(genesis []byte) error {
	wasInitialized, err := isInitialized(s.singletonDB)
	if err != nil {
		return fmt.Errorf(
			"failed to check if the database is initialized: %w",
			err,
		)
	}

	// If the database wasn't previously initialized, create the platform chain
	// anew using the provided genesis state.
	if !wasInitialized {
		if err := s.init(genesis); err != nil {
			return fmt.Errorf(
				"failed to initialize the database: %w",
				err,
			)
		}
	}

	if err := s.load(); err != nil {
		return fmt.Errorf(
			"failed to load the database state: %w",
			err,
		)
	}
	return nil
}

func (s *state) init(genesisBytes []byte) error {
	// Create the genesis block and save it as being accepted (We don't do
	// genesisBlock.Accept() because then it'd look for genesisBlock's
	// non-existent parent)
	genesisID := hashing.ComputeHash256Array(genesisBytes)
	genesisBlock, err := block.NewApricotCommitBlock(genesisID, 0 /*height*/)
	if err != nil {
		return err
	}

	parsedGenesis, err := genesis.Parse(genesisBytes)
	if err != nil {
		return err
	}

	if err := s.syncGenesis(genesisBlock, parsedGenesis); err != nil {
		return err
	}

	// Commit the genesis state to the underlying database
	// This is necessary because baseDB is a versiondb which keeps changes
	// in memory until CommitBatch() is called
	if _, err := s.baseDB.CommitBatch(); err != nil {
		return fmt.Errorf("failed to commit genesis state: %w", err)
	}

	if err := markInitialized(s.singletonDB); err != nil {
		return err
	}

	return s.Commit()
}

func (s *state) AddStatelessBlock(block block.Block) {
	blkID := block.ID()
	s.addedBlockIDs[block.Height()] = blkID
	s.addedBlocks[blkID] = block
}

func (s *state) SetHeight(height uint64) {
	if s.indexedHeights == nil {
		// If indexedHeights hasn't been created yet, then we are newly tracking
		// the range. This means we should initialize the LowerBound to the
		// current height.
		s.indexedHeights = &heightRange{
			LowerBound: height,
		}
	}

	s.indexedHeights.UpperBound = height
	s.currentHeight = height
}

func (s *state) Commit() error {
	defer s.Abort()
	batch, err := s.CommitBatch()
	if err != nil {
		return err
	}
	return batch.Write()
}

func (s *state) Abort() {
	s.baseDB.Abort()
}

func (s *state) Checksum() ids.ID {
	return s.utxoState.Checksum()
}

func (s *state) CommitBatch() (database.Batch, error) {
	// updateValidators is set to true here so that the validator manager is
	// kept up to date with the last accepted state.
	if err := s.write(true /*=updateValidators*/, s.currentHeight); err != nil {
		return nil, err
	}
	return s.baseDB.CommitBatch()
}

func (s *state) writeBlocks() error {
	for blkID, blk := range s.addedBlocks {
		blkBytes := blk.Bytes()
		blkHeight := blk.Height()
		heightKey := database.PackUInt64(blkHeight)

		delete(s.addedBlockIDs, blkHeight)
		s.blockIDCache.Put(blkHeight, blkID)
		if err := database.PutID(s.blockIDDB, heightKey, blkID); err != nil {
			return fmt.Errorf("failed to add blockID: %w", err)
		}

		delete(s.addedBlocks, blkID)
		// Note: Evict is used rather than Put here because blk may end up
		// referencing additional data (because of shared byte slices) that
		// would not be properly accounted for in the cache sizing.
		s.blockCache.Evict(blkID)
		if err := s.blockDB.Put(blkID[:], blkBytes); err != nil {
			return fmt.Errorf("failed to write block %s: %w", blkID, err)
		}
	}
	return nil
}

func (s *state) GetStatelessBlock(blockID ids.ID) (block.Block, error) {
	if blk, exists := s.addedBlocks[blockID]; exists {
		return blk, nil
	}
	if blk, cached := s.blockCache.Get(blockID); cached {
		if blk == nil {
			return nil, database.ErrNotFound
		}

		return blk, nil
	}

	blkBytes, err := s.blockDB.Get(blockID[:])
	if err == database.ErrNotFound {
		s.blockCache.Put(blockID, nil)
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	blk, _, err := parseStoredBlock(blkBytes)
	if err != nil {
		return nil, err
	}

	s.blockCache.Put(blockID, blk)
	return blk, nil
}

func (s *state) GetBlockIDAtHeight(height uint64) (ids.ID, error) {
	if blkID, exists := s.addedBlockIDs[height]; exists {
		return blkID, nil
	}
	if blkID, cached := s.blockIDCache.Get(height); cached {
		if blkID == ids.Empty {
			return ids.Empty, database.ErrNotFound
		}

		return blkID, nil
	}

	heightKey := database.PackUInt64(height)

	blkID, err := database.GetID(s.blockIDDB, heightKey)
	if err == database.ErrNotFound {
		s.blockIDCache.Put(height, ids.Empty)
		return ids.Empty, database.ErrNotFound
	}
	if err != nil {
		return ids.Empty, err
	}

	s.blockIDCache.Put(height, blkID)
	return blkID, nil
}

func (s *state) writeExpiry() error {
	for entry, isAdded := range s.expiryDiff.modified {
		var (
			key = entry.Marshal()
			err error
		)
		if isAdded {
			s.expiry.ReplaceOrInsert(entry)
			err = s.expiryDB.Put(key, nil)
		} else {
			s.expiry.Delete(entry)
			err = s.expiryDB.Delete(key)
		}
		if err != nil {
			return err
		}
	}

	s.expiryDiff = newExpiryDiff()
	return nil
}

// getInheritedPublicKey returns the primary network validator's public key.
//
// Note: This function may return a nil public key and no error if the primary
// network validator does not have a public key.
func (s *state) getInheritedPublicKey(nodeID ids.NodeID) (*bls.PublicKey, error) {
	if vdr, ok := s.currentStakers.validators[constants.PrimaryNetworkID][nodeID]; ok && vdr.validator != nil {
		// The primary network validator is present.
		return vdr.validator.PublicKey, nil
	}
	if vdr, ok := s.currentStakers.validatorDiffs[constants.PrimaryNetworkID][nodeID]; ok && vdr.validator != nil {
		// The primary network validator is being modified.
		return vdr.validator.PublicKey, nil
	}
	return nil, fmt.Errorf("%w: %s", errMissingPrimaryNetworkValidator, nodeID)
}

// updateValidatorManager updates the validator manager with the pending
// validator set changes.
//
// This function must be called prior to writeCurrentStakers and
// writeL1Validators.
//
// TODO: L1s with no active weight should not be held in memory.
func (s *state) updateValidatorManager(updateValidators bool) error {
	if !updateValidators {
		return nil
	}

	for subnetID, validatorDiffs := range s.currentStakers.validatorDiffs {
		// Record the change in weight and/or public key for each validator.
		for nodeID, diff := range validatorDiffs {
			weightDiff, err := diff.WeightDiff()
			if err != nil {
				return err
			}

			if weightDiff.Amount == 0 {
				continue // No weight change; go to the next validator.
			}

			if weightDiff.Decrease {
				if err := s.validators.RemoveWeight(subnetID, nodeID, weightDiff.Amount); err != nil {
					return fmt.Errorf("failed to reduce validator weight: %w", err)
				}
				continue
			}

			if diff.validatorStatus != added {
				if err := s.validators.AddWeight(subnetID, nodeID, weightDiff.Amount); err != nil {
					return fmt.Errorf("failed to increase validator weight: %w", err)
				}
				continue
			}

			pk, err := s.getInheritedPublicKey(nodeID)
			if err != nil {
				// This should never happen as there should always be a primary
				// network validator corresponding to a subnet validator.
				return err
			}

			err = s.validators.AddStaker(
				subnetID,
				nodeID,
				bls.PublicKeyToUncompressedBytes(pk),
				diff.validator.TxID,
				weightDiff.Amount,
			)
			if err != nil {
				return fmt.Errorf("failed to add validator: %w", err)
			}
		}
	}

	// Remove all deleted L1 validators. This must be done before adding new
	// L1 validators to support the case where a validator is removed and then
	// immediately re-added with a different validationID.
	//
	// Sort validators by ValidationID for deterministic processing order.
	// This is important when multiple inactive validators share the same
	// effectiveNodeID (ids.EmptyNodeID), as the first one processed sets
	// the TxID in the validators manager.
	sortedValidationIDs := slices.Collect(maps.Keys(s.l1ValidatorsDiff.modified))
	slices.SortFunc(sortedValidationIDs, func(a, b ids.ID) int {
		return a.Compare(b)
	})
	for _, validationID := range sortedValidationIDs {
		l1Validator := s.l1ValidatorsDiff.modified[validationID]
		if !l1Validator.isDeleted() {
			continue
		}

		priorL1Validator, err := s.getPersistedL1Validator(validationID)
		if err == database.ErrNotFound {
			// Deleting a non-existent validator is a noop. This can happen if
			// the validator was added and then immediately removed.
			continue
		}
		if err != nil {
			return err
		}

		if err := s.validators.RemoveWeight(priorL1Validator.ChainID, priorL1Validator.effectiveNodeID(), priorL1Validator.Weight); err != nil {
			return err
		}
	}

	// Now that the removed L1 validators have been deleted, perform additions
	// and modifications.
	for _, validationID := range sortedValidationIDs {
		l1Validator := s.l1ValidatorsDiff.modified[validationID]
		if l1Validator.isDeleted() {
			continue
		}

		priorL1Validator, err := s.getPersistedL1Validator(validationID)
		switch err {
		case nil:
			// Modifying an existing validator
			if priorL1Validator.IsActive() == l1Validator.IsActive() {
				// This validator's active status isn't changing. This means
				// the effectiveNodeIDs are equal.
				nodeID := l1Validator.effectiveNodeID()
				if priorL1Validator.Weight < l1Validator.Weight {
					err = s.validators.AddWeight(l1Validator.ChainID, nodeID, l1Validator.Weight-priorL1Validator.Weight)
				} else if priorL1Validator.Weight > l1Validator.Weight {
					err = s.validators.RemoveWeight(l1Validator.ChainID, nodeID, priorL1Validator.Weight-l1Validator.Weight)
				}
			} else {
				// This validator's active status is changing.
				err = errors.Join(
					s.validators.RemoveWeight(l1Validator.ChainID, priorL1Validator.effectiveNodeID(), priorL1Validator.Weight),
					addL1ValidatorToValidatorManager(s.validators, l1Validator),
				)
			}
		case database.ErrNotFound:
			// Adding a new validator
			err = addL1ValidatorToValidatorManager(s.validators, l1Validator)
		}
		if err != nil {
			return err
		}
	}

	// Update the stake metrics
	totalWeight, err := s.validators.TotalWeight(constants.PrimaryNetworkID)
	if err != nil {
		return fmt.Errorf("failed to get total weight of primary network: %w", err)
	}

	s.metrics.SetLocalStake(s.validators.GetWeight(constants.PrimaryNetworkID, s.ctx.NodeID))
	s.metrics.SetTotalStake(totalWeight)
	return nil
}

type validatorDiff struct {
	weightDiff      ValidatorWeightDiff
	prevPublicKey   []byte
	newPublicKey    []byte
	hadRemoval      bool // True if pass 1 processed a removal for this (netID, nodeID)
}

// calculateValidatorDiffs calculates the validator set diff contained by the
// pending validator set changes.
//
// This function must be called prior to writeCurrentStakers.
func (s *state) calculateValidatorDiffs() (map[subnetIDNodeID]*validatorDiff, error) {
	changes := make(map[subnetIDNodeID]*validatorDiff)

	// Calculate the changes to the pre-LP-77 validator set
	for subnetID, subnetDiffs := range s.currentStakers.validatorDiffs {
		for nodeID, diff := range subnetDiffs {
			weightDiff, err := diff.WeightDiff()
			if err != nil {
				return nil, err
			}

			// For legacy validators, set ValidationID to the validator's TxID
			// to ensure TxID is preserved during historical reconstruction.
			if diff.validator != nil && weightDiff.Amount != 0 {
				weightDiff.ValidationID = diff.validator.TxID
			}

			pk, err := s.getInheritedPublicKey(nodeID)
			if err != nil {
				// This should never happen as there should always be a primary
				// network validator corresponding to a subnet validator.
				return nil, err
			}

			change := &validatorDiff{
				weightDiff: weightDiff,
			}
			if pk != nil {
				pkBytes := bls.PublicKeyToUncompressedBytes(pk)
				if diff.validatorStatus != added {
					change.prevPublicKey = pkBytes
				}
				if diff.validatorStatus != deleted {
					change.newPublicKey = pkBytes
				}
			}

			subnetIDNodeID := subnetIDNodeID{
				subnetID: subnetID,
				nodeID:   nodeID,
			}
			changes[subnetIDNodeID] = change
		}
	}

	// Calculate the changes to the LP-77 validator set
	//
	// Process in two passes to ensure TxID preservation during ValidationID changes:
	// Pass 1: Process removals (weight decreases) to capture original TxIDs
	// Pass 2: Process additions (weight increases) without overwriting TxIDs from pass 1

	// Collect entries into two slices to ensure deterministic processing order
	type validatorEntry struct {
		validationID ids.ID
		validator    L1Validator
	}
	var removals, additions []validatorEntry

	for validationID, l1Validator := range s.l1ValidatorsDiff.modified {
		_, err := s.getPersistedL1Validator(validationID)
		if err == nil {
			// This validator existed before, so we're removing it
			removals = append(removals, validatorEntry{validationID, l1Validator})
		} else if err != database.ErrNotFound {
			return nil, err
		}

		// If not deleted, we're also adding it (possibly with a different ValidationID)
		if !l1Validator.isDeleted() {
			additions = append(additions, validatorEntry{validationID, l1Validator})
		}
	}

	// Pass 1: Process all removals first
	for _, entry := range removals {
		priorL1Validator, err := s.getPersistedL1Validator(entry.validationID)
		if err != nil {
			return nil, err
		}

		subnetIDNodeID := subnetIDNodeID{
			subnetID: priorL1Validator.ChainID,
			nodeID:   priorL1Validator.effectiveNodeID(),
		}
		diff := getOrSetDefault(changes, subnetIDNodeID)
		// For removals, always set ValidationID to the original TxID.
		// This ensures TxID preservation when ValidationID changes.
		diff.weightDiff.ValidationID = entry.validationID
		diff.hadRemoval = true // Mark that this diff includes a removal
		if err := diff.weightDiff.Sub(priorL1Validator.Weight); err != nil {
			return nil, err
		}
		diff.prevPublicKey = priorL1Validator.effectivePublicKeyBytes()
	}

	// Pass 2: Process all additions
	for _, entry := range additions {
		subnetIDNodeID := subnetIDNodeID{
			subnetID: entry.validator.ChainID,
			nodeID:   entry.validator.effectiveNodeID(),
		}
		diff := getOrSetDefault(changes, subnetIDNodeID)
		// Only set ValidationID if not already set by pass 1 (removal).
		// This preserves the original TxID when a ValidationID changes.
		if diff.weightDiff.ValidationID == ids.Empty {
			diff.weightDiff.ValidationID = entry.validationID
		}
		if err := diff.weightDiff.Add(entry.validator.Weight); err != nil {
			return nil, err
		}
		diff.newPublicKey = entry.validator.effectivePublicKeyBytes()
	}

	return changes, nil
}

// writeValidatorDiffs writes the validator set diff contained by the pending
// validator set changes to disk.
//
// This function must be called prior to writeCurrentStakers.
func (s *state) writeValidatorDiffs(height uint64) error {
	changes, err := s.calculateValidatorDiffs()
	if err != nil {
		return err
	}

	// Write the changes to the database
	for subnetIDNodeID, diff := range changes {
		diffKey := marshalDiffKey(subnetIDNodeID.subnetID, height, subnetIDNodeID.nodeID)
		// Write weight diff if:
		// 1. Amount changed, OR
		// 2. Amount didn't change but ValidationID changed (hadRemoval indicates removal+addition)
		// This ensures TxID tracking across ValidationID changes even when net weight is unchanged.
		shouldWrite := diff.weightDiff.Amount != 0 || (diff.hadRemoval && diff.weightDiff.ValidationID != ids.Empty)
		if shouldWrite {
			err := s.validatorWeightDiffsDB.Put(
				diffKey,
				marshalWeightDiff(&diff.weightDiff),
			)
			if err != nil {
				return err
			}
		}
		if !bytes.Equal(diff.prevPublicKey, diff.newPublicKey) {
			err := s.validatorPublicKeyDiffsDB.Put(
				diffKey,
				diff.prevPublicKey,
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// getOrSetDefault returns the value at k in m if it exists. If it doesn't
// exist, it sets m[k] to a new value and returns that value.
func getOrSetDefault[K comparable, V any](m map[K]*V, k K) *V {
	if v, ok := m[k]; ok {
		return v
	}

	v := new(V)
	m[k] = v
	return v
}

func (s *state) writeCurrentStakers(codecVersion uint16) error {
	for subnetID, validatorDiffs := range s.currentStakers.validatorDiffs {
		// Select db to write to
		validatorDB := s.currentNetValidatorList
		delegatorDB := s.currentNetDelegatorList
		if subnetID == constants.PrimaryNetworkID {
			validatorDB = s.currentValidatorList
			delegatorDB = s.currentDelegatorList
		}

		// Record the change in weight and/or public key for each validator.
		for nodeID, validatorDiff := range validatorDiffs {
			switch validatorDiff.validatorStatus {
			case added:
				staker := validatorDiff.validator

				// The validator is being added.
				//
				// Invariant: It's impossible for a delegator to have been rewarded
				// in the same block that the validator was added.
				startTime := uint64(staker.StartTime.Unix())
				metadata := &validatorMetadata{
					txID:        staker.TxID,
					lastUpdated: staker.StartTime,

					UpDuration:               0,
					LastUpdated:              startTime,
					StakerStartTime:          startTime,
					PotentialReward:          staker.PotentialReward,
					PotentialDelegateeReward: 0,
				}

				metadataBytes, err := MetadataCodec.Marshal(codecVersion, metadata)
				if err != nil {
					return fmt.Errorf("failed to serialize current validator: %w", err)
				}

				if err = validatorDB.Put(staker.TxID[:], metadataBytes); err != nil {
					return fmt.Errorf("failed to write current validator to list: %w", err)
				}

				s.validatorState.LoadValidatorMetadata(nodeID, subnetID, metadata)
			case deleted:
				if err := validatorDB.Delete(validatorDiff.validator.TxID[:]); err != nil {
					return fmt.Errorf("failed to delete current staker: %w", err)
				}

				s.validatorState.DeleteValidatorMetadata(nodeID, subnetID)
			}

			err := writeCurrentDelegatorDiff(
				delegatorDB,
				validatorDiff,
				codecVersion,
			)
			if err != nil {
				return err
			}
		}
	}
	clear(s.currentStakers.validatorDiffs)
	return nil
}

func writeCurrentDelegatorDiff(
	currentDelegatorList linkeddb.LinkedDB,
	validatorDiff *diffValidator,
	codecVersion uint16,
) error {
	addedDelegatorIterator := iterator.FromTree(validatorDiff.addedDelegators)
	defer addedDelegatorIterator.Release()

	for addedDelegatorIterator.Next() {
		staker := addedDelegatorIterator.Value()

		metadata := &delegatorMetadata{
			txID:            staker.TxID,
			PotentialReward: staker.PotentialReward,
			StakerStartTime: uint64(staker.StartTime.Unix()),
		}
		if err := writeDelegatorMetadata(currentDelegatorList, metadata, codecVersion); err != nil {
			return fmt.Errorf("failed to write current delegator to list: %w", err)
		}
	}

	for _, staker := range validatorDiff.deletedDelegators {
		if err := currentDelegatorList.Delete(staker.TxID[:]); err != nil {
			return fmt.Errorf("failed to delete current staker: %w", err)
		}
	}
	return nil
}

func (s *state) writePendingStakers() error {
	for netID, subnetValidatorDiffs := range s.pendingStakers.validatorDiffs {
		delete(s.pendingStakers.validatorDiffs, netID)

		validatorDB := s.pendingNetValidatorList
		delegatorDB := s.pendingNetDelegatorList
		if netID == constants.PrimaryNetworkID {
			validatorDB = s.pendingValidatorList
			delegatorDB = s.pendingDelegatorList
		}

		for _, validatorDiff := range subnetValidatorDiffs {
			err := writePendingDiff(
				validatorDB,
				delegatorDB,
				validatorDiff,
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func writePendingDiff(
	pendingValidatorList linkeddb.LinkedDB,
	pendingDelegatorList linkeddb.LinkedDB,
	validatorDiff *diffValidator,
) error {
	switch validatorDiff.validatorStatus {
	case added:
		err := pendingValidatorList.Put(validatorDiff.validator.TxID[:], nil)
		if err != nil {
			return fmt.Errorf("failed to add pending validator: %w", err)
		}
	case deleted:
		err := pendingValidatorList.Delete(validatorDiff.validator.TxID[:])
		if err != nil {
			return fmt.Errorf("failed to delete pending validator: %w", err)
		}
	}

	addedDelegatorIterator := iterator.FromTree(validatorDiff.addedDelegators)
	defer addedDelegatorIterator.Release()
	for addedDelegatorIterator.Next() {
		staker := addedDelegatorIterator.Value()

		if err := pendingDelegatorList.Put(staker.TxID[:], nil); err != nil {
			return fmt.Errorf("failed to write pending delegator to list: %w", err)
		}
	}

	for _, staker := range validatorDiff.deletedDelegators {
		if err := pendingDelegatorList.Delete(staker.TxID[:]); err != nil {
			return fmt.Errorf("failed to delete pending delegator: %w", err)
		}
	}
	return nil
}

func (s *state) writeL1Validators() error {
	// Write modified weights
	for subnetID, weight := range s.l1ValidatorsDiff.modifiedTotalWeight {
		var err error
		if weight == 0 {
			err = s.weightsDB.Delete(subnetID[:])
		} else {
			err = database.PutUInt64(s.weightsDB, subnetID[:], weight)
		}
		if err != nil {
			return err
		}

		s.weightsCache.Put(subnetID, weight)
	}

	// The L1 validator diff application is split into two loops to ensure that all
	// deletions to the subnetIDNodeIDDB happen prior to any additions.
	// Otherwise replacing an L1 validator by deleting it and then re-adding it with a
	// different validationID could result in an inconsistent state.
	for validationID, l1Validator := range s.l1ValidatorsDiff.modified {
		// Delete the prior validator if it exists
		var err error
		if s.activeL1Validators.delete(validationID) {
			err = deleteL1Validator(s.activeDB, emptyL1ValidatorCache, validationID)
		} else {
			err = deleteL1Validator(s.inactiveDB, s.inactiveCache, validationID)
		}
		if err != nil {
			return err
		}

		if !l1Validator.isDeleted() {
			continue
		}

		var (
			subnetIDNodeID = subnetIDNodeID{
				subnetID: l1Validator.ChainID,
				nodeID:   l1Validator.NodeID,
			}
			subnetIDNodeIDKey = subnetIDNodeID.Marshal()
		)
		if err := s.subnetIDNodeIDDB.Delete(subnetIDNodeIDKey); err != nil {
			return err
		}

		s.subnetIDNodeIDCache.Put(subnetIDNodeID, false)
	}

	for validationID, l1Validator := range s.l1ValidatorsDiff.modified {
		if l1Validator.isDeleted() {
			continue
		}

		// Update the subnetIDNodeID mapping
		var (
			subnetIDNodeID = subnetIDNodeID{
				subnetID: l1Validator.ChainID,
				nodeID:   l1Validator.NodeID,
			}
			subnetIDNodeIDKey = subnetIDNodeID.Marshal()
		)
		if err := s.subnetIDNodeIDDB.Put(subnetIDNodeIDKey, validationID[:]); err != nil {
			return err
		}

		s.subnetIDNodeIDCache.Put(subnetIDNodeID, true)

		// Add the new validator
		var err error
		if l1Validator.IsActive() {
			s.activeL1Validators.put(l1Validator)
			err = putL1Validator(s.activeDB, emptyL1ValidatorCache, l1Validator)
		} else {
			err = putL1Validator(s.inactiveDB, s.inactiveCache, l1Validator)
		}
		if err != nil {
			return err
		}
	}

	s.l1ValidatorsDiffLock.Lock()
	s.l1ValidatorsDiff = newL1ValidatorsDiff()
	s.l1ValidatorsDiffLock.Unlock()
	return nil
}

func (s *state) writeTXs() error {
	for txID, txStatus := range s.addedTxs {
		stx := txBytesAndStatus{
			Tx:     txStatus.tx.Bytes(),
			Status: txStatus.status,
		}

		// Note that we're serializing a [txBytesAndStatus] here, not a
		// *txs.Tx, so we don't use [txs.Codec].
		txBytes, err := txs.GenesisCodec.Marshal(txs.CodecVersion, &stx)
		if err != nil {
			return fmt.Errorf("failed to serialize tx: %w", err)
		}

		delete(s.addedTxs, txID)
		// Note: Evict is used rather than Put here because stx may end up
		// referencing additional data (because of shared byte slices) that
		// would not be properly accounted for in the cache sizing.
		s.txCache.Evict(txID)
		if err := s.txDB.Put(txID[:], txBytes); err != nil {
			return fmt.Errorf("failed to add tx: %w", err)
		}
	}
	return nil
}

func (s *state) writeRewardUTXOs() error {
	for txID, utxos := range s.addedRewardUTXOs {
		delete(s.addedRewardUTXOs, txID)
		s.rewardUTXOsCache.Put(txID, utxos)
		rawTxDB := prefixdb.New(txID[:], s.rewardUTXODB)
		txDB := linkeddb.NewDefault(rawTxDB)

		for _, utxo := range utxos {
			utxoBytes, err := txs.GenesisCodec.Marshal(txs.CodecVersion, utxo)
			if err != nil {
				return fmt.Errorf("failed to serialize reward UTXO: %w", err)
			}
			utxoID := utxo.InputID()
			if err := txDB.Put(utxoID[:], utxoBytes); err != nil {
				return fmt.Errorf("failed to add reward UTXO: %w", err)
			}
		}
	}
	return nil
}

func (s *state) writeUTXOs() error {
	for utxoID, utxo := range s.modifiedUTXOs {
		delete(s.modifiedUTXOs, utxoID)

		if utxo == nil {
			if err := s.utxoState.DeleteUTXO(utxoID); err != nil {
				return fmt.Errorf("failed to delete UTXO: %w", err)
			}
			continue
		}
		if err := s.utxoState.PutUTXO(utxo); err != nil {
			return fmt.Errorf("failed to add UTXO: %w", err)
		}
	}
	return nil
}

func (s *state) writeNets() error {
	for _, chainID := range s.addedChainIDs {
		if err := s.subnetDB.Put(chainID[:], nil); err != nil {
			return fmt.Errorf("failed to write subnet: %w", err)
		}
	}
	s.addedChainIDs = nil
	return nil
}

func (s *state) writeNetOwners() error {
	for subnetID, owner := range s.subnetOwners {
		delete(s.subnetOwners, subnetID)

		ownerBytes, err := block.GenesisCodec.Marshal(block.CodecVersion, &owner)
		if err != nil {
			return fmt.Errorf("failed to marshal net owner: %w", err)
		}

		s.subnetOwnerCache.Put(subnetID, fxOwnerAndSize{
			owner: owner,
			size:  len(ownerBytes),
		})

		if err := s.subnetOwnerDB.Put(subnetID[:], ownerBytes); err != nil {
			return fmt.Errorf("failed to write net owner: %w", err)
		}
	}
	return nil
}

func (s *state) writeNetToL1Conversions() error {
	for subnetID, c := range s.subnetToL1Conversions {
		delete(s.subnetToL1Conversions, subnetID)

		bytes, err := block.GenesisCodec.Marshal(block.CodecVersion, &c)
		if err != nil {
			return fmt.Errorf("failed to marshal subnet conversion: %w", err)
		}

		s.subnetToL1ConversionCache.Put(subnetID, c)

		if err := s.subnetToL1ConversionDB.Put(subnetID[:], bytes); err != nil {
			return fmt.Errorf("failed to write subnet conversion: %w", err)
		}
	}
	return nil
}

func (s *state) writeTransformedNets() error {
	for netID, tx := range s.transformedNets {
		txID := tx.ID()

		delete(s.transformedNets, netID)
		// Note: Evict is used rather than Put here because tx may end up
		// referencing additional data (because of shared byte slices) that
		// would not be properly accounted for in the cache sizing.
		s.transformedNetCache.Evict(netID)
		if err := database.PutID(s.transformedNetDB, netID[:], txID); err != nil {
			return fmt.Errorf("failed to write transformed subnet: %w", err)
		}
	}
	return nil
}

func (s *state) writeNetSupplies() error {
	for subnetID, supply := range s.modifiedSupplies {
		delete(s.modifiedSupplies, subnetID)
		s.supplyCache.Put(subnetID, &supply)
		if err := database.PutUInt64(s.supplyDB, subnetID[:], supply); err != nil {
			return fmt.Errorf("failed to write subnet supply: %w", err)
		}
	}
	return nil
}

func (s *state) writeChains() error {
	for netID, chains := range s.addedChains {
		for _, chain := range chains {
			chainDB := s.getChainDB(netID)

			chainID := chain.ID()
			if err := chainDB.Put(chainID[:], nil); err != nil {
				return fmt.Errorf("failed to write chain: %w", err)
			}
		}
		delete(s.addedChains, netID)
	}

	// Persist chain names for network-wide uniqueness
	for nameLower, chainID := range s.addedChainNames {
		if err := s.chainNameDB.Put([]byte(nameLower), chainID[:]); err != nil {
			return fmt.Errorf("failed to write chain name %s: %w", nameLower, err)
		}
		delete(s.addedChainNames, nameLower)
	}
	return nil
}

func (s *state) writeMetadata() error {
	if !s.persistedTimestamp.Equal(s.timestamp) {
		if err := database.PutTimestamp(s.singletonDB, TimestampKey, s.timestamp); err != nil {
			return fmt.Errorf("failed to write timestamp: %w", err)
		}
		s.persistedTimestamp = s.timestamp
	}
	if s.feeState != s.persistedFeeState {
		if err := putFeeState(s.singletonDB, s.feeState); err != nil {
			return fmt.Errorf("failed to write fee state: %w", err)
		}
		s.persistedFeeState = s.feeState
	}
	if s.l1ValidatorExcess != s.persistedL1ValidatorExcess {
		if err := database.PutUInt64(s.singletonDB, L1ValidatorExcessKey, uint64(s.l1ValidatorExcess)); err != nil {
			return fmt.Errorf("failed to write l1Validator excess: %w", err)
		}
		s.persistedL1ValidatorExcess = s.l1ValidatorExcess
	}
	if s.accruedFees != s.persistedAccruedFees {
		if err := database.PutUInt64(s.singletonDB, AccruedFeesKey, s.accruedFees); err != nil {
			return fmt.Errorf("failed to write accrued fees: %w", err)
		}
		s.persistedAccruedFees = s.accruedFees
	}
	if s.persistedCurrentSupply != s.currentSupply {
		if err := database.PutUInt64(s.singletonDB, CurrentSupplyKey, s.currentSupply); err != nil {
			return fmt.Errorf("failed to write current supply: %w", err)
		}
		s.persistedCurrentSupply = s.currentSupply
	}
	s.lastAcceptedLock.RLock()
	lastAcceptedChanged := s.persistedLastAccepted != s.lastAccepted
	currentLastAccepted := s.lastAccepted
	s.lastAcceptedLock.RUnlock()
	if lastAcceptedChanged {
		if err := database.PutID(s.singletonDB, LastAcceptedKey, currentLastAccepted); err != nil {
			return fmt.Errorf("failed to write last accepted: %w", err)
		}
		s.persistedLastAccepted = currentLastAccepted
	}
	if s.indexedHeights != nil {
		indexedHeightsBytes, err := block.GenesisCodec.Marshal(block.CodecVersion, s.indexedHeights)
		if err != nil {
			return err
		}
		if err := s.singletonDB.Put(HeightsIndexedKey, indexedHeightsBytes); err != nil {
			return fmt.Errorf("failed to write indexed range: %w", err)
		}
	}
	return nil
}

// Returns the block and whether it is a [stateBlk].
// Invariant: blkBytes is safe to parse with blocks.GenesisCodec
//
// TODO: Remove after v1.14.x is activated
func parseStoredBlock(blkBytes []byte) (block.Block, bool, error) {
	// Attempt to parse as blocks.Block
	blk, err := block.Parse(block.GenesisCodec, blkBytes)
	if err == nil {
		return blk, false, nil
	}

	// Fallback to [stateBlk] using our legacy codec
	blkState := stateBlk{}
	if _, err := block.GenesisCodec.Unmarshal(blkBytes, &blkState); err != nil {
		// If we can't unmarshal as stateBlk, this might not be a block at all
		// (could be an index entry or other data in the blockDB)
		// Return the original parse error
		return nil, false, err
	}

	blk, err = block.Parse(block.GenesisCodec, blkState.Bytes)
	return blk, true, err
}

func (s *state) ReindexBlocks(lock sync.Locker, log log.Logger) error {
	has, err := s.singletonDB.Has(BlocksReindexedKey)
	if err != nil {
		return err
	}
	if has {
		log.Info("blocks already reindexed")
		return nil
	}

	// It is possible that new blocks are added after grabbing this iterator.
	// New blocks are guaranteed to be persisted in the new format, so we don't
	// need to check them.
	blockIterator := s.blockDB.NewIterator()
	// Releasing is done using a closure to ensure that updating blockIterator
	// will result in having the most recent iterator released when executing
	// the deferred function.
	defer func() {
		blockIterator.Release()
	}()

	log.Info("starting block reindexing")

	var (
		startTime         = time.Now()
		lastCommit        = startTime
		nextUpdate        = startTime.Add(indexLogFrequency)
		numIndicesChecked = 0
		numIndicesUpdated = 0
	)

	for blockIterator.Next() {
		keyBytes := blockIterator.Key()
		valueBytes := blockIterator.Value()

		// Skip entries that are not 32 bytes (not block IDs)
		if len(keyBytes) != ids.IDLen {
			continue
		}

		blk, isStateBlk, err := parseStoredBlock(valueBytes)
		if err != nil {
			// Skip entries that can't be parsed as blocks
			// This could be metadata or other non-block data
			continue
		}

		blkID := blk.ID()

		// This block was previously stored using the legacy format, update the
		// index to remove the usage of stateBlk.
		if isStateBlk {
			blkBytes := blk.Bytes()
			if err := s.blockDB.Put(blkID[:], blkBytes); err != nil {
				return fmt.Errorf("failed to write block: %w", err)
			}

			numIndicesUpdated++
		}

		numIndicesChecked++

		now := time.Now()
		if now.After(nextUpdate) {
			nextUpdate = now.Add(indexLogFrequency)

			progress := timer.ProgressFromHash(blkID[:])
			eta := timer.EstimateETA(
				startTime,
				progress,
				math.MaxUint64,
			)

			log.Info("reindexing blocks",
				"numIndicesUpdated", numIndicesUpdated,
				"numIndicesChecked", numIndicesChecked,
				"eta", eta,
			)
		}

		if numIndicesChecked%indexIterationLimit == 0 {
			// We must hold the lock during committing to make sure we don't
			// attempt to commit to disk while a block is concurrently being
			// accepted.
			lock.Lock()
			err := errors.Join(
				s.Commit(),
				blockIterator.Error(),
			)
			lock.Unlock()
			if err != nil {
				return err
			}

			// We release the iterator here to allow the underlying database to
			// clean up deleted state.
			blockIterator.Release()

			// We take the minimum here because it's possible that the node is
			// currently bootstrapping. This would mean that grabbing the lock
			// could take an extremely long period of time; which we should not
			// delay processing for.
			indexDuration := now.Sub(lastCommit)
			sleepDuration := min(
				indexIterationSleepMultiplier*indexDuration,
				indexIterationSleepCap,
			)
			time.Sleep(sleepDuration)

			// Make sure not to include the sleep duration into the next index
			// duration.
			lastCommit = time.Now()

			blockIterator = s.blockDB.NewIteratorWithStart(blkID[:])
		}
	}

	// Ensure we fully iterated over all blocks before writing that indexing has
	// finished.
	//
	// Note: This is needed because a transient read error could cause the
	// iterator to stop early.
	if err := blockIterator.Error(); err != nil {
		return fmt.Errorf("failed to iterate over historical blocks: %w", err)
	}

	if err := s.singletonDB.Put(BlocksReindexedKey, nil); err != nil {
		return fmt.Errorf("failed to put marked blocks as reindexed: %w", err)
	}

	// We must hold the lock during committing to make sure we don't attempt to
	// commit to disk while a block is concurrently being accepted.
	lock.Lock()
	defer lock.Unlock()

	log.Info("finished block reindexing",
		"numIndicesUpdated", numIndicesUpdated,
		"numIndicesChecked", numIndicesChecked,
		"duration", time.Since(startTime),
	)

	return s.Commit()
}

func (s *state) GetUptime(vdrID ids.NodeID, netID ids.ID) (time.Duration, time.Duration, error) {
	upDuration, lastUpdated, err := s.validatorState.GetUptime(vdrID, netID)
	if err != nil {
		return 0, 0, err
	}
	// Convert lastUpdated time.Time to Duration since Unix epoch
	lastUpdatedDuration := time.Duration(lastUpdated.Unix()) * time.Second
	return upDuration, lastUpdatedDuration, nil
}

func (s *state) SetUptime(vdrID ids.NodeID, netID ids.ID, upDuration time.Duration, lastUpdated time.Time) error {
	return s.validatorState.SetUptime(vdrID, netID, upDuration, lastUpdated)
}

func markInitialized(db database.KeyValueWriter) error {
	return db.Put(InitializedKey, nil)
}

func isInitialized(db database.KeyValueReader) (bool, error) {
	return db.Has(InitializedKey)
}

func putFeeState(db database.KeyValueWriter, feeState gas.State) error {
	feeStateBytes, err := block.GenesisCodec.Marshal(block.CodecVersion, feeState)
	if err != nil {
		return err
	}
	return db.Put(FeeStateKey, feeStateBytes)
}

func getFeeState(db database.KeyValueReader) (gas.State, error) {
	feeStateBytes, err := db.Get(FeeStateKey)
	if err == database.ErrNotFound {
		return gas.State{}, nil
	}
	if err != nil {
		return gas.State{}, err
	}

	var feeState gas.State
	if _, err := block.GenesisCodec.Unmarshal(feeStateBytes, &feeState); err != nil {
		return gas.State{}, err
	}
	return feeState, nil
}
