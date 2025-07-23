// (c) 2023, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package message

import (
	"context"
	"fmt"

	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
)

var (
	_ Request = MessageSignatureRequest{}
	_ Request = BlockSignatureRequest{}
)

// MessageSignatureRequest is used to request a warp message's signature.
type MessageSignatureRequest struct {
	MessageID interfaces.ID `serialize:"true"`
}

func (s MessageSignatureRequest) String() string {
	return fmt.Sprintf("MessageSignatureRequest(MessageID=%s)", s.MessageID.String())
}

func (s MessageSignatureRequest) Handle(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, handler RequestHandler) ([]byte, error) {
	return handler.HandleMessageSignatureRequest(ctx, nodeID, requestID, s)
}

// BlockSignatureRequest is used to request a warp message's signature.
type BlockSignatureRequest struct {
	BlockID interfaces.ID `serialize:"true"`
}

func (s BlockSignatureRequest) String() string {
	return fmt.Sprintf("BlockSignatureRequest(BlockID=%s)", s.BlockID.String())
}

func (s BlockSignatureRequest) Handle(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, handler RequestHandler) ([]byte, error) {
	return handler.HandleBlockSignatureRequest(ctx, nodeID, requestID, s)
}

// SignatureResponse is the response to a BlockSignatureRequest or MessageSignatureRequest.
// The response contains a BLS signature of the requested message, signed by the responding node's BLS private key.
type SignatureResponse struct {
	Signature [interfaces.SignatureLen]byte `serialize:"true"`
}
