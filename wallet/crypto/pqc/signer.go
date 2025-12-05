// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package pqc

import (
	"fmt"

	"github.com/luxfi/crypto/mldsa"
	"github.com/luxfi/crypto/mlkem"
	"github.com/luxfi/crypto/slhdsa"
	"github.com/luxfi/ids"
	walletcrypto "github.com/luxfi/sdk/wallet/crypto"
)

// signer implements post-quantum cryptography
type signer struct {
	scheme     walletcrypto.SignatureScheme
	privateKey []byte
	publicKey  []byte
	address    ids.ShortID
}

// NewSigner creates a new PQC signer
func NewSigner(scheme walletcrypto.SignatureScheme, privateKeyBytes []byte) (walletcrypto.Signer, error) {
	s := &signer{
		scheme:     scheme,
		privateKey: privateKeyBytes,
	}

	// Generate public key and address based on scheme
	switch scheme {
	case walletcrypto.MLDSA87:
		pubKey, err := mldsa.GeneratePublicKey87(privateKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ML-DSA-87 public key: %w", err)
		}
		s.publicKey = pubKey
		s.address = deriveAddress(pubKey)

	case walletcrypto.MLDSA65:
		pubKey, err := mldsa.GeneratePublicKey65(privateKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ML-DSA-65 public key: %w", err)
		}
		s.publicKey = pubKey
		s.address = deriveAddress(pubKey)

	case walletcrypto.MLDSA44:
		pubKey, err := mldsa.GeneratePublicKey44(privateKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ML-DSA-44 public key: %w", err)
		}
		s.publicKey = pubKey
		s.address = deriveAddress(pubKey)

	case walletcrypto.SLHDSA256:
		pubKey, err := slhdsa.GeneratePublicKey256(privateKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to generate SLH-DSA-256 public key: %w", err)
		}
		s.publicKey = pubKey
		s.address = deriveAddress(pubKey)

	default:
		return nil, fmt.Errorf("unsupported PQC scheme: %d", scheme)
	}

	return s, nil
}

func (s *signer) Sign(msg []byte) ([]byte, error) {
	switch s.scheme {
	case walletcrypto.MLDSA87:
		return mldsa.Sign87(s.privateKey, msg)
	case walletcrypto.MLDSA65:
		return mldsa.Sign65(s.privateKey, msg)
	case walletcrypto.MLDSA44:
		return mldsa.Sign44(s.privateKey, msg)
	case walletcrypto.SLHDSA256:
		return slhdsa.Sign256(s.privateKey, msg)
	default:
		return nil, fmt.Errorf("unsupported scheme: %d", s.scheme)
	}
}

func (s *signer) PublicKey() []byte {
	return s.publicKey
}

func (s *signer) Address() ids.ShortID {
	return s.address
}

func (s *signer) Scheme() walletcrypto.SignatureScheme {
	return s.scheme
}

func (s *signer) Verify(msg, sig []byte) bool {
	switch s.scheme {
	case walletcrypto.MLDSA87:
		return mldsa.Verify87(s.publicKey, msg, sig)
	case walletcrypto.MLDSA65:
		return mldsa.Verify65(s.publicKey, msg, sig)
	case walletcrypto.MLDSA44:
		return mldsa.Verify44(s.publicKey, msg, sig)
	case walletcrypto.SLHDSA256:
		return slhdsa.Verify256(s.publicKey, msg, sig)
	default:
		return false
	}
}

// deriveAddress creates an address from a public key
// Uses the same hashing as classic crypto for compatibility
func deriveAddress(publicKey []byte) ids.ShortID {
	// Hash the public key to create address
	// This ensures PQC addresses are the same format as classic
	hashed := ids.ToShortID(ids.HashBytes(publicKey))
	return hashed
}
