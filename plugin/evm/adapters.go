// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"time"

	"github.com/luxfi/ids"

	// Node interfaces that the VM plugin must implement (from engine/chain/block)
	nodeblock "github.com/luxfi/consensus/engine/chain/block"

	// Consensus interfaces that our implementation uses (from protocol/chain)
	consensusblock "github.com/luxfi/consensus/protocol/chain"

	// Network interfaces
	"github.com/luxfi/consensus/core/appsender"
	"github.com/luxfi/consensus/utils/set"
)

// BlockAdapter adapts consensus Block to node Block interface
type BlockAdapter struct {
	consensus consensusblock.Block
}

// NewBlockAdapter creates a new block adapter
func NewBlockAdapter(consensusBlock consensusblock.Block) nodeblock.Block {
	return &BlockAdapter{consensus: consensusBlock}
}

// ID returns the block ID
func (b *BlockAdapter) ID() ids.ID {
	return b.consensus.ID()
}

// Parent returns the parent block ID (alias for ParentID)
func (b *BlockAdapter) Parent() ids.ID {
	return b.consensus.ParentID()
}

// ParentID returns the parent block ID
func (b *BlockAdapter) ParentID() ids.ID {
	return b.consensus.ParentID()
}

// Height returns the block height
func (b *BlockAdapter) Height() uint64 {
	return b.consensus.Height()
}

// Timestamp returns the block timestamp
func (b *BlockAdapter) Timestamp() time.Time {
	return b.consensus.Timestamp()
}

// Status returns the block status
func (b *BlockAdapter) Status() uint8 {
	return uint8(b.consensus.Status())
}

// Verify verifies the block
func (b *BlockAdapter) Verify(ctx context.Context) error {
	return b.consensus.Verify(ctx)
}

// Accept accepts the block
func (b *BlockAdapter) Accept(ctx context.Context) error {
	return b.consensus.Accept(ctx)
}

// Reject rejects the block
func (b *BlockAdapter) Reject(ctx context.Context) error {
	return b.consensus.Reject(ctx)
}

// Bytes returns the block bytes
func (b *BlockAdapter) Bytes() []byte {
	return b.consensus.Bytes()
}

// Unwrap returns the underlying consensus block.
// This is useful for tests that need access to the internal *Block type.
func (b *BlockAdapter) Unwrap() consensusblock.Block {
	return b.consensus
}

// ShouldVerifyWithContext implements the block.WithVerifyContext interface
// by delegating to the underlying block if it supports the interface
func (b *BlockAdapter) ShouldVerifyWithContext(ctx context.Context) (bool, error) {
	// Check if the underlying block implements WithVerifyContext
	if verifiable, ok := b.consensus.(nodeblock.WithVerifyContext); ok {
		return verifiable.ShouldVerifyWithContext(ctx)
	}
	// Default to false if not supported
	return false, nil
}

// VerifyWithContext implements the block.WithVerifyContext interface
// by delegating to the underlying block if it supports the interface
func (b *BlockAdapter) VerifyWithContext(ctx context.Context, blockCtx *nodeblock.Context) error {
	// Check if the underlying block implements WithVerifyContext
	if verifiable, ok := b.consensus.(nodeblock.WithVerifyContext); ok {
		return verifiable.VerifyWithContext(ctx, blockCtx)
	}
	// Fall back to regular verification
	return b.consensus.Verify(ctx)
}

// ContextAdapter adapts between node and consensus context types
type ContextAdapter struct {
	nodeCtx *nodeblock.Context
}

// NewContextAdapter creates a context adapter from node to consensus
func NewContextAdapter(nodeCtx *nodeblock.Context) *nodeblock.Context {
	// Both use the same Context type from engine/chain/block package
	return nodeCtx
}

// NewNodeContextAdapter creates a node context from consensus context
func NewNodeContextAdapter(consensusCtx *nodeblock.Context) *nodeblock.Context {
	// Both use the same Context type from engine/chain/block package
	return consensusCtx
}

// AppSenderAdapter adapts consensus AppSender to node AppSender interface
type AppSenderAdapter struct {
	consensus appsender.AppSender
}

// NewAppSenderAdapter creates an AppSender adapter
func NewAppSenderAdapter(consensusAppSender appsender.AppSender) nodeblock.AppSender {
	return &AppSenderAdapter{consensus: consensusAppSender}
}

// SendAppRequest sends an app request
func (a *AppSenderAdapter) SendAppRequest(ctx context.Context, nodeIDs []ids.NodeID, requestID uint32, appRequestBytes []byte) error {
	// Convert slice to set for consensus interface
	nodeIDSet := set.NewSet[ids.NodeID](len(nodeIDs))
	for _, nodeID := range nodeIDs {
		nodeIDSet.Add(nodeID)
	}

	return a.consensus.SendAppRequest(ctx, nodeIDSet, requestID, appRequestBytes)
}

// SendAppResponse sends an app response
func (a *AppSenderAdapter) SendAppResponse(ctx context.Context, nodeID ids.NodeID, requestID uint32, appResponseBytes []byte) error {
	return a.consensus.SendAppResponse(ctx, nodeID, requestID, appResponseBytes)
}

// SendAppError sends an app error
func (a *AppSenderAdapter) SendAppError(ctx context.Context, nodeID ids.NodeID, requestID uint32, errorCode int32, errorMessage string) error {
	return a.consensus.SendAppError(ctx, nodeID, requestID, errorCode, errorMessage)
}

// SendAppGossip sends app gossip
func (a *AppSenderAdapter) SendAppGossip(ctx context.Context, nodeIDs []ids.NodeID, appGossipBytes []byte) error {
	// Convert slice to set for consensus interface
	nodeIDSet := set.NewSet[ids.NodeID](len(nodeIDs))
	for _, nodeID := range nodeIDs {
		nodeIDSet.Add(nodeID)
	}

	return a.consensus.SendAppGossip(ctx, nodeIDSet, appGossipBytes)
}

// ReverseAppSenderAdapter adapts node AppSender to consensus AppSender interface
type ReverseAppSenderAdapter struct {
	node nodeblock.AppSender
}

// NewReverseAppSenderAdapter creates a reverse AppSender adapter
func NewReverseAppSenderAdapter(nodeAppSender nodeblock.AppSender) appsender.AppSender {
	return &ReverseAppSenderAdapter{node: nodeAppSender}
}

// SendAppRequest sends an app request (consensus to node)
func (a *ReverseAppSenderAdapter) SendAppRequest(ctx context.Context, nodeIDs set.Set[ids.NodeID], requestID uint32, appRequestBytes []byte) error {
	// Convert set to slice for node interface
	nodeIDSlice := make([]ids.NodeID, 0, nodeIDs.Len())
	for nodeID := range nodeIDs {
		nodeIDSlice = append(nodeIDSlice, nodeID)
	}

	return a.node.SendAppRequest(ctx, nodeIDSlice, requestID, appRequestBytes)
}

// SendAppResponse sends an app response (consensus to node)
func (a *ReverseAppSenderAdapter) SendAppResponse(ctx context.Context, nodeID ids.NodeID, requestID uint32, appResponseBytes []byte) error {
	return a.node.SendAppResponse(ctx, nodeID, requestID, appResponseBytes)
}

// SendAppError sends an app error (consensus to node)
func (a *ReverseAppSenderAdapter) SendAppError(ctx context.Context, nodeID ids.NodeID, requestID uint32, errorCode int32, errorMessage string) error {
	return a.node.SendAppError(ctx, nodeID, requestID, errorCode, errorMessage)
}

// SendAppGossip sends app gossip (consensus to node)
func (a *ReverseAppSenderAdapter) SendAppGossip(ctx context.Context, nodeIDs set.Set[ids.NodeID], appGossipBytes []byte) error {
	// Convert set to slice for node interface
	nodeIDSlice := make([]ids.NodeID, 0, nodeIDs.Len())
	for nodeID := range nodeIDs {
		nodeIDSlice = append(nodeIDSlice, nodeID)
	}

	return a.node.SendAppGossip(ctx, nodeIDSlice, appGossipBytes)
}

// SendAppGossipSpecific sends app gossip to specific nodes (consensus to node)
func (a *ReverseAppSenderAdapter) SendAppGossipSpecific(ctx context.Context, nodeIDs set.Set[ids.NodeID], appGossipBytes []byte) error {
	// Convert set to slice for node interface
	nodeIDSlice := make([]ids.NodeID, 0, nodeIDs.Len())
	for nodeID := range nodeIDs {
		nodeIDSlice = append(nodeIDSlice, nodeID)
	}

	return a.node.SendAppGossip(ctx, nodeIDSlice, appGossipBytes)
}
