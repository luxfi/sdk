// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
)

// LocalSigner implements signing with luxfi/crypto/bls
type LocalSigner struct {
	sk *bls.SecretKey
	pk *bls.PublicKey
}

// NewLocalSigner creates a new local signer using luxfi/crypto/bls
func NewLocalSigner(sk *bls.SecretKey) *LocalSigner {
	return &LocalSigner{
		sk: sk,
		pk: bls.PublicFromSecretKey(sk),
	}
}

// Sign signs the message with the private key
func (s *LocalSigner) Sign(msg []byte) ([]byte, error) {
	sig := bls.Sign(s.sk, msg)
	return bls.SignatureToBytes(sig), nil
}

// GetPublicKey returns the public key
func (s *LocalSigner) GetPublicKey() *bls.PublicKey {
	return s.pk
}

// PublicKey returns the public key as bytes
func (s *LocalSigner) PublicKey() []byte {
	return bls.PublicKeyToUncompressedBytes(s.pk)
}

// NodeID returns the node ID derived from the public key
func (s *LocalSigner) NodeID() ids.NodeID {
	// Create a dummy NodeID for testing
	return ids.GenerateTestNodeID()
}