// Copyright (C) 2023-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"github.com/luxdefi/vmsdk/crypto"

	"github.com/luxdefi/vmsdk/examples/tokenvm/consts"
)

func Address(pk crypto.PublicKey) string {
	return crypto.Address(consts.HRP, pk)
}

func ParseAddress(s string) (crypto.PublicKey, error) {
	return crypto.ParseAddress(consts.HRP, s)
}
