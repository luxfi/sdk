// Copyright (C) 2020-2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

package deploy

import (
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/node/vms/secp256k1fx"
	"github.com/luxfi/node/wallet/net/primary"
)

// EWOQKeyBytes are the genesis key bytes used in local networks
var EWOQKeyBytes = []byte{
	0x56, 0x28, 0x9e, 0x99, 0xc9, 0x4b, 0x69, 0x12,
	0xbf, 0xc1, 0x2a, 0xdc, 0x09, 0x3c, 0x9b, 0x51,
	0x12, 0x4f, 0x0d, 0xc5, 0x4a, 0xc7, 0xa7, 0x66,
	0xb2, 0xbc, 0x5c, 0xcf, 0x55, 0x8d, 0x80, 0x27,
}

// KeychainFromPrivateKey creates a keychain adapter from a single private key.
// The returned adapter implements both wallet/keychain.Keychain and c.EthKeychain.
func KeychainFromPrivateKey(key *secp256k1.PrivateKey) *primary.KeychainAdapter {
	return primary.NewKeychainAdapter(secp256k1fx.NewKeychain(key))
}

// KeychainFromPrivateKeyBytes creates a keychain adapter from private key bytes.
func KeychainFromPrivateKeyBytes(keyBytes []byte) (*primary.KeychainAdapter, error) {
	key, err := secp256k1.ToPrivateKey(keyBytes)
	if err != nil {
		return nil, err
	}
	return KeychainFromPrivateKey(key), nil
}

// EWOQKeychain returns a keychain containing the EWOQ (genesis) key.
// This is useful for local development and testing.
func EWOQKeychain() *primary.KeychainAdapter {
	kc, _ := KeychainFromPrivateKeyBytes(EWOQKeyBytes)
	return kc
}
