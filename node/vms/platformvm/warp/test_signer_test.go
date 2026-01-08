// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


//go:build test

package warp

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/const"
	"github.com/luxfi/crypto/bls"
)

// SignerTests is a list of all signer tests
var SignerTests = map[string]func(t *testing.T, s Signer, sk *bls.SecretKey, networkID uint32, chainID ids.ID){
	"WrongChainID":   testWrongChainID,
	"WrongNetworkID": testWrongNetworkID,
	"Verifies":       testVerifies,
}

// testWrongChainID tests that using a random SourceChainID results in an error
func testWrongChainID(t *testing.T, s Signer, _ *bls.SecretKey, _ uint32, _ ids.ID) {
	require := require.New(t)

	msg, err := NewUnsignedMessage(
		constants.UnitTestID,
		ids.GenerateTestID(),
		[]byte("payload"),
	)
	require.NoError(err)

	_, err = s.Sign(msg)
	require.Error(err) //nolint:forbidigo // currently returns grpc errors too
}

// testWrongNetworkID tests that using a different networkID results in an error
func testWrongNetworkID(t *testing.T, s Signer, _ *bls.SecretKey, networkID uint32, blockchainID ids.ID) {
	require := require.New(t)

	msg, err := NewUnsignedMessage(
		networkID+1,
		blockchainID,
		[]byte("payload"),
	)
	require.NoError(err)

	_, err = s.Sign(msg)
	require.Error(err) //nolint:forbidigo // currently returns grpc errors too
}

// testVerifies tests that a signature generated with the signer verifies correctly
func testVerifies(t *testing.T, s Signer, sk *bls.SecretKey, networkID uint32, chainID ids.ID) {
	require := require.New(t)

	msg, err := NewUnsignedMessage(
		networkID,
		chainID,
		[]byte("payload"),
	)
	require.NoError(err)

	sigBytes, err := s.Sign(msg)
	require.NoError(err)

	sig, err := bls.SignatureFromBytes(sigBytes)
	require.NoError(err)

	pk := bls.PublicFromSecretKey(sk)
	msgBytes := msg.Bytes()
	require.True(bls.Verify(pk, sig, msgBytes))
}
