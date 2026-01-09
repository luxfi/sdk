// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keychain

import (
	"testing"

	"github.com/luxfi/ids"
	"github.com/stretchr/testify/require"
)

func TestCryptoKeychain_Secp256k1(t *testing.T) {
	require := require.New(t)

	// Create keychain with secp256k1 as default
	kc := NewPQKeychain(KeyTypeSecp256k1)

	// Generate a key
	addr, err := kc.GenerateKey()
	require.NoError(err)
	require.NotEqual(ids.ShortEmpty, addr)

	// Get the signer
	signer, exists := kc.Get(addr)
	require.True(exists)
	require.NotNil(signer)

	// Sign a message
	msg := []byte("test message")
	sig, err := signer.Sign(msg)
	require.NoError(err)
	require.NotEmpty(sig)

	// Check algorithm
	keySigner := signer.(*PQSigner)
	require.Equal(KeyTypeSecp256k1, keySigner.keyType)
}

func TestCryptoKeychain_MLDSA(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		name string
		algo KeyType
	}{
		{"ML-DSA-44", KeyTypeMLDSA44},
		{"ML-DSA-65", KeyTypeMLDSA65},
		{"ML-DSA-87", KeyTypeMLDSA87},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			kc := NewPQKeychain(tc.algo)

			addr, err := kc.GenerateKey()
			require.NoError(err)
			require.NotEqual(ids.ShortEmpty, addr)

			signer, exists := kc.Get(addr)
			require.True(exists)

			msg := []byte("test message for ML-DSA")
			sig, err := signer.Sign(msg)
			require.NoError(err)
			require.NotEmpty(sig)

			keySigner := signer.(*PQSigner)
			require.Equal(tc.algo, keySigner.keyType)
		})
	}
}

func TestCryptoKeychain_SLHDSA(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		name string
		algo KeyType
	}{
		{"SLH-DSA-128", KeyTypeSLHDSA128},
		{"SLH-DSA-192", KeyTypeSLHDSA192},
		{"SLH-DSA-256", KeyTypeSLHDSA256},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			kc := NewPQKeychain(tc.algo)

			addr, err := kc.GenerateKey()
			require.NoError(err)
			require.NotEqual(ids.ShortEmpty, addr)

			signer, exists := kc.Get(addr)
			require.True(exists)

			msg := []byte("test message for SLH-DSA")
			sig, err := signer.Sign(msg)
			require.NoError(err)
			require.NotEmpty(sig)

			keySigner := signer.(*PQSigner)
			require.Equal(tc.algo, keySigner.keyType)
		})
	}
}

// ML-KEM tests are removed as ML-KEM is for key encapsulation, not signing
// and PQKeychain doesn't support generating ML-KEM keys

func TestCryptoKeychain_MultipleAlgorithms(t *testing.T) {
	require := require.New(t)

	// Test different keychains with different default algorithms
	kc1 := NewPQKeychain(KeyTypeSecp256k1)
	addr1, err := kc1.GenerateKey()
	require.NoError(err)

	kc2 := NewPQKeychain(KeyTypeMLDSA44)
	addr2, err := kc2.GenerateKey()
	require.NoError(err)

	kc3 := NewPQKeychain(KeyTypeSLHDSA128)
	addr3, err := kc3.GenerateKey()
	require.NoError(err)

	// All addresses should be different
	require.NotEqual(addr1, addr2)
	require.NotEqual(addr2, addr3)
	require.NotEqual(addr1, addr3)

	// Check that each keychain has the right type
	signer1, exists := kc1.Get(addr1)
	require.True(exists)
	require.Equal(KeyTypeSecp256k1, signer1.(*PQSigner).keyType)

	signer2, exists := kc2.Get(addr2)
	require.True(exists)
	require.Equal(KeyTypeMLDSA44, signer2.(*PQSigner).keyType)

	signer3, exists := kc3.Get(addr3)
	require.True(exists)
	require.Equal(KeyTypeSLHDSA128, signer3.(*PQSigner).keyType)

	// Check Addresses method
	addrs1 := kc1.Addresses()
	require.Equal(1, len(addrs1))
	require.Equal(addr1, addrs1[0])
}

func TestCryptoKeychain_Compatibility(t *testing.T) {
	require := require.New(t)

	// Test that NewPQKeychain with secp256k1 provides backward compatibility
	kc := NewPQKeychain(KeyTypeSecp256k1)

	addr, err := kc.GenerateKey()
	require.NoError(err)

	signer, exists := kc.Get(addr)
	require.True(exists)

	keySigner := signer.(*PQSigner)
	require.Equal(KeyTypeSecp256k1, keySigner.keyType)

	// Test SignHash for secp256k1
	hash := make([]byte, 32)
	copy(hash, []byte("test hash 32 bytes long........!"))

	sig, err := signer.SignHash(hash)
	require.NoError(err)
	require.NotEmpty(sig)
}
