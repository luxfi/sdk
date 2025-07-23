// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package localsigner

import (
	"crypto/rand"
	"github.com/luxfi/evm/interfaces"
)

// SecretKey represents a BLS secret key
type SecretKey struct {
	bytes [32]byte
}

// New generates a new random BLS secret key
func New() (*SecretKey, error) {
	sk := &SecretKey{}
	_, err := rand.Read(sk.bytes[:])
	if err != nil {
		return nil, err
	}
	return sk, nil
}

// Bytes returns the secret key bytes
func (sk *SecretKey) Bytes() []byte {
	return sk.bytes[:]
}

// ToPublicKey converts the secret key to a public key
func (sk *SecretKey) ToPublicKey() interfaces.PublicKey {
	// For testing purposes, we just use part of the secret key as public key
	var pk interfaces.PublicKey
	copy(pk[:], sk.bytes[:48])
	return pk
}