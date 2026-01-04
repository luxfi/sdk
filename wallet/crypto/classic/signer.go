// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package classic provides classic cryptographic signing implementation.
package classic

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/luxfi/crypto"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/ids"
	walletcrypto "github.com/luxfi/sdk/wallet/crypto"
)

// signer implements classic cryptography (secp256k1)
type signer struct {
	scheme     walletcrypto.SignatureScheme
	privateKey *ecdsa.PrivateKey
}

// NewSigner creates a new classic crypto signer
func NewSigner(scheme walletcrypto.SignatureScheme, privateKeyBytes []byte) (walletcrypto.Signer, error) {
	switch scheme {
	case walletcrypto.SECP256K1:
		privateKey, err := crypto.ToECDSA(privateKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse secp256k1 private key: %w", err)
		}
		return &signer{
			scheme:     scheme,
			privateKey: privateKey,
		}, nil
	case walletcrypto.SECP256R1:
		// TODO: Implement secp256r1
		return nil, fmt.Errorf("secp256r1 not yet implemented")
	case walletcrypto.BLS:
		// TODO: Implement BLS
		return nil, fmt.Errorf("BLS not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported classic scheme: %d", scheme)
	}
}

func (s *signer) Sign(msg []byte) ([]byte, error) {
	// Hash the message and sign
	hash := crypto.Keccak256(msg)
	return secp256k1.Sign(hash, crypto.FromECDSA(s.privateKey))
}

func (s *signer) PublicKey() []byte {
	return crypto.FromECDSAPub(&s.privateKey.PublicKey)
}

func (s *signer) Address() ids.ShortID {
	addr := crypto.PubkeyToAddress(s.privateKey.PublicKey)
	var shortID ids.ShortID
	copy(shortID[:], addr[:])
	return shortID
}

func (s *signer) Scheme() walletcrypto.SignatureScheme {
	return s.scheme
}

func (s *signer) Verify(msg, sig []byte) bool {
	hash := crypto.Keccak256(msg)
	pubKey, err := secp256k1.RecoverPubkey(hash, sig)
	if err != nil {
		return false
	}
	// Compare with our public key
	return string(pubKey) == string(s.PublicKey())
}
