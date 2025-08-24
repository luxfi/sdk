// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"context"
	"testing"

	"github.com/luxfi/node/cache/lru"
	"github.com/luxfi/database"
	"github.com/luxfi/database/memdb"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/utils"
	"github.com/luxfi/crypto/bls"
	luxWarp "github.com/luxfi/warp"
	"github.com/luxfi/warp/payload"
	"github.com/luxfi/evm/warp/warptest"
	"github.com/stretchr/testify/require"
)

var (
	networkID           uint32 = 54321
	sourceChainID              = ids.GenerateTestID()
	testSourceAddress          = utils.RandomBytes(20)
	testPayload                = []byte("test")
	testUnsignedMessage *luxWarp.UnsignedMessage
)

func init() {
	testAddressedCallPayload, err := payload.NewAddressedCall(testSourceAddress, testPayload)
	if err != nil {
		panic(err)
	}
	testUnsignedMessage, err = luxWarp.NewUnsignedMessage(networkID, sourceChainID[:], testAddressedCallPayload.Bytes())
	if err != nil {
		panic(err)
	}
}

func TestAddAndGetValidMessage(t *testing.T) {
	db := memdb.New()

	// Create a BLS private key for signing
	blsKey, err := bls.NewSecretKey()
	require.NoError(t, err)
	// Create a LocalSigner from the BLS key
	localSigner := NewLocalSigner(blsKey)
	messageSignatureCache := lru.NewCache[ids.ID, []byte](500)
	backend, err := NewBackend(networkID, sourceChainID, localSigner, nil, warptest.NoOpValidatorReader{}, db, messageSignatureCache, nil)
	require.NoError(t, err)

	// Add testUnsignedMessage to the warp backend
	require.NoError(t, backend.AddMessage(testUnsignedMessage))

	// Verify that a signature is returned successfully, and compare to expected signature.
	signature, err := backend.GetMessageSignature(context.TODO(), testUnsignedMessage)
	require.NoError(t, err)

	expectedSigBytes, err := localSigner.Sign(testUnsignedMessage.Bytes())
	require.NoError(t, err)
	require.Equal(t, expectedSigBytes, signature[:])
}

func TestAddAndGetUnknownMessage(t *testing.T) {
	db := memdb.New()

	// Create a BLS private key for signing
	blsKey, err := bls.NewSecretKey()
	require.NoError(t, err)
	// Create a LocalSigner from the BLS key
	localSigner := NewLocalSigner(blsKey)
	messageSignatureCache := lru.NewCache[ids.ID, []byte](500)
	backend, err := NewBackend(networkID, sourceChainID, localSigner, nil, warptest.NoOpValidatorReader{}, db, messageSignatureCache, nil)
	require.NoError(t, err)

	// Try getting a signature for a message that was not added.
	_, err = backend.GetMessageSignature(context.TODO(), testUnsignedMessage)
	require.Error(t, err)
}

func TestGetBlockSignature(t *testing.T) {
	require := require.New(t)

	blkID := ids.GenerateTestID()
	blockClient := warptest.MakeBlockClient(blkID)
	db := memdb.New()

	// Create a BLS private key for signing
	blsKey, err := bls.NewSecretKey()
	require.NoError(err)
	// Create a LocalSigner from the BLS key
	localSigner := NewLocalSigner(blsKey)
	messageSignatureCache := lru.NewCache[ids.ID, []byte](500)
	backend, err := NewBackend(networkID, sourceChainID, localSigner, blockClient, warptest.NoOpValidatorReader{}, db, messageSignatureCache, nil)
	require.NoError(err)

	blockHashPayload, err := payload.NewHash(blkID[:])
	require.NoError(err)
	unsignedMessage, err := luxWarp.NewUnsignedMessage(networkID, sourceChainID[:], blockHashPayload.Bytes())
	require.NoError(err)
	msgBytes := unsignedMessage.Bytes()
	expectedSigBytes, err := localSigner.Sign(msgBytes)
	require.NoError(err)

	signature, err := backend.GetBlockSignature(context.TODO(), blkID)
	require.NoError(err)
	require.Equal(expectedSigBytes, signature[:])

	_, err = backend.GetBlockSignature(context.TODO(), ids.GenerateTestID())
	require.Error(err)
}

func TestZeroSizedCache(t *testing.T) {
	db := memdb.New()

	// Create a BLS private key for signing
	blsKey, err := bls.NewSecretKey()
	require.NoError(t, err)
	// Create a LocalSigner from the BLS key
	localSigner := NewLocalSigner(blsKey)

	// Verify zero sized cache works normally, because the lru cache will be initialized to size 1 for any size parameter <= 0.
	messageSignatureCache := lru.NewCache[ids.ID, []byte](0)
	backend, err := NewBackend(networkID, sourceChainID, localSigner, nil, warptest.NoOpValidatorReader{}, db, messageSignatureCache, nil)
	require.NoError(t, err)

	// Add testUnsignedMessage to the warp backend
	require.NoError(t, backend.AddMessage(testUnsignedMessage))

	// Verify that a signature is returned successfully, and compare to expected signature.
	signature, err := backend.GetMessageSignature(context.TODO(), testUnsignedMessage)
	require.NoError(t, err)

	expectedSigBytes, err := localSigner.Sign(testUnsignedMessage.Bytes())
	require.NoError(t, err)
	require.Equal(t, expectedSigBytes, signature[:])
}

func TestOffChainMessages(t *testing.T) {
	type test struct {
		offchainMessages [][]byte
		check            func(require *require.Assertions, b Backend)
		err              error
	}
	sk, err := bls.NewSecretKey()
	require.NoError(t, err)
	warpSigner := NewLocalSigner(sk)

	for name, test := range map[string]test{
		"no offchain messages": {},
		"single off-chain message": {
			offchainMessages: [][]byte{
				testUnsignedMessage.Bytes(),
			},
			check: func(require *require.Assertions, b Backend) {
				msgID, _ := ids.ToID(testUnsignedMessage.ID())
				msg, err := b.GetMessage(msgID)
				require.NoError(err)
				require.Equal(testUnsignedMessage.Bytes(), msg.Bytes())

				signature, err := b.GetMessageSignature(context.TODO(), testUnsignedMessage)
				require.NoError(err)
				expectedSignatureBytes, err := warpSigner.Sign(testUnsignedMessage.Bytes())
				require.NoError(err)
				require.Equal(expectedSignatureBytes, signature[:])
			},
		},
		"unknown message": {
			check: func(require *require.Assertions, b Backend) {
				msgID, _ := ids.ToID(testUnsignedMessage.ID())
				_, err := b.GetMessage(msgID)
				require.ErrorIs(err, database.ErrNotFound)
			},
		},
		"invalid message": {
			offchainMessages: [][]byte{{1, 2, 3}},
			err:              errParsingOffChainMessage,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			db := memdb.New()

			messageSignatureCache := lru.NewCache[ids.ID, []byte](0)
			backend, err := NewBackend(networkID, sourceChainID, warpSigner, nil, warptest.NoOpValidatorReader{}, db, messageSignatureCache, test.offchainMessages)
			require.ErrorIs(err, test.err)
			if test.check != nil {
				test.check(require, backend)
			}
		})
	}
}
