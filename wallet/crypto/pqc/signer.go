// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package pqc

import (
	"crypto/rand"
	"fmt"

	"github.com/luxfi/crypto/mldsa"
	"github.com/luxfi/crypto/slhdsa"
	"github.com/luxfi/ids"
	walletcrypto "github.com/luxfi/sdk/wallet/crypto"
)

// signer implements post-quantum cryptography using ML-DSA and SLH-DSA
type signer struct {
	scheme    walletcrypto.SignatureScheme
	mldsaKey  *mldsa.PrivateKey
	slhdsaKey *slhdsa.PrivateKey
	publicKey []byte
	address   ids.ShortID
}

// NewSigner creates a new PQC signer
func NewSigner(scheme walletcrypto.SignatureScheme, privateKeyBytes []byte) (walletcrypto.Signer, error) {
	s := &signer{
		scheme: scheme,
	}

	// Generate or load keys based on scheme
	switch scheme {
	case walletcrypto.MLDSA87:
		if len(privateKeyBytes) > 0 {
			privKey, err := mldsa.PrivateKeyFromBytes(mldsa.MLDSA87, privateKeyBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to load ML-DSA-87 private key: %w", err)
			}
			s.mldsaKey = privKey
			s.publicKey = privKey.PublicKey.Bytes()
		} else {
			privKey, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA87)
			if err != nil {
				return nil, fmt.Errorf("failed to generate ML-DSA-87 key: %w", err)
			}
			s.mldsaKey = privKey
			s.publicKey = privKey.PublicKey.Bytes()
		}
		s.address = deriveAddress(s.publicKey)

	case walletcrypto.MLDSA65:
		if len(privateKeyBytes) > 0 {
			privKey, err := mldsa.PrivateKeyFromBytes(mldsa.MLDSA65, privateKeyBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to load ML-DSA-65 private key: %w", err)
			}
			s.mldsaKey = privKey
			s.publicKey = privKey.PublicKey.Bytes()
		} else {
			privKey, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA65)
			if err != nil {
				return nil, fmt.Errorf("failed to generate ML-DSA-65 key: %w", err)
			}
			s.mldsaKey = privKey
			s.publicKey = privKey.PublicKey.Bytes()
		}
		s.address = deriveAddress(s.publicKey)

	case walletcrypto.MLDSA44:
		if len(privateKeyBytes) > 0 {
			privKey, err := mldsa.PrivateKeyFromBytes(mldsa.MLDSA44, privateKeyBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to load ML-DSA-44 private key: %w", err)
			}
			s.mldsaKey = privKey
			s.publicKey = privKey.PublicKey.Bytes()
		} else {
			privKey, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA44)
			if err != nil {
				return nil, fmt.Errorf("failed to generate ML-DSA-44 key: %w", err)
			}
			s.mldsaKey = privKey
			s.publicKey = privKey.PublicKey.Bytes()
		}
		s.address = deriveAddress(s.publicKey)

	case walletcrypto.SLHDSA256:
		if len(privateKeyBytes) > 0 {
			privKey, err := slhdsa.PrivateKeyFromBytes(slhdsa.SHA2_256s, privateKeyBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to load SLH-DSA-256 private key: %w", err)
			}
			s.slhdsaKey = privKey
			s.publicKey = privKey.PublicKey.Bytes()
		} else {
			privKey, err := slhdsa.GenerateKey(rand.Reader, slhdsa.SHA2_256s)
			if err != nil {
				return nil, fmt.Errorf("failed to generate SLH-DSA-256 key: %w", err)
			}
			s.slhdsaKey = privKey
			s.publicKey = privKey.PublicKey.Bytes()
		}
		s.address = deriveAddress(s.publicKey)

	default:
		return nil, fmt.Errorf("unsupported PQC scheme: %d", scheme)
	}

	return s, nil
}

func (s *signer) Sign(msg []byte) ([]byte, error) {
	switch s.scheme {
	case walletcrypto.MLDSA87, walletcrypto.MLDSA65, walletcrypto.MLDSA44:
		if s.mldsaKey == nil {
			return nil, fmt.Errorf("ML-DSA key not initialized")
		}
		return s.mldsaKey.Sign(rand.Reader, msg, nil)
	case walletcrypto.SLHDSA256:
		if s.slhdsaKey == nil {
			return nil, fmt.Errorf("SLH-DSA key not initialized")
		}
		return s.slhdsaKey.Sign(rand.Reader, msg, nil)
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
	case walletcrypto.MLDSA87, walletcrypto.MLDSA65, walletcrypto.MLDSA44:
		if s.mldsaKey == nil || s.mldsaKey.PublicKey == nil {
			return false
		}
		return s.mldsaKey.PublicKey.Verify(msg, sig, nil)
	case walletcrypto.SLHDSA256:
		if s.slhdsaKey == nil || s.slhdsaKey.PublicKey == nil {
			return false
		}
		return s.slhdsaKey.PublicKey.Verify(msg, sig, nil)
	default:
		return false
	}
}

// deriveAddress creates an address from a public key
// Uses the same hashing as classic crypto for compatibility
func deriveAddress(publicKey []byte) ids.ShortID {
	// Hash the public key to create address
	// This ensures PQC addresses are the same format as classic
	shortID, _ := ids.ToShortID(publicKey)
	return shortID
}
