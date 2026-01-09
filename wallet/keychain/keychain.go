// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keychain

import (
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
)

// Keychain interface that wallet signers can use
// This allows both secp256k1fx.Keychain and ledger-lux-go/keychain.Keychain to be used
// Generic across chains, DAGs, and post-quantum crypto
type Keychain interface {
	Addresses() set.Set[ids.ShortID]
	Get(ids.ShortID) (Signer, bool)
}

// Signer interface for signing operations
// Generic interface for all signing needs (classical and post-quantum)
type Signer interface {
	SignHash([]byte) ([]byte, error)
	Sign([]byte) ([]byte, error)
	Address() ids.ShortID
}
