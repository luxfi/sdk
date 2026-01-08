// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"context"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/consensus/engine/chain/block"
	platformblock "github.com/luxfi/sdk/node/vms/platformvm/block"
)

var (
	_ block.Block             = (*Block)(nil)
	// _ block.OracleBlock       = (*Block)(nil) // TODO: Check if OracleBlock interface exists
	_ block.WithVerifyContext = (*Block)(nil)
)

// Exported for testing in platformvm package.
type Block struct {
	platformblock.Block
	manager *manager
}

// ParentID implements block.Block interface by delegating to Parent()
func (b *Block) ParentID() ids.ID {
	return b.Parent()
}

// Status implements block.Block interface
func (b *Block) Status() uint8 {
	// TODO: Implement proper status tracking
	// For now, return 0 (status tracking needs to be added to blockState)
	return 0
}

func (*Block) ShouldVerifyWithContext(context.Context) (bool, error) {
	return true, nil
}

func (b *Block) VerifyWithContext(ctx context.Context, blockContext *block.Context) error {
	blkID := b.ID()
	blkState, previouslyExecuted := b.manager.backend.getBlockState(blkID)
	warpAlreadyVerified := previouslyExecuted && blkState.verifiedHeights.Contains(blockContext.PChainHeight)

	// If the chain is bootstrapped and the warp messages haven't been verified,
	// we must verify them.
	if !warpAlreadyVerified && b.manager.txExecutorBackend.Bootstrapped.Get() {
		err := VerifyWarpMessages(
			ctx,
			b.manager.ctx.NetworkID,
			b.manager.validatorManager,
			blockContext.PChainHeight,
			b,
		)
		if err != nil {
			return err
		}
	}

	// If the block was previously executed, we don't need to execute it again,
	// we can just mark that the warp messages are valid at this height.
	if previouslyExecuted {
		blkState.verifiedHeights.Add(blockContext.PChainHeight)
		return nil
	}

	// Since this is the first time we are verifying this block, we must execute
	// the state transitions to generate the state diffs.
	return b.Visit(&verifier{
		backend:           b.manager.backend,
		txExecutorBackend: b.manager.txExecutorBackend,
		pChainHeight:      blockContext.PChainHeight,
	})
}

func (b *Block) Verify(ctx context.Context) error {
	return b.VerifyWithContext(
		ctx,
		&block.Context{
			PChainHeight: 0,
		},
	)
}

func (b *Block) Accept(context.Context) error {
	return b.Visit(b.manager.acceptor)
}

func (b *Block) Reject(context.Context) error {
	return b.Visit(b.manager.rejector)
}

func (b *Block) Timestamp() time.Time {
	return b.manager.getTimestamp(b.ID())
}

func (b *Block) Options(context.Context) ([2]block.Block, error) {
	options := options{
		log:                     b.manager.Log,
		primaryUptimePercentage: b.manager.txExecutorBackend.Config.UptimePercentage,
		uptimes:                 b.manager.txExecutorBackend.Uptimes,
		state:                   b.manager.backend.state,
	}
	if err := b.Block.Visit(&options); err != nil {
		return [2]block.Block{}, err
	}

	return [2]block.Block{
		b.manager.NewBlock(options.preferredBlock),
		b.manager.NewBlock(options.alternateBlock),
	}, nil
}

// FPCVotes implements the chain.Block interface
// Returns embedded fast-path consensus vote references
func (b *Block) FPCVotes() [][]byte {
	return nil
}

// EpochBit implements the chain.Block interface
// Returns the epoch fence bit for FPC
func (b *Block) EpochBit() bool {
	return false
}
