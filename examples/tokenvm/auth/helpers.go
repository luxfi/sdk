// Copyright (C) 2023-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package auth

import (
	"github.com/luxdefi/vmsdk/chain"
	"github.com/luxdefi/vmsdk/crypto"
)

func GetActor(auth chain.Auth) crypto.PublicKey {
	switch a := auth.(type) {
	case *ED25519:
		return a.Signer
	default:
		return crypto.EmptyPublicKey
	}
}

func GetSigner(auth chain.Auth) crypto.PublicKey {
	switch a := auth.(type) {
	case *ED25519:
		return a.Signer
	default:
		return crypto.EmptyPublicKey
	}
}
