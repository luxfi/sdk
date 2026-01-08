// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package signer

import (
	"github.com/luxfi/crypto/bls/signer/localsigner"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/crypto/bls"
)

// TestBLSSingleNodeProofOfPossession tests BLS PoP for single node BFT
func TestBLSSingleNodeProofOfPossession(t *testing.T) {
	require := require.New(t)

	// Generate BLS key
	sk, err := bls.NewSecretKey()
	require.NoError(err)

	// Create proof of possession
	pop, err := NewProofOfPossession(func() bls.Signer { s, _ := localsigner.FromBytes(bls.SecretKeyToBytes(sk)); return s }())
	require.NotNil(pop)

	// Verify proof of possession
	err = pop.Verify()
	require.NoError(err, "Single node BLS proof of possession must be valid")

	// Get public key
	pk := pop.Key()
	require.NotNil(pk)

	// Verify it's the same as original
	originalPk := sk.PublicKey()
	require.Equal(bls.PublicKeyToCompressedBytes(originalPk), pop.PublicKey[:])
}

// TestBLSAggregateOfOne tests BLS aggregate signature with single validator
func TestBLSAggregateOfOne(t *testing.T) {
	require := require.New(t)

	// Generate key
	sk, err := bls.NewSecretKey()
	require.NoError(err)

	// Message to sign
	msg := []byte("consensus block data")

	// Sign
	sig, err := sk.Sign(msg)
	require.NoError(err)
	require.NotNil(sig)

	// Create aggregate of 1
	aggSig, err := bls.AggregateSignatures([]*bls.Signature{sig})
	require.NoError(err, "Must support aggregate signature of 1")

	// Aggregate public key of 1
	pk := sk.PublicKey()
	aggPk, err := bls.AggregatePublicKeys([]*bls.PublicKey{pk})
	require.NoError(err, "Must support aggregate public key of 1")

	// Verify
	valid := bls.Verify(aggPk, aggSig, msg)
	require.True(valid, "Aggregate signature of 1 must verify")
}

// TestInvalidProofOfPossession tests invalid PoP scenarios
func TestInvalidProofOfPossession(t *testing.T) {
	tests := []struct {
		name      string
		setupPoP  func() *ProofOfPossession
		expectErr bool
	}{
		{
			name: "valid PoP",
			setupPoP: func() *ProofOfPossession {
				sk, _ := bls.NewSecretKey()
				pop, _ := NewProofOfPossession(func() bls.Signer { s, _ := localsigner.FromBytes(bls.SecretKeyToBytes(sk)); return s }())
				return pop
			},
			expectErr: false,
		},
		{
			name: "corrupted signature",
			setupPoP: func() *ProofOfPossession {
				sk, _ := bls.NewSecretKey()
				pop, _ := NewProofOfPossession(func() bls.Signer { s, _ := localsigner.FromBytes(bls.SecretKeyToBytes(sk)); return s }())
				// Corrupt signature
				pop.ProofOfPossession[0] ^= 0xFF
				return pop
			},
			expectErr: true,
		},
		{
			name: "corrupted public key",
			setupPoP: func() *ProofOfPossession {
				sk, _ := bls.NewSecretKey()
				pop, _ := NewProofOfPossession(func() bls.Signer { s, _ := localsigner.FromBytes(bls.SecretKeyToBytes(sk)); return s }())
				// Corrupt public key
				pop.PublicKey[0] ^= 0xFF
				return pop
			},
			expectErr: true,
		},
		{
			name: "mismatched signature",
			setupPoP: func() *ProofOfPossession {
				sk1, _ := bls.NewSecretKey()
				sk2, _ := bls.NewSecretKey()

				// Use pk from sk1 but signature from sk2
				pop, _ := NewProofOfPossession(func() bls.Signer { s, _ := localsigner.FromBytes(bls.SecretKeyToBytes(sk1)); return s }())
				pk2 := sk2.PublicKey()
				pk2Bytes := bls.PublicKeyToCompressedBytes(pk2)
				sig2, _ := sk2.SignProofOfPossession(pk2Bytes)
				copy(pop.ProofOfPossession[:], bls.SignatureToBytes(sig2))

				return pop
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			pop := tt.setupPoP()
			err := pop.Verify()

			if tt.expectErr {
				require.Error(err, "Invalid PoP must fail verification")
				// Different corruption types can cause different errors
				// from the BLS library or from our validation
			} else {
				require.NoError(err, "Valid PoP must pass verification")
			}
		})
	}
}
