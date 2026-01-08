// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/http"
	"slices"
	"time"

	"github.com/luxfi/log"

	validators "github.com/luxfi/consensus/validator"
	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/api"
	"luxfi/cache/lru"
	"github.com/luxfi/const"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/sdk/utils/formatting"
	safemath "github.com/luxfi/math"
	"github.com/luxfi/sdk/node/vms/components/gas"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/platformvm/fx"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
	"github.com/luxfi/sdk/node/vms/platformvm/signer"
	"github.com/luxfi/sdk/node/vms/platformvm/stakeable"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/status"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/platformvm/validators/fee"
	"github.com/luxfi/sdk/node/vms/platformvm/warp/message"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/node/vms/types"

	avajson "github.com/luxfi/sdk/utils/json"
	platformapi "github.com/luxfi/sdk/node/vms/platformvm/api"
)

const (
	// Max number of addresses that can be passed in as argument to GetUTXOs
	maxGetUTXOsAddrs = 1024

	// Max number of addresses that can be passed in as argument to GetStake
	maxGetStakeAddrs = 256

	// Max number of items allowed in a page
	maxPageSize = 1024

	// Note: Staker attributes cache should be large enough so that no evictions
	// happen when the API loops through all stakers.
	stakerAttributesCacheSize = 100_000
)

var (
	errMissingDecisionBlock    = errors.New("should have a decision block within the past two blocks")
	errPrimaryNetworkIsNotANet = errors.New("the primary network isn't a net")
	errNoAddresses             = errors.New("no addresses provided")
	errMissingBlockchainID     = errors.New("argument 'blockchainID' not given")
)

// Service defines the API calls that can be made to the platform chain
type Service struct {
	vm                    *VM
	addrManager           lux.AddressManager
	stakerAttributesCache *lru.Cache[ids.ID, *stakerAttributes]
}

// All attributes are optional and may not be filled for each stakerTx.
type stakerAttributes struct {
	shares                 uint32
	rewardsOwner           fx.Owner
	validationRewardsOwner fx.Owner
	delegationRewardsOwner fx.Owner
	proofOfPossession      *signer.ProofOfPossession
}

// GetHeight returns the height of the last accepted block
func (s *Service) GetHeight(r *http.Request, _ *struct{}, response *api.GetHeightResponse) error {

	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "getHeight",
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	ctx := r.Context()
	height, err := s.vm.GetCurrentHeight(ctx)
	response.Height = avajson.Uint64(height)
	return err
}

// GetProposedHeight returns the current ProposerVM height
func (s *Service) GetProposedHeight(r *http.Request, _ *struct{}, reply *api.GetHeightResponse) error {

	s.vm.log.Debug("API called",
		log.String("service", "platform"),
		log.String("method", "getProposedHeight"),
	)
	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	lastAcceptedID := s.vm.state.GetLastAccepted()
	lastAcceptedBlock, err := s.vm.manager.GetStatelessBlock(lastAcceptedID)
	if err != nil {
		return err
	}
	reply.Height = avajson.Uint64(lastAcceptedBlock.Height())
	return nil
}

type GetBalanceRequest struct {
	Addresses []string `json:"addresses"`
}

// Note: We explicitly duplicate LUX out of the maps to ensure backwards
// compatibility.
type GetBalanceResponse struct {
	// Balance, in nLUX, of the address
	Balance             avajson.Uint64            `json:"balance"`
	Unlocked            avajson.Uint64            `json:"unlocked"`
	LockedStakeable     avajson.Uint64            `json:"lockedStakeable"`
	LockedNotStakeable  avajson.Uint64            `json:"lockedNotStakeable"`
	Balances            map[ids.ID]avajson.Uint64 `json:"balances"`
	Unlockeds           map[ids.ID]avajson.Uint64 `json:"unlockeds"`
	LockedStakeables    map[ids.ID]avajson.Uint64 `json:"lockedStakeables"`
	LockedNotStakeables map[ids.ID]avajson.Uint64 `json:"lockedNotStakeables"`
	UTXOIDs             []*lux.UTXOID             `json:"utxoIDs"`
}

// GetBalance gets the balance of an address
func (s *Service) GetBalance(_ *http.Request, args *GetBalanceRequest, response *GetBalanceResponse) error {

	s.vm.log.Debug("deprecated API called",
		"service", "platform",
		"method", "getBalance",
		"addresses", args.Addresses,
	)

	addrs, err := lux.ParseServiceAddresses(s.addrManager, args.Addresses)
	if err != nil {
		return err
	}

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	utxos, err := lux.GetAllUTXOs(s.vm.state, addrs)
	if err != nil {
		return fmt.Errorf("couldn't get UTXO set of %v: %w", args.Addresses, err)
	}

	currentTime := s.vm.nodeClock.Unix()

	unlockeds := map[ids.ID]uint64{}
	lockedStakeables := map[ids.ID]uint64{}
	lockedNotStakeables := map[ids.ID]uint64{}

utxoFor:
	for _, utxo := range utxos {
		assetID := utxo.AssetID()
		switch out := utxo.Out.(type) {
		case *secp256k1fx.TransferOutput:
			if out.Locktime <= currentTime {
				newBalance, err := safemath.Add(unlockeds[assetID], out.Amount())
				if err != nil {
					unlockeds[assetID] = math.MaxUint64
				} else {
					unlockeds[assetID] = newBalance
				}
			} else {
				newBalance, err := safemath.Add(lockedNotStakeables[assetID], out.Amount())
				if err != nil {
					lockedNotStakeables[assetID] = math.MaxUint64
				} else {
					lockedNotStakeables[assetID] = newBalance
				}
			}
		case *stakeable.LockOut:
			innerOut, ok := out.TransferableOut.(*secp256k1fx.TransferOutput)
			switch {
			case !ok:
				s.vm.log.Warn("unexpected output type in UTXO",
					"type", fmt.Sprintf("%T", out.TransferableOut),
				)
				continue utxoFor
			case innerOut.Locktime > currentTime:
				newBalance, err := safemath.Add(lockedNotStakeables[assetID], out.Amount())
				if err != nil {
					lockedNotStakeables[assetID] = math.MaxUint64
				} else {
					lockedNotStakeables[assetID] = newBalance
				}
			case out.Locktime <= currentTime:
				newBalance, err := safemath.Add(unlockeds[assetID], out.Amount())
				if err != nil {
					unlockeds[assetID] = math.MaxUint64
				} else {
					unlockeds[assetID] = newBalance
				}
			default:
				newBalance, err := safemath.Add(lockedStakeables[assetID], out.Amount())
				if err != nil {
					lockedStakeables[assetID] = math.MaxUint64
				} else {
					lockedStakeables[assetID] = newBalance
				}
			}
		default:
			continue utxoFor
		}

		response.UTXOIDs = append(response.UTXOIDs, &utxo.UTXOID)
	}

	balances := maps.Clone(lockedStakeables)
	for assetID, amount := range lockedNotStakeables {
		newBalance, err := safemath.Add(balances[assetID], amount)
		if err != nil {
			balances[assetID] = math.MaxUint64
		} else {
			balances[assetID] = newBalance
		}
	}
	for assetID, amount := range unlockeds {
		newBalance, err := safemath.Add(balances[assetID], amount)
		if err != nil {
			balances[assetID] = math.MaxUint64
		} else {
			balances[assetID] = newBalance
		}
	}

	response.Balances = newJSONBalanceMap(balances)
	response.Unlockeds = newJSONBalanceMap(unlockeds)
	response.LockedStakeables = newJSONBalanceMap(lockedStakeables)
	response.LockedNotStakeables = newJSONBalanceMap(lockedNotStakeables)
	response.Balance = response.Balances[s.vm.luxAssetID]
	response.Unlocked = response.Unlockeds[s.vm.luxAssetID]
	response.LockedStakeable = response.LockedStakeables[s.vm.luxAssetID]
	response.LockedNotStakeable = response.LockedNotStakeables[s.vm.luxAssetID]
	return nil
}

func newJSONBalanceMap(balanceMap map[ids.ID]uint64) map[ids.ID]avajson.Uint64 {
	jsonBalanceMap := make(map[ids.ID]avajson.Uint64, len(balanceMap))
	for assetID, amount := range balanceMap {
		jsonBalanceMap[assetID] = avajson.Uint64(amount)
	}
	return jsonBalanceMap
}

// Index is an address and an associated UTXO.
// Marks a starting or stopping point when fetching UTXOs. Used for pagination.
type Index struct {
	Address string `json:"address"` // The address as a string
	UTXO    string `json:"utxo"`    // The UTXO ID as a string
}

// GetUTXOs returns the UTXOs controlled by the given addresses
func (s *Service) GetUTXOs(_ *http.Request, args *api.GetUTXOsArgs, response *api.GetUTXOsReply) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "getUTXOs",
	)

	if len(args.Addresses) == 0 {
		return errNoAddresses
	}
	if len(args.Addresses) > maxGetUTXOsAddrs {
		return fmt.Errorf("number of addresses given, %d, exceeds maximum, %d", len(args.Addresses), maxGetUTXOsAddrs)
	}

	var sourceChain ids.ID
	if args.SourceChain == "" {
		sourceChain = s.vm.chainID
	} else {
		// Try to parse as ID first
		chainID, err := ids.FromString(args.SourceChain)
		if err != nil {
			// If not a valid ID, try as an alias
			// Note: bcLookup doesn't have Lookup method, would need reverse lookup
			// For now, just return error
			return fmt.Errorf("problem parsing source chainID %q: %w", args.SourceChain, err)
		}
		sourceChain = chainID
	}

	addrSet, err := lux.ParseServiceAddresses(s.addrManager, args.Addresses)
	if err != nil {
		return err
	}

	startAddr := ids.ShortEmpty
	startUTXO := ids.Empty
	if args.StartIndex.Address != "" || args.StartIndex.UTXO != "" {
		startAddr, err = lux.ParseServiceAddress(s.addrManager, args.StartIndex.Address)
		if err != nil {
			return fmt.Errorf("couldn't parse start index address %q: %w", args.StartIndex.Address, err)
		}
		startUTXO, err = ids.FromString(args.StartIndex.UTXO)
		if err != nil {
			return fmt.Errorf("couldn't parse start index utxo: %w", err)
		}
	}

	var (
		utxos     []*lux.UTXO
		endAddr   ids.ShortID
		endUTXOID ids.ID
	)
	limit := int(args.Limit)
	if limit <= 0 || maxPageSize < limit {
		limit = maxPageSize
	}

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	if sourceChain == s.vm.chainID {
		utxos, endAddr, endUTXOID, err = lux.GetPaginatedUTXOs(
			s.vm.state,
			addrSet,
			startAddr,
			startUTXO,
			limit,
		)
	} else {
		// For now, return empty results when shared memory is used
		utxos = []*lux.UTXO{}
		endAddr = ids.ShortEmpty
		endUTXOID = ids.Empty
		err = nil
	}
	if err != nil {
		return fmt.Errorf("problem retrieving UTXOs: %w", err)
	}

	response.UTXOs = make([]string, len(utxos))
	for i, utxo := range utxos {
		bytes, err := txs.Codec.Marshal(txs.CodecVersion, utxo)
		if err != nil {
			return fmt.Errorf("couldn't serialize UTXO %q: %w", utxo.InputID(), err)
		}
		response.UTXOs[i], err = formatting.Encode(args.Encoding, bytes)
		if err != nil {
			return fmt.Errorf("couldn't encode UTXO %s as %s: %w", utxo.InputID(), args.Encoding, err)
		}
	}

	endAddress, err := s.addrManager.FormatLocalAddress(endAddr)
	if err != nil {
		return fmt.Errorf("problem formatting address: %w", err)
	}

	response.EndIndex.Address = endAddress
	response.EndIndex.UTXO = endUTXOID.String()
	response.NumFetched = avajson.Uint64(len(utxos))
	response.Encoding = args.Encoding
	return nil
}

// GetNetArgs are the arguments to GetNet
type GetNetArgs struct {
	// ID of the net to retrieve information about
	NetID ids.ID `json:"netID"`
}

// GetNetResponse is the response from calling GetNet
type GetNetResponse struct {
	// whether it is permissioned or not
	IsPermissioned bool `json:"isPermissioned"`
	// net auth information for a permissioned net
	ControlKeys []string       `json:"controlKeys"`
	Threshold   avajson.Uint32 `json:"threshold"`
	Locktime    avajson.Uint64 `json:"locktime"`
	// net transformation tx ID for an elastic net
	NetTransformationTxID ids.ID `json:"netTransformationTxID"`
	// net conversion information for an L1
	ConversionID   ids.ID              `json:"conversionID"`
	ManagerChainID ids.ID              `json:"managerChainID"`
	ManagerAddress types.JSONByteSlice `json:"managerAddress"`
}

func (s *Service) GetNet(_ *http.Request, args *GetNetArgs, response *GetNetResponse) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "getNet",
		"netID", args.NetID,
	)

	if args.NetID == constants.PrimaryNetworkID {
		return errPrimaryNetworkIsNotANet
	}

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	netOwner, err := s.vm.state.GetNetOwner(args.NetID)
	if err != nil {
		return err
	}
	owner, ok := netOwner.(*secp256k1fx.OutputOwners)
	if !ok {
		return fmt.Errorf("expected *secp256k1fx.OutputOwners but got %T", netOwner)
	}
	controlAddrs := make([]string, len(owner.Addrs))
	for i, controlKeyID := range owner.Addrs {
		addr, err := s.addrManager.FormatLocalAddress(controlKeyID)
		if err != nil {
			return fmt.Errorf("problem formatting address: %w", err)
		}
		controlAddrs[i] = addr
	}

	response.ControlKeys = controlAddrs
	response.Threshold = avajson.Uint32(owner.Threshold)
	response.Locktime = avajson.Uint64(owner.Locktime)

	switch netTransformationTx, err := s.vm.state.GetNetTransformation(args.NetID); err {
	case nil:
		response.IsPermissioned = false
		response.NetTransformationTxID = netTransformationTx.ID()
	case database.ErrNotFound:
		response.IsPermissioned = true
		response.NetTransformationTxID = ids.Empty
	default:
		return err
	}

	switch c, err := s.vm.state.GetNetToL1Conversion(args.NetID); err {
	case nil:
		response.IsPermissioned = false
		response.ConversionID = c.ConversionID
		response.ManagerChainID = c.ChainID
		response.ManagerAddress = c.Addr
	case database.ErrNotFound:
		response.ConversionID = ids.Empty
		response.ManagerChainID = ids.Empty
		response.ManagerAddress = []byte(nil)
	default:
		return err
	}

	return nil
}

// APINet is a representation of a net used in API calls
type APINet struct {
	// ID of the net
	ID ids.ID `json:"id"`

	// Each element of [ControlKeys] the address of a public key.
	// A transaction to add a validator to this net requires
	// signatures from [Threshold] of these keys to be valid.
	ControlKeys []string       `json:"controlKeys"`
	Threshold   avajson.Uint32 `json:"threshold"`
}

// GetNetsArgs are the arguments to GetNets
type GetNetsArgs struct {
	// IDs of the nets to retrieve information about
	// If omitted, gets all nets
	IDs []ids.ID `json:"ids"`
}

// GetNetsResponse is the response from calling GetNets
type GetNetsResponse struct {
	// Each element is a net that exists
	// Null if there are no nets other than the primary network
	Nets []APINet `json:"nets"`
}

// GetNets returns the nets whose ID are in [args.IDs]
// The response will include the primary network
func (s *Service) GetNets(_ *http.Request, args *GetNetsArgs, response *GetNetsResponse) error {
	s.vm.log.Debug("deprecated API called",
		"service", "platform",
		"method", "getNets",
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	getAll := len(args.IDs) == 0
	if getAll {
		netIDs, err := s.vm.state.GetNetIDs() // all nets
		if err != nil {
			return fmt.Errorf("error getting nets from database: %w", err)
		}

		response.Nets = make([]APINet, len(netIDs)+1)
		for i, netID := range netIDs {
			if _, err := s.vm.state.GetNetTransformation(netID); err == nil {
				response.Nets[i] = APINet{
					ID:          netID,
					ControlKeys: []string{},
					Threshold:   avajson.Uint32(0),
				}
				continue
			}

			netOwner, err := s.vm.state.GetNetOwner(netID)
			if err != nil {
				return err
			}

			owner, ok := netOwner.(*secp256k1fx.OutputOwners)
			if !ok {
				return fmt.Errorf("expected *secp256k1fx.OutputOwners but got %T", netOwner)
			}

			controlAddrs := make([]string, len(owner.Addrs))
			for i, controlKeyID := range owner.Addrs {
				addr, err := s.addrManager.FormatLocalAddress(controlKeyID)
				if err != nil {
					return fmt.Errorf("problem formatting address: %w", err)
				}
				controlAddrs[i] = addr
			}
			response.Nets[i] = APINet{
				ID:          netID,
				ControlKeys: controlAddrs,
				Threshold:   avajson.Uint32(owner.Threshold),
			}
		}
		// Include primary network
		response.Nets[len(netIDs)] = APINet{
			ID:          constants.PrimaryNetworkID,
			ControlKeys: []string{},
			Threshold:   avajson.Uint32(0),
		}
		return nil
	}

	netSet := set.NewSet[ids.ID](len(args.IDs))
	for _, netID := range args.IDs {
		if netSet.Contains(netID) {
			continue
		}
		netSet.Add(netID)

		if netID == constants.PrimaryNetworkID {
			response.Nets = append(response.Nets,
				APINet{
					ID:          constants.PrimaryNetworkID,
					ControlKeys: []string{},
					Threshold:   avajson.Uint32(0),
				},
			)
			continue
		}

		if _, err := s.vm.state.GetNetTransformation(netID); err == nil {
			response.Nets = append(response.Nets, APINet{
				ID:          netID,
				ControlKeys: []string{},
				Threshold:   avajson.Uint32(0),
			})
			continue
		}

		netOwner, err := s.vm.state.GetNetOwner(netID)
		if err == database.ErrNotFound {
			continue
		}
		if err != nil {
			return err
		}

		owner, ok := netOwner.(*secp256k1fx.OutputOwners)
		if !ok {
			return fmt.Errorf("expected *secp256k1fx.OutputOwners but got %T", netOwner)
		}

		controlAddrs := make([]string, len(owner.Addrs))
		for i, controlKeyID := range owner.Addrs {
			addr, err := s.addrManager.FormatLocalAddress(controlKeyID)
			if err != nil {
				return fmt.Errorf("problem formatting address: %w", err)
			}
			controlAddrs[i] = addr
		}

		response.Nets = append(response.Nets, APINet{
			ID:          netID,
			ControlKeys: controlAddrs,
			Threshold:   avajson.Uint32(owner.Threshold),
		})
	}
	return nil
}

// GetStakingAssetIDArgs are the arguments to GetStakingAssetID
type GetStakingAssetIDArgs struct {
	NetID ids.ID `json:"netID"`
}

// GetStakingAssetIDResponse is the response from calling GetStakingAssetID
type GetStakingAssetIDResponse struct {
	AssetID ids.ID `json:"assetID"`
}

// GetStakingAssetID returns the assetID of the token used to stake on the
// provided net
func (s *Service) GetStakingAssetID(_ *http.Request, args *GetStakingAssetIDArgs, response *GetStakingAssetIDResponse) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "getStakingAssetID",
	)

	if args.NetID == constants.PrimaryNetworkID {
		response.AssetID = s.vm.luxAssetID
		return nil
	}

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	transformNetIntf, err := s.vm.state.GetNetTransformation(args.NetID)
	if err != nil {
		return fmt.Errorf(
			"failed fetching net transformation for %s: %w",
			args.NetID,
			err,
		)
	}
	transformNet, ok := transformNetIntf.Unsigned.(*txs.TransformChainTx)
	if !ok {
		return fmt.Errorf(
			"unexpected net transformation tx type fetched %T",
			transformNetIntf.Unsigned,
		)
	}

	response.AssetID = transformNet.AssetID
	return nil
}

// GetCurrentValidatorsArgs are the arguments for calling GetCurrentValidators
type GetCurrentValidatorsArgs struct {
	// Net we're listing the validators of
	// If omitted, defaults to primary network
	NetID ids.ID `json:"netID"`
	// NodeIDs of validators to request. If [NodeIDs]
	// is empty, it fetches all current validators. If
	// some nodeIDs are not currently validators, they
	// will be omitted from the response.
	NodeIDs []ids.NodeID `json:"nodeIDs"`
}

// GetCurrentValidatorsReply are the results from calling GetCurrentValidators.
// Each validator contains a list of delegators to itself.
type GetCurrentValidatorsReply struct {
	Validators []any `json:"validators"`
}

func (s *Service) loadStakerTxAttributes(txID ids.ID) (*stakerAttributes, error) {
	// Lookup tx from the cache first.
	attr, found := s.stakerAttributesCache.Get(txID)
	if found {
		return attr, nil
	}

	// Tx not available in cache; pull it from disk and populate the cache.
	tx, _, err := s.vm.state.GetTx(txID)
	if err != nil {
		return nil, err
	}

	switch stakerTx := tx.Unsigned.(type) {
	case txs.ValidatorTx:
		var pop *signer.ProofOfPossession
		if staker, ok := stakerTx.(*txs.AddPermissionlessValidatorTx); ok {
			if s, ok := staker.Signer.(*signer.ProofOfPossession); ok {
				pop = s
			}
		}

		attr = &stakerAttributes{
			shares:                 stakerTx.Shares(),
			validationRewardsOwner: stakerTx.ValidationRewardsOwner(),
			delegationRewardsOwner: stakerTx.DelegationRewardsOwner(),
			proofOfPossession:      pop,
		}

	case txs.DelegatorTx:
		attr = &stakerAttributes{
			rewardsOwner: stakerTx.RewardsOwner(),
		}

	default:
		return nil, fmt.Errorf("unexpected staker tx type %T", tx.Unsigned)
	}

	s.stakerAttributesCache.Put(txID, attr)
	return attr, nil
}

// GetCurrentValidators returns the current validators. If a single nodeID
// is provided, full delegators information is also returned. Otherwise only
// delegators' number and total weight is returned.
func (s *Service) GetCurrentValidators(request *http.Request, args *GetCurrentValidatorsArgs, reply *GetCurrentValidatorsReply) error {
	s.vm.log.Debug("API called",
		log.String("service", "platform"),
		log.String("method", "getCurrentValidators"),
	)

	// Create set of nodeIDs
	nodeIDs := set.Of(args.NodeIDs...)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	// Check if net is L1
	_, err := s.vm.state.GetNetToL1Conversion(args.NetID)
	if errors.Is(err, database.ErrNotFound) {
		// Net is not L1, get validators for the net
		reply.Validators, err = s.getPrimaryOrNetValidators(
			args.NetID,
			nodeIDs,
		)
		if err != nil {
			return fmt.Errorf("failed to get primary or net validators: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get net to L1 conversion: %w", err)
	}

	// Net is L1, get validators for L1
	reply.Validators, err = s.getL1Validators(
		request.Context(),
		args.NetID,
		nodeIDs,
	)
	if err != nil {
		return fmt.Errorf("failed to get L1 validators: %w", err)
	}
	return nil
}

func (s *Service) getL1Validators(
	ctx context.Context,
	netID ids.ID,
	nodeIDs set.Set[ids.NodeID],
) ([]any, error) {
	validators := []any{}
	baseStakers, l1Validators, _, err := s.vm.state.GetCurrentValidators(ctx, netID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current validators: %w", err)
	}

	fetchAll := nodeIDs.Len() == 0

	for _, staker := range baseStakers {
		if !fetchAll && !nodeIDs.Contains(staker.NodeID) {
			continue
		}

		apiStaker := toPlatformStaker(staker)
		validators = append(validators, apiStaker)
	}

	for _, l1Validator := range l1Validators {
		if !fetchAll && !nodeIDs.Contains(l1Validator.NodeID) {
			continue
		}

		apiL1Vdr, err := s.convertL1ValidatorToAPI(l1Validator)
		if err != nil {
			return nil, fmt.Errorf("converting L1 validator to API format: %w", err)
		}

		validators = append(validators, apiL1Vdr)
	}

	return validators, nil
}

func (s *Service) getPrimaryOrNetValidators(netID ids.ID, nodeIDs set.Set[ids.NodeID]) ([]any, error) {
	numNodeIDs := nodeIDs.Len()

	targetStakers := make([]*state.Staker, 0, numNodeIDs)

	// Validator's node ID as string --> Delegators to them
	vdrToDelegators := map[ids.NodeID][]platformapi.PrimaryDelegator{}

	validators := []any{}

	if numNodeIDs == 0 { // Include all nodes
		currentStakerIterator, err := s.vm.state.GetCurrentStakerIterator()
		if err != nil {
			return nil, err
		}
		for currentStakerIterator.Next() {
			staker := currentStakerIterator.Value()
			if netID != staker.ChainID {
				continue
			}
			targetStakers = append(targetStakers, staker)
		}
		currentStakerIterator.Release()
	} else {
		for nodeID := range nodeIDs {
			staker, err := s.vm.state.GetCurrentValidator(netID, nodeID)
			switch err {
			case nil:
			case database.ErrNotFound:
				// nothing to do, continue
				continue
			default:
				return nil, err
			}
			targetStakers = append(targetStakers, staker)

			// TODO: avoid iterating over delegators when numNodeIDs > 1.
			delegatorsIt, err := s.vm.state.GetCurrentDelegatorIterator(netID, nodeID)
			if err != nil {
				return nil, err
			}
			for delegatorsIt.Next() {
				staker := delegatorsIt.Value()
				targetStakers = append(targetStakers, staker)
			}
			delegatorsIt.Release()
		}
	}

	for _, currentStaker := range targetStakers {
		apiStaker := toPlatformStaker(currentStaker)
		potentialReward := avajson.Uint64(currentStaker.PotentialReward)

		delegateeReward, err := s.vm.state.GetDelegateeReward(currentStaker.ChainID, currentStaker.NodeID)
		if err != nil {
			return nil, err
		}
		jsonDelegateeReward := avajson.Uint64(delegateeReward)

		switch currentStaker.Priority {
		case txs.PrimaryNetworkValidatorCurrentPriority, txs.NetPermissionlessValidatorCurrentPriority:
			attr, err := s.loadStakerTxAttributes(currentStaker.TxID)
			if err != nil {
				return nil, err
			}

			shares := attr.shares
			delegationFee := avajson.Float32(100 * float32(shares) / float32(reward.PercentDenominator))
			var (
				uptime    *avajson.Float32
				connected *bool
			)
			if netID == constants.PrimaryNetworkID {
				rawUptime, err := s.vm.uptimeManager.CalculateUptimePercentFrom(currentStaker.NodeID, netID, currentStaker.StartTime)
				if err != nil {
					return nil, err
				}
				// Transform this to a percentage (0-100) to make it consistent
				// with observedUptime in info.peers API
				currentUptime := avajson.Float32(rawUptime * 100)
				if err != nil {
					return nil, err
				}
				// connected field left nil - IsConnected method no longer exists
				uptime = &currentUptime
			}

			var (
				validationRewardOwner *platformapi.Owner
				delegationRewardOwner *platformapi.Owner
			)
			validationOwner, ok := attr.validationRewardsOwner.(*secp256k1fx.OutputOwners)
			if ok {
				validationRewardOwner, err = s.getAPIOwner(validationOwner)
				if err != nil {
					return nil, err
				}
			}
			delegationOwner, ok := attr.delegationRewardsOwner.(*secp256k1fx.OutputOwners)
			if ok {
				delegationRewardOwner, err = s.getAPIOwner(delegationOwner)
				if err != nil {
					return nil, err
				}
			}

			vdr := platformapi.PermissionlessValidator{
				Staker:                 apiStaker,
				Uptime:                 uptime,
				Connected:              connected,
				PotentialReward:        &potentialReward,
				AccruedDelegateeReward: &jsonDelegateeReward,
				ValidationRewardOwner:  validationRewardOwner,
				DelegationRewardOwner:  delegationRewardOwner,
				DelegationFee:          delegationFee,
				Signer:                 attr.proofOfPossession,
			}
			validators = append(validators, vdr)

		case txs.PrimaryNetworkDelegatorCurrentPriority, txs.NetPermissionlessDelegatorCurrentPriority:
			var rewardOwner *platformapi.Owner
			// If we are handling multiple nodeIDs, we don't return the
			// delegator information.
			if numNodeIDs == 1 {
				attr, err := s.loadStakerTxAttributes(currentStaker.TxID)
				if err != nil {
					return nil, err
				}
				owner, ok := attr.rewardsOwner.(*secp256k1fx.OutputOwners)
				if ok {
					rewardOwner, err = s.getAPIOwner(owner)
					if err != nil {
						return nil, err
					}
				}
			}

			delegator := platformapi.PrimaryDelegator{
				Staker:          apiStaker,
				RewardOwner:     rewardOwner,
				PotentialReward: &potentialReward,
			}
			vdrToDelegators[delegator.NodeID] = append(vdrToDelegators[delegator.NodeID], delegator)

		case txs.NetPermissionedValidatorCurrentPriority:
			validators = append(validators, apiStaker)

		default:
			return nil, fmt.Errorf("unexpected staker priority %d", currentStaker.Priority)
		}
	}

	// handle delegators' information
	for i, vdrIntf := range validators {
		vdr, ok := vdrIntf.(platformapi.PermissionlessValidator)
		if !ok {
			continue
		}
		delegators, ok := vdrToDelegators[vdr.NodeID]
		if !ok {
			// If we are expected to populate the delegators field, we should
			// always return a non-nil value.
			delegators = []platformapi.PrimaryDelegator{}
		}
		delegatorCount := avajson.Uint64(len(delegators))
		delegatorWeight := avajson.Uint64(0)
		for _, d := range delegators {
			delegatorWeight += d.Weight
		}

		vdr.DelegatorCount = &delegatorCount
		vdr.DelegatorWeight = &delegatorWeight

		if numNodeIDs == 1 {
			// queried a specific validator, load all of its delegators
			vdr.Delegators = &delegators
		}
		validators[i] = vdr
	}

	return validators, nil
}

type GetL1ValidatorArgs struct {
	ValidationID ids.ID `json:"validationID"`
}

type GetL1ValidatorReply struct {
	platformapi.APIL1Validator
	NetID ids.ID `json:"netID"`
	// Height is the height of the last accepted block
	Height avajson.Uint64 `json:"height"`
}

// GetL1Validator returns the L1 validator if it exists
func (s *Service) GetL1Validator(r *http.Request, args *GetL1ValidatorArgs, reply *GetL1ValidatorReply) error {
	s.vm.log.Debug("API called",
		log.String("service", "platform"),
		log.String("method", "getL1Validator"),
		log.Stringer("validationID", args.ValidationID),
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	l1Validator, err := s.vm.state.GetL1Validator(args.ValidationID)
	if err != nil {
		return fmt.Errorf("fetching L1 validator %q failed: %w", args.ValidationID, err)
	}

	ctx := r.Context()
	height, err := s.vm.GetCurrentHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the current height: %w", err)
	}
	apiVdr, err := s.convertL1ValidatorToAPI(l1Validator)
	if err != nil {
		return fmt.Errorf("failed to convert L1 validator to API format: %w", err)
	}

	reply.APIL1Validator = apiVdr
	reply.NetID = l1Validator.ChainID
	reply.Height = avajson.Uint64(height)
	return nil
}

func (s *Service) convertL1ValidatorToAPI(vdr state.L1Validator) (platformapi.APIL1Validator, error) {
	var remainingBalanceOwner message.PChainOwner
	if _, err := txs.Codec.Unmarshal(vdr.RemainingBalanceOwner, &remainingBalanceOwner); err != nil {
		return platformapi.APIL1Validator{}, fmt.Errorf("failed unmarshalling remaining balance owner: %w", err)
	}
	remainingBalanceAPIOwner, err := s.getAPIOwner(&secp256k1fx.OutputOwners{
		Threshold: remainingBalanceOwner.Threshold,
		Addrs:     remainingBalanceOwner.Addresses,
	})
	if err != nil {
		return platformapi.APIL1Validator{}, fmt.Errorf("failed formatting remaining balance owner: %w", err)
	}

	var deactivationOwner message.PChainOwner
	if _, err := txs.Codec.Unmarshal(vdr.DeactivationOwner, &deactivationOwner); err != nil {
		return platformapi.APIL1Validator{}, fmt.Errorf("failed unmarshalling deactivation owner: %w", err)
	}
	deactivationAPIOwner, err := s.getAPIOwner(&secp256k1fx.OutputOwners{
		Threshold: deactivationOwner.Threshold,
		Addrs:     deactivationOwner.Addresses,
	})
	if err != nil {
		return platformapi.APIL1Validator{}, fmt.Errorf("failed formatting deactivation owner: %w", err)
	}

	pubKey := types.JSONByteSlice(bls.PublicKeyToCompressedBytes(
		bls.PublicKeyFromValidUncompressedBytes(vdr.PublicKey),
	))
	minNonce := avajson.Uint64(vdr.MinNonce)

	apiVdr := platformapi.APIL1Validator{
		NodeID:    vdr.NodeID,
		StartTime: avajson.Uint64(vdr.StartTime),
		Weight:    avajson.Uint64(vdr.Weight),
		BaseL1Validator: platformapi.BaseL1Validator{
			ValidationID:          &vdr.ValidationID,
			PublicKey:             &pubKey,
			RemainingBalanceOwner: remainingBalanceAPIOwner,
			DeactivationOwner:     deactivationAPIOwner,
			MinNonce:              &minNonce,
		},
	}
	zero := avajson.Uint64(0)
	apiVdr.Balance = &zero
	if vdr.EndAccumulatedFee != 0 {
		accruedFees := s.vm.state.GetAccruedFees()
		balance := avajson.Uint64(vdr.EndAccumulatedFee - accruedFees)
		apiVdr.Balance = &balance
	}
	return apiVdr, nil
}

// GetCurrentSupplyArgs are the arguments for calling GetCurrentSupply
type GetCurrentSupplyArgs struct {
	NetID ids.ID `json:"netID"`
}

// GetCurrentSupplyReply are the results from calling GetCurrentSupply
type GetCurrentSupplyReply struct {
	Supply avajson.Uint64 `json:"supply"`
	Height avajson.Uint64 `json:"height"`
}

// GetCurrentSupply returns an upper bound on the supply of LUX in the system
func (s *Service) GetCurrentSupply(r *http.Request, args *GetCurrentSupplyArgs, reply *GetCurrentSupplyReply) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "getCurrentSupply",
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	supply, err := s.vm.state.GetCurrentSupply(args.NetID)
	if err != nil {
		return fmt.Errorf("fetching current supply failed: %w", err)
	}
	reply.Supply = avajson.Uint64(supply)

	ctx := r.Context()
	height, err := s.vm.GetCurrentHeight(ctx)
	if err != nil {
		return fmt.Errorf("fetching current height failed: %w", err)
	}
	reply.Height = avajson.Uint64(height)

	return nil
}

// SampleValidatorsArgs are the arguments for calling SampleValidators
type SampleValidatorsArgs struct {
	// Number of validators in the sample
	Size avajson.Uint16 `json:"size"`

	// ID of net to sample validators from
	// If omitted, defaults to the primary network
	NetID ids.ID `json:"netID"`
}

// SampleValidatorsReply are the results from calling Sample
type SampleValidatorsReply struct {
	Validators []ids.NodeID `json:"validators"`
}

// SampleValidators returns a sampling of the list of current validators
func (s *Service) SampleValidators(_ *http.Request, args *SampleValidatorsArgs, reply *SampleValidatorsReply) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "sampleValidators",
		"size", uint16(args.Size),
	)

	// Sample is not available in consensus validators.Manager
	// For now, return empty list
	// TODO: Implement sampling when validators.Manager is properly integrated
	// sample, err := s.vm.Validators.Sample(args.NetID, int(args.Size))
	// if err != nil {
	// 	return fmt.Errorf("sampling %s errored with %w", args.NetID, err)
	// }

	reply.Validators = []ids.NodeID{}
	return nil
}

// GetBlockchainStatusArgs is the arguments for calling GetBlockchainStatus
// [BlockchainID] is the ID of or an alias of the blockchain to get the status of.
type GetBlockchainStatusArgs struct {
	BlockchainID string `json:"blockchainID"`
}

// GetBlockchainStatusReply is the reply from calling GetBlockchainStatus
// [Status] is the blockchain's status.
type GetBlockchainStatusReply struct {
	Status status.BlockchainStatus `json:"status"`
}

// GetBlockchainStatus gets the status of a blockchain with the ID [args.BlockchainID].
func (s *Service) GetBlockchainStatus(r *http.Request, args *GetBlockchainStatusArgs, reply *GetBlockchainStatusReply) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "getBlockchainStatus",
	)

	if args.BlockchainID == "" {
		return errMissingBlockchainID
	}

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	// if its aliased then vm created this chain.
	if aliasedID, err := s.vm.Chains.Lookup(args.BlockchainID); err == nil {
		if s.nodeValidates(aliasedID) {
			reply.Status = status.Validating
			return nil
		}

		reply.Status = status.Syncing
		return nil
	}

	blockchainID, err := ids.FromString(args.BlockchainID)
	if err != nil {
		return fmt.Errorf("problem parsing blockchainID %q: %w", args.BlockchainID, err)
	}

	ctx := r.Context()
	lastAcceptedID, err := s.vm.LastAccepted(ctx)
	if err != nil {
		return fmt.Errorf("problem loading last accepted ID: %w", err)
	}

	exists, err := s.chainExists(ctx, lastAcceptedID, blockchainID)
	if err != nil {
		return fmt.Errorf("problem looking up blockchain: %w", err)
	}
	if exists {
		reply.Status = status.Created
		return nil
	}

	preferredBlkID := s.vm.manager.Preferred()
	preferred, err := s.chainExists(ctx, preferredBlkID, blockchainID)
	if err != nil {
		return fmt.Errorf("problem looking up blockchain: %w", err)
	}
	if preferred {
		reply.Status = status.Preferred
	} else {
		reply.Status = status.UnknownChain
	}
	return nil
}

func (s *Service) nodeValidates(blockchainID ids.ID) bool {
	chainTx, _, err := s.vm.state.GetTx(blockchainID)
	if err != nil {
		return false
	}

	chain, ok := chainTx.Unsigned.(*txs.CreateChainTx)
	if !ok {
		return false
	}

	_, isValidator := s.vm.Validators.GetValidator(chain.ChainID, s.vm.nodeID)
	return isValidator
}

func (s *Service) chainExists(ctx context.Context, blockID ids.ID, chainID ids.ID) (bool, error) {
	state, ok := s.vm.manager.GetState(blockID)
	if !ok {
		block, err := s.vm.GetBlock(ctx, blockID)
		if err != nil {
			return false, err
		}
		state, ok = s.vm.manager.GetState(block.Parent())
		if !ok {
			return false, errMissingDecisionBlock
		}
	}

	tx, _, err := state.GetTx(chainID)
	if err == database.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, ok = tx.Unsigned.(*txs.CreateChainTx)
	return ok, nil
}

// ValidatedByArgs is the arguments for calling ValidatedBy
type ValidatedByArgs struct {
	// ValidatedBy returns the ID of the Net validating the blockchain with this ID
	BlockchainID ids.ID `json:"blockchainID"`
}

// ValidatedByResponse is the reply from calling ValidatedBy
type ValidatedByResponse struct {
	// ID of the Net validating the specified blockchain
	NetID ids.ID `json:"netID"`
}

// ValidatedBy returns the ID of the Net that validates [args.BlockchainID]
func (s *Service) ValidatedBy(r *http.Request, args *ValidatedByArgs, response *ValidatedByResponse) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "validatedBy",
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	// GetNetID is not available in the current validators.Manager interface
	// Return primary network for now
	response.NetID = constants.PrimaryNetworkID
	return nil
}

// ValidatesArgs are the arguments to Validates
type ValidatesArgs struct {
	NetID ids.ID `json:"netID"`
}

// ValidatesResponse is the response from calling Validates
type ValidatesResponse struct {
	BlockchainIDs []ids.ID `json:"blockchainIDs"`
}

// Validates returns the IDs of the blockchains validated by [args.NetID]
func (s *Service) Validates(_ *http.Request, args *ValidatesArgs, response *ValidatesResponse) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "validates",
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	if args.NetID != constants.PrimaryNetworkID {
		netTx, _, err := s.vm.state.GetTx(args.NetID)
		if err != nil {
			return fmt.Errorf(
				"problem retrieving net %q: %w",
				args.NetID,
				err,
			)
		}
		_, ok := netTx.Unsigned.(*txs.CreateSubnetTx)
		if !ok {
			return fmt.Errorf("%q is not a net", args.NetID)
		}
	}

	// Get the chains that exist
	chains, err := s.vm.state.GetChains(args.NetID)
	if err != nil {
		return fmt.Errorf("problem retrieving chains for net %q: %w", args.NetID, err)
	}

	response.BlockchainIDs = make([]ids.ID, len(chains))
	for i, chain := range chains {
		response.BlockchainIDs[i] = chain.ID()
	}
	return nil
}

// APIBlockchain is the representation of a blockchain used in API calls
type APIBlockchain struct {
	// Blockchain's ID
	ID ids.ID `json:"id"`

	// Blockchain's (non-unique) human-readable name
	Name string `json:"name"`

	// Net that validates the blockchain
	NetID ids.ID `json:"netID"`

	// Virtual Machine the blockchain runs
	VMID ids.ID `json:"vmID"`
}

// GetBlockchainsResponse is the response from a call to GetBlockchains
type GetBlockchainsResponse struct {
	// blockchains that exist
	Blockchains []APIBlockchain `json:"blockchains"`
}

// GetBlockchains returns all of the blockchains that exist
func (s *Service) GetBlockchains(_ *http.Request, _ *struct{}, response *GetBlockchainsResponse) error {
	s.vm.log.Debug("deprecated API called",
		"service", "platform",
		"method", "getBlockchains",
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	netIDs, err := s.vm.state.GetNetIDs()
	if err != nil {
		return fmt.Errorf("couldn't retrieve nets: %w", err)
	}

	response.Blockchains = []APIBlockchain{}
	for _, netID := range netIDs {
		chains, err := s.vm.state.GetChains(netID)
		if err != nil {
			return fmt.Errorf(
				"couldn't retrieve chains for net %q: %w",
				netID,
				err,
			)
		}

		for _, chainTx := range chains {
			chainID := chainTx.ID()
			chain, ok := chainTx.Unsigned.(*txs.CreateChainTx)
			if !ok {
				return fmt.Errorf("expected tx type *txs.CreateChainTx but got %T", chainTx.Unsigned)
			}
			response.Blockchains = append(response.Blockchains, APIBlockchain{
				ID:    chainID,
				Name:  chain.BlockchainName,
				NetID: netID,
				VMID:  chain.VMID,
			})
		}
	}

	chains, err := s.vm.state.GetChains(constants.PrimaryNetworkID)
	if err != nil {
		return fmt.Errorf("couldn't retrieve nets: %w", err)
	}
	for _, chainTx := range chains {
		chainID := chainTx.ID()
		chain, ok := chainTx.Unsigned.(*txs.CreateChainTx)
		if !ok {
			return fmt.Errorf("expected tx type *txs.CreateChainTx but got %T", chainTx.Unsigned)
		}
		response.Blockchains = append(response.Blockchains, APIBlockchain{
			ID:    chainID,
			Name:  chain.BlockchainName,
			NetID: constants.PrimaryNetworkID,
			VMID:  chain.VMID,
		})
	}

	return nil
}

func (s *Service) IssueTx(_ *http.Request, args *api.FormattedTx, response *api.JSONTxID) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "issueTx",
	)

	txBytes, err := formatting.Decode(args.Encoding, args.Tx)
	if err != nil {
		return fmt.Errorf("problem decoding transaction: %w", err)
	}
	tx, err := txs.Parse(txs.Codec, txBytes)
	if err != nil {
		return fmt.Errorf("couldn't parse tx: %w", err)
	}

	if err := s.vm.issueTxFromRPC(tx); err != nil {
		return fmt.Errorf("couldn't issue tx: %w", err)
	}

	response.TxID = tx.ID()
	return nil
}

func (s *Service) GetTx(_ *http.Request, args *api.GetTxArgs, response *api.GetTxReply) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "getTx",
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	tx, _, err := s.vm.state.GetTx(args.TxID)
	if err != nil {
		return fmt.Errorf("couldn't get tx: %w", err)
	}
	response.Encoding = args.Encoding

	var result any
	if args.Encoding == formatting.JSON {
		tx.Unsigned.InitCtx(s.vm.ctx)
		result = tx
	} else {
		result, err = formatting.Encode(args.Encoding, tx.Bytes())
		if err != nil {
			return fmt.Errorf("couldn't encode tx as %s: %w", args.Encoding, err)
		}
	}

	response.Tx, err = json.Marshal(result)
	return err
}

type GetTxStatusArgs struct {
	TxID ids.ID `json:"txID"`
}

type GetTxStatusResponse struct {
	Status status.Status `json:"status"`
	// Reason this tx was dropped.
	// Only non-empty if Status is dropped
	Reason string `json:"reason,omitempty"`
}

// GetTxStatus gets a tx's status
func (s *Service) GetTxStatus(_ *http.Request, args *GetTxStatusArgs, response *GetTxStatusResponse) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "getTxStatus",
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	_, txStatus, err := s.vm.state.GetTx(args.TxID)
	if err == nil { // Found the status. Report it.
		response.Status = txStatus
		return nil
	}
	if err != database.ErrNotFound {
		return err
	}

	// The status of this transaction is not in the database - check if the tx
	// is in the preferred block's db. If so, return that it's processing.
	preferredID := s.vm.manager.Preferred()
	onAccept, ok := s.vm.manager.GetState(preferredID)
	if !ok {
		return fmt.Errorf("could not retrieve state for block %s", preferredID)
	}

	_, _, err = onAccept.GetTx(args.TxID)
	if err == nil {
		// Found the status in the preferred block's db. Report tx is processing.
		response.Status = status.Processing
		return nil
	}
	if err != database.ErrNotFound {
		return err
	}

	if _, ok := s.vm.Builder.Get(args.TxID); ok {
		// Found the tx in the mempool. Report tx is processing.
		response.Status = status.Processing
		return nil
	}

	// Note: we check if tx is dropped only after having looked for it
	// in the database and the mempool, because dropped txs may be re-issued.
	reason := s.vm.Builder.GetDropReason(args.TxID)
	if reason == nil {
		// The tx isn't being tracked by the node.
		response.Status = status.Unknown
		return nil
	}

	// The tx was recently dropped because it was invalid.
	response.Status = status.Dropped
	response.Reason = reason.Error()
	return nil
}

type GetStakeArgs struct {
	api.JSONAddresses
	ValidatorsOnly bool                `json:"validatorsOnly"`
	Encoding       formatting.Encoding `json:"encoding"`
}

// GetStakeReply is the response from calling GetStake.
type GetStakeReply struct {
	Staked  avajson.Uint64            `json:"staked"`
	Stakeds map[ids.ID]avajson.Uint64 `json:"stakeds"`
	// String representation of staked outputs
	// Each is of type lux.TransferableOutput
	Outputs []string `json:"stakedOutputs"`
	// Encoding of [Outputs]
	Encoding formatting.Encoding `json:"encoding"`
}

// GetStake returns the amount of nLUX that [args.Addresses] have cumulatively
// staked on the Primary Network.
//
// This method assumes that each stake output has only owner
// This method assumes only LUX can be staked
// This method only concerns itself with the Primary Network, not nets
// in a data structure rather than re-calculating it by iterating over stakers
func (s *Service) GetStake(_ *http.Request, args *GetStakeArgs, response *GetStakeReply) error {
	s.vm.log.Debug("deprecated API called",
		"service", "platform",
		"method", "getStake",
	)

	if len(args.Addresses) > maxGetStakeAddrs {
		return fmt.Errorf("%d addresses provided but this method can take at most %d", len(args.Addresses), maxGetStakeAddrs)
	}

	addrs, err := lux.ParseServiceAddresses(s.addrManager, args.Addresses)
	if err != nil {
		return err
	}

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	currentStakerIterator, err := s.vm.state.GetCurrentStakerIterator()
	if err != nil {
		return err
	}
	defer currentStakerIterator.Release()

	var (
		totalAmountStaked = make(map[ids.ID]uint64)
		stakedOuts        []lux.TransferableOutput
	)
	for currentStakerIterator.Next() { // Iterates over current stakers
		staker := currentStakerIterator.Value()

		if args.ValidatorsOnly && !staker.Priority.IsValidator() {
			continue
		}

		tx, _, err := s.vm.state.GetTx(staker.TxID)
		if err != nil {
			return err
		}

		stakedOuts = append(stakedOuts, getStakeHelper(tx, addrs, totalAmountStaked)...)
	}

	pendingStakerIterator, err := s.vm.state.GetPendingStakerIterator()
	if err != nil {
		return err
	}
	defer pendingStakerIterator.Release()

	for pendingStakerIterator.Next() { // Iterates over pending stakers
		staker := pendingStakerIterator.Value()

		if args.ValidatorsOnly && !staker.Priority.IsValidator() {
			continue
		}

		tx, _, err := s.vm.state.GetTx(staker.TxID)
		if err != nil {
			return err
		}

		stakedOuts = append(stakedOuts, getStakeHelper(tx, addrs, totalAmountStaked)...)
	}

	response.Stakeds = newJSONBalanceMap(totalAmountStaked)
	response.Staked = response.Stakeds[s.vm.luxAssetID]
	response.Outputs = make([]string, len(stakedOuts))
	for i, output := range stakedOuts {
		bytes, err := txs.Codec.Marshal(txs.CodecVersion, output)
		if err != nil {
			return fmt.Errorf("couldn't serialize output %s: %w", output.ID, err)
		}
		response.Outputs[i], err = formatting.Encode(args.Encoding, bytes)
		if err != nil {
			return fmt.Errorf("couldn't encode output %s as %s: %w", output.ID, args.Encoding, err)
		}
	}
	response.Encoding = args.Encoding

	return nil
}

// GetMinStakeArgs are the arguments for calling GetMinStake.
type GetMinStakeArgs struct {
	NetID ids.ID `json:"netID"`
}

// GetMinStakeReply is the response from calling GetMinStake.
type GetMinStakeReply struct {
	//  The minimum amount of tokens one must bond to be a validator
	MinValidatorStake avajson.Uint64 `json:"minValidatorStake"`
	// Minimum stake, in nLUX, that can be delegated on the primary network
	MinDelegatorStake avajson.Uint64 `json:"minDelegatorStake"`
}

// GetMinStake returns the minimum staking amount in nLUX.
func (s *Service) GetMinStake(_ *http.Request, args *GetMinStakeArgs, reply *GetMinStakeReply) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "getMinStake",
	)

	if args.NetID == constants.PrimaryNetworkID {
		reply.MinValidatorStake = avajson.Uint64(s.vm.MinValidatorStake)
		reply.MinDelegatorStake = avajson.Uint64(s.vm.MinDelegatorStake)
		return nil
	}

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	transformNetIntf, err := s.vm.state.GetNetTransformation(args.NetID)
	if err != nil {
		return fmt.Errorf(
			"failed fetching net transformation for %s: %w",
			args.NetID,
			err,
		)
	}
	transformNet, ok := transformNetIntf.Unsigned.(*txs.TransformChainTx)
	if !ok {
		return fmt.Errorf(
			"unexpected net transformation tx type fetched %T",
			transformNetIntf.Unsigned,
		)
	}

	reply.MinValidatorStake = avajson.Uint64(transformNet.MinValidatorStake)
	reply.MinDelegatorStake = avajson.Uint64(transformNet.MinDelegatorStake)

	return nil
}

// GetTotalStakeArgs are the arguments for calling GetTotalStake
type GetTotalStakeArgs struct {
	// Net we're getting the total stake
	// If omitted returns Primary network weight
	NetID ids.ID `json:"netID"`
}

// GetTotalStakeReply is the response from calling GetTotalStake.
type GetTotalStakeReply struct {
	// Deprecated: Use Weight instead.
	Stake avajson.Uint64 `json:"stake"`

	Weight avajson.Uint64 `json:"weight"`
}

// GetTotalStake returns the total amount staked on the Primary Network
func (s *Service) GetTotalStake(_ *http.Request, args *GetTotalStakeArgs, reply *GetTotalStakeReply) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "getTotalStake",
	)

	totalWeight, err := s.vm.Validators.TotalWeight(args.NetID)
	if err != nil {
		return fmt.Errorf("couldn't get total weight: %w", err)
	}
	weight := avajson.Uint64(totalWeight)
	reply.Weight = weight
	reply.Stake = weight
	return nil
}

// GetRewardUTXOsReply defines the GetRewardUTXOs replies returned from the API
type GetRewardUTXOsReply struct {
	// Number of UTXOs returned
	NumFetched avajson.Uint64 `json:"numFetched"`
	// The UTXOs
	UTXOs []string `json:"utxos"`
	// Encoding specifies the encoding format the UTXOs are returned in
	Encoding formatting.Encoding `json:"encoding"`
}

// GetRewardUTXOs returns the UTXOs that were rewarded after the provided
// transaction's staking period ended.
func (s *Service) GetRewardUTXOs(_ *http.Request, args *api.GetTxArgs, reply *GetRewardUTXOsReply) error {
	s.vm.log.Debug("deprecated API called",
		"service", "platform",
		"method", "getRewardUTXOs",
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	utxos, err := s.vm.state.GetRewardUTXOs(args.TxID)
	if err != nil {
		return fmt.Errorf("couldn't get reward UTXOs: %w", err)
	}

	reply.NumFetched = avajson.Uint64(len(utxos))
	reply.UTXOs = make([]string, len(utxos))
	for i, utxo := range utxos {
		utxoBytes, err := txs.GenesisCodec.Marshal(txs.CodecVersion, utxo)
		if err != nil {
			return fmt.Errorf("couldn't encode UTXO to bytes: %w", err)
		}

		utxoStr, err := formatting.Encode(args.Encoding, utxoBytes)
		if err != nil {
			return fmt.Errorf("couldn't encode utxo as %s: %w", args.Encoding, err)
		}
		reply.UTXOs[i] = utxoStr
	}
	reply.Encoding = args.Encoding
	return nil
}

// GetTimestampReply is the response from GetTimestamp
type GetTimestampReply struct {
	// Current timestamp
	Timestamp time.Time `json:"timestamp"`
}

// GetTimestamp returns the current timestamp on chain.
func (s *Service) GetTimestamp(_ *http.Request, _ *struct{}, reply *GetTimestampReply) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "getTimestamp",
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	reply.Timestamp = s.vm.state.GetTimestamp()
	return nil
}

// GetValidatorsAtArgs is the response from GetValidatorsAt
type GetValidatorsAtArgs struct {
	Height platformapi.Height `json:"height"`
	NetID  ids.ID             `json:"netID"`
}

type jsonGetValidatorOutput struct {
	PublicKey *string        `json:"publicKey"`
	Weight    avajson.Uint64 `json:"weight"`
}

func (v *GetValidatorsAtReply) MarshalJSON() ([]byte, error) {
	m := make(map[ids.NodeID]*jsonGetValidatorOutput, len(v.Validators))
	for _, vdr := range v.Validators {
		vdrJSON := &jsonGetValidatorOutput{
			Weight: avajson.Uint64(vdr.Weight),
		}

		if vdr.PublicKey != nil {
			pk, err := formatting.Encode(formatting.HexNC, vdr.PublicKey)
			if err != nil {
				return nil, err
			}
			vdrJSON.PublicKey = &pk
		}

		m[vdr.NodeID] = vdrJSON
	}
	return json.Marshal(m)
}

func (v *GetValidatorsAtReply) UnmarshalJSON(b []byte) error {
	var m map[ids.NodeID]*jsonGetValidatorOutput
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	if m == nil {
		v.Validators = nil
		return nil
	}

	v.Validators = make(map[ids.NodeID]*validators.GetValidatorOutput, len(m))
	for nodeID, vdrJSON := range m {
		vdr := &validators.GetValidatorOutput{
			NodeID: nodeID,
			Weight: uint64(vdrJSON.Weight),
		}

		if vdrJSON.PublicKey != nil {
			pkBytes, err := formatting.Decode(formatting.HexNC, *vdrJSON.PublicKey)
			if err != nil {
				return err
			}
			vdr.PublicKey = pkBytes
		}

		v.Validators[nodeID] = vdr
	}
	return nil
}

// GetValidatorsAtReply is the response from GetValidatorsAt
type GetValidatorsAtReply struct {
	Validators map[ids.NodeID]*validators.GetValidatorOutput
}

// GetValidatorsAt returns the weights of the validator set of a provided net
// at the specified height.
func (s *Service) GetValidatorsAt(r *http.Request, args *GetValidatorsAtArgs, reply *GetValidatorsAtReply) error {
	s.vm.log.Debug("API called",
		log.String("service", "platform"),
		log.String("method", "getValidatorsAt"),
		log.Uint64("height", uint64(args.Height)),
		log.Bool("isProposed", args.Height.IsProposed()),
		log.Stringer("netID", args.NetID),
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	ctx := r.Context()
	var err error
	height := uint64(args.Height)
	if args.Height.IsProposed() {
		// Get the proposed height from the last accepted block
		lastAcceptedID := s.vm.state.GetLastAccepted()
		lastAcceptedBlock, err := s.vm.manager.GetStatelessBlock(lastAcceptedID)
		if err != nil {
			return fmt.Errorf("failed to get last accepted block: %w", err)
		}
		height = lastAcceptedBlock.Height()
	}

	reply.Validators, err = s.vm.GetValidatorSet(ctx, height, args.NetID)
	if err != nil {
		return fmt.Errorf("failed to get validator set: %w", err)
	}
	return nil
}

// GetAllValidatorsAtArgs are the arguments to GetAllValidatorsAt
type GetAllValidatorsAtArgs struct {
	Height platformapi.Height `json:"height"`
}

// GetAllValidatorsAtReply is the response from GetAllValidatorsAt
type GetAllValidatorsAtReply struct {
	// Map of NetID -> ValidatorSet
	ValidatorSets map[ids.ID]map[ids.NodeID]*validators.GetValidatorOutput `json:"validatorSets"`
}

// GetAllValidatorsAt returns the validator sets of all nets (including primary network)
// at the specified height.
func (s *Service) GetAllValidatorsAt(r *http.Request, args *GetAllValidatorsAtArgs, reply *GetAllValidatorsAtReply) error {
	s.vm.log.Debug("API called",
		log.String("service", "platform"),
		log.String("method", "getAllValidatorsAt"),
		log.Uint64("height", uint64(args.Height)),
		log.Bool("isProposed", args.Height.IsProposed()),
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	ctx := r.Context()
	height := uint64(args.Height)
	if args.Height.IsProposed() {
		// Get the proposed height from the last accepted block
		lastAcceptedID := s.vm.state.GetLastAccepted()
		lastAcceptedBlock, err := s.vm.manager.GetStatelessBlock(lastAcceptedID)
		if err != nil {
			return fmt.Errorf("failed to get last accepted block: %w", err)
		}
		height = lastAcceptedBlock.Height()
	}

	// Get all net IDs
	netIDs, err := s.vm.state.GetNetIDs()
	if err != nil {
		return fmt.Errorf("failed to get net IDs: %w", err)
	}

	// Initialize the result map
	reply.ValidatorSets = make(map[ids.ID]map[ids.NodeID]*validators.GetValidatorOutput)

	// Add primary network first
	primaryValidators, err := s.vm.GetValidatorSet(ctx, height, constants.PrimaryNetworkID)
	if err != nil {
		return fmt.Errorf("failed to get primary network validator set: %w", err)
	}
	reply.ValidatorSets[constants.PrimaryNetworkID] = primaryValidators

	// Add all nets
	for _, netID := range netIDs {
		netValidators, err := s.vm.GetValidatorSet(ctx, height, netID)
		if err != nil {
			return fmt.Errorf("failed to get validator set for net %s: %w", netID, err)
		}
		reply.ValidatorSets[netID] = netValidators
	}

	return nil
}

func (s *Service) GetBlock(_ *http.Request, args *api.GetBlockArgs, response *api.GetBlockResponse) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "getBlock",
		"blkID", args.BlockID,
		"encoding", args.Encoding,
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	block, err := s.vm.manager.GetStatelessBlock(args.BlockID)
	if err != nil {
		return fmt.Errorf("couldn't get block with id %s: %w", args.BlockID, err)
	}
	response.Encoding = args.Encoding

	var result any
	if args.Encoding == formatting.JSON {
		// block.InitCtx(s.vm.ctx)
		result = block
	} else {
		result, err = formatting.Encode(args.Encoding, block.Bytes())
		if err != nil {
			return fmt.Errorf("couldn't encode block %s as %s: %w", args.BlockID, args.Encoding, err)
		}
	}

	response.Block, err = json.Marshal(result)
	return err
}

// GetBlockByHeight returns the block at the given height.
func (s *Service) GetBlockByHeight(_ *http.Request, args *api.GetBlockByHeightArgs, response *api.GetBlockResponse) error {
	s.vm.log.Debug("API called",
		"service", "platform",
		"method", "getBlockByHeight",
		"height", uint64(args.Height),
		"encoding", args.Encoding,
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	blockID, err := s.vm.state.GetBlockIDAtHeight(uint64(args.Height))
	if err != nil {
		return fmt.Errorf("couldn't get block at height %d: %w", args.Height, err)
	}

	block, err := s.vm.manager.GetStatelessBlock(blockID)
	if err != nil {
		s.vm.log.Error("couldn't get accepted block",
			"blkID", blockID,
			"error", err,
		)
		return fmt.Errorf("couldn't get block with id %s: %w", blockID, err)
	}
	response.Encoding = args.Encoding

	var result any
	if args.Encoding == formatting.JSON {
		// block.InitCtx(s.vm.ctx)
		result = block
	} else {
		result, err = formatting.Encode(args.Encoding, block.Bytes())
		if err != nil {
			return fmt.Errorf("couldn't encode block %s as %s: %w", blockID, args.Encoding, err)
		}
	}

	response.Block, err = json.Marshal(result)
	return err
}

// GetFeeConfig returns the dynamic fee config of the chain.
func (s *Service) GetFeeConfig(_ *http.Request, _ *struct{}, reply *gas.Config) error {
	s.vm.log.Debug("API called",
		log.String("service", "platform"),
		log.String("method", "getFeeConfig"),
	)

	*reply = s.vm.DynamicFeeConfig
	return nil
}

type GetFeeStateReply struct {
	gas.State
	Price gas.Price `json:"price"`
	Time  time.Time `json:"timestamp"`
}

// GetFeeState returns the current fee state of the chain.
func (s *Service) GetFeeState(_ *http.Request, _ *struct{}, reply *GetFeeStateReply) error {
	s.vm.log.Debug("API called",
		log.String("service", "platform"),
		log.String("method", "getFeeState"),
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	reply.State = s.vm.state.GetFeeState()
	reply.Price = gas.CalculatePrice(
		s.vm.DynamicFeeConfig.MinPrice,
		reply.State.Excess,
		s.vm.DynamicFeeConfig.ExcessConversionConstant,
	)
	reply.Time = s.vm.state.GetTimestamp()
	return nil
}

// GetValidatorFeeConfig returns the validator fee config of the chain.
func (s *Service) GetValidatorFeeConfig(_ *http.Request, _ *struct{}, reply *fee.Config) error {
	s.vm.log.Debug("API called",
		log.String("service", "platform"),
		log.String("method", "getValidatorFeeConfig"),
	)

	*reply = s.vm.ValidatorFeeConfig
	return nil
}

type GetValidatorFeeStateReply struct {
	Excess gas.Gas   `json:"excess"`
	Price  gas.Price `json:"price"`
	Time   time.Time `json:"timestamp"`
}

// GetValidatorFeeState returns the current validator fee state of the chain.
func (s *Service) GetValidatorFeeState(_ *http.Request, _ *struct{}, reply *GetValidatorFeeStateReply) error {
	s.vm.log.Debug("API called",
		log.String("service", "platform"),
		log.String("method", "getValidatorFeeState"),
	)

	s.vm.lock.Lock()
	defer s.vm.lock.Unlock()

	reply.Excess = s.vm.state.GetL1ValidatorExcess()
	reply.Price = gas.CalculatePrice(
		s.vm.ValidatorFeeConfig.MinPrice,
		reply.Excess,
		s.vm.ValidatorFeeConfig.ExcessConversionConstant,
	)
	reply.Time = s.vm.state.GetTimestamp()
	return nil
}

func (s *Service) getAPIOwner(owner *secp256k1fx.OutputOwners) (*platformapi.Owner, error) {
	apiOwner := &platformapi.Owner{
		Locktime:  avajson.Uint64(owner.Locktime),
		Threshold: avajson.Uint32(owner.Threshold),
		Addresses: make([]string, 0, len(owner.Addrs)),
	}
	for _, addr := range owner.Addrs {
		addrStr, err := s.addrManager.FormatLocalAddress(addr)
		if err != nil {
			return nil, err
		}
		apiOwner.Addresses = append(apiOwner.Addresses, addrStr)
	}
	return apiOwner, nil
}

// Takes in a staker and a set of addresses
// Returns:
// 1) The total amount staked by addresses in [addrs]
// 2) The staked outputs
func getStakeHelper(tx *txs.Tx, addrs set.Set[ids.ShortID], totalAmountStaked map[ids.ID]uint64) []lux.TransferableOutput {
	staker, ok := tx.Unsigned.(txs.PermissionlessStaker)
	if !ok {
		return nil
	}

	stake := staker.Stake()
	stakedOuts := make([]lux.TransferableOutput, 0, len(stake))
	// Go through all of the staked outputs
	for _, output := range stake {
		out := output.Out
		if lockedOut, ok := out.(*stakeable.LockOut); ok {
			// This output can only be used for staking until [stakeOnlyUntil]
			out = lockedOut.TransferableOut
		}
		secpOut, ok := out.(*secp256k1fx.TransferOutput)
		if !ok {
			continue
		}

		// Check whether this output is owned by one of the given addresses
		contains := slices.ContainsFunc(secpOut.Addrs, addrs.Contains)
		if !contains {
			// This output isn't owned by one of the given addresses. Ignore.
			continue
		}

		assetID := output.AssetID()
		newAmount, err := safemath.Add(totalAmountStaked[assetID], secpOut.Amt)
		if err != nil {
			newAmount = math.MaxUint64
		}
		totalAmountStaked[assetID] = newAmount

		stakedOuts = append(
			stakedOuts,
			*output,
		)
	}
	return stakedOuts
}

func toPlatformStaker(staker *state.Staker) platformapi.Staker {
	return platformapi.Staker{
		TxID:      staker.TxID,
		StartTime: avajson.Uint64(staker.StartTime.Unix()),
		EndTime:   avajson.Uint64(staker.EndTime.Unix()),
		Weight:    avajson.Uint64(staker.Weight),
		NodeID:    staker.NodeID,
	}
}
