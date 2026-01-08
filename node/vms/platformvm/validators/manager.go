// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validators

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	validators "github.com/luxfi/consensus/validator"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
	"github.com/luxfi/container"
	"github.com/luxfi/cache/lru"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/utils/timer/mockable"
	"github.com/luxfi/sdk/utils/window"
	"github.com/luxfi/sdk/node/vms/platformvm/block"
	"github.com/luxfi/sdk/node/vms/platformvm/config"
	"github.com/luxfi/sdk/node/vms/platformvm/metrics"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/status"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
)

const (
	// MaxRecentlyAcceptedWindowSize is the maximum number of blocks that the
	// recommended minimum height will lag behind the last accepted block.
	MaxRecentlyAcceptedWindowSize = 64
	// MinRecentlyAcceptedWindowSize is the minimum number of blocks that the
	// recommended minimum height will lag behind the last accepted block.
	MinRecentlyAcceptedWindowSize = 0
	// RecentlyAcceptedWindowTTL is the amount of time after a block is accepted
	// to avoid recommending it as the minimum height. The size constraints take
	// precedence over this time constraint.
	RecentlyAcceptedWindowTTL = 30 * time.Second

	validatorSetsCacheSize = 64
)

var (
	_ validators.State = (*manager)(nil)

	errUnfinalizedHeight = errors.New("failed to fetch validator set at unfinalized height")
)

// Manager adds the ability to introduce newly accepted blocks IDs to the State
// interface.
type Manager interface {
	validators.State

	// OnAcceptedBlockID registers the ID of the latest accepted block.
	// It is used to update the [recentlyAccepted] sliding window.
	OnAcceptedBlockID(blkID ids.ID)
}

type State interface {
	GetTx(txID ids.ID) (*txs.Tx, status.Status, error)

	GetLastAccepted() ids.ID
	GetStatelessBlock(blockID ids.ID) (block.Block, error)

	// ApplyValidatorWeightDiffs iterates from [startHeight] towards the genesis
	// block until it has applied all of the diffs up to and including
	// [endHeight]. Applying the diffs modifies [validators].
	//
	// Invariant: If attempting to generate the validator set for
	// [endHeight - 1], [validators] must initially contain the validator
	// weights for [startHeight].
	//
	// Note: Because this function iterates towards the genesis, [startHeight]
	// should normally be greater than or equal to [endHeight].
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
	// should normally be greater than or equal to [endHeight].
	ApplyValidatorPublicKeyDiffs(
		ctx context.Context,
		validators map[ids.NodeID]*validators.GetValidatorOutput,
		startHeight uint64,
		endHeight uint64,
		subnetID ids.ID,
	) error

	GetCurrentValidators(ctx context.Context, subnetID ids.ID) ([]*state.Staker, []state.L1Validator, uint64, error)
}

func NewManager(
	cfg config.Internal,
	state State,
	metrics metrics.Metrics,
	clk *mockable.Clock,
) Manager {
	return &manager{
		cfg:     cfg,
		state:   state,
		metrics: metrics,
		clk:     clk,
		caches:  make(map[ids.ID]cache.Cacher[uint64, map[ids.NodeID]*validators.GetValidatorOutput]),
		recentlyAccepted: window.New[ids.ID](
			window.Config{
				Clock:   clk,
				MaxSize: MaxRecentlyAcceptedWindowSize,
				MinSize: MinRecentlyAcceptedWindowSize,
				TTL:     RecentlyAcceptedWindowTTL,
			},
		),
	}
}

// calling exported functions.
type manager struct {
	cfg     config.Internal
	state   State
	metrics metrics.Metrics
	clk     *mockable.Clock

	// Maps caches for each net that is currently tracked.
	// Key: Net ID
	// Value: cache mapping height -> validator set map
	caches map[ids.ID]cache.Cacher[uint64, map[ids.NodeID]*validators.GetValidatorOutput]

	// sliding window of blocks that were recently accepted
	recentlyAccepted window.Window[ids.ID]
}

// GetMinimumHeight returns the height of the most recent block beyond the
// horizon of our recentlyAccepted window.
//
// Because the time between blocks is arbitrary, we're only guaranteed that
// the window's configured TTL amount of time has passed once an element
// expires from the window.
//
// To try to always return a block older than the window's TTL, we return the
// parent of the oldest element in the window (as an expired element is always
// guaranteed to be sufficiently stale). If we haven't expired an element yet
// in the case of a process restart, we default to the lastAccepted block's
// height which is likely (but not guaranteed) to also be older than the
// window's configured TTL.
//
// If [UseCurrentHeight] is true, we override the block selection policy
// described above and we will always return the last accepted block height
// as the minimum.
func (m *manager) GetMinimumHeight(ctx context.Context) (uint64, error) {
	if m.cfg.UseCurrentHeight {
		return m.getCurrentHeight(ctx)
	}

	oldest, ok := m.recentlyAccepted.Oldest()
	if !ok {
		return m.getCurrentHeight(ctx)
	}

	blk, err := m.state.GetStatelessBlock(oldest)
	if err != nil {
		return 0, err
	}

	// We subtract 1 from the height of [oldest] because we want the height of
	// the last block accepted before the [recentlyAccepted] window.
	//
	// There is guaranteed to be a block accepted before this window because the
	// first block added to [recentlyAccepted] window is >= height 1.
	return blk.Height() - 1, nil
}

// GetCurrentHeight without context to implement validators.State
func (m *manager) GetCurrentHeight(ctx context.Context) (uint64, error) {
	return m.getCurrentHeight(ctx)
}

// GetCurrentHeightWithContext with context for internal use
func (m *manager) GetCurrentHeightWithContext(ctx context.Context) (uint64, error) {
	return m.getCurrentHeight(ctx)
}

func (m *manager) getCurrentHeight(context.Context) (uint64, error) {
	if m.state == nil {
		return 0, fmt.Errorf("state not initialized")
	}
	lastAcceptedID := m.state.GetLastAccepted()
	lastAccepted, err := m.state.GetStatelessBlock(lastAcceptedID)
	if err != nil {
		return 0, err
	}
	return lastAccepted.Height(), nil
}

// GetValidatorSet implements validators.State
func (m *manager) GetValidatorSet(
	ctx context.Context,
	targetHeight uint64,
	netID ids.ID,
) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	return m.GetValidatorSetWithContext(ctx, targetHeight, netID)
}

// GetCurrentValidators implements validators.State
func (m *manager) GetCurrentValidators(
	ctx context.Context,
	height uint64,
	netID ids.ID,
) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	return m.GetValidatorSet(ctx, height, netID)
}

// GetValidatorSetWithContext returns detailed validator information
func (m *manager) GetValidatorSetWithContext(
	ctx context.Context,
	targetHeight uint64,
	netID ids.ID,
) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	validatorSetsCache := m.getValidatorSetCache(netID)

	if validatorSet, ok := validatorSetsCache.Get(targetHeight); ok {
		m.metrics.IncValidatorSetsCached()
		return maps.Clone(validatorSet), nil
	}

	// get the start time to track metrics
	startTime := m.clk.Time()

	validatorSet, currentHeight, err := m.makeValidatorSet(ctx, targetHeight, netID)
	if err != nil {
		return nil, err
	}

	// cache the validator set
	validatorSetsCache.Put(targetHeight, validatorSet)

	duration := m.clk.Time().Sub(startTime)
	m.metrics.IncValidatorSetsCreated()
	m.metrics.AddValidatorSetsDuration(duration)
	m.metrics.AddValidatorSetsHeightDiff(currentHeight - targetHeight)
	return maps.Clone(validatorSet), nil
}

func (m *manager) getValidatorSetCache(chainID ids.ID) cache.Cacher[uint64, map[ids.NodeID]*validators.GetValidatorOutput] {
	// Only cache tracked chains
	if chainID != constants.PrimaryNetworkID && !m.cfg.TrackedChains.Contains(chainID) {
		return &cache.Empty[uint64, map[ids.NodeID]*validators.GetValidatorOutput]{}
	}

	validatorSetsCache, exists := m.caches[chainID]
	if exists {
		return validatorSetsCache
	}

	validatorSetsCache = lru.NewCache[uint64, map[ids.NodeID]*validators.GetValidatorOutput](validatorSetsCacheSize)
	m.caches[chainID] = validatorSetsCache
	return validatorSetsCache
}

func (m *manager) makeValidatorSet(
	ctx context.Context,
	targetHeight uint64,
	netID ids.ID,
) (map[ids.NodeID]*validators.GetValidatorOutput, uint64, error) {
	// Get the current validator set at the current height
	currentValidatorSet, err := m.getCurrentValidatorSet(ctx, netID)
	if err != nil {
		return nil, 0, err
	}

	currentHeight, err := m.getCurrentHeight(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Verify that the target height is not in the future
	if currentHeight < targetHeight {
		return nil, 0, fmt.Errorf(
			"%w with NetID = %s: current P-chain height (%d) < requested P-Chain height (%d)",
			errUnfinalizedHeight,
			netID,
			currentHeight,
			targetHeight,
		)
	}

	// If requesting current height, return immediately
	if targetHeight == currentHeight {
		return maps.Clone(currentValidatorSet), currentHeight, nil
	}

	// Rebuild validators at [targetHeight]
	//
	// Note: Since we are attempting to generate the validator set at
	// [targetHeight], we want to apply the diffs from
	// (targetHeight, currentHeight]. Because the state interface is implemented
	// to be inclusive, we apply diffs in [targetHeight + 1, currentHeight].
	lastDiffHeight := targetHeight + 1
	validatorSet := maps.Clone(currentValidatorSet)

	err = m.state.ApplyValidatorWeightDiffs(
		ctx,
		validatorSet,
		currentHeight,
		lastDiffHeight,
		netID,
	)
	if err != nil {
		return nil, 0, err
	}

	err = m.state.ApplyValidatorPublicKeyDiffs(
		ctx,
		validatorSet,
		currentHeight,
		lastDiffHeight,
		netID,
	)
	if err != nil {
		return nil, 0, err
	}

	return validatorSet, currentHeight, nil
}

func (m *manager) getCurrentValidatorSet(
	ctx context.Context,
	netID ids.ID,
) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	baseStakers, l1Validators, _, err := m.state.GetCurrentValidators(ctx, netID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current validators: %w", err)
	}

	result := make(map[ids.NodeID]*validators.GetValidatorOutput)

	// Add base (legacy) validators
	for _, staker := range baseStakers {
		var pkBytes []byte
		if staker.PublicKey != nil {
			pkBytes = bls.PublicKeyToUncompressedBytes(staker.PublicKey)
		}
		result[staker.NodeID] = &validators.GetValidatorOutput{
			NodeID:    staker.NodeID,
			PublicKey: pkBytes,
			Light:     staker.Weight, // Light is kept in sync with Weight
			Weight:    staker.Weight,
			TxID:      staker.TxID,
		}
	}

	// Add L1 validators
	for _, validator := range l1Validators {
		result[validator.NodeID] = &validators.GetValidatorOutput{
			NodeID:    validator.NodeID,
			PublicKey: validator.PublicKey,
			Light:     validator.Weight, // Light is kept in sync with Weight
			Weight:    validator.Weight,
			TxID:      validator.ValidationID,
		}
	}

	return result, nil
}

func (m *manager) GetNetID(_ context.Context, chainID ids.ID) (ids.ID, error) {
	if chainID == constants.PlatformChainID {
		return constants.PrimaryNetworkID, nil
	}

	chainTx, _, err := m.state.GetTx(chainID)
	if err != nil {
		return ids.Empty, fmt.Errorf(
			"problem retrieving blockchain %q: %w",
			chainID,
			err,
		)
	}
	chain, ok := chainTx.Unsigned.(*txs.CreateChainTx)
	if !ok {
		return ids.Empty, fmt.Errorf("%q is not a blockchain", chainID)
	}
	return chain.ChainID, nil
}

func (m *manager) OnAcceptedBlockID(blkID ids.ID) {
	m.recentlyAccepted.Add(blkID)
}

func (m *manager) GetWarpValidatorSet(ctx context.Context, height uint64, netID ids.ID) (*validators.WarpSet, error) {
	// Get the validator set at the requested height
	vdrSet, err := m.GetValidatorSet(ctx, height, netID)
	if err != nil {
		return nil, err
	}

	// Convert to WarpSet format (Height + Validators map)
	warpValidators := make(map[ids.NodeID]*validators.WarpValidator, len(vdrSet))
	for nodeID, vdr := range vdrSet {
		// Only include validators with BLS public keys
		if len(vdr.PublicKey) > 0 {
			warpValidators[nodeID] = &validators.WarpValidator{
				NodeID:    nodeID,
				PublicKey: vdr.PublicKey,
				Weight:    vdr.Weight,
			}
		}
	}

	return &validators.WarpSet{
		Height:     height,
		Validators: warpValidators,
	}, nil
}

func (m *manager) GetWarpValidatorSets(ctx context.Context, heights []uint64, netIDs []ids.ID) (map[ids.ID]map[uint64]*validators.WarpSet, error) {
	result := make(map[ids.ID]map[uint64]*validators.WarpSet)

	// For each netID, get validator sets for all requested heights
	for _, netID := range netIDs {
		heightMap := make(map[uint64]*validators.WarpSet)
		for _, height := range heights {
			warpSet, err := m.GetWarpValidatorSet(ctx, height, netID)
			if err != nil {
				return nil, err
			}
			heightMap[height] = warpSet
		}
		result[netID] = heightMap
	}

	return result, nil
}
