package compat

import (
	"context"

	nodecore "github.com/luxfi/node/consensus/engine/core"
	commonEng "github.com/luxfi/consensus/core"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
)

// NodeAppSenderAdapter adapts nodecore.AppSender to commonEng.AppSender
type NodeAppSenderAdapter struct {
	NodeSender nodecore.AppSender
}

// NewNodeAppSenderAdapter creates a new adapter
func NewNodeAppSenderAdapter(sender nodecore.AppSender) commonEng.AppSender {
	return &NodeAppSenderAdapter{NodeSender: sender}
}

// SendAppRequest sends an application-level request
func (a *NodeAppSenderAdapter) SendAppRequest(ctx context.Context, nodeIDs set.Set[ids.NodeID], requestID uint32, appRequestBytes []byte) error {
	// Convert set.Set to SendConfig
	config := nodecore.SendConfig{
		NodeIDs: nodeIDs,
	}
	return a.NodeSender.SendAppRequest(ctx, config, requestID, appRequestBytes)
}

// SendAppResponse sends an application-level response
func (a *NodeAppSenderAdapter) SendAppResponse(ctx context.Context, nodeID ids.NodeID, requestID uint32, appResponseBytes []byte) error {
	return a.NodeSender.SendAppResponse(ctx, nodeID, requestID, appResponseBytes)
}

// SendAppError sends an application-level error
func (a *NodeAppSenderAdapter) SendAppError(ctx context.Context, nodeID ids.NodeID, requestID uint32, errorCode int32, errorMessage string) error {
	return a.NodeSender.SendAppError(ctx, nodeID, requestID, errorCode, errorMessage)
}

// SendAppGossip sends an application-level gossip message
func (a *NodeAppSenderAdapter) SendAppGossip(ctx context.Context, nodeIDs set.Set[ids.NodeID], appGossipBytes []byte) error {
	// Convert set.Set to SendConfig
	config := nodecore.SendConfig{
		NodeIDs: nodeIDs,
	}
	return a.NodeSender.SendAppGossip(ctx, config, appGossipBytes)
}

// SendAppGossipSpecific sends a gossip message to specific nodes
func (a *NodeAppSenderAdapter) SendAppGossipSpecific(ctx context.Context, nodeIDs set.Set[ids.NodeID], appGossipBytes []byte) error {
	// Convert set.Set to SendConfig
	config := nodecore.SendConfig{
		NodeIDs: nodeIDs,
	}
	return a.NodeSender.SendAppGossipSpecific(ctx, config, appGossipBytes)
}

// SendCrossChainAppRequest sends a cross-chain application request
func (a *NodeAppSenderAdapter) SendCrossChainAppRequest(ctx context.Context, chainID ids.ID, requestID uint32, appRequestBytes []byte) error {
	return a.NodeSender.SendCrossChainAppRequest(ctx, chainID, requestID, appRequestBytes)
}

// SendCrossChainAppResponse sends a cross-chain application response
func (a *NodeAppSenderAdapter) SendCrossChainAppResponse(ctx context.Context, chainID ids.ID, requestID uint32, appResponseBytes []byte) error {
	return a.NodeSender.SendCrossChainAppResponse(ctx, chainID, requestID, appResponseBytes)
}

// SendCrossChainAppError sends a cross-chain application error
func (a *NodeAppSenderAdapter) SendCrossChainAppError(ctx context.Context, chainID ids.ID, requestID uint32, errorCode int32, errorMessage string) error {
	return a.NodeSender.SendCrossChainAppError(ctx, chainID, requestID, errorCode, errorMessage)
}