// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package classic

import (
	"fmt"

	"github.com/luxfi/crypto"
	"github.com/luxfi/ids"
	walletcrypto "github.com/luxfi/sdk/wallet/crypto"
)

// signer implements classic cryptography (secp256k1, secp256r1, BLS)
type signer struct {
	scheme     walletcrypto.SignatureScheme
	privateKey crypto.PrivateKey
	factory    crypto.Factory
}

// NewSigner creates a new classic crypto signer
func NewSigner(scheme walletcrypto.SignatureScheme, privateKeyBytes []byte) (walletcrypto.Signer, error) {
	factory, err := getFactory(scheme)
	if err != nil {
		return nil, err
	}

	privateKey, err := factory.ToPrivateKey(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &signer{
		scheme:     scheme,
		privateKey: privateKey,
		factory:    factory,
	}, nil
}

func (s *signer) Sign(msg []byte) ([]byte, error) {
	return s.privateKey.Sign(msg)
}

func (s *signer) PublicKey() []byte {
	return s.privateKey.PublicKey().Bytes()
}

func (s *signer) Address() ids.ShortID {
	return s.privateKey.Address()
}

func (s *signer) Scheme() walletcrypto.SignatureScheme {
	return s.scheme
}

func (s *signer) Verify(msg, sig []byte) bool {
	pubKey := s.privateKey.PublicKey()
	return pubKey.Verify(msg, sig)
}

func getFactory(scheme walletcrypto.SignatureScheme) (crypto.Factory, error) {
	switch scheme {
	case walletcrypto.SECP256K1:
		return crypto.FactorySECP256K1R{}, nil
	case walletcrypto.BLS:
		return crypto.FactoryBLS{}, nil
	default:
		return nil, fmt.Errorf("unsupported classic scheme: %d", scheme)
	}
}
