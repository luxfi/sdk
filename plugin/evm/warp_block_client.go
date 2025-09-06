// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"time"

	consensusChain "github.com/luxfi/consensus/protocol/chain"
	"github.com/luxfi/database"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/ids"
)

// warpBlockClient wraps VM to provide the warp.BlockClient interface
type warpBlockClient struct {
	vm *VM
}

// GetAcceptedBlock returns a block that implements consensus/chain.Block
func (w *warpBlockClient) GetAcceptedBlock(ctx context.Context, blkID ids.ID) (consensusChain.Block, error) {
	// First verify the block is accepted
	ethBlock := w.vm.blockChain.GetBlockByHash(common.BytesToHash(blkID[:]))
	if ethBlock == nil {
		return nil, database.ErrNotFound
	}

	// Check if this block is accepted by comparing with canonical chain
	acceptedHash := w.vm.blockChain.GetCanonicalHash(ethBlock.NumberU64())
	if acceptedHash != ethBlock.Hash() {
		return nil, database.ErrNotFound
	}

	// Create a wrapper that implements consensus/chain.Block
	return &warpConsensusBlockWrapper{
		block: w.vm.newBlock(ethBlock),
	}, nil
}

// warpConsensusBlockWrapper wraps a Block to implement consensus/chain.Block
type warpConsensusBlockWrapper struct {
	block *Block
}

// ID returns the block's ID (consensus/chain.Block interface)
func (b *warpConsensusBlockWrapper) ID() ids.ID {
	return b.block.ID()
}

// Height returns the block's height (consensus/chain.Block interface)
func (b *warpConsensusBlockWrapper) Height() uint64 {
	return b.block.Height()
}

// Parent returns the parent block's ID (consensus/chain.Block interface)
func (b *warpConsensusBlockWrapper) Parent() ids.ID {
	return b.block.Parent()
}

// Accept implements consensus/chain.Block interface
func (b *warpConsensusBlockWrapper) Accept(ctx context.Context) error {
	// Block is already accepted (we only return accepted blocks)
	// This is a no-op since we already verified the block is in the canonical chain
	return nil
}

// Bytes implements consensus/chain.Block interface
func (b *warpConsensusBlockWrapper) Bytes() []byte {
	return b.block.Bytes()
}

// Timestamp implements consensus/chain.Block interface
func (b *warpConsensusBlockWrapper) Timestamp() time.Time {
	return b.block.Timestamp()
}

// Reject implements consensus/chain.Block interface
func (b *warpConsensusBlockWrapper) Reject(ctx context.Context) error {
	// Block is already accepted, cannot reject
	return nil
}

// Verify implements consensus/chain.Block interface
func (b *warpConsensusBlockWrapper) Verify(ctx context.Context) error {
	// Block is already accepted, no need to verify
	return nil
}
