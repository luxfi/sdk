// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/const"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/crypto/bls/signer/localsigner"
	"github.com/luxfi/sdk/node/vms/platformvm/warp"
	"github.com/luxfi/sdk/node/vms/platformvm/warp/signertest"
)

func TestSigner(t *testing.T) {
	for name, test := range signertest.SignerTests {
		t.Run(name, func(t *testing.T) {
			sk, err := localsigner.New()
			require.NoError(t, err)

			chainID := ids.GenerateTestID()
			s := warp.NewSigner(sk, constants.UnitTestID, chainID)

			test(t, s, sk, constants.UnitTestID, chainID)
		})
	}
}

// Test that using a random SourceChainID results in an error
func testWrongChainID(t *testing.T, s warp.Signer, _ *localsigner.LocalSigner, _ uint32, _ ids.ID) {
	require := require.New(t)

	msg, err := warp.NewUnsignedMessage(
		constants.UnitTestID,
		ids.GenerateTestID(),
		[]byte("payload"),
	)
	require.NoError(err)

	_, err = s.Sign(msg)
	require.Error(err) //nolint:forbidigo // currently returns grpc errors too
}

// Test that using a different networkID results in an error
func testWrongNetworkID(t *testing.T, s warp.Signer, _ *localsigner.LocalSigner, networkID uint32, blockchainID ids.ID) {
	require := require.New(t)

	msg, err := warp.NewUnsignedMessage(
		networkID+1,
		blockchainID,
		[]byte("payload"),
	)
	require.NoError(err)

	_, err = s.Sign(msg)
	require.Error(err) //nolint:forbidigo // currently returns grpc errors too
}

// Test that a signature generated with the signer verifies correctly
func testVerifies(t *testing.T, s warp.Signer, sk *localsigner.LocalSigner, networkID uint32, chainID ids.ID) {
	require := require.New(t)

	msg, err := warp.NewUnsignedMessage(
		networkID,
		chainID,
		[]byte("payload"),
	)
	require.NoError(err)

	sigBytes, err := s.Sign(msg)
	require.NoError(err)

	sig, err := bls.SignatureFromBytes(sigBytes)
	require.NoError(err)

	pk := sk.PublicKey()
	msgBytes := msg.Bytes()
	require.True(bls.Verify(pk, sig, msgBytes))
}
