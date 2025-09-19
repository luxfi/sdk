package compat

import (
	"context"
	"errors"

	consensusCore "github.com/luxfi/consensus/core/appsender"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
)

var (
	// ErrCrossChainNotSupported indicates that cross-chain communication is not supported
	ErrCrossChainNotSupported = errors.New("cross-chain communication not supported by consensus module")
)

// AppSenderAdapter adapts between consensus AppSender and compat AppSender
type AppSenderAdapter struct {
	wrapped consensusCore.AppSender
}

// NewAppSenderAdapter creates a new adapter
func NewAppSenderAdapter(sender consensusCore.AppSender) AppSender {
	return &AppSenderAdapter{wrapped: sender}
}

// SendAppRequest sends an application-level request
func (a *AppSenderAdapter) SendAppRequest(ctx context.Context, nodeIDs set.Set[ids.NodeID], requestID uint32, appRequestBytes []byte) error {
	// No conversion needed - same underlying type
	return a.wrapped.SendAppRequest(ctx, nodeIDs, requestID, appRequestBytes)
}

// SendAppResponse sends an application-level response
func (a *AppSenderAdapter) SendAppResponse(ctx context.Context, nodeID ids.NodeID, requestID uint32, appResponseBytes []byte) error {
	return a.wrapped.SendAppResponse(ctx, nodeID, requestID, appResponseBytes)
}

// SendAppError sends an application-level error
func (a *AppSenderAdapter) SendAppError(ctx context.Context, nodeID ids.NodeID, requestID uint32, errorCode int32, errorMessage string) error {
	return a.wrapped.SendAppError(ctx, nodeID, requestID, errorCode, errorMessage)
}

// SendAppGossip sends an application-level gossip message
func (a *AppSenderAdapter) SendAppGossip(ctx context.Context, nodeIDs set.Set[ids.NodeID], appGossipBytes []byte) error {
	// No conversion needed - same underlying type
	return a.wrapped.SendAppGossip(ctx, nodeIDs, appGossipBytes)
}

// SendAppGossipSpecific sends a gossip message to specific nodes
func (a *AppSenderAdapter) SendAppGossipSpecific(ctx context.Context, nodeIDs set.Set[ids.NodeID], appGossipBytes []byte) error {
	// No conversion needed - same underlying type
	return a.wrapped.SendAppGossipSpecific(ctx, nodeIDs, appGossipBytes)
}

// SendCrossChainAppRequest sends a cross-chain app request
func (a *AppSenderAdapter) SendCrossChainAppRequest(ctx context.Context, chainID ids.ID, requestID uint32, appRequestBytes []byte) error {
	// Cross-chain communication not supported by consensus module
	return ErrCrossChainNotSupported
}

// SendCrossChainAppResponse sends a cross-chain app response
func (a *AppSenderAdapter) SendCrossChainAppResponse(ctx context.Context, chainID ids.ID, requestID uint32, appResponseBytes []byte) error {
	// Cross-chain communication not supported by consensus module
	return ErrCrossChainNotSupported
}

// SendCrossChainAppError sends a cross-chain app error
func (a *AppSenderAdapter) SendCrossChainAppError(ctx context.Context, chainID ids.ID, requestID uint32, errorCode int32, errorMessage string) error {
	// Cross-chain communication not supported by consensus module
	return ErrCrossChainNotSupported
}
