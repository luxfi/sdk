// Package compat provides compatibility shims for consensus types
// that are expected by the EVM but not available in the local consensus module.
package compat

import (
	"context"

	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
)

// AppSender sends application messages
type AppSender interface {
	// Send an application-level request.
	SendAppRequest(ctx context.Context, nodeIDs set.Set[ids.NodeID], requestID uint32, appRequestBytes []byte) error
	// Send an application-level response to a request.
	SendAppResponse(ctx context.Context, nodeID ids.NodeID, requestID uint32, appResponseBytes []byte) error
	// SendAppError sends an application-level error to an AppRequest
	SendAppError(ctx context.Context, nodeID ids.NodeID, requestID uint32, errorCode int32, errorMessage string) error
	// Gossip an application-level message.
	SendAppGossip(ctx context.Context, nodeIDs set.Set[ids.NodeID], appGossipBytes []byte) error
	// SendAppGossipSpecific sends a gossip message to a list of nodeIDs
	SendAppGossipSpecific(ctx context.Context, nodeIDs set.Set[ids.NodeID], appGossipBytes []byte) error

	// Cross-chain communication
	// Send a cross-chain app request to another chain
	SendCrossChainAppRequest(ctx context.Context, chainID ids.ID, requestID uint32, appRequestBytes []byte) error
	// Send a cross-chain app response to a request from another chain
	SendCrossChainAppResponse(ctx context.Context, chainID ids.ID, requestID uint32, appResponseBytes []byte) error
	// Send a cross-chain app error in response to a request from another chain
	SendCrossChainAppError(ctx context.Context, chainID ids.ID, requestID uint32, errorCode int32, errorMessage string) error
}

// AppHandler handles application messages
type AppHandler interface {
	// Handle an application-level request
	AppRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, deadline int64, appRequestBytes []byte) error
	// Handle an application-level response
	AppResponse(ctx context.Context, nodeID ids.NodeID, requestID uint32, appResponseBytes []byte) error
	// Handle an application-level error
	AppRequestFailed(ctx context.Context, nodeID ids.NodeID, requestID uint32, appErr *AppError) error
	// Handle an application-level gossip message
	AppGossip(ctx context.Context, nodeID ids.NodeID, appGossipBytes []byte) error

	// Cross-chain
	CrossChainAppRequest(ctx context.Context, chainID ids.ID, requestID uint32, deadline int64, appRequestBytes []byte) error
	CrossChainAppResponse(ctx context.Context, chainID ids.ID, requestID uint32, appResponseBytes []byte) error
	CrossChainAppRequestFailed(ctx context.Context, chainID ids.ID, requestID uint32, appErr *AppError) error
}

// AppError represents an application-level error
type AppError struct {
	Code    int32
	Message string
}

// Error implements the error interface
func (e *AppError) Error() string {
	return e.Message
}
