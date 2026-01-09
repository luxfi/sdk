// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build ignore

package keychain

import (
	"errors"

	"github.com/luxfi/crypto"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
)

// UnifiedKeychain uses the unified crypto interface for all key operations
type UnifiedKeychain struct {
	crypto        crypto.UnifiedCrypto
	keysByAddress map[ids.ShortID]*UnifiedSigner
	addressSet    set.Set[ids.ShortID]
	defaultAlgo   crypto.Algorithm
}

// NewUnifiedKeychain creates a new keychain using the unified crypto interface
func NewUnifiedKeychain(defaultAlgo crypto.Algorithm) *UnifiedKeychain {
	return &UnifiedKeychain{
		crypto:        crypto.NewUnifiedCrypto(),
		keysByAddress: make(map[ids.ShortID]*UnifiedSigner),
		addressSet:    set.NewSet[ids.ShortID](0),
		defaultAlgo:   defaultAlgo,
	}
}

// GenerateKey generates a new key with the default algorithm
func (kc *UnifiedKeychain) GenerateKey() (ids.ShortID, error) {
	privKey, pubKey, err := kc.crypto.GenerateKey(kc.defaultAlgo)
	if err != nil {
		return ids.ShortEmpty, err
	}

	return kc.AddKey(privKey, pubKey), nil
}

// GenerateKeyWithAlgo generates a new key with the specified algorithm
func (kc *UnifiedKeychain) GenerateKeyWithAlgo(algo crypto.Algorithm) (ids.ShortID, error) {
	privKey, pubKey, err := kc.crypto.GenerateKey(algo)
	if err != nil {
		return ids.ShortEmpty, err
	}

	return kc.AddKey(privKey, pubKey), nil
}

// AddKey adds a key pair to the keychain
func (kc *UnifiedKeychain) AddKey(privKey crypto.PrivateKey, pubKey crypto.PublicKey) ids.ShortID {
	addrBytes := pubKey.Address()
	if len(addrBytes) < 20 {
		// Pad if necessary
		padded := make([]byte, 20)
		copy(padded, addrBytes)
		addrBytes = padded
	}

	addr := ids.ShortID{}
	copy(addr[:], addrBytes[:20])

	signer := &UnifiedSigner{
		privKey: privKey,
		pubKey:  pubKey,
		address: addr,
		crypto:  kc.crypto,
	}

	kc.keysByAddress[addr] = signer
	kc.addressSet.Add(addr)

	return addr
}

// Get returns the signer for the given address
func (kc *UnifiedKeychain) Get(addr ids.ShortID) (Signer, bool) {
	signer, exists := kc.keysByAddress[addr]
	if !exists {
		return nil, false
	}
	return signer, true
}

// Addresses returns all addresses in the keychain
func (kc *UnifiedKeychain) Addresses() []ids.ShortID {
	addrs := make([]ids.ShortID, 0, kc.addressSet.Len())
	for addr := range kc.addressSet {
		addrs = append(addrs, addr)
	}
	return addrs
}

// GetUnifiedSigner returns the unified signer for advanced operations
func (kc *UnifiedKeychain) GetUnifiedSigner(addr ids.ShortID) (*UnifiedSigner, bool) {
	signer, exists := kc.keysByAddress[addr]
	return signer, exists
}

// UnifiedSigner implements the Signer interface using unified crypto
type UnifiedSigner struct {
	privKey crypto.PrivateKey
	pubKey  crypto.PublicKey
	address ids.ShortID
	crypto  crypto.UnifiedCrypto
}

// Sign signs a message
func (s *UnifiedSigner) Sign(msg []byte) ([]byte, error) {
	return s.crypto.Sign(s.privKey, msg)
}

// SignHash signs a hash (for compatibility)
func (s *UnifiedSigner) SignHash(hash []byte) ([]byte, error) {
	// For algorithms that expect raw hashes
	switch s.privKey.Algorithm() {
	case crypto.AlgoSecp256k1:
		// secp256k1 expects 32-byte hash
		if len(hash) != 32 {
			return nil, errors.New("invalid hash length for secp256k1")
		}
	}
	return s.privKey.Sign(hash)
}

// Address returns the address
func (s *UnifiedSigner) Address() ids.ShortID {
	return s.address
}

// Algorithm returns the algorithm used by this signer
func (s *UnifiedSigner) Algorithm() crypto.Algorithm {
	return s.privKey.Algorithm()
}

// PublicKey returns the public key
func (s *UnifiedSigner) PublicKey() crypto.PublicKey {
	return s.pubKey
}

// SignRing creates a ring signature (for privacy-preserving signatures)
func (s *UnifiedSigner) SignRing(message []byte, ring []crypto.PublicKey) ([]byte, error) {
	return s.crypto.SignRing(s.privKey, message, ring)
}

// Encapsulate creates a ciphertext and shared secret (for ML-KEM)
func (s *UnifiedSigner) Encapsulate(pubKey crypto.PublicKey) (ciphertext []byte, sharedSecret []byte, err error) {
	return s.crypto.Encapsulate(pubKey)
}

// Decapsulate recovers the shared secret from ciphertext (for ML-KEM)
func (s *UnifiedSigner) Decapsulate(ciphertext []byte) (sharedSecret []byte, err error) {
	return s.crypto.Decapsulate(s.privKey, ciphertext)
}
