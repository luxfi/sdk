// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"time"

	"github.com/luxfi/ids"
	consensusChain "github.com/luxfi/consensus/chain"
	"github.com/luxfi/consensus/choices"
	"github.com/luxfi/database"
	"github.com/luxfi/geth/common"
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

// warpConsensusBlockWrapper wraps a Block to implement consensus/chain.Block (with string IDs)
type warpConsensusBlockWrapper struct {
	block *Block
}

// ID returns the block's ID as string (consensus/chain.Block interface)
func (b *warpConsensusBlockWrapper) ID() string {
	return b.block.ID().String()
}

// Height returns the block's height (consensus/chain.Block interface)
func (b *warpConsensusBlockWrapper) Height() uint64 {
	return b.block.Height()
}

// Parent returns the parent block's ID as string (consensus/chain.Block interface)
func (b *warpConsensusBlockWrapper) Parent() string {
	return b.block.Parent().String()
}