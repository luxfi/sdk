// (c) 2019-2021, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package message

import (
	"context"
	"github.com/luxfi/evm/interfaces"
)

var (
	_ RequestHandler = NoopRequestHandler{}
)

// RequestHandler interface handles incoming requests from peers
// Must have methods in format of handleType(context.Context, interfaces.NodeID, uint32, request Type) error
// so that the Request object of relevant Type can invoke its respective handle method
// on this struct.
// Also see GossipHandler for implementation style.
type RequestHandler interface {
	HandleStateTrieLeafsRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, leafsRequest LeafsRequest) ([]byte, error)
	HandleBlockRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, request BlockRequest) ([]byte, error)
	HandleCodeRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, codeRequest CodeRequest) ([]byte, error)
	HandleMessageSignatureRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, signatureRequest MessageSignatureRequest) ([]byte, error)
	HandleBlockSignatureRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, signatureRequest BlockSignatureRequest) ([]byte, error)
}

// ResponseHandler handles response for a sent request
// Only one of OnResponse or OnFailure is called for a given requestID, not both
type ResponseHandler interface {
	// OnResponse is invoked when the peer responded to a request
	OnResponse(response []byte) error
	// OnFailure is invoked when there was a failure in processing a request
	OnFailure() error
}

// CrossChainRequestHandler interface handles incoming cross chain requests
type CrossChainRequestHandler interface {
	HandleCrossChainRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, request []byte) ([]byte, error)
}

// GossipHandler interface handles incoming gossip messages
type GossipHandler interface {
	HandleGossip(ctx context.Context, nodeID interfaces.NodeID, gossipBytes []byte)
}

// GossipMessage is a marker interface for gossip messages
type GossipMessage interface {
	// Handle is called to process this gossip message
	Handle(handler GossipHandler, nodeID interfaces.NodeID) error
}

type NoopRequestHandler struct{}

func (NoopRequestHandler) HandleStateTrieLeafsRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, leafsRequest LeafsRequest) ([]byte, error) {
	return nil, nil
}

func (NoopRequestHandler) HandleBlockRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, request BlockRequest) ([]byte, error) {
	return nil, nil
}

func (NoopRequestHandler) HandleCodeRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, codeRequest CodeRequest) ([]byte, error) {
	return nil, nil
}

func (NoopRequestHandler) HandleMessageSignatureRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, signatureRequest MessageSignatureRequest) ([]byte, error) {
	return nil, nil
}

func (NoopRequestHandler) HandleBlockSignatureRequest(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, signatureRequest BlockSignatureRequest) ([]byte, error) {
	return nil, nil
}
