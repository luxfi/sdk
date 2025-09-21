// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	platformWarp "github.com/luxfi/node/vms/platformvm/warp"
	luxWarp "github.com/luxfi/warp"
)

// LP118SignerAdapter adapts a LocalSigner to the lp118.Signer interface
type LP118SignerAdapter struct {
	signer *LocalSigner
}

// NewLP118SignerAdapter creates a new adapter
func NewLP118SignerAdapter(signer *LocalSigner) *LP118SignerAdapter {
	return &LP118SignerAdapter{
		signer: signer,
	}
}

// Sign implements the lp118.Signer interface (using platformvm/warp.UnsignedMessage)
func (a *LP118SignerAdapter) Sign(unsignedMsg *platformWarp.UnsignedMessage) ([]byte, error) {
	// Convert platformvm/warp.UnsignedMessage to luxfi/warp.UnsignedMessage
	msg, err := luxWarp.NewUnsignedMessage(
		unsignedMsg.NetworkID,
		unsignedMsg.SourceChainID[:],
		unsignedMsg.Payload,
	)
	if err != nil {
		return nil, err
	}
	return a.signer.SignUnsignedMessage(msg)
}
