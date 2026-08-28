// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package aichain

import (
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/common"
)

// parseKey parses a 0x-optional hex secp256k1 private key into a key + its EVM
// address.
func parseKey(hexKey string) (*ecdsa.PrivateKey, common.Address, error) {
	key, err := crypto.HexToECDSA(strings.TrimPrefix(hexKey, "0x"))
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("aichain: bad private key: %w", err)
	}
	// crypto.PubkeyToAddress returns luxfi/crypto/common.Address ([20]byte);
	// convert via bytes to the luxfi/geth/common.Address used throughout the SDK.
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return key, common.BytesToAddress(addr.Bytes()), nil
}
