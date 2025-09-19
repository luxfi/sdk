// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"github.com/luxfi/ids"
	luxWarp "github.com/luxfi/warp"
)

// WarpSigner defines the interface for signing warp messages
type WarpSigner interface {
	// Sign signs a message and returns the signature bytes
	Sign(msg []byte) ([]byte, error)

	// PublicKey returns the public key bytes
	PublicKey() []byte

	// NodeID returns the node ID
	NodeID() ids.NodeID
}

// SignMessage signs a warp unsigned message
func SignMessage(signer WarpSigner, msg *luxWarp.UnsignedMessage) ([]byte, error) {
	return signer.Sign(msg.Bytes())
}
