// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package localsigner

import (
	"github.com/luxfi/node/utils/crypto/bls"
	"github.com/luxfi/node/utils/crypto/bls/signer/localsigner"
)

// SecretKey is a wrapper around the BLS local signer
type SecretKey struct {
	signer *localsigner.LocalSigner
}

// New generates a new random BLS secret key
func New() (*SecretKey, error) {
	signer, err := localsigner.New()
	if err != nil {
		return nil, err
	}
	return &SecretKey{signer: signer}, nil
}

// Bytes returns the secret key bytes
func (sk *SecretKey) Bytes() []byte {
	return sk.signer.ToBytes()
}

// PublicKey returns the public key corresponding to this secret key
func (sk *SecretKey) PublicKey() *bls.PublicKey {
	return sk.signer.PublicKey()
}

// Sign signs a message with this secret key
func (sk *SecretKey) Sign(msg []byte) (*bls.Signature, error) {
	return sk.signer.Sign(msg)
}

// SignProofOfPossession signs a proof of possession message
func (sk *SecretKey) SignProofOfPossession(msg []byte) (*bls.Signature, error) {
	return sk.signer.SignProofOfPossession(msg)
}