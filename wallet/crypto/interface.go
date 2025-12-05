// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package crypto

import (
	"crypto/rand"

	"github.com/luxfi/crypto"
	"github.com/luxfi/ids"
)

// SignatureScheme represents the type of cryptographic signature scheme
type SignatureScheme uint8

const (
	// Classic cryptography
	SECP256K1 SignatureScheme = iota
	SECP256R1
	BLS

	// Post-quantum cryptography (NIST standards)
	MLDSA87   // ML-DSA-87 (Dilithium5 equivalent)
	MLDSA65   // ML-DSA-65 (Dilithium3 equivalent)
	MLDSA44   // ML-DSA-44 (Dilithium2 equivalent)
	SLHDSA256 // SLH-DSA-256 (SPHINCS+ equivalent)
	SLHDSA192 // SLH-DSA-192
	SLHDSA128 // SLH-DSA-128

	// Hybrid schemes (classic + PQC)
	SECP256K1_MLDSA87 // secp256k1 + ML-DSA-87
	BLS_MLDSA87       // BLS + ML-DSA-87
)

// Signer represents a unified interface for signing transactions
// across all chains (P, X, C, EVM) with both classic and PQC support
type Signer interface {
	// Sign signs the message with the private key
	Sign(msg []byte) ([]byte, error)

	// PublicKey returns the public key
	PublicKey() []byte

	// Address returns the address derived from the public key
	Address() ids.ShortID

	// Scheme returns the signature scheme used
	Scheme() SignatureScheme

	// Verify verifies a signature
	Verify(msg, sig []byte) bool
}

// Factory creates signers for different schemes
type Factory interface {
	// NewSigner creates a new signer from a private key
	NewSigner(scheme SignatureScheme, privateKey []byte) (Signer, error)

	// NewSignerFromSeed creates a signer from a seed
	NewSignerFromSeed(scheme SignatureScheme, seed []byte) (Signer, error)

	// GenerateKey generates a new private key
	GenerateKey(scheme SignatureScheme) (privateKey []byte, err error)
}

// KeyManager handles key derivation and storage
type KeyManager interface {
	// DeriveKey derives a key using BIP32/BIP44 path
	DeriveKey(path string) ([]byte, error)

	// SupportsScheme checks if the key manager supports the scheme
	SupportsScheme(scheme SignatureScheme) bool
}

// NewFactory creates a new crypto factory
func NewFactory() Factory {
	return &factory{}
}

type factory struct{}

func (f *factory) NewSigner(scheme SignatureScheme, privateKey []byte) (Signer, error) {
	// Route to classic or PQC implementation based on scheme
	if isPQCScheme(scheme) {
		return newPQCSigner(scheme, privateKey)
	}
	return newClassicSigner(scheme, privateKey)
}

func (f *factory) NewSignerFromSeed(scheme SignatureScheme, seed []byte) (Signer, error) {
	// Derive private key from seed based on scheme
	if isPQCScheme(scheme) {
		return newPQCSigner(scheme, seed)
	}
	return newClassicSigner(scheme, seed)
}

func isPQCScheme(scheme SignatureScheme) bool {
	return scheme >= MLDSA87 && scheme <= BLS_MLDSA87
}

func (f *factory) GenerateKey(scheme SignatureScheme) ([]byte, error) {
	// Generate key based on scheme
	switch scheme {
	case SECP256K1:
		// Generate secp256k1 private key using luxfi/crypto
		key, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}
		return crypto.FromECDSA(key), nil
	default:
		// For other schemes, generate random bytes of appropriate length
		privKey := make([]byte, 32)
		_, err := rand.Read(privKey)
		return privKey, err
	}
}

// Helper functions to instantiate signers from sub-packages
// These will be implemented by importing classic and pqc packages
func newClassicSigner(scheme SignatureScheme, privateKey []byte) (Signer, error) {
	// Import from classic package
	// return classic.NewSigner(scheme, privateKey)
	return nil, nil // Placeholder
}

func newPQCSigner(scheme SignatureScheme, privateKey []byte) (Signer, error) {
	// Import from pqc package
	// return pqc.NewSigner(scheme, privateKey)
	return nil, nil // Placeholder
}
