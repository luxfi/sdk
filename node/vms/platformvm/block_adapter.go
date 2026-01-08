// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package platformvm

import (
	"context"
	"time"

	"github.com/luxfi/consensus/engine/chain/block"
	"github.com/luxfi/consensus/protocol/chain"
	"github.com/luxfi/ids"
)

// blockAdapter wraps a chain.Block to implement linearblock.Block
type blockAdapter struct {
	chain.Block
}

// Status returns the block status as choices.Status
func (b *blockAdapter) Status() uint8 {
	// Return the uint8 status from the underlying block
	return b.Block.Status()
}

// All other methods are delegated to the underlying block
func (b *blockAdapter) ID() ids.ID {
	return b.Block.ID()
}

func (b *blockAdapter) ParentID() ids.ID {
	return b.Block.ParentID()
}

func (b *blockAdapter) Height() uint64 {
	return b.Block.Height()
}

func (b *blockAdapter) Timestamp() time.Time {
	return b.Block.Timestamp()
}

func (b *blockAdapter) Bytes() []byte {
	return b.Block.Bytes()
}

func (b *blockAdapter) Verify(ctx context.Context) error {
	return b.Block.Verify(ctx)
}

func (b *blockAdapter) Accept(ctx context.Context) error {
	return b.Block.Accept(ctx)
}

func (b *blockAdapter) Reject(ctx context.Context) error {
	return b.Block.Reject(ctx)
}

func (b *blockAdapter) Options(ctx context.Context) ([2]block.Block, error) {
	// Options is not available in the chain.Block interface
	// Return empty options for now
	return [2]block.Block{nil, nil}, nil
}

// wrapBlock wraps a chain.Block in an adapter that implements linearblock.Block
func wrapBlock(blk chain.Block) block.Block {
	if blk == nil {
		return nil
	}
	return &blockAdapter{Block: blk}
}
