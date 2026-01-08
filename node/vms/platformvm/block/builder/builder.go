// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"go.uber.org/zap"

	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/luxfi/log"


	"github.com/luxfi/ids"
	consensuscore "github.com/luxfi/consensus/core"
	validators "github.com/luxfi/consensus/validator"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/utils/timer/mockable"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/sdk/node/vms/components/gas"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/status"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/platformvm/txs/fee"
	"github.com/luxfi/sdk/node/vms/txs/mempool"

	chainblock "github.com/luxfi/consensus/engine/chain/block"
	platformblock "github.com/luxfi/sdk/node/vms/platformvm/block"
	blockexecutor "github.com/luxfi/sdk/node/vms/platformvm/block/executor"
	consensusctx "github.com/luxfi/consensus/context"
	txexecutor "github.com/luxfi/sdk/node/vms/platformvm/txs/executor"
)

// validatorStateAdapter adapts consensusctx.ValidatorState to validators.State
type validatorStateAdapter struct {
	state consensusctx.ValidatorState
}

func (a *validatorStateAdapter) GetValidatorSet(ctx context.Context, height uint64, netID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	weights, err := a.state.GetValidatorSet(height, netID)
	if err != nil {
		return nil, err
	}
	
	result := make(map[ids.NodeID]*validators.GetValidatorOutput, len(weights))
	for nodeID, weight := range weights {
		result[nodeID] = &validators.GetValidatorOutput{
			NodeID: nodeID,
			Light:  weight,
		}
	}
	return result, nil
}

func (a *validatorStateAdapter) GetCurrentValidators(ctx context.Context, height uint64, netID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	// Use GetValidatorSet for current validators
	return a.GetValidatorSet(ctx, height, netID)
}

func (a *validatorStateAdapter) GetCurrentHeight(ctx context.Context) (uint64, error) {
	return a.state.GetCurrentHeight(ctx)
}

func (a *validatorStateAdapter) GetWarpValidatorSet(ctx context.Context, height uint64, netID ids.ID) (*validators.WarpSet, error) {
	// Get the validator set at the requested height
	vdrSet, err := a.GetValidatorSet(ctx, height, netID)
	if err != nil {
		return nil, err
	}

	// Convert to WarpSet format
	// Note: This adapter doesn't have BLS public keys, so we return empty WarpSet
	// Real implementations should query for BLS keys
	warpValidators := make(map[ids.NodeID]*validators.WarpValidator, len(vdrSet))
	for nodeID, vdr := range vdrSet {
		// Only include validators with BLS public keys (none in this adapter)
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

func (a *validatorStateAdapter) GetWarpValidatorSets(ctx context.Context, heights []uint64, netIDs []ids.ID) (map[ids.ID]map[uint64]*validators.WarpSet, error) {
	result := make(map[ids.ID]map[uint64]*validators.WarpSet)

	// For each netID, get validator sets for all requested heights
	for _, netID := range netIDs {
		heightMap := make(map[uint64]*validators.WarpSet)
		for _, height := range heights {
			warpSet, err := a.GetWarpValidatorSet(ctx, height, netID)
			if err != nil {
				return nil, err
			}
			heightMap[height] = warpSet
		}
		result[netID] = heightMap
	}

	return result, nil
}

const (
	// targetBlockSize is maximum number of transaction bytes to place into a
	// StandardBlock
	targetBlockSize = 128 * units.KiB

	// maxTimeToSleep is the maximum time to sleep between checking if a block
	// should be produced.
	maxTimeToSleep = time.Hour
)

var (
	_ Builder = (*builder)(nil)

	ErrEndOfTime                 = errors.New("program time is suspiciously far in the future")
	ErrNoPendingBlocks           = errors.New("no pending blocks")
	errMissingPreferredState     = errors.New("missing preferred block state")
	errCalculatingNextStakerTime = errors.New("failed calculating next staker time")
)

type Builder interface {
	mempool.Mempool[*txs.Tx]

	// BuildBlock can be called to attempt to create a new block
	BuildBlock(context.Context) (chainblock.Block, error)

	// BuildBlockWithContext builds a block with context
	BuildBlockWithContext(context.Context, *chainblock.Context) (chainblock.Block, error)

	// Connected is called when a node connects
	Connected(context.Context, ids.NodeID, interface{}) error

	// Disconnected is called when a node disconnects
	Disconnected(context.Context, ids.NodeID) error

	// PackAllBlockTxs returns an array of all txs that could be packed into a
	// valid block of infinite size. The returned txs are all verified against
	// the preferred state.
	//
	// Note: This function does not call the consensus engine.
	PackAllBlockTxs() ([]*txs.Tx, error)
}

// builder implements a simple builder to convert txs into valid blocks
type builder struct {
	mempool.Mempool[*txs.Tx]

	txExecutorBackend *txexecutor.Backend
	blkManager        blockexecutor.Manager
}

func New(
	mempool mempool.Mempool[*txs.Tx],
	txExecutorBackend *txexecutor.Backend,
	blkManager blockexecutor.Manager,
) Builder {
	return &builder{
		Mempool:           mempool,
		txExecutorBackend: txExecutorBackend,
		blkManager:        blkManager,
	}
}

func (b *builder) Connected(ctx context.Context, nodeID ids.NodeID, version interface{}) error {
	// No-op implementation for builder
	return nil
}

func (b *builder) Disconnected(ctx context.Context, nodeID ids.NodeID) error {
	// No-op implementation for builder
	return nil
}

func (b *builder) WaitForEvent(ctx context.Context) (consensuscore.Message, error) {
	logger := b.txExecutorBackend.Ctx.Log.(log.Logger)
	consecutiveErrors := 0
	for {
		if err := ctx.Err(); err != nil {
			return consensuscore.Message{}, err
		}

		duration, err := b.durationToSleep()
		if err != nil {
			consecutiveErrors++
			// Log the error but don't crash - use exponential backoff
			if consecutiveErrors <= 5 {
				logger.Error("block builder failed to calculate next staker change time",
					zap.Error(err),
					zap.Int("consecutiveErrors", consecutiveErrors),
				)
			}
			// Use exponential backoff with max of 30 seconds
			backoff := time.Duration(math.Min(float64(time.Second)*float64(consecutiveErrors*consecutiveErrors), float64(30*time.Second)))
			select {
			case <-ctx.Done():
				return consensuscore.Message{}, ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}
		consecutiveErrors = 0 // Reset on success
		if duration <= 0 {
			logger.Debug("Skipping block build wait, next staker change is ready")
			// The next staker change is ready to be performed.
			return consensuscore.Message{Type: consensuscore.PendingTxs}, nil
		}

		logger.Debug("Will wait until a transaction comes", log.Duration("maxWait", duration))

		// Wait for a transaction in the mempool until there is a next staker
		// change ready to be performed.
		newCtx, cancel := context.WithTimeout(ctx, duration)
		msg, err := b.Mempool.WaitForEvent(newCtx)
		cancel()

		switch {
		case err == nil:
			logger.Debug("New transaction received")
			return msg, nil
		case errors.Is(err, context.DeadlineExceeded):
			continue // Recheck the staker change time before returning
		default:
			// Error could have been due to the parent context being cancelled
			// or another unexpected error.
			return consensuscore.Message{}, err
		}
	}
}

func (b *builder) durationToSleep() (time.Duration, error) {
	// Check if builder is properly initialized
	if b.txExecutorBackend == nil {
		return 0, nil
	}

	preferredID := b.blkManager.Preferred()
	preferredState, ok := b.blkManager.GetState(preferredID)
	if !ok {
		return 0, fmt.Errorf("%w: %s", errMissingPreferredState, preferredID)
	}

	now := b.txExecutorBackend.Clk.Time()
	maxTimeToAwake := now.Add(maxTimeToSleep)
	nextStakerChangeTime, err := state.GetNextStakerChangeTime(
		b.txExecutorBackend.Config.ValidatorFeeConfig,
		preferredState,
		maxTimeToAwake,
	)
	if err != nil {
		return 0, fmt.Errorf("%w of %s: %w", errCalculatingNextStakerTime, preferredID, err)
	}

	return nextStakerChangeTime.Sub(now), nil
}

func (b *builder) BuildBlock(ctx context.Context) (chainblock.Block, error) {
	return b.BuildBlockWithContext(
		ctx,
		&chainblock.Context{
			PChainHeight: 0,
		},
	)
}

func (b *builder) BuildBlockWithContext(
	ctx context.Context,
	blockContext *chainblock.Context,
) (chainblock.Block, error) {
	logger := b.txExecutorBackend.Ctx.Log.(log.Logger)
	logger.Debug("starting to attempt to build a block")

	// Get the block to build on top of and retrieve the new block's context.
	preferredID := b.blkManager.Preferred()
	preferred, err := b.blkManager.GetBlock(preferredID)
	if err != nil {
		return nil, err
	}
	nextHeight := preferred.Height() + 1
	preferredState, ok := b.blkManager.GetState(preferredID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", state.ErrMissingParentState, preferredID)
	}

	timestamp, timeWasCapped, err := state.NextBlockTime(
		b.txExecutorBackend.Config.ValidatorFeeConfig,
		preferredState,
		b.txExecutorBackend.Clk,
	)
	if err != nil {
		return nil, fmt.Errorf("could not calculate next staker change time: %w", err)
	}

	statelessBlk, err := buildBlock(
		ctx,
		b,
		preferredID,
		nextHeight,
		timestamp,
		timeWasCapped,
		preferredState,
		blockContext.PChainHeight,
	)
	if err != nil {
		return nil, err
	}

	return b.blkManager.NewBlock(statelessBlk), nil
}

func (b *builder) PackAllBlockTxs() ([]*txs.Tx, error) {
	preferredID := b.blkManager.Preferred()
	preferredState, ok := b.blkManager.GetState(preferredID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", errMissingPreferredState, preferredID)
	}

	timestamp, _, err := state.NextBlockTime(
		b.txExecutorBackend.Config.ValidatorFeeConfig,
		preferredState,
		b.txExecutorBackend.Clk,
	)
	if err != nil {
		return nil, fmt.Errorf("could not calculate next staker change time: %w", err)
	}

	// Type assert ValidatorState to get GetMinimumHeight method
	// ValidatorState may be nil during initialization, use 0 as default
	var recommendedPChainHeight uint64
	if b.txExecutorBackend.Ctx.ValidatorState != nil {
		validatorState := b.txExecutorBackend.Ctx.ValidatorState.(interface {
			GetMinimumHeight(context.Context) (uint64, error)
		})
		var err error
		recommendedPChainHeight, err = validatorState.GetMinimumHeight(context.TODO())
		if err != nil {
			return nil, err
		}
	}

	if !b.txExecutorBackend.Config.UpgradeConfig.IsEtnaActivated(timestamp) {
		return packDurangoBlockTxs(
			context.TODO(),
			preferredID,
			preferredState,
			b.Mempool,
			b.txExecutorBackend,
			b.blkManager,
			timestamp,
			recommendedPChainHeight,
			math.MaxInt,
		)
	}
	return packEtnaBlockTxs(
		context.TODO(),
		preferredID,
		preferredState,
		b.Mempool,
		b.txExecutorBackend,
		b.blkManager,
		timestamp,
		recommendedPChainHeight,
		math.MaxUint64,
	)
}

// [timestamp] is min(max(now, parent timestamp), next staker change time)
func buildBlock(
	ctx context.Context,
	builder *builder,
	parentID ids.ID,
	height uint64,
	timestamp time.Time,
	forceAdvanceTime bool,
	parentState state.Chain,
	pChainHeight uint64,
) (platformblock.Block, error) {
	var (
		blockTxs []*txs.Tx
		err      error
	)
	if builder.txExecutorBackend.Config.UpgradeConfig.IsEtnaActivated(timestamp) {
		blockTxs, err = packEtnaBlockTxs(
			ctx,
			parentID,
			parentState,
			builder.Mempool,
			builder.txExecutorBackend,
			builder.blkManager,
			timestamp,
			pChainHeight,
			0, // minCapacity is 0 as we want to honor the capacity in state.
		)
	} else {
		blockTxs, err = packDurangoBlockTxs(
			ctx,
			parentID,
			parentState,
			builder.Mempool,
			builder.txExecutorBackend,
			builder.blkManager,
			timestamp,
			pChainHeight,
			targetBlockSize,
		)
	}
	if err != nil {
		logger := builder.txExecutorBackend.Ctx.Log.(log.Logger)
		logger.Warn("failed to pack block transactions: " + err.Error())
		return nil, fmt.Errorf("failed to pack block txs: %w", err)
	}

	// Try rewarding stakers whose staking period ends at the new chain time.
	// This is done first to prioritize advancing the timestamp as quickly as
	// possible.
	stakerTxID, shouldReward, err := getNextStakerToReward(timestamp, parentState)
	if err != nil {
		return nil, fmt.Errorf("could not find next staker to reward: %w", err)
	}
	if shouldReward {
		rewardValidatorTx, err := NewRewardValidatorTx(context.TODO(), stakerTxID)
		if err != nil {
			return nil, fmt.Errorf("could not build tx to reward staker: %w", err)
		}

		return platformblock.NewBanffProposalBlock(
			timestamp,
			parentID,
			height,
			rewardValidatorTx,
			blockTxs,
		)
	}

	// If there is no reason to build a block, don't.
	if len(blockTxs) == 0 && !forceAdvanceTime {
		log.Debug("no pending txs to issue into a block")
		return nil, ErrNoPendingBlocks
	}

	// Issue a block with as many transactions as possible.
	return platformblock.NewBanffStandardBlock(
		timestamp,
		parentID,
		height,
		blockTxs,
	)
}

func packDurangoBlockTxs(
	ctx context.Context,
	parentID ids.ID,
	parentState state.Chain,
	mempool mempool.Mempool[*txs.Tx],
	backend *txexecutor.Backend,
	manager blockexecutor.Manager,
	timestamp time.Time,
	pChainHeight uint64,
	remainingSize int,
) ([]*txs.Tx, error) {
	logger := backend.Ctx.Log.(log.Logger)
	logger.Debug("packDurangoBlockTxs starting",
		log.Time("timestamp", timestamp),
		log.Uint64("pChainHeight", pChainHeight),
	)
	stateDiff, err := state.NewDiffOn(parentState)
	if err != nil {
		logger.Warn("packDurangoBlockTxs NewDiffOn failed: " + err.Error())
		return nil, err
	}
	logger.Debug("packDurangoBlockTxs NewDiffOn succeeded")

	if _, err := txexecutor.AdvanceTimeTo(backend, stateDiff, timestamp); err != nil {
		logger.Warn("packDurangoBlockTxs AdvanceTimeTo failed: " + err.Error())
		return nil, err
	}
	logger.Debug("packDurangoBlockTxs AdvanceTimeTo succeeded")

	var (
		blockTxs      []*txs.Tx
		inputs        set.Set[ids.ID]
		feeCalculator = state.PickFeeCalculator(backend.Config, stateDiff)
	)
	for {
		tx, exists := mempool.Peek()
		if !exists {
			break
		}
		txSize := len(tx.Bytes())
		if txSize > remainingSize {
			break
		}

		shouldAdd, err := executeTx(
			ctx,
			parentID,
			stateDiff,
			mempool,
			backend,
			manager,
			pChainHeight,
			&inputs,
			feeCalculator,
			tx,
		)
		if err != nil {
			return nil, err
		}
		if !shouldAdd {
			continue
		}

		remainingSize -= txSize
		blockTxs = append(blockTxs, tx)
	}

	return blockTxs, nil
}

func packEtnaBlockTxs(
	ctx context.Context,
	parentID ids.ID,
	parentState state.Chain,
	mempool mempool.Mempool[*txs.Tx],
	backend *txexecutor.Backend,
	manager blockexecutor.Manager,
	timestamp time.Time,
	pChainHeight uint64,
	minCapacity gas.Gas,
) ([]*txs.Tx, error) {
	stateDiff, err := state.NewDiffOn(parentState)
	if err != nil {
		return nil, err
	}

	if _, err := txexecutor.AdvanceTimeTo(backend, stateDiff, timestamp); err != nil {
		return nil, err
	}

	feeState := stateDiff.GetFeeState()
	capacity := max(feeState.Capacity, minCapacity)

	var (
		blockTxs        []*txs.Tx
		inputs          set.Set[ids.ID]
		blockComplexity gas.Dimensions
		feeCalculator   = state.PickFeeCalculator(backend.Config, stateDiff)
	)

	logger := backend.Ctx.Log.(log.Logger)
	logger.Debug("starting to pack block txs",
		log.Stringer("parentID", parentID),
		log.Time("blockTimestamp", timestamp),
		log.Uint64("capacity", uint64(capacity)),
		log.Int("mempoolLen", mempool.Len()),
	)
	for {
		currentBlockGas, err := blockComplexity.ToGas(backend.Config.DynamicFeeConfig.Weights)
		if err != nil {
			return nil, err
		}

		tx, exists := mempool.Peek()
		if !exists {
			logger.Debug("mempool is empty",
				log.Uint64("capacity", uint64(capacity)),
				log.Uint64("blockGas", uint64(currentBlockGas)),
				log.Int("blockLen", len(blockTxs)),
			)
			break
		}

		txComplexity, err := fee.TxComplexity(tx.Unsigned)
		if err != nil {
			return nil, err
		}
		newBlockComplexity, err := blockComplexity.Add(&txComplexity)
		if err != nil {
			return nil, err
		}
		newBlockGas, err := newBlockComplexity.ToGas(backend.Config.DynamicFeeConfig.Weights)
		if err != nil {
			return nil, err
		}
		if newBlockGas > capacity {
			logger.Debug("block is full",
				log.Uint64("nextBlockGas", uint64(newBlockGas)),
				log.Uint64("capacity", uint64(capacity)),
				log.Uint64("blockGas", uint64(currentBlockGas)),
				log.Int("blockLen", len(blockTxs)),
			)
			break
		}

		shouldAdd, err := executeTx(
			ctx,
			parentID,
			stateDiff,
			mempool,
			backend,
			manager,
			pChainHeight,
			&inputs,
			feeCalculator,
			tx,
		)
		if err != nil {
			return nil, err
		}
		if !shouldAdd {
			continue
		}

		blockComplexity = newBlockComplexity
		blockTxs = append(blockTxs, tx)
	}

	return blockTxs, nil
}

func executeTx(
	ctx context.Context,
	parentID ids.ID,
	stateDiff state.Diff,
	mempool mempool.Mempool[*txs.Tx],
	backend *txexecutor.Backend,
	manager blockexecutor.Manager,
	pChainHeight uint64,
	inputs *set.Set[ids.ID],
	feeCalculator fee.Calculator,
	tx *txs.Tx,
) (bool, error) {
	mempool.Remove(tx)

	// Invariant: [tx] has already been syntactically verified.

	logger := backend.Ctx.Log.(log.Logger)
	txID := tx.ID()

	// Get validator state - handle both validators.State (from node) and consensusctx.ValidatorState (from tests)
	var stateAdapter validators.State
	if vs, ok := backend.Ctx.ValidatorState.(validators.State); ok {
		// Node provides validators.State directly
		stateAdapter = vs
	} else if vs, ok := backend.Ctx.ValidatorState.(consensusctx.ValidatorState); ok {
		// Tests may provide consensusctx.ValidatorState, wrap it
		stateAdapter = &validatorStateAdapter{state: vs}
	} else {
		return false, fmt.Errorf("invalid validator state type: %T", backend.Ctx.ValidatorState)
	}

	err := txexecutor.VerifyWarpMessages(
		ctx,
		backend.Ctx.NetworkID,
		stateAdapter,
		pChainHeight,
		tx.Unsigned,
	)
	if err != nil {
		logger.Debug("transaction failed warp verification",
			log.Stringer("txID", txID),
			zap.Error(err),
		)

		mempool.MarkDropped(txID, err)
		return false, nil
	}

	txDiff, err := state.NewDiffOn(stateDiff)
	if err != nil {
		return false, err
	}

	txInputs, _, _, err := txexecutor.StandardTx(
		backend,
		feeCalculator,
		tx,
		txDiff,
	)
	if err != nil {
		logger.Debug("transaction failed execution",
			log.Stringer("txID", txID),
			zap.Error(err),
		)

		mempool.MarkDropped(txID, err)
		return false, nil
	}

	if inputs.Overlaps(txInputs) {
		// This log is a warn because the mempool should not have allowed this
		// transaction to be included.
		logger.Warn("transaction conflicts with prior transaction",
			log.Stringer("txID", txID),
			zap.Error(err),
		)

		mempool.MarkDropped(txID, blockexecutor.ErrConflictingBlockTxs)
		return false, nil
	}
	if err := manager.VerifyUniqueInputs(parentID, txInputs); err != nil {
		logger.Debug("transaction conflicts with ancestor's import transaction",
			log.Stringer("txID", txID),
			zap.Error(err),
		)

		mempool.MarkDropped(txID, err)
		return false, nil
	}
	inputs.Union(txInputs)

	logger.Debug("successfully executed transaction",
		log.Stringer("txID", txID),
		zap.Error(err),
	)
	txDiff.AddTx(tx, status.Committed)
	return true, txDiff.Apply(stateDiff)
}

// getNextStakerToReward returns the next staker txID to remove from the staking
// set with a RewardValidatorTx rather than an AdvanceTimeTx. [chainTimestamp]
// is the timestamp of the chain at the time this validator would be getting
// removed and is used to calculate [shouldReward].
// Returns:
// - [txID] of the next staker to reward
// - [shouldReward] if the txID exists and is ready to be rewarded
// - [err] if something bad happened
func getNextStakerToReward(
	chainTimestamp time.Time,
	preferredState state.Chain,
) (ids.ID, bool, error) {
	if !chainTimestamp.Before(mockable.MaxTime) {
		return ids.Empty, false, ErrEndOfTime
	}

	currentStakerIterator, err := preferredState.GetCurrentStakerIterator()
	if err != nil {
		return ids.Empty, false, err
	}
	defer currentStakerIterator.Release()

	for currentStakerIterator.Next() {
		currentStaker := currentStakerIterator.Value()
		priority := currentStaker.Priority
		// If the staker is a permissionless staker (not a permissioned net
		// validator), it's the next staker we will want to remove with a
		// RewardValidatorTx rather than an AdvanceTimeTx.
		if priority != txs.NetPermissionedValidatorCurrentPriority {
			return currentStaker.TxID, chainTimestamp.Equal(currentStaker.EndTime), nil
		}
	}
	return ids.Empty, false, nil
}

func NewRewardValidatorTx(ctx context.Context, txID ids.ID) (*txs.Tx, error) {
	utx := &txs.RewardValidatorTx{TxID: txID}
	tx, err := txs.NewSigned(utx, txs.Codec, nil)
	if err != nil {
		return nil, err
	}
	// RewardValidatorTx doesn't need context for syntactic verification
	return tx, tx.SyntacticVerify(nil)
}
