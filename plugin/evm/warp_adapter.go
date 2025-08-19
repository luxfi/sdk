// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"time"

	"github.com/luxfi/ids"
	consensusBlock "github.com/luxfi/consensus/engine/chain/block"
	consensusChain "github.com/luxfi/consensus/chain"
	"github.com/luxfi/node/vms/components/chain"
	nodeWarp "github.com/luxfi/node/vms/platformvm/warp"
	nodeCore "github.com/luxfi/node/consensus/engine/core"
	luxWarp "github.com/luxfi/warp"
	"github.com/luxfi/evm/warp"
)

// warpVerifierAdapter adapts warp.Backend to lp118.Verifier
type warpVerifierAdapter struct {
	backend warp.Backend
}

func (w *warpVerifierAdapter) Verify(ctx context.Context, msg *nodeWarp.UnsignedMessage, signature []byte) *nodeCore.AppError {
	// Convert node warp message to consensus warp message
	luxMsg := &luxWarp.UnsignedMessage{
		NetworkID:     msg.NetworkID,
		SourceChainID: msg.SourceChainID,
		Payload:       msg.Payload,
	}
	
	err := w.backend.Verify(ctx, luxMsg, signature)
	if err != nil {
		return &nodeCore.AppError{
			Code:    1,
			Message: err.Error(),
		}
	}
	return nil
}

// warpSignerAdapter adapts consensus warp signer to node warp signer
type warpSignerAdapter struct {
	signer interface{}
}

func (w *warpSignerAdapter) Sign(msg *nodeWarp.UnsignedMessage) ([]byte, error) {
	// For now, return empty signature as we don't have the actual signer interface
	return []byte{}, nil
}

// warpBlockClientWrapper wraps a function to implement warp.BlockClient interface
type warpBlockClientWrapper struct {
	getBlock func(context.Context, ids.ID) (consensusBlock.Block, error)
}

func (w *warpBlockClientWrapper) GetAcceptedBlock(ctx context.Context, blockID ids.ID) (consensusChain.Block, error) {
	// The warp backend expects consensus chain.Block
	// Our getBlock returns consensusBlock.Block which extends it
	block, err := w.getBlock(ctx, blockID)
	if err != nil {
		return nil, err
	}
	// consensusBlock.Block implements consensusChain.Block
	return block, nil
}

// blockAdapter wraps a consensus block to implement node chain.Block interface
type blockAdapter struct {
	block consensusBlock.Block
}

func (b *blockAdapter) ID() ids.ID {
	// consensus block ID returns string, convert to ids.ID
	idStr := b.block.ID()
	id, _ := ids.FromString(idStr)
	return id
}

func (b *blockAdapter) Parent() ids.ID {
	// consensus block Parent returns string, convert to ids.ID
	parentStr := b.block.Parent()
	id, _ := ids.FromString(parentStr)
	return id
}

func (b *blockAdapter) Height() uint64 {
	return b.block.Height()
}

func (b *blockAdapter) Bytes() []byte {
	// Node chain.Block expects Bytes method
	// This is not in consensus block interface, return empty for now
	return []byte{}
}

func (b *blockAdapter) Timestamp() time.Time {
	// Node chain.Block expects Timestamp method
	// This is not in consensus block interface, return zero time for now
	return time.Time{}
}