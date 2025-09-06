// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	luxWarp "github.com/luxfi/node/vms/platformvm/warp"
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

// Sign implements the lp118.Signer interface
func (a *LP118SignerAdapter) Sign(unsignedMsg *luxWarp.UnsignedMessage) ([]byte, error) {
	return a.signer.SignUnsignedMessage(unsignedMsg)
}