// (c) 2023, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"context"
	"testing"
	
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/localsigner"
	"github.com/luxfi/evm/utils"
	"github.com/luxfi/evm/warp/warptest"
	"github.com/stretchr/testify/require"
)

var (
	networkID           uint32 = 54321
	sourceChainID              = interfaces.GenerateTestID()
	testSourceAddress          = utils.RandomBytes(20)
	testPayload                = []byte("test")
	testUnsignedMessage *interfaces.UnsignedMessage
)

func init() {
	testAddressedCallPayload, err := interfaces.NewAddressedCall(testSourceAddress, testPayload)
	if err != nil {
		panic(err)
	}
	testUnsignedMessage, err = interfaces.NewUnsignedMessage(networkID, sourceChainID, testAddressedCallPayload.Bytes())
	if err != nil {
		panic(err)
	}
}

func TestClearDB(t *testing.T) {
	db := interfaces.New()

	sk, err := interfaces.NewSecretKey()
	require.NoError(t, err)
	warpSigner := interfaces.NewSigner(sk, networkID, sourceChainID)
	backendIntf, err := NewBackend(networkID, sourceChainID, warpSigner, nil, warptest.NoOpValidatorReader{}, db, utils.NewLRUCache[interfaces.ID, []byte](500), nil)
	require.NoError(t, err)
	backend, ok := backendIntf.(*backend)
	require.True(t, ok)

	// use multiple messages to test that all messages get cleared
	payloads := [][]byte{[]byte("test1"), []byte("test2"), []byte("test3"), []byte("test4"), []byte("test5")}
	messageIDs := []interfaces.ID{}

	// add all messages
	for _, payload := range payloads {
		unsignedMsg, err := interfaces.NewUnsignedMessage(networkID, sourceChainID, payload)
		require.NoError(t, err)
		messageID := utils.ComputeHash256Array(unsignedMsg.Bytes())
		messageIDs = append(messageIDs, messageID)
		err = backend.AddMessage(unsignedMsg)
		require.NoError(t, err)
		// ensure that the message was added
		_, err = backend.GetMessageSignature(messageID)
		require.NoError(t, err)
	}

	err = backend.Clear()
	require.NoError(t, err)
	require.Zero(t, backend.messageCache.Len())
	require.Zero(t, backend.messageSignatureCache.Len())
	require.Zero(t, backend.blockSignatureCache.Len())
	it := db.NewIterator()
	defer it.Release()
	require.False(t, it.Next())

	// ensure all messages have been deleted
	for _, messageID := range messageIDs {
		_, err := backend.GetMessageSignature(messageID)
		require.ErrorContains(t, err, "failed to get warp message")
	}
}

func TestAddAndGetValidMessage(t *testing.T) {
	db := interfaces.New()

	sk, err := localsigner.New()
	require.NoError(t, err)
	warpSigner := interfaces.NewSigner(sk, networkID, sourceChainID)
	backend, err := NewBackend(networkID, sourceChainID, warpSigner, nil, warptest.NoOpValidatorReader{}, db, utils.NewLRUCache[interfaces.ID, []byte](500), nil)
	require.NoError(t, err)

	// Add testUnsignedMessage to the warp backend
	err = backend.AddMessage(testUnsignedMessage)
	require.NoError(t, err)

	// Verify that a signature is returned successfully, and compare to expected signature.
	signature, err := backend.GetMessageSignature(context.TODO(), testUnsignedMessage)
	require.NoError(t, err)

	expectedSig, err := warpSigner.Sign(testUnsignedMessage)
	require.NoError(t, err)
	require.Equal(t, expectedSig, signature[:])
}

func TestAddAndGetUnknownMessage(t *testing.T) {
	db := interfaces.New()

	sk, err := localsigner.New()
	require.NoError(t, err)
	warpSigner := interfaces.NewSigner(sk, networkID, sourceChainID)
	backend, err := NewBackend(networkID, sourceChainID, warpSigner, nil, warptest.NoOpValidatorReader{}, db, utils.NewLRUCache[interfaces.ID, []byte](500), nil)
	require.NoError(t, err)

	// Try getting a signature for a message that was not added.
	_, err = backend.GetMessageSignature(context.TODO(), testUnsignedMessage)
	require.Error(t, err)
}

func TestGetBlockSignature(t *testing.T) {
	require := require.New(t)

	blkID := interfaces.GenerateTestID()
	blockClient := warptest.MakeBlockClient(blkID)
	db := interfaces.New()

	sk, err := localsigner.New()
	require.NoError(err)
	warpSigner := interfaces.NewSigner(sk, networkID, sourceChainID)
	backend, err := NewBackend(networkID, sourceChainID, warpSigner, blockClient, warptest.NoOpValidatorReader{}, db, utils.NewLRUCache[interfaces.ID, []byte](500), nil)
	require.NoError(err)

	blockHashPayload, err := interfaces.NewHash(blkID)
	require.NoError(err)
	unsignedMessage, err := interfaces.NewUnsignedMessage(networkID, sourceChainID, blockHashPayload.Bytes())
	require.NoError(err)
	expectedSig, err := warpSigner.Sign(unsignedMessage)
	require.NoError(err)

	signature, err := backend.GetBlockSignature(context.TODO(), blkID)
	require.NoError(err)
	require.Equal(expectedSig, signature[:])

	_, err = backend.GetBlockSignature(context.TODO(), interfaces.GenerateTestID())
	require.Error(err)
}

func TestZeroSizedCache(t *testing.T) {
	db := interfaces.New()

	sk, err := localsigner.New()
	require.NoError(t, err)
	warpSigner := interfaces.NewSigner(sk, networkID, sourceChainID)

	// Verify zero sized cache works normally, because the lru cache will be initialized to size 1 for any size parameter <= 0.
	messageSignatureCache := utils.NewLRUCache[interfaces.ID, []byte](0)
	backend, err := NewBackend(networkID, sourceChainID, warpSigner, nil, warptest.NoOpValidatorReader{}, db, messageSignatureCache, nil)
	require.NoError(t, err)

	// Add testUnsignedMessage to the warp backend
	err = backend.AddMessage(testUnsignedMessage)
	require.NoError(t, err)

	// Verify that a signature is returned successfully, and compare to expected signature.
	signature, err := backend.GetMessageSignature(context.TODO(), testUnsignedMessage)
	require.NoError(t, err)

	expectedSig, err := warpSigner.Sign(testUnsignedMessage)
	require.NoError(t, err)
	require.Equal(t, expectedSig, signature[:])
}

func TestOffChainMessages(t *testing.T) {
	type test struct {
		offchainMessages [][]byte
		check            func(require *require.Assertions, b Backend)
		err              error
	}
	sk, err := localsigner.New()
	require.NoError(t, err)
	warpSigner := interfaces.NewSigner(sk, networkID, sourceChainID)

	for name, test := range map[string]test{
		"no offchain messages": {},
		"single off-chain message": {
			offchainMessages: [][]byte{
				testUnsignedMessage.Bytes(),
			},
			check: func(require *require.Assertions, b Backend) {
				msg, err := b.GetMessage(testUnsignedMessage.ID())
				require.NoError(err)
				require.Equal(testUnsignedMessage.Bytes(), msg.Bytes())

				signature, err := b.GetMessageSignature(context.TODO(), testUnsignedMessage)
				require.NoError(err)
				expectedSignatureBytes, err := warpSigner.Sign(msg)
				require.NoError(err)
				require.Equal(expectedSignatureBytes, signature[:])
			},
		},
		"unknown message": {
			check: func(require *require.Assertions, b Backend) {
				_, err := b.GetMessage(testUnsignedMessage.ID())
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
			db := interfaces.New()

			messageSignatureCache := utils.NewLRUCache[interfaces.ID, []byte](0)
			backend, err := NewBackend(networkID, sourceChainID, warpSigner, nil, warptest.NoOpValidatorReader{}, db, messageSignatureCache, test.offchainMessages)
			require.ErrorIs(err, test.err)
			if test.check != nil {
				test.check(require, backend)
			}
		})
	}
}
