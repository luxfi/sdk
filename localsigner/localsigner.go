// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package localsigner

import (
	"github.com/luxfi/node/utils/crypto/bls"
	"github.com/luxfi/node/utils/crypto/bls/signer/localsigner"
)

// SecretKey is a wrapper around the BLS local signer
// SecretKey is a wrapper around the BLS secret key.
type SecretKey struct {
	sk *bls.SecretKey
}

// Inner returns the underlying BLS SecretKey.
func (sk *SecretKey) Inner() *bls.SecretKey {
	return sk.sk
}

// New generates a new random BLS secret key
// New generates a new random BLS secret key.
func New() (*SecretKey, error) {
	sk, err := localsigner.New()
	if err != nil {
		return nil, err
	}
	return &SecretKey{sk: sk}, nil
}

// Bytes returns the secret key bytes
// Bytes returns the secret key bytes.
func (sk *SecretKey) Bytes() []byte {
	return bls.SecretKeyToBytes(sk.sk)
}

// PublicKey returns the public key corresponding to this secret key
// PublicKey returns the public key corresponding to this secret key.
func (sk *SecretKey) PublicKey() *bls.PublicKey {
	return bls.PublicFromSecretKey(sk.sk)
}

// Sign signs a message with this secret key
// Sign signs a message with this secret key.
func (sk *SecretKey) Sign(msg []byte) *bls.Signature {
	return bls.Sign(sk.sk, msg)
}

// SignProofOfPossession signs a proof of possession message
// SignProofOfPossession signs a proof of possession message.
func (sk *SecretKey) SignProofOfPossession(msg []byte) *bls.Signature {
	return bls.SignProofOfPossession(sk.sk, msg)
}
