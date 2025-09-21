// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/luxfi/node/cache/lru"
	"github.com/luxfi/node/cache/metercacher"
	"github.com/luxfi/node/network/p2p"
	"github.com/luxfi/node/network/p2p/gossip"
	"github.com/luxfi/node/network/p2p/lp118"

	// "github.com/luxfi/firewood-go-ethhash/ffi"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/luxfi/evm/commontype"
	"github.com/luxfi/evm/consensus/dummy"
	"github.com/luxfi/evm/constants"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/evm/core/txpool"
	"github.com/luxfi/evm/eth"
	"github.com/luxfi/evm/eth/ethconfig"
	subnetevmprometheus "github.com/luxfi/evm/metrics/prometheus"
	"github.com/luxfi/evm/miner"
	"github.com/luxfi/evm/network"
	"github.com/luxfi/evm/node"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/evm/params/extras"
	"github.com/luxfi/evm/plugin/evm/config"
	gossipHandler "github.com/luxfi/evm/plugin/evm/gossip"
	subnetevmlog "github.com/luxfi/evm/plugin/evm/log"
	"github.com/luxfi/evm/plugin/evm/message"
	"github.com/luxfi/evm/plugin/evm/validators"
	"github.com/luxfi/evm/plugin/evm/validators/interfaces"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/metrics"
	"github.com/luxfi/geth/triedb"
	"github.com/luxfi/geth/triedb/hashdb"

	warpcontract "github.com/luxfi/evm/precompile/contracts/warp"
	"github.com/luxfi/evm/rpc"
	statesyncclient "github.com/luxfi/evm/sync/client"
	"github.com/luxfi/evm/sync/client/stats"
	"github.com/luxfi/evm/utils"
	"github.com/luxfi/evm/warp"
	nodeWarp "github.com/luxfi/node/vms/platformvm/warp"
	luxWarp "github.com/luxfi/warp"
	"github.com/luxfi/warp/signer"

	// Force-load tracer engine to trigger registration
	//
	// We must import this package (not referenced elsewhere) so that the native "callTracer"
	// is added to a map of client-accessible tracers. In geth, this is done
	// inside of cmd/geth.
	_ "github.com/luxfi/geth/eth/tracers/js"
	_ "github.com/luxfi/geth/eth/tracers/native"

	"github.com/luxfi/evm/precompile/precompileconfig"
	// Force-load precompiles to trigger registration
	_ "github.com/luxfi/evm/precompile/registry"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/rlp"
	"github.com/luxfi/log"

	luxRPC "github.com/gorilla/rpc/v2"

	nodeConsensus "github.com/luxfi/consensus"
	"github.com/luxfi/consensus/engine/chain/block"
	nodeblock "github.com/luxfi/consensus/engine/chain/block"
	consensusInterfaces "github.com/luxfi/consensus/interfaces"
	nodechain "github.com/luxfi/consensus/protocol/chain"
	nodeConsensusChain "github.com/luxfi/node/consensus/chain"
	consensusmockable "github.com/luxfi/consensus/utils/timer/mockable"
	"github.com/luxfi/database/versiondb"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/codec"
	"github.com/luxfi/node/upgrade"
	"github.com/luxfi/node/utils/perms"
	"github.com/luxfi/node/utils/profiler"
	nodemockable "github.com/luxfi/node/utils/timer/mockable"
	"github.com/luxfi/node/utils/units"
	"github.com/luxfi/node/version"
	nodeChain "github.com/luxfi/node/vms/components/chain"

	commonEng "github.com/luxfi/consensus/core"
	"github.com/luxfi/consensus/core/appsender"
	nodeCommonEng "github.com/luxfi/node/consensus/engine/core"
	"github.com/luxfi/math/set"

	"github.com/luxfi/database"
	luxUtils "github.com/luxfi/node/utils"
	luxJSON "github.com/luxfi/node/utils/json"
)

var (
	// Interface compatibility resolved with node v1.16.15
	_ nodeblock.ChainVM                      = (*VM)(nil)
	_ nodeblock.BuildBlockWithContextChainVM = (*VM)(nil)
	_ nodeblock.StateSyncableVM              = (*VM)(nil)
	_ statesyncclient.EthBlockParser         = (*VM)(nil)
)

const (
	// Max time from current time allowed for blocks, before they're considered future blocks
	// and fail verification
	maxFutureBlockTime     = 10 * time.Second
	decidedCacheSize       = 10 * units.MiB
	missingCacheSize       = 50
	unverifiedCacheSize    = 5 * units.MiB
	bytesToIDCacheSize     = 5 * units.MiB
	warpSignatureCacheSize = 500

	// Prefixes for metrics gatherers
	ethMetricsPrefix        = "eth"
	sdkMetricsPrefix        = "sdk"
	chainStateMetricsPrefix = "chain_state"

	// TxGossipHandlerID is the handler ID for transaction gossip
	TxGossipHandlerID = uint64(0x1)
)

// Define the API endpoints for the VM
const (
	adminEndpoint        = "/admin"
	ethRPCEndpoint       = "/rpc"
	ethWSEndpoint        = "/ws"
	validatorsEndpoint   = "/validators"
	ethTxGossipNamespace = "eth_tx_gossip"
)

var (
	// Set last accepted key to be longer than the keys used to store accepted block IDs.
	lastAcceptedKey    = []byte("last_accepted_key")
	acceptedPrefix     = []byte("chain_accepted")
	metadataPrefix     = []byte("metadata")
	warpPrefix         = []byte("warp")
	ethDBPrefix        = []byte("ethdb")
	validatorsDBPrefix = []byte("validators")
)

var (
	errEmptyBlock                    = errors.New("empty block")
	errUnsupportedFXs                = errors.New("unsupported feature extensions")
	errInvalidBlock                  = errors.New("invalid block")
	errInvalidNonce                  = errors.New("invalid nonce")
	errUnclesUnsupported             = errors.New("uncles unsupported")
	errNilBaseFeeSubnetEVM           = errors.New("nil base fee is invalid after subnetEVM")
	errNilBlockGasCostSubnetEVM      = errors.New("nil blockGasCost is invalid after subnetEVM")
	errInvalidHeaderPredicateResults = errors.New("invalid header predicate results")
	errInitializingLogger            = errors.New("failed to initialize logger")
	errShuttingDownVM                = errors.New("shutting down VM")
)

// legacyApiNames maps pre geth v1.10.20 api names to their updated counterparts.
// used in attachEthService for backward configuration compatibility.
var legacyApiNames = map[string]string{
	"internal-public-eth":              "internal-eth",
	"internal-public-blockchain":       "internal-blockchain",
	"internal-public-transaction-pool": "internal-transaction",
	"internal-public-tx-pool":          "internal-tx-pool",
	"internal-public-debug":            "internal-debug",
	"internal-private-debug":           "internal-debug",
	"internal-public-account":          "internal-account",
	"internal-private-personal":        "internal-personal",

	"public-eth":        "eth",
	"public-eth-filter": "eth-filter",
	"private-admin":     "admin",
	"public-debug":      "debug",
	"private-debug":     "debug",
}

// VM implements the chain.ChainVM interface
// warpSignerAdapter adapts a signer.Signer to warp.WarpSigner
type warpSignerAdapter struct {
	signer signer.Signer
	nodeID ids.NodeID
}

func (w *warpSignerAdapter) Sign(msg []byte) ([]byte, error) {
	// Parse the message as an unsigned warp message
	unsignedMsg, err := luxWarp.ParseUnsignedMessage(msg)
	if err != nil {
		return nil, err
	}

	// Sign using the signer
	sig, err := w.signer.Sign(unsignedMsg)
	if err != nil {
		return nil, err
	}

	// Return the signature bytes
	return sig.Bytes(), nil
}

func (w *warpSignerAdapter) PublicKey() []byte {
	pk := w.signer.GetPublicKey()
	if pk == nil {
		return nil
	}
	return pk.Bytes()
}

func (w *warpSignerAdapter) NodeID() ids.NodeID {
	return w.nodeID
}

// appSenderWrapper wraps a consensus AppSender to implement node's AppSender interface
type appSenderWrapper struct {
	appSender appsender.AppSender
}

func (w *appSenderWrapper) SendAppRequest(ctx context.Context, nodeIDs set.Set[ids.NodeID], requestID uint32, appRequestBytes []byte) error {
	return w.appSender.SendAppRequest(ctx, nodeIDs, requestID, appRequestBytes)
}

func (w *appSenderWrapper) SendAppResponse(ctx context.Context, nodeID ids.NodeID, requestID uint32, appResponseBytes []byte) error {
	return w.appSender.SendAppResponse(ctx, nodeID, requestID, appResponseBytes)
}

func (w *appSenderWrapper) SendAppError(ctx context.Context, nodeID ids.NodeID, requestID uint32, errorCode int32, errorMessage string) error {
	return w.appSender.SendAppError(ctx, nodeID, requestID, errorCode, errorMessage)
}

func (w *appSenderWrapper) SendAppGossip(ctx context.Context, nodeIDs set.Set[ids.NodeID], appGossipBytes []byte) error {
	return w.appSender.SendAppGossip(ctx, nodeIDs, appGossipBytes)
}

func (w *appSenderWrapper) SendAppGossipSpecific(ctx context.Context, nodeIDs set.Set[ids.NodeID], appGossipBytes []byte) error {
	return w.appSender.SendAppGossipSpecific(ctx, nodeIDs, appGossipBytes)
}

// SendCrossChainAppError implements node's AppSender interface
func (w *appSenderWrapper) SendCrossChainAppError(ctx context.Context, chainID ids.ID, requestID uint32, errorCode int32, errorMessage string) error {
	// consensus AppSender doesn't have this method, so just return nil
	// Cross-chain app messages are not supported in this VM
	return nil
}

// SendCrossChainAppRequest implements node's AppSender interface
func (w *appSenderWrapper) SendCrossChainAppRequest(ctx context.Context, chainID ids.ID, requestID uint32, appRequestBytes []byte) error {
	// consensus AppSender doesn't have this method, so just return nil
	// Cross-chain app messages are not supported in this VM
	return nil
}

// SendCrossChainAppResponse implements node's AppSender interface
func (w *appSenderWrapper) SendCrossChainAppResponse(ctx context.Context, chainID ids.ID, requestID uint32, appResponseBytes []byte) error {
	// consensus AppSender doesn't have this method, so just return nil
	// Cross-chain app messages are not supported in this VM
	return nil
}

type VM struct {
	ctx      context.Context
	chainCtx *nodeConsensus.Context
	// contextLock is used to coordinate global VM operations.
	// This can be used safely instead of context.Context.Lock which is deprecated and should not be used in rpcchainvm.
	vmLock sync.RWMutex
	// [cancel] may be nil until [consensus.NormalOp] starts
	cancel context.CancelFunc
	// *nodeChain.State helps to implement the VM interface by wrapping blocks
	// with an efficient caching layer.
	*nodeChain.State

	config config.Config

	genesisHash common.Hash
	chainConfig *params.ChainConfig
	ethConfig   ethconfig.Config

	// pointers to eth constructs
	eth        *eth.Ethereum
	txPool     *txpool.TxPool
	blockChain *core.BlockChain
	miner      *miner.Miner

	// [versiondb] is the VM's current versioned database
	versiondb *versiondb.Database

	// [db] is the VM's current database
	db database.Database

	// metadataDB is used to store one off keys.
	metadataDB database.Database

	// [chaindb] is the database supplied to the Ethereum backend
	chaindb ethdb.Database

	usingStandaloneDB bool

	// [acceptedBlockDB] is the database to store the last accepted
	// block.
	acceptedBlockDB database.Database
	// [warpDB] is used to store warp message signatures
	// set to a prefixDB with the prefix [warpPrefix]
	warpDB database.Database

	validatorsDB database.Database

	syntacticBlockValidator BlockValidator

	// builderLock is used to synchronize access to the block builder,
	// as it is uninitialized at first and is only initialized when onNormalOperationsStarted is called.
	builderLock sync.Mutex
	builder     *blockBuilder

	clock          nodemockable.Clock
	consensusClock consensusmockable.Clock

	shutdownChan chan struct{}
	shutdownWg   sync.WaitGroup

	// Continuous Profiler
	profiler profiler.ContinuousProfiler

	network.Network
	networkCodec codec.Manager

	// Metrics
	sdkMetrics *prometheus.Registry

	bootstrapped luxUtils.Atomic[bool]

	stateSyncDone chan struct{}

	logger subnetevmlog.Logger
	// State sync server and client
	StateSyncServer
	StateSyncClient

	// Lux Warp Messaging backend
	// Used to serve BLS signatures of warp messages over RPC
	warpBackend warp.Backend

	// Initialize only sets these if nil so they can be overridden in tests
	p2pValidators      *p2p.Validators
	ethTxGossipHandler p2p.Handler
	ethTxPushGossiper  luxUtils.Atomic[*gossip.PushGossiper[*GossipEthTx]]
	ethTxPullGossiper  gossip.Gossiper

	validatorsManager interfaces.Manager

	chainAlias string
	// RPC handlers (should be stopped before closing chaindb)
	rpcHandlers []interface{ Stop() }
}

// ParseBlock implements nodeblock.ChainVM interface
func (vm *VM) ParseBlock(ctx context.Context, b []byte) (nodeblock.Block, error) {
	// Call the embedded State's ParseBlock and convert the result
	blk, err := vm.State.ParseBlock(ctx, b)
	if err != nil {
		return nil, err
	}
	// Adapt the consensus block to node block interface
	return NewBlockAdapter(blk.(nodechain.Block)), nil
}

// Initialize implements the chain.ChainVM interface with generic interface{} parameters
func (vm *VM) Initialize(
	ctx context.Context,
	chainCtx interface{},
	dbManager interface{},
	genesisBytes []byte,
	upgradeBytes []byte,
	configBytes []byte,
	toEngine interface{},
	fxs []interface{},
	appSender interface{},
) error {
	// Convert interface{} parameters to strongly typed ones
	typedChainCtx := chainCtx.(*nodeConsensus.Context)
	typedDB := dbManager.(database.Database)

	// Convert fxs from []interface{} to []*commonEng.Fx
	typedFxs := make([]*commonEng.Fx, len(fxs))
	for i, fx := range fxs {
		typedFxs[i] = fx.(*commonEng.Fx)
	}

	// Convert appSender interface to typed AppSender
	typedAppSender := appSender.(commonEng.AppSender)

	return vm.initializeInternal(ctx, typedChainCtx, typedDB, genesisBytes, upgradeBytes, configBytes, typedFxs, typedAppSender)
}

// initializeInternal contains the actual initialization logic with strongly typed parameters
func (vm *VM) initializeInternal(
	ctx context.Context,
	chainCtx *nodeConsensus.Context,
	db database.Database,
	genesisBytes []byte,
	upgradeBytes []byte,
	configBytes []byte,
	fxs []*commonEng.Fx,
	appSender commonEng.AppSender,
) error {
	vm.stateSyncDone = make(chan struct{})
	vm.config.SetDefaults(defaultTxPoolConfig)
	if len(configBytes) > 0 {
		if err := json.Unmarshal(configBytes, &vm.config); err != nil {
			return fmt.Errorf("failed to unmarshal config %s: %w", string(configBytes), err)
		}
	}
	if err := vm.config.Validate(); err != nil {
		return err
	}
	// We should deprecate config flags as the first thing, before we do anything else
	// because this can set old flags to new flags. log the message after we have
	// initialized the logger.
	deprecateMsg := vm.config.Deprecate()

	// Store the chain context
	vm.chainCtx = chainCtx
	// Create a regular context for operations
	vm.ctx = ctx

	// Use ChainID from chainCtx for alias
	var alias string
	if chainCtx != nil && chainCtx.ChainID != ids.Empty {
		alias = chainCtx.ChainID.String()
	} else {
		alias = "evm"
	}
	vm.chainAlias = alias

	// Create a logger since consensus Context doesn't have Log field
	// TODO: Integrate with proper logging from consensus package
	contextLogger := log.New()
	logWriter := newLoggerWriter(contextLogger)
	subnetEVMLogger, err := subnetevmlog.InitLogger(vm.chainAlias, vm.config.LogLevel, vm.config.LogJSONFormat, logWriter)
	if err != nil {
		return fmt.Errorf("%w: %w ", errInitializingLogger, err)
	}
	vm.logger = subnetEVMLogger

	log.Info("Initializing Subnet EVM VM", "Version", Version, "geth version", params.VersionWithMeta, "Config", vm.config)

	if deprecateMsg != "" {
		log.Warn("Deprecation Warning", "msg", deprecateMsg)
	}

	if len(fxs) > 0 {
		return errUnsupportedFXs
	}

	// Enable debug-level metrics that might impact runtime performance
	// Note: metrics.EnabledExpensive is not a global in our geth fork
	// Expensive metrics configuration handled via config

	vm.shutdownChan = make(chan struct{}, 1)

	if err := vm.initializeMetrics(); err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}

	// Initialize the database
	if err := vm.initializeDBs(db); err != nil {
		return fmt.Errorf("failed to initialize databases: %w", err)
	}

	if vm.config.InspectDatabase {
		if err := vm.inspectDatabases(); err != nil {
			return err
		}
	}

	g, err := parseGenesis(vm.ctx, genesisBytes, upgradeBytes, vm.config.AirdropFile)
	if err != nil {
		return err
	}

	vm.syntacticBlockValidator = NewBlockValidator()

	vm.ethConfig = ethconfig.NewDefaultConfig()
	vm.ethConfig.Genesis = g
	// NetworkID here is different than Lux's NetworkID.
	// Lux's NetworkID represents the Lux network is running on
	// like Testnet, Mainnet, Local, etc.
	// The NetworkId here is kept same as ChainID to be compatible with
	// Ethereum tooling.
	vm.ethConfig.NetworkId = g.Config.ChainID.Uint64()

	// Set minimum price for mining and default gas price oracle value to the min
	// gas price to prevent so transactions and blocks all use the correct fees
	vm.ethConfig.RPCGasCap = vm.config.RPCGasCap
	vm.ethConfig.RPCEVMTimeout = vm.config.APIMaxDuration.Duration
	vm.ethConfig.RPCTxFeeCap = vm.config.RPCTxFeeCap

	vm.ethConfig.TxPool.Locals = vm.config.PriorityRegossipAddresses
	vm.ethConfig.TxPool.NoLocals = !vm.config.LocalTxsEnabled
	vm.ethConfig.TxPool.PriceLimit = vm.config.TxPoolPriceLimit
	vm.ethConfig.TxPool.PriceBump = vm.config.TxPoolPriceBump
	vm.ethConfig.TxPool.AccountSlots = vm.config.TxPoolAccountSlots
	vm.ethConfig.TxPool.GlobalSlots = vm.config.TxPoolGlobalSlots
	vm.ethConfig.TxPool.AccountQueue = vm.config.TxPoolAccountQueue
	vm.ethConfig.TxPool.GlobalQueue = vm.config.TxPoolGlobalQueue
	vm.ethConfig.TxPool.Lifetime = vm.config.TxPoolLifetime.Duration

	vm.ethConfig.AllowUnfinalizedQueries = vm.config.AllowUnfinalizedQueries
	vm.ethConfig.AllowUnprotectedTxs = vm.config.AllowUnprotectedTxs
	vm.ethConfig.AllowUnprotectedTxHashes = vm.config.AllowUnprotectedTxHashes
	vm.ethConfig.Preimages = vm.config.Preimages
	vm.ethConfig.Pruning = vm.config.Pruning
	vm.ethConfig.TrieCleanCache = vm.config.TrieCleanCache
	vm.ethConfig.TrieDirtyCache = vm.config.TrieDirtyCache
	vm.ethConfig.TrieDirtyCommitTarget = vm.config.TrieDirtyCommitTarget
	vm.ethConfig.TriePrefetcherParallelism = vm.config.TriePrefetcherParallelism
	vm.ethConfig.SnapshotCache = vm.config.SnapshotCache
	vm.ethConfig.AcceptorQueueLimit = vm.config.AcceptorQueueLimit
	vm.ethConfig.PopulateMissingTries = vm.config.PopulateMissingTries
	vm.ethConfig.PopulateMissingTriesParallelism = vm.config.PopulateMissingTriesParallelism
	vm.ethConfig.AllowMissingTries = vm.config.AllowMissingTries
	vm.ethConfig.SnapshotDelayInit = vm.config.StateSyncEnabled
	vm.ethConfig.SnapshotWait = vm.config.SnapshotWait
	vm.ethConfig.SnapshotVerify = vm.config.SnapshotVerify
	vm.ethConfig.HistoricalProofQueryWindow = vm.config.HistoricalProofQueryWindow
	vm.ethConfig.OfflinePruning = vm.config.OfflinePruning
	vm.ethConfig.OfflinePruningBloomFilterSize = vm.config.OfflinePruningBloomFilterSize
	vm.ethConfig.OfflinePruningDataDirectory = vm.config.OfflinePruningDataDirectory
	vm.ethConfig.CommitInterval = vm.config.CommitInterval
	vm.ethConfig.SkipUpgradeCheck = vm.config.SkipUpgradeCheck
	vm.ethConfig.AcceptedCacheSize = vm.config.AcceptedCacheSize
	vm.ethConfig.StateHistory = vm.config.StateHistory
	vm.ethConfig.TransactionHistory = vm.config.TransactionHistory
	vm.ethConfig.SkipTxIndexing = vm.config.SkipTxIndexing
	vm.ethConfig.StateScheme = vm.config.StateScheme

	// if vm.ethConfig.StateScheme == customrawdb.FirewoodScheme {
	// 	log.Warn("Firewood state scheme is enabled")
	// 	log.Warn("This is untested in production, use at your own risk")
	// 	// Firewood only supports pruning for now.
	// 	if !vm.config.Pruning {
	// 		return errors.New("Pruning must be enabled for Firewood")
	// 	}
	// 	// Firewood does not support iterators, so the snapshot cannot be constructed
	// 	if vm.config.SnapshotCache > 0 {
	// 		return errors.New("Snapshot cache must be disabled for Firewood")
	// 	}
	// 	if vm.config.OfflinePruning {
	// 		return errors.New("Offline pruning is not supported for Firewood")
	// 	}
	// 	if vm.config.StateSyncEnabled {
	// 		return errors.New("State sync is not yet supported for Firewood")
	// 	}
	// }
	if vm.ethConfig.StateScheme == rawdb.PathScheme {
		log.Error("Path state scheme is not supported. Please use HashDB state scheme instead")
		return errors.New("Path state scheme is not supported")
	}

	// Create directory for offline pruning
	if len(vm.ethConfig.OfflinePruningDataDirectory) != 0 {
		if err := os.MkdirAll(vm.ethConfig.OfflinePruningDataDirectory, perms.ReadWriteExecute); err != nil {
			log.Error("failed to create offline pruning data directory", "error", err)
			return err
		}
	}

	// Handle custom fee recipient
	if common.IsHexAddress(vm.config.FeeRecipient) {
		address := common.HexToAddress(vm.config.FeeRecipient)
		log.Info("Setting fee recipient", "address", address)
		vm.ethConfig.Miner.Etherbase = address
	} else {
		log.Info("Config has not specified any coinbase address. Defaulting to the blackhole address.")
		vm.ethConfig.Miner.Etherbase = constants.BlackholeAddr
	}

	vm.chainConfig = g.Config

	// create genesisHash after applying upgradeBytes in case
	// upgradeBytes modifies genesis.
	vm.genesisHash = vm.ethConfig.Genesis.ToBlock().Hash() // must create genesis hash before [vm.readLastAccepted]
	lastAcceptedHash, lastAcceptedHeight, err := vm.readLastAccepted()
	if err != nil {
		return err
	}
	log.Info("read last accepted",
		"hash", lastAcceptedHash,
		"height", lastAcceptedHeight,
	)

	vm.networkCodec = message.Codec
	// Convert block.AppSender to appsender.AppSender - they should be compatible
	coreAppSender, ok := appSender.(appsender.AppSender)
	if !ok {
		return fmt.Errorf("appSender does not implement appsender.AppSender")
	}
	// Wrap the consensus AppSender to match node's AppSender interface
	wrappedAppSender := &appSenderWrapper{appSender: coreAppSender}
	vm.Network, err = network.NewNetwork(context.Background(), wrappedAppSender, vm.networkCodec, vm.config.MaxOutboundActiveRequests, vm.sdkMetrics)
	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}
	// P2PValidators might be nil in test environments
	p2pValidatorsInterface := vm.Network.P2PValidators()
	if p2pValidatorsInterface != nil {
		vm.p2pValidators = p2pValidatorsInterface.(*p2p.Validators)
	}

	vm.validatorsManager, err = validators.NewManager(vm.ctx, vm.validatorsDB, &vm.clock)
	if err != nil {
		return fmt.Errorf("failed to initialize validators manager: %w", err)
	}

	// Initialize warp backend
	offchainWarpMessages := make([][]byte, len(vm.config.WarpOffChainMessages))
	for i, hexMsg := range vm.config.WarpOffChainMessages {
		offchainWarpMessages[i] = []byte(hexMsg)
	}
	warpSignatureCache := lru.NewCache[ids.ID, []byte](warpSignatureCacheSize)
	meteredCache, err := metercacher.New("warp_signature_cache", vm.sdkMetrics, warpSignatureCache)
	if err != nil {
		return fmt.Errorf("failed to create warp signature cache: %w", err)
	}

	// clear warpdb on initialization if config enabled
	if vm.config.PruneWarpDB {
		if err := database.Clear(vm.warpDB, ethdb.IdealBatchSize); err != nil {
			return fmt.Errorf("failed to prune warpDB: %w", err)
		}
	}

	// VM implements warp.BlockClient directly

	// Get warp signer from context
	// Warp signer is not directly available in chainCtx, skip for now
	var warpSignerInterface interface{}
	var warpAdapter *warpSignerAdapter
	if warpSignerInterface != nil {
		warpSigner, ok := warpSignerInterface.(signer.Signer)
		if !ok {
			return fmt.Errorf("invalid warp signer type: %T", warpSignerInterface)
		}
		// Create a wrapper that implements WarpSigner
		warpAdapter = &warpSignerAdapter{
			signer: warpSigner,
			nodeID: chainCtx.NodeID,
		}
	}

	// Only create warp backend if we have a signer
	if warpAdapter != nil {
		vm.warpBackend, err = warp.NewBackend(
			chainCtx.QuantumID, // Use QuantumID as network ID from consensus Context
			chainCtx.ChainID,
			warpAdapter,
			&warpBlockClient{vm: vm}, // Wrapper that implements warp.BlockClient
			validators.NewLockedValidatorReader(vm.validatorsManager, &vm.vmLock),
			vm.warpDB,
			meteredCache,
			offchainWarpMessages,
		)
		if err != nil {
			return err
		}
	}

	if err := vm.initializeChain(lastAcceptedHash, vm.ethConfig); err != nil {
		return err
	}

	// Start continuous profiler in a goroutine with recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				contextLogger.Error("continuous profiler panicked", "panic", r)
			}
		}()
		vm.startContinuousProfiler()
	}()

	// Add p2p warp message warpHandler
	// Create adapter to convert our warp backend to lp118.Verifier
	warpVerifier := &warpVerifierAdapter{backend: vm.warpBackend}
	// Pass nil for signer since lp118 doesn't use it the same way
	warpHandler := lp118.NewCachedHandler(meteredCache, warpVerifier, nil)
	// Use adapter to convert lp118.Handler to p2p.Handler
	p2pHandler := newLP118HandlerAdapter(warpHandler)
	vm.Network.AddHandler(lp118.HandlerID, p2pHandler)

	vm.setAppRequestHandlers()

	vm.StateSyncServer = NewStateSyncServer(&stateSyncServerConfig{
		Chain:            vm.blockChain,
		SyncableInterval: vm.config.StateSyncCommitInterval,
	})
	return vm.initializeStateSyncClient(lastAcceptedHeight)
}

func parseGenesis(ctx context.Context, genesisBytes []byte, upgradeBytes []byte, airdropFile string) (*core.Genesis, error) {
	// First check if this is a database replay genesis
	var genesisMap map[string]interface{}
	if err := json.Unmarshal(genesisBytes, &genesisMap); err == nil {
		if replay, ok := genesisMap["replay"].(bool); ok && replay {
			// This is a database replay genesis
			dbPath, _ := genesisMap["dbPath"].(string)
			dbType, _ := genesisMap["dbType"].(string)
			log.Info("Database replay genesis detected", "dbPath", dbPath, "dbType", dbType)

			// Return a special genesis that signals database replay
			g := &core.Genesis{
				Config: &params.ChainConfig{},
			}

			// Extract the chain config from the genesis map
			if configData, ok := genesisMap["config"].(map[string]interface{}); ok {
				configBytes, _ := json.Marshal(configData)
				if err := json.Unmarshal(configBytes, g.Config); err != nil {
					return nil, fmt.Errorf("failed to parse chain config: %w", err)
				}
			}

			// Note: Database replay fields are not present in our ChainConfig
			// These would need to be added if database replay functionality is needed

			return g, nil
		}
	}

	// Normal genesis parsing
	g := new(core.Genesis)
	if err := json.Unmarshal(genesisBytes, g); err != nil {
		return nil, fmt.Errorf("parsing genesis: %w", err)
	}

	// Set the default chain config if not provided
	if g.Config == nil {
		g.Config = params.SubnetEVMDefaultChainConfig
	}

	// Populate the Lux config extras.
	configExtra := params.GetExtra(g.Config)
	configExtra.LuxContext = extras.LuxContext{
		ConsensusCtx: ctx,
	}

	if configExtra.FeeConfig == commontype.EmptyFeeConfig {
		log.Info("No fee config given in genesis, setting default fee config", "DefaultFeeConfig", params.DefaultFeeConfig)
		configExtra.FeeConfig = params.DefaultFeeConfig
	}

	// Load airdrop file if provided
	if airdropFile != "" {
		var err error
		g.AirdropData, err = os.ReadFile(airdropFile)
		if err != nil {
			return nil, fmt.Errorf("could not read airdrop file '%s': %w", airdropFile, err)
		}
	}

	// Set network upgrade defaults
	// Network upgrades are managed through chain config
	configExtra.SetDefaults(upgrade.Config{})

	// Parse network upgrades from the genesis JSON if present
	// They won't be in g.Config because geth's ChainConfig doesn't know about them
	// This must be done after SetDefaults to override defaults with genesis values
	genesisMap = make(map[string]interface{})
	if err := json.Unmarshal(genesisBytes, &genesisMap); err == nil {
		if configData, ok := genesisMap["config"].(map[string]interface{}); ok {
			// Extract network upgrade timestamps
			if val, ok := configData["subnetEVMTimestamp"]; ok {
				if ts, ok := val.(float64); ok {
					configExtra.SubnetEVMTimestamp = utils.NewUint64(uint64(ts))
				}
			}
			if val, ok := configData["durangoTimestamp"]; ok {
				if ts, ok := val.(float64); ok {
					configExtra.DurangoTimestamp = utils.NewUint64(uint64(ts))
				}
			}
			if val, ok := configData["etnaTimestamp"]; ok {
				if ts, ok := val.(float64); ok {
					configExtra.EtnaTimestamp = utils.NewUint64(uint64(ts))
				}
			}
			if val, ok := configData["fortunaTimestamp"]; ok {
				if ts, ok := val.(float64); ok {
					configExtra.FortunaTimestamp = utils.NewUint64(uint64(ts))
				}
			}
			if val, ok := configData["graniteTimestamp"]; ok {
				if ts, ok := val.(float64); ok {
					configExtra.GraniteTimestamp = utils.NewUint64(uint64(ts))
				}
			}
		}
	}

	// Apply upgradeBytes (if any) by unmarshalling them into [chainConfig.UpgradeConfig].
	// Initializing the chain will verify upgradeBytes are compatible with existing values.
	// This should be called before g.Verify().
	if len(upgradeBytes) > 0 {
		var upgradeConfig extras.UpgradeConfig
		if err := json.Unmarshal(upgradeBytes, &upgradeConfig); err != nil {
			return nil, fmt.Errorf("failed to parse upgrade bytes: %w", err)
		}
		configExtra.UpgradeConfig = upgradeConfig
	}

	if configExtra.UpgradeConfig.NetworkUpgradeOverrides != nil {
		overrides := configExtra.UpgradeConfig.NetworkUpgradeOverrides
		marshaled, err := json.Marshal(overrides)
		if err != nil {
			log.Warn("Failed to marshal network upgrade overrides", "error", err, "overrides", overrides)
		} else {
			log.Info("Applying network upgrade overrides", "overrides", string(marshaled))
		}
		configExtra.Override(overrides)
	}

	if err := configExtra.Verify(); err != nil {
		return nil, fmt.Errorf("invalid chain config: %w", err)
	}

	// Align all the Ethereum upgrades to the Lux upgrades
	if err := params.SetEthUpgrades(g.Config); err != nil {
		return nil, fmt.Errorf("setting eth upgrades: %w", err)
	}
	return g, nil
}

func (vm *VM) initializeMetrics() error {
	// Enable metrics collection using our geth's Enable function
	metrics.Enable()
	vm.sdkMetrics = prometheus.NewRegistry()
	gatherer := subnetevmprometheus.NewGatherer(metrics.DefaultRegistry)
	// Metrics are handled through sdkMetrics parameter
	_ = gatherer

	// if vm.config.MetricsExpensiveEnabled && vm.config.StateScheme == customrawdb.FirewoodScheme {
	// 	if err := ffi.StartMetrics(); err != nil {
	// 		return fmt.Errorf("failed to start firewood metrics collection: %w", err)
	// 	}
	// 	// Firewood metrics registration deferred
	// }
	// SDK metrics registered via sdkMetrics parameter
	return nil
}

func (vm *VM) initializeChain(lastAcceptedHash common.Hash, ethConfig ethconfig.Config) error {
	nodecfg := &node.Config{
		SubnetEVMVersion:      Version,
		KeyStoreDir:           vm.config.KeystoreDirectory,
		ExternalSigner:        vm.config.KeystoreExternalSigner,
		InsecureUnlockAllowed: vm.config.KeystoreInsecureUnlockAllowed,
	}
	node, err := node.New(nodecfg)
	if err != nil {
		return err
	}
	vm.eth, err = eth.New(
		node,
		&vm.ethConfig,
		&EthPushGossiper{vm: vm},
		vm.chaindb,
		eth.Settings{MaxBlocksPerRequest: vm.config.MaxBlocksPerRequest},
		lastAcceptedHash,
		dummy.NewFakerWithClock(&vm.clock),
		&vm.clock,
	)
	if err != nil {
		return err
	}
	vm.eth.SetEtherbase(ethConfig.Miner.Etherbase)
	vm.txPool = vm.eth.TxPool()
	vm.blockChain = vm.eth.BlockChain()
	vm.miner = vm.eth.Miner()
	lastAccepted := vm.blockChain.LastAcceptedBlock()
	feeConfig, _, err := vm.blockChain.GetFeeConfigAt(lastAccepted.Header())
	if err != nil {
		return err
	}
	vm.txPool.SetMinFee(feeConfig.MinBaseFee)
	vm.txPool.SetGasTip(big.NewInt(0))

	vm.eth.Start()
	return vm.initChainState(lastAccepted)
}

// initializeStateSyncClient initializes the client for performing state sync.
// If state sync is disabled, this function will wipe any ongoing summary from
// disk to ensure that we do not continue syncing from an invalid snapshot.
func (vm *VM) initializeStateSyncClient(lastAcceptedHeight uint64) error {
	// parse nodeIDs from state sync IDs in vm config
	var stateSyncIDs []ids.NodeID
	if vm.config.StateSyncEnabled && len(vm.config.StateSyncIDs) > 0 {
		nodeIDs := strings.Split(vm.config.StateSyncIDs, ",")
		stateSyncIDs = make([]ids.NodeID, len(nodeIDs))
		for i, nodeIDString := range nodeIDs {
			nodeID, err := ids.NodeIDFromString(nodeIDString)
			if err != nil {
				return fmt.Errorf("failed to parse %s as NodeID: %w", nodeIDString, err)
			}
			stateSyncIDs[i] = nodeID
		}
	}

	vm.StateSyncClient = NewStateSyncClient(&stateSyncClientConfig{
		chain:         vm.eth,
		state:         vm.State,
		stateSyncDone: vm.stateSyncDone,
		client: statesyncclient.NewClient(
			&statesyncclient.ClientConfig{
				NetworkClient:    vm.Network,
				Codec:            vm.networkCodec,
				Stats:            stats.NewClientSyncerStats(),
				StateSyncNodeIDs: stateSyncIDs,
				BlockParser:      vm,
			},
		),
		enabled:              vm.config.StateSyncEnabled,
		skipResume:           vm.config.StateSyncSkipResume,
		stateSyncMinBlocks:   vm.config.StateSyncMinBlocks,
		stateSyncRequestSize: vm.config.StateSyncRequestSize,
		lastAcceptedHeight:   lastAcceptedHeight, // TODO clean up how this is passed around
		chaindb:              vm.chaindb,
		metadataDB:           vm.metadataDB,
		acceptedBlockDB:      vm.acceptedBlockDB,
		db:                   vm.versiondb,
	})

	// If StateSync is disabled, clear any ongoing summary so that we will not attempt to resume
	// sync using a snapshot that has been modified by the node running normal operations.
	if !vm.config.StateSyncEnabled {
		return vm.StateSyncClient.ClearOngoingSummary()
	}

	return nil
}

func (vm *VM) initChainState(lastAcceptedBlock *types.Block) error {
	block := vm.newBlock(lastAcceptedBlock)

	config := &nodeChain.Config{
		DecidedCacheSize:    decidedCacheSize,
		MissingCacheSize:    missingCacheSize,
		UnverifiedCacheSize: unverifiedCacheSize,
		BytesToIDCacheSize:  bytesToIDCacheSize,
		// Our vm methods return *Block which needs to implement the node's chain.Block
		GetBlock: func(ctx context.Context, id ids.ID) (nodeConsensusChain.Block, error) {
			// getBlock returns consensus block, we need to return node block
			ethBlock := vm.blockChain.GetBlockByHash(common.Hash(id))
			if ethBlock == nil {
				return nil, database.ErrNotFound
			}
			return vm.newBlock(ethBlock), nil
		},
		UnmarshalBlock: func(ctx context.Context, b []byte) (nodeConsensusChain.Block, error) {
			// parseBlock returns consensus block, we need to return node block
			ethBlock := &types.Block{}
			if err := rlp.DecodeBytes(b, ethBlock); err != nil {
				return nil, err
			}
			return vm.newBlock(ethBlock), nil
		},
		BuildBlock: func(ctx context.Context) (nodeConsensusChain.Block, error) {
			// Call VM's BuildBlock directly which returns the right type
			return vm.BuildBlock(ctx)
		},
		// BuildBlockWithContext: func(ctx context.Context, proposerVMBlockCtx *block.Context) (chain.Block, error) {
		// 	// Call VM's BuildBlockWithContext directly which returns the right type
		// 	return vm.BuildBlockWithContext(ctx, proposerVMBlockCtx)
		// },
		LastAcceptedBlock: block,
	}

	// Register chain state metrics
	chainStateRegisterer := prometheus.NewRegistry()
	state, err := nodeChain.NewMeteredState(chainStateRegisterer, config)
	if err != nil {
		return fmt.Errorf("could not create metered state: %w", err)
	}
	vm.State = state

	if !metrics.Enabled() {
		return nil
	}

	// Chain state metrics registered through initializeMetrics
	_ = chainStateRegisterer
	return nil
}

func (vm *VM) SetState(_ context.Context, state consensusInterfaces.State) error {
	vm.vmLock.Lock()
	defer vm.vmLock.Unlock()

	switch state {
	case consensusInterfaces.StateSyncing:
		vm.bootstrapped.Set(false)
		return nil
	case consensusInterfaces.Bootstrapping:
		return vm.onBootstrapStarted()
	case consensusInterfaces.NormalOp:
		return vm.onNormalOperationsStarted()
	default:
		return fmt.Errorf("unknown state: %v", state)
	}
}

// onBootstrapStarted marks this VM as bootstrapping
func (vm *VM) onBootstrapStarted() error {
	vm.bootstrapped.Set(false)
	if err := vm.StateSyncClient.Error(); err != nil {
		return err
	}
	// After starting bootstrapping, do not attempt to resume a previous state sync.
	if err := vm.StateSyncClient.ClearOngoingSummary(); err != nil {
		return err
	}
	// Ensure snapshots are initialized before bootstrapping (i.e., if state sync is skipped).
	// Note calling this function has no effect if snapshots are already initialized.
	vm.blockChain.InitializeSnapshots()

	return nil
}

// onNormalOperationsStarted marks this VM as bootstrapped
func (vm *VM) onNormalOperationsStarted() error {
	if vm.bootstrapped.Get() {
		return nil
	}
	vm.bootstrapped.Set(true)

	ctx, cancel := context.WithCancel(context.TODO())
	vm.cancel = cancel

	// Start the validators manager
	if err := vm.validatorsManager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize validators manager: %w", err)
	}

	// dispatch validator set update
	vm.shutdownWg.Add(1)
	go func() {
		vm.validatorsManager.DispatchSync(ctx, &vm.vmLock)
		vm.shutdownWg.Done()
	}()

	// Initialize goroutines related to block building
	// once we enter normal operation as there is no need to handle mempool gossip before this point.
	ethTxGossipMarshaller := GossipEthTxMarshaller{}

	// P2PValidators might be nil in test environments
	var p2pValidators *p2p.Validators
	p2pValidatorsInterface := vm.P2PValidators()
	var ethTxGossipClient *p2p.Client
	if p2pValidatorsInterface != nil {
		var ok bool
		p2pValidators, ok = p2pValidatorsInterface.(*p2p.Validators)
		if !ok {
			return fmt.Errorf("failed to get P2P validators")
		}
		ethTxGossipClient = vm.Network.NewClient(TxGossipHandlerID, p2p.WithValidatorSampling(p2pValidators))
	} else {
		// In test mode, use a client without validator sampling
		ethTxGossipClient = vm.Network.NewClient(TxGossipHandlerID)
	}
	ethTxGossipMetrics, err := gossip.NewMetrics(vm.sdkMetrics, ethTxGossipNamespace)
	if err != nil {
		return fmt.Errorf("failed to initialize eth tx gossip metrics: %w", err)
	}
	ethTxPool, err := NewGossipEthTxPool(vm.txPool, vm.sdkMetrics)
	if err != nil {
		return fmt.Errorf("failed to initialize gossip eth tx pool: %w", err)
	}
	vm.shutdownWg.Add(1)
	go func() {
		ethTxPool.Subscribe(ctx)
		vm.shutdownWg.Done()
	}()

	pushGossipParams := gossip.BranchingFactor{
		StakePercentage: vm.config.PushGossipPercentStake,
		Validators:      vm.config.PushGossipNumValidators,
		Peers:           vm.config.PushGossipNumPeers,
	}
	pushRegossipParams := gossip.BranchingFactor{
		Validators: vm.config.PushRegossipNumValidators,
		Peers:      vm.config.PushRegossipNumPeers,
	}

	ethTxPushGossiper := vm.ethTxPushGossiper.Get()
	if ethTxPushGossiper == nil && p2pValidatorsInterface != nil {
		// Only create push gossiper if we have P2P validators
		p2pValidators, _ := p2pValidatorsInterface.(*p2p.Validators)
		ethTxPushGossiper, err = gossip.NewPushGossiper[*GossipEthTx](
			ethTxGossipMarshaller,
			ethTxPool,
			p2pValidators,
			ethTxGossipClient,
			ethTxGossipMetrics,
			pushGossipParams,
			pushRegossipParams,
			config.PushGossipDiscardedElements,
			config.TxGossipTargetMessageSize,
			vm.config.RegossipFrequency.Duration,
		)
		if err != nil {
			return fmt.Errorf("failed to initialize eth tx push gossiper: %w", err)
		}
		vm.ethTxPushGossiper.Set(ethTxPushGossiper)
	}

	// NOTE: gossip network must be initialized first otherwise ETH tx gossip will not work.
	vm.builderLock.Lock()
	vm.builder = vm.NewBlockBuilder()
	vm.builder.awaitSubmittedTxs()
	vm.builderLock.Unlock()

	if vm.ethTxGossipHandler == nil {
		// Get logger from context for gossip handler
		// Use VM's logger instead of consensus logger
		handler, err := gossipHandler.NewTxGossipHandler[*GossipEthTx](
			log.Root(),
			ethTxGossipMarshaller,
			ethTxPool,
			ethTxGossipMetrics,
			config.TxGossipTargetMessageSize,
			config.TxGossipThrottlingPeriod,
			float64(config.TxGossipThrottlingLimit),
			p2pValidators,
			vm.sdkMetrics,
			ethTxGossipNamespace,
		)
		if err != nil {
			return fmt.Errorf("failed to create tx gossip handler: %w", err)
		}
		vm.ethTxGossipHandler = handler
	}

	if err := vm.Network.AddHandler(TxGossipHandlerID, vm.ethTxGossipHandler); err != nil {
		return fmt.Errorf("failed to add eth tx gossip handler: %w", err)
	}

	if vm.ethTxPullGossiper == nil && p2pValidators != nil {
		// Only create pull gossiper if we have P2P validators
		// Use VM's logger instead of consensus logger
		ethTxPullGossiper := gossip.NewPullGossiper[*GossipEthTx](
			vm.logger,
			ethTxGossipMarshaller,
			ethTxPool,
			ethTxGossipClient,
			ethTxGossipMetrics,
			config.TxGossipPollSize,
		)

		vm.ethTxPullGossiper = gossip.ValidatorGossiper{
			Gossiper:   ethTxPullGossiper,
			NodeID:     vm.chainCtx.NodeID,
			Validators: p2pValidators,
		}
	}

	// Get logger for gossip routines
	// Use VM's logger for gossip routines

	vm.shutdownWg.Add(1)
	go func() {
		gossip.Every(ctx, vm.logger, ethTxPushGossiper, vm.config.PushGossipFrequency.Duration)
		vm.shutdownWg.Done()
	}()
	vm.shutdownWg.Add(1)
	go func() {
		gossip.Every(ctx, vm.logger, vm.ethTxPullGossiper, vm.config.PullGossipFrequency.Duration)
		vm.shutdownWg.Done()
	}()

	return nil
}

// setAppRequestHandlers sets the request handlers for the VM to serve state sync
// requests.
func (vm *VM) setAppRequestHandlers() {
	// Create standalone EVM TrieDB (read only) for serving leafs requests.
	// We create a standalone TrieDB here, so that it has a standalone cache from the one
	// used by the node when processing blocks.
	evmTrieDB := triedb.NewDatabase(
		vm.chaindb,
		&triedb.Config{
			HashDB: &hashdb.Config{
				CleanCacheSize: vm.config.StateSyncServerTrieCache * units.MiB,
			},
		},
	)

	networkHandler := newNetworkHandler(vm.blockChain, vm.chaindb, evmTrieDB, vm.warpBackend, vm.networkCodec)
	vm.Network.SetRequestHandler(networkHandler)
}

func (vm *VM) WaitForEvent(ctx context.Context) (commonEng.Message, error) {
	vm.builderLock.Lock()
	builder := vm.builder
	vm.builderLock.Unlock()

	// Block building is not initialized yet, so we haven't finished syncing or bootstrapping.
	if builder == nil {
		select {
		case <-ctx.Done():
			return commonEng.Message{}, ctx.Err()
		case <-vm.stateSyncDone:
			// Return empty message to indicate state sync is done
			return commonEng.Message{}, nil
		case <-vm.shutdownChan:
			return commonEng.Message{}, errShuttingDownVM
		}
	}

	return builder.waitForEvent(ctx)
}

// Shutdown implements the chain.ChainVM interface
func (vm *VM) Shutdown(context.Context) error {
	vm.vmLock.Lock()
	defer vm.vmLock.Unlock()
	if vm.ctx == nil {
		return nil
	}
	if vm.cancel != nil {
		vm.cancel()
	}
	if vm.bootstrapped.Get() {
		if err := vm.validatorsManager.Shutdown(); err != nil {
			return fmt.Errorf("failed to shutdown validators manager: %w", err)
		}
	}
	vm.Network.Shutdown()
	if err := vm.StateSyncClient.Shutdown(); err != nil {
		log.Error("error stopping state syncer", "err", err)
	}
	close(vm.shutdownChan)
	// Stop RPC handlers before eth.Stop which will close the database
	for _, handler := range vm.rpcHandlers {
		handler.Stop()
	}
	vm.eth.Stop()
	log.Info("Ethereum backend stop completed")
	if vm.usingStandaloneDB {
		if err := vm.db.Close(); err != nil {
			log.Error("failed to close database: %w", err)
		} else {
			log.Info("Database closed")
		}
	}
	vm.shutdownWg.Wait()
	log.Info("Subnet-EVM Shutdown completed")
	return nil
}

// BuildBlock implements the ChainVM interface
func (vm *VM) BuildBlock(ctx context.Context) (nodeblock.Block, error) {
	blk, err := vm.buildBlock(ctx)
	if err != nil {
		return nil, err
	}
	// Adapt the consensus block to node block interface
	return NewBlockAdapter(blk.(*Block)), nil
}

// BuildBlockWithContext implements the BuildBlockWithContextChainVM interface
func (vm *VM) BuildBlockWithContext(ctx context.Context, proposerVMBlockCtx *nodeblock.Context) (nodeblock.Block, error) {
	// Convert node context to consensus context
	var consensusCtx *block.Context
	if proposerVMBlockCtx != nil {
		consensusCtx = &block.Context{
			PChainHeight: proposerVMBlockCtx.PChainHeight,
		}
	}
	blk, err := vm.buildBlockWithContext(ctx, consensusCtx)
	if err != nil {
		return nil, err
	}
	// Adapt the consensus block to node block interface
	return NewBlockAdapter(blk.(*Block)), nil
}

// buildBlock builds a block to be wrapped by ChainState
func (vm *VM) buildBlock(ctx context.Context) (block.Block, error) {
	return vm.buildBlockWithContext(ctx, nil)
}

func (vm *VM) buildBlockWithContext(ctx context.Context, proposerVMBlockCtx *block.Context) (block.Block, error) {
	if proposerVMBlockCtx != nil {
		log.Debug("Building block with context", "pChainBlockHeight", proposerVMBlockCtx.PChainHeight)
	} else {
		log.Debug("Building block without context")
	}
	predicateCtx := &precompileconfig.PredicateContext{
		ConsensusCtx:       context.Background(),
		ProposerVMBlockCtx: proposerVMBlockCtx,
	}

	block, err := vm.miner.GenerateBlock(predicateCtx)
	vm.builder.handleGenerateBlock()
	if err != nil {
		return nil, err
	}

	// Note: the status of block is set by ChainState
	blk := vm.newBlock(block)

	// Verify is called on a non-wrapped block here, such that this
	// does not add [blk] to the processing blocks map in ChainState.
	//
	// TODO cache verification since Verify() will be called by the
	// consensus engine as well.
	//
	// Note: this is only called when building a new block, so caching
	// verification will only be a significant optimization for nodes
	// that produce a large number of blocks.
	// We call verify without writes here to avoid generating a reference
	// to the blk state root in the triedb when we are going to call verify
	// again from the consensus engine with writes enabled.
	if err := blk.verify(predicateCtx, false /*=writes*/); err != nil {
		return nil, fmt.Errorf("block failed verification due to: %w", err)
	}

	log.Debug("built block",
		"id", blk.ID(),
	)
	// Marks the current transactions from the mempool as being successfully issued
	// into a block.
	return blk, nil
}

// parseBlock parses [b] into a block to be wrapped by ChainState.
func (vm *VM) parseBlock(_ context.Context, b []byte) (block.Block, error) {
	ethBlock := new(types.Block)
	if err := rlp.DecodeBytes(b, ethBlock); err != nil {
		return nil, err
	}

	// Note: the status of block is set by ChainState
	block := vm.newBlock(ethBlock)
	// Performing syntactic verification in ParseBlock allows for
	// short-circuiting bad blocks before they are processed by the VM.
	if err := block.syntacticVerify(); err != nil {
		return nil, fmt.Errorf("syntactic block verification failed: %w", err)
	}
	return block, nil
}

func (vm *VM) ParseEthBlock(b []byte) (*types.Block, error) {
	// Parse the raw bytes into an Ethereum block
	ethBlock := new(types.Block)
	if err := rlp.DecodeBytes(b, ethBlock); err != nil {
		return nil, err
	}
	return ethBlock, nil
}

// getBlock attempts to retrieve block [id] from the VM to be wrapped
// by ChainState.
// GetBlock implements the ChainVM interface
func (vm *VM) GetBlock(ctx context.Context, id ids.ID) (nodeblock.Block, error) {
	blk, err := vm.getBlock(ctx, id)
	if err != nil {
		return nil, err
	}
	// Adapt the consensus block to node block interface
	return NewBlockAdapter(blk.(nodechain.Block)), nil
}

func (vm *VM) getBlock(_ context.Context, id ids.ID) (block.Block, error) {
	ethBlock := vm.blockChain.GetBlockByHash(common.Hash(id))
	// If [ethBlock] is nil, return [database.ErrNotFound] here
	// so that the miss is considered cacheable.
	if ethBlock == nil {
		return nil, database.ErrNotFound
	}
	// Note: the status of block is set by ChainState
	return vm.newBlock(ethBlock), nil
}

// GetAcceptedBlock attempts to retrieve block [blkID] from the VM. This method
// only returns accepted blocks.
func (vm *VM) GetAcceptedBlock(ctx context.Context, blkID ids.ID) (nodeConsensusChain.Block, error) {
	// First verify the block is accepted
	ethBlock := vm.blockChain.GetBlockByHash(common.BytesToHash(blkID[:]))
	if ethBlock == nil {
		return nil, database.ErrNotFound
	}

	// Check if this block is accepted by comparing with canonical chain
	acceptedHash := vm.blockChain.GetCanonicalHash(ethBlock.NumberU64())
	if acceptedHash != ethBlock.Hash() {
		return nil, database.ErrNotFound
	}

	// Get the block from our State
	blk, err := vm.State.GetBlock(ctx, blkID)
	if err != nil {
		return nil, err
	}

	// Extract the actual Block from the nodeChain.BlockWrapper
	switch b := blk.(type) {
	case *nodeChain.BlockWrapper:
		if evmBlock, ok := b.Block.(*Block); ok {
			return evmBlock, nil
		}
		return nil, fmt.Errorf("unexpected block type in wrapper: %T", b.Block)
	case *Block:
		return b, nil
	default:
		return nil, fmt.Errorf("unexpected block type: %T", blk)
	}
}

// SetPreference sets what the current tail of the chain is
func (vm *VM) SetPreference(ctx context.Context, blkID ids.ID) error {
	// Since each internal handler used by [vm.State] always returns a block
	// with non-nil ethBlock value, GetBlockInternal should never return a
	// (*Block) with a nil ethBlock value.
	block, err := vm.GetBlockInternal(ctx, blkID)
	if err != nil {
		return fmt.Errorf("failed to set preference to %s: %w", blkID, err)
	}

	return vm.blockChain.SetPreference(block.(*Block).ethBlock)
}

// GetBlockIDAtHeight returns the canonical block at [height].
// Note: the engine assumes that if a block is not found at [height], then
// [database.ErrNotFound] will be returned. This indicates that the VM has state
// synced and does not have all historical blocks available.
func (vm *VM) GetBlockIDAtHeight(_ context.Context, height uint64) (ids.ID, error) {
	lastAcceptedBlock := vm.blockChain.LastAcceptedBlock()
	if lastAcceptedBlock.NumberU64() < height {
		return ids.ID{}, database.ErrNotFound
	}

	hash := vm.blockChain.GetCanonicalHash(height)
	if hash == (common.Hash{}) {
		return ids.ID{}, database.ErrNotFound
	}
	return ids.ID(hash), nil
}

func (vm *VM) Version(context.Context) (string, error) {
	return Version, nil
}

// NewHandler returns a new Handler for a service where:
//   - The handler's functionality is defined by [service]
//     [service] should be a gorilla RPC service (see https://www.gorillatoolkit.org/pkg/rpc/v2)
//   - The name of the service is [name]
func newHandler(name string, service interface{}) (http.Handler, error) {
	server := luxRPC.NewServer()
	server.RegisterCodec(luxJSON.NewCodec(), "application/json")
	server.RegisterCodec(luxJSON.NewCodec(), "application/json;charset=UTF-8")
	return server, server.RegisterService(service, name)
}

// CreateHandlers makes new http handlers that can handle API calls
func (vm *VM) CreateHandlers(context.Context) (map[string]http.Handler, error) {
	handler := rpc.NewServer(vm.config.APIMaxDuration.Duration)
	if vm.config.BatchRequestLimit > 0 && vm.config.BatchResponseMaxSize > 0 {
		handler.SetBatchLimits(int(vm.config.BatchRequestLimit), int(vm.config.BatchResponseMaxSize))
	}
	if vm.config.HttpBodyLimit > 0 {
		handler.SetHTTPBodyLimit(int(vm.config.HttpBodyLimit))
	}

	enabledAPIs := vm.config.EthAPIs()
	if err := attachEthService(handler, vm.eth.APIs(), enabledAPIs); err != nil {
		return nil, err
	}

	apis := make(map[string]http.Handler)
	if vm.config.AdminAPIEnabled {
		adminAPI, err := newHandler("admin", NewAdminService(vm, os.ExpandEnv(fmt.Sprintf("%s_subnet_evm_performance_%s", vm.config.AdminAPIDir, vm.chainAlias))))
		if err != nil {
			return nil, fmt.Errorf("failed to register service for admin API due to %w", err)
		}
		apis[adminEndpoint] = adminAPI
		enabledAPIs = append(enabledAPIs, "evm-admin")
	}

	if vm.config.ValidatorsAPIEnabled {
		validatorsAPI, err := newHandler("validators", &ValidatorsAPI{vm})
		if err != nil {
			return nil, fmt.Errorf("failed to register service for validators API due to %w", err)
		}
		apis[validatorsEndpoint] = validatorsAPI
		enabledAPIs = append(enabledAPIs, "validators")
	}

	if vm.config.WarpAPIEnabled {
		warpSDKClient := vm.Network.NewClient(lp118.HandlerID)
		// lp118.NewSignatureAggregator expects a node/utils/logging.Logger
		// For now, pass nil as the logger is optional
		signatureAggregator := lp118.NewSignatureAggregator(nil, warpSDKClient)

		if err := handler.RegisterName("warp", warp.NewAPI(vm.ctx, vm.warpBackend, signatureAggregator, vm.requirePrimaryNetworkSigners)); err != nil {
			return nil, err
		}
		enabledAPIs = append(enabledAPIs, "warp")
	}

	log.Info("enabling apis",
		"apis", enabledAPIs,
	)
	apis[ethRPCEndpoint] = handler
	apis[ethWSEndpoint] = handler.WebsocketHandlerWithDuration(
		[]string{"*"},
		vm.config.APIMaxDuration.Duration,
		vm.config.WSCPURefillRate.Duration,
		vm.config.WSCPUMaxStored.Duration,
	)

	vm.rpcHandlers = append(vm.rpcHandlers, handler)
	return apis, nil
}

// NewHTTPHandler implements the block.ChainVM interface
func (vm *VM) NewHTTPHandler(ctx context.Context) (http.Handler, error) {
	handlers, err := vm.CreateHandlers(ctx)
	if err != nil {
		return nil, err
	}

	// Return the main RPC handler as the primary HTTP handler
	if handler, exists := handlers[ethRPCEndpoint]; exists {
		return handler, nil
	}

	// Fallback to a default handler if no RPC handler exists
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "No HTTP handler available", http.StatusNotFound)
	}), nil
}

func (vm *VM) CreateHTTP2Handler(context.Context) (http.Handler, error) {
	return nil, nil
}

/*
 ******************************************************************************
 *********************************** Helpers **********************************
 ******************************************************************************
 */

// GetCurrentNonce returns the nonce associated with the address at the
// preferred block
func (vm *VM) GetCurrentNonce(address common.Address) (uint64, error) {
	// Note: current state uses the state of the preferred block.
	state, err := vm.blockChain.State()
	if err != nil {
		return 0, err
	}
	return state.GetNonce(address), nil
}

func (vm *VM) chainConfigExtra() *extras.ChainConfig {
	return params.GetExtra(vm.chainConfig)
}

func (vm *VM) rules(number *big.Int, time uint64) extras.Rules {
	ethrules := vm.chainConfig.Rules(number, params.IsMergeTODO, time)
	return *params.GetExtrasRules(ethrules, vm.chainConfig, time)
}

// currentRules returns the chain rules for the current block.
func (vm *VM) currentRules() extras.Rules {
	header := vm.eth.APIBackend.CurrentHeader()
	return vm.rules(header.Number, header.Time)
}

// requirePrimaryNetworkSigners returns true if warp messages from the primary
// network must be signed by the primary network validators.
// This is necessary when the subnet is not validating the primary network.
func (vm *VM) requirePrimaryNetworkSigners() bool {
	switch c := vm.currentRules().Precompiles[warpcontract.ContractAddress].(type) {
	case *warpcontract.Config:
		return c.RequirePrimaryNetworkSigners
	default: // includes nil due to non-presence
		return false
	}
}

func (vm *VM) startContinuousProfiler() {
	// If the profiler directory is empty, return immediately
	// without creating or starting a continuous profiler.
	if vm.config.ContinuousProfilerDir == "" {
		return
	}
	vm.profiler = profiler.NewContinuous(
		filepath.Join(vm.config.ContinuousProfilerDir),
		vm.config.ContinuousProfilerFrequency.Duration,
		vm.config.ContinuousProfilerMaxFiles,
	)
	defer vm.profiler.Shutdown()

	vm.shutdownWg.Add(1)
	go func() {
		defer vm.shutdownWg.Done()
		log.Info("Dispatching continuous profiler", "dir", vm.config.ContinuousProfilerDir, "freq", vm.config.ContinuousProfilerFrequency, "maxFiles", vm.config.ContinuousProfilerMaxFiles)
		err := vm.profiler.Dispatch()
		if err != nil {
			log.Error("continuous profiler failed", "err", err)
		}
	}()
	// Wait for shutdownChan to be closed
	<-vm.shutdownChan
}

// readLastAccepted reads the last accepted hash from [acceptedBlockDB] and returns the
// last accepted block hash and height by reading directly from [vm.chaindb] instead of relying
// on [chain].
// Note: assumes [vm.chaindb] and [vm.genesisHash] have been initialized.
func (vm *VM) readLastAccepted() (common.Hash, uint64, error) {
	// Attempt to load last accepted block to determine if it is necessary to
	// initialize state with the genesis block.
	lastAcceptedBytes, lastAcceptedErr := vm.acceptedBlockDB.Get(lastAcceptedKey)
	switch {
	case lastAcceptedErr == database.ErrNotFound:
		// If there is nothing in the database, return the genesis block hash and height
		return vm.genesisHash, 0, nil
	case lastAcceptedErr != nil:
		return common.Hash{}, 0, fmt.Errorf("failed to get last accepted block ID due to: %w", lastAcceptedErr)
	case len(lastAcceptedBytes) != common.HashLength:
		return common.Hash{}, 0, fmt.Errorf("last accepted bytes should have been length %d, but found %d", common.HashLength, len(lastAcceptedBytes))
	default:
		lastAcceptedHash := common.BytesToHash(lastAcceptedBytes)
		height, found := rawdb.ReadHeaderNumber(vm.chaindb, lastAcceptedHash)
		if !found {
			return common.Hash{}, 0, fmt.Errorf("failed to retrieve header number of last accepted block: %s", lastAcceptedHash)
		}
		return lastAcceptedHash, height, nil
	}
}

// attachEthService registers the backend RPC services provided by Ethereum
// to the provided handler under their assigned namespaces.
func attachEthService(handler *rpc.Server, apis []rpc.API, names []string) error {
	enabledServicesSet := make(map[string]struct{})
	for _, ns := range names {
		// handle pre geth v1.10.20 api names as aliases for their updated values
		// to allow configurations to be backwards compatible.
		if newName, isLegacy := legacyApiNames[ns]; isLegacy {
			log.Info("deprecated api name referenced in configuration.", "deprecated", ns, "new", newName)
			enabledServicesSet[newName] = struct{}{}
			continue
		}

		enabledServicesSet[ns] = struct{}{}
	}

	apiSet := make(map[string]rpc.API)
	for _, api := range apis {
		if existingAPI, exists := apiSet[api.Name]; exists {
			return fmt.Errorf("duplicated API name: %s, namespaces %s and %s", api.Name, api.Namespace, existingAPI.Namespace)
		}
		apiSet[api.Name] = api
	}

	for name := range enabledServicesSet {
		api, exists := apiSet[name]
		if !exists {
			return fmt.Errorf("API service %s not found", name)
		}
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
	}

	return nil
}

func (vm *VM) Connected(ctx context.Context, nodeID ids.NodeID, version *version.Application) error {
	vm.vmLock.Lock()
	defer vm.vmLock.Unlock()

	if err := vm.validatorsManager.Connect(nodeID); err != nil {
		return fmt.Errorf("uptime manager failed to connect node %s: %w", nodeID, err)
	}
	// Network.Connected doesn't use version parameter in this implementation
	return vm.Network.Connected(ctx, nodeID, nil)
}

func (vm *VM) Disconnected(ctx context.Context, nodeID ids.NodeID) error {
	vm.vmLock.Lock()
	defer vm.vmLock.Unlock()

	if err := vm.validatorsManager.Disconnect(nodeID); err != nil {
		return fmt.Errorf("uptime manager failed to disconnect node %s: %w", nodeID, err)
	}

	return vm.Network.Disconnected(ctx, nodeID)
}

// AppRequestFailed implements the VM interface
func (vm *VM) AppRequestFailed(ctx context.Context, nodeID ids.NodeID, requestID uint32, appErr *commonEng.AppError) error {
	// The Network interface doesn't expose AppRequestFailed directly
	// We need to handle this at the VM level by logging the error
	log.Debug("AppRequestFailed", "nodeID", nodeID, "requestID", requestID, "error", appErr)
	// The network's response handler will handle the timeout internally
	return nil
}

// CrossChainAppRequestFailed implements the VM interface
func (vm *VM) CrossChainAppRequestFailed(ctx context.Context, chainID ids.ID, requestID uint32, appErr *commonEng.AppError) error {
	// Cross-chain app requests are not currently supported
	// Just log and return nil to satisfy the interface
	log.Debug("CrossChainAppRequestFailed called", "chainID", chainID, "requestID", requestID, "error", appErr.Message)
	return nil
}

// StateSyncEnabled implements the StateSyncableVM interface
func (vm *VM) StateSyncEnabled(ctx context.Context) (bool, error) {
	return vm.config.StateSyncEnabled, nil
}

// GetOngoingSyncStateSummary implements the StateSyncableVM interface
func (vm *VM) GetOngoingSyncStateSummary(ctx context.Context) (nodeblock.StateSummary, error) {
	// TODO: Implement ongoing sync support
	return nil, database.ErrNotFound
}

// GetLastStateSummary implements the StateSyncableVM interface
func (vm *VM) GetLastStateSummary(ctx context.Context) (nodeblock.StateSummary, error) {
	summary, err := vm.StateSyncServer.GetLastStateSummary(ctx)
	if err != nil {
		return nil, err
	}
	// Cast to node's StateSummary interface
	return summary.(nodeblock.StateSummary), nil
}

// stateSummaryWrapper wraps message.SyncSummary to implement nodeblock.StateSummary
type stateSummaryWrapper struct {
	summary message.SyncSummary
}

func (s *stateSummaryWrapper) Accept(ctx context.Context) (nodeblock.StateSyncMode, error) {
	consensusMode, err := s.summary.Accept(ctx)
	if err != nil {
		return 0, err
	}
	// Convert consensus StateSyncMode to node StateSyncMode
	// The values should be the same, just different types
	return nodeblock.StateSyncMode(consensusMode), nil
}

func (s *stateSummaryWrapper) Bytes() []byte {
	return s.summary.Bytes()
}

func (s *stateSummaryWrapper) Height() uint64 {
	return s.summary.Height()
}

func (s *stateSummaryWrapper) ID() ids.ID {
	return s.summary.ID()
}

// ParseStateSummary implements the StateSyncableVM interface
func (vm *VM) ParseStateSummary(ctx context.Context, summaryBytes []byte) (nodeblock.StateSummary, error) {
	// Parse the summary bytes
	summary, err := message.NewSyncSummaryFromBytes(summaryBytes, nil)
	if err != nil {
		return nil, err
	}
	return &stateSummaryWrapper{summary: summary}, nil
}

// GetStateSummary implements the StateSyncableVM interface
func (vm *VM) GetStateSummary(ctx context.Context, height uint64) (nodeblock.StateSummary, error) {
	summary, err := vm.StateSyncServer.GetStateSummary(ctx, height)
	if err != nil {
		return nil, err
	}
	// Cast to node's StateSummary interface
	return summary.(nodeblock.StateSummary), nil
}

// warpVerifierAdapter adapts our warp.Backend to lp118.Verifier
type warpVerifierAdapter struct {
	backend warp.Backend
}

func (w *warpVerifierAdapter) Verify(ctx context.Context, msg *nodeWarp.UnsignedMessage, justification []byte) *nodeCommonEng.AppError {
	// Convert node warp message to consensus warp message
	luxMsg := &luxWarp.UnsignedMessage{
		NetworkID:     msg.NetworkID,
		SourceChainID: msg.SourceChainID,
		Payload:       msg.Payload,
	}

	if err := w.backend.Verify(ctx, luxMsg, justification); err != nil {
		return &nodeCommonEng.AppError{
			Code:    1,
			Message: err.Error(),
		}
	}
	return nil
}
