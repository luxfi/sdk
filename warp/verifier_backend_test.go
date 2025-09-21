// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"context"
	"sync"
	"testing"
	"time"

	luxConsensus "github.com/luxfi/consensus"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/database/memdb"
	"github.com/luxfi/evm/consensus"
	"github.com/luxfi/evm/metrics/metricstest"
	"github.com/luxfi/evm/plugin/evm/validators"
	stateinterfaces "github.com/luxfi/evm/plugin/evm/validators/state/interfaces"
	"github.com/luxfi/evm/utils/utilstest"
	"github.com/luxfi/evm/warp/messages"
	"github.com/luxfi/evm/warp/warptest"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/cache"
	"github.com/luxfi/node/cache/lru"
	"github.com/luxfi/node/network/p2p/lp118"
	"github.com/luxfi/node/proto/pb/sdk"
	"github.com/luxfi/node/utils/timer/mockable"
	luxWarp "github.com/luxfi/warp"
	"github.com/luxfi/warp/payload"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestAddressedCallSignatures(t *testing.T) {
	metricstest.WithMetrics(t)

	database := memdb.New()

	offChainPayload, err := payload.NewAddressedCall([]byte{1, 2, 3}, []byte{1, 2, 3})
	require.NoError(t, err)
	networkID := uint32(1337) // Use a test network ID
	chainID := ids.GenerateTestID()
	offchainMessage, err := luxWarp.NewUnsignedMessage(networkID, chainID, offChainPayload.Bytes())
	require.NoError(t, err)
	// Create a BLS key for signing
	blsKey, err := bls.NewSecretKey()
	require.NoError(t, err)
	localSigner := NewLocalSigner(blsKey)
	offchainSignature, err := localSigner.Sign(offchainMessage.Bytes())
	require.NoError(t, err)

	tests := map[string]struct {
		setup       func(backend Backend) (request []byte, expectedResponse []byte)
		verifyStats func(t *testing.T, stats *verifierStats)
		err         error
	}{
		"known message": {
			setup: func(backend Backend) (request []byte, expectedResponse []byte) {
				knownPayload, err := payload.NewAddressedCall([]byte{0, 0, 0}, []byte("test"))
				require.NoError(t, err)
				msg, err := luxWarp.NewUnsignedMessage(networkID, chainID, knownPayload.Bytes())
				require.NoError(t, err)
				signature, err := localSigner.Sign(msg.Bytes())
				require.NoError(t, err)

				backend.AddMessage(msg)
				return msg.Bytes(), signature[:]
			},
			verifyStats: func(t *testing.T, stats *verifierStats) {
				require.EqualValues(t, 0, stats.messageParseFail.Snapshot().Count())
				require.EqualValues(t, 0, stats.blockValidationFail.Snapshot().Count())
			},
		},
		"offchain message": {
			setup: func(_ Backend) (request []byte, expectedResponse []byte) {
				return offchainMessage.Bytes(), offchainSignature[:]
			},
			verifyStats: func(t *testing.T, stats *verifierStats) {
				require.EqualValues(t, 0, stats.messageParseFail.Snapshot().Count())
				require.EqualValues(t, 0, stats.blockValidationFail.Snapshot().Count())
			},
		},
		"unknown message": {
			setup: func(_ Backend) (request []byte, expectedResponse []byte) {
				unknownPayload, err := payload.NewAddressedCall([]byte{0, 0, 0}, []byte("unknown message"))
				require.NoError(t, err)
				unknownMessage, err := luxWarp.NewUnsignedMessage(networkID, chainID, unknownPayload.Bytes())
				require.NoError(t, err)
				return unknownMessage.Bytes(), nil
			},
			verifyStats: func(t *testing.T, stats *verifierStats) {
				require.EqualValues(t, 1, stats.messageParseFail.Snapshot().Count())
				require.EqualValues(t, 0, stats.blockValidationFail.Snapshot().Count())
			},
			err: &compat.AppError{Code: ParseErrCode},
		},
	}

	for name, test := range tests {
		for _, withCache := range []bool{true, false} {
			if withCache {
				name += "_with_cache"
			} else {
				name += "_no_cache"
			}
			t.Run(name, func(t *testing.T) {
				var sigCache cache.Cacher[ids.ID, []byte]
				if withCache {
					sigCache = lru.NewCache[ids.ID, []byte](100)
				} else {
					sigCache = &cache.Empty[ids.ID, []byte]{}
				}
				warpBackend, err := NewBackend(
					networkID,
					chainID,
					localSigner,
					warptest.EmptyBlockClient,
					nil,
					database,
					sigCache,
					[][]byte{offchainMessage.Bytes()},
				)
				require.NoError(t, err)
				lp118Signer := NewLP118SignerAdapter(localSigner)
				handler := lp118.NewCachedHandler(sigCache, warpBackend, lp118Signer)

				requestBytes, expectedResponse := test.setup(warpBackend)
				protoMsg := &sdk.SignatureRequest{Message: requestBytes}
				protoBytes, err := proto.Marshal(protoMsg)
				require.NoError(t, err)
				responseBytes, appErr := handler.AppRequest(context.Background(), ids.GenerateTestNodeID(), time.Time{}, protoBytes)
				if test.err != nil {
					require.Error(t, appErr)
					require.ErrorIs(t, appErr, test.err)
				} else {
					require.Nil(t, appErr)
				}

				test.verifyStats(t, warpBackend.(*backend).stats)

				// If the expected response is empty, assert that the handler returns an empty response and return early.
				if len(expectedResponse) == 0 {
					require.Len(t, responseBytes, 0, "expected response to be empty")
					return
				}
				// check cache is populated
				if withCache {
					require.NotZero(t, warpBackend.(*backend).signatureCache.Len())
				} else {
					require.Zero(t, warpBackend.(*backend).signatureCache.Len())
				}
				response := &sdk.SignatureResponse{}
				require.NoError(t, proto.Unmarshal(responseBytes, response))
				require.NoError(t, err, "error unmarshalling SignatureResponse")

				require.Equal(t, expectedResponse, response.Signature)
			})
		}
	}
}

func TestBlockSignatures(t *testing.T) {
	metricstest.WithMetrics(t)

	database := memdb.New()
	consensusCtx := utilstest.NewTestConsensusContext(t)
	networkID := consensus.GetNetworkID(consensusCtx)
	chainID := consensus.GetChainID(consensusCtx)

	// Create a local signer for testing
	sk, err := bls.NewSecretKey()
	require.NoError(t, err)
	localSigner := NewLocalSigner(sk)

	knownBlkID := ids.GenerateTestID()
	blockClient := warptest.MakeBlockClient(knownBlkID)

	toMessageBytes := func(id ids.ID) []byte {
		idPayload, err := payload.NewHash(id[:])
		if err != nil {
			panic(err)
		}

		msg, err := luxWarp.NewUnsignedMessage(networkID, chainID, idPayload.Bytes())
		if err != nil {
			panic(err)
		}

		return msg.Bytes()
	}

	tests := map[string]struct {
		setup       func() (request []byte, expectedResponse []byte)
		verifyStats func(t *testing.T, stats *verifierStats)
		err         error
	}{
		"known block": {
			setup: func() (request []byte, expectedResponse []byte) {
				hashPayload, err := payload.NewHash(knownBlkID[:])
				require.NoError(t, err)
				unsignedMessage, err := luxWarp.NewUnsignedMessage(networkID, chainID, hashPayload.Bytes())
				require.NoError(t, err)
				warpSignerInterface := consensus.GetWarpSigner(consensusCtx)
				warpSigner := warpSignerInterface.(luxWarp.Signer)
				signature, err := warpSigner.Sign(unsignedMessage)
				require.NoError(t, err)
				return toMessageBytes(knownBlkID), signature[:]
			},
			verifyStats: func(t *testing.T, stats *verifierStats) {
				require.EqualValues(t, 0, stats.blockValidationFail.Snapshot().Count())
				require.EqualValues(t, 0, stats.messageParseFail.Snapshot().Count())
			},
		},
		"unknown block": {
			setup: func() (request []byte, expectedResponse []byte) {
				unknownBlockID := ids.GenerateTestID()
				return toMessageBytes(unknownBlockID), nil
			},
			verifyStats: func(t *testing.T, stats *verifierStats) {
				require.EqualValues(t, 1, stats.blockValidationFail.Snapshot().Count())
				require.EqualValues(t, 0, stats.messageParseFail.Snapshot().Count())
			},
			err: &AppError{Code: VerifyErrCode},
		},
	}

	for name, test := range tests {
		for _, withCache := range []bool{true, false} {
			if withCache {
				name += "_with_cache"
			} else {
				name += "_no_cache"
			}
			t.Run(name, func(t *testing.T) {
				var sigCache cache.Cacher[ids.ID, []byte]
				if withCache {
					sigCache = lru.NewCache[ids.ID, []byte](100)
				} else {
					sigCache = &cache.Empty[ids.ID, []byte]{}
				}
				warpBackend, err := NewBackend(
					networkID,
					chainID,
					localSigner,
					blockClient,
					warptest.NoOpValidatorReader{},
					database,
					sigCache,
					nil,
				)
				require.NoError(t, err)
				lp118Signer := NewLP118SignerAdapter(localSigner)
				handler := lp118.NewCachedHandler(sigCache, warpBackend, lp118Signer)

				requestBytes, expectedResponse := test.setup()
				protoMsg := &sdk.SignatureRequest{Message: requestBytes}
				protoBytes, err := proto.Marshal(protoMsg)
				require.NoError(t, err)
				responseBytes, appErr := handler.AppRequest(context.Background(), ids.GenerateTestNodeID(), time.Time{}, protoBytes)
				if test.err != nil {
					require.NotNil(t, appErr)
					require.ErrorIs(t, test.err, appErr)
				} else {
					require.Nil(t, appErr)
				}

				test.verifyStats(t, warpBackend.(*backend).stats)

				// If the expected response is empty, assert that the handler returns an empty response and return early.
				if len(expectedResponse) == 0 {
					require.Len(t, responseBytes, 0, "expected response to be empty")
					return
				}
				// check cache is populated
				if withCache {
					require.NotZero(t, warpBackend.(*backend).signatureCache.Len())
				} else {
					require.Zero(t, warpBackend.(*backend).signatureCache.Len())
				}
				var response sdk.SignatureResponse
				err = proto.Unmarshal(responseBytes, &response)
				require.NoError(t, err, "error unmarshalling SignatureResponse")
				require.Equal(t, expectedResponse, response.Signature)
			})
		}
	}
}

func TestUptimeSignatures(t *testing.T) {
	database := memdb.New()
	consensusCtx := utilstest.NewTestConsensusContext(t)
	networkID := consensus.GetNetworkID(consensusCtx)
	chainID := consensus.GetChainID(consensusCtx)

	getUptimeMessageBytes := func(sourceAddress []byte, vID ids.ID, totalUptime uint64) ([]byte, *luxWarp.UnsignedMessage) {
		uptimePayload, err := messages.NewValidatorUptime(vID, 80)
		require.NoError(t, err)
		addressedCall, err := payload.NewAddressedCall(sourceAddress, uptimePayload.Bytes())
		require.NoError(t, err)
		unsignedMessage, err := luxWarp.NewUnsignedMessage(networkID, chainID, addressedCall.Bytes())
		require.NoError(t, err)

		protoMsg := &sdk.SignatureRequest{Message: unsignedMessage.Bytes()}
		protoBytes, err := proto.Marshal(protoMsg)
		require.NoError(t, err)
		return protoBytes, unsignedMessage
	}

	for _, withCache := range []bool{true, false} {
		var sigCache cache.Cacher[ids.ID, []byte]
		if withCache {
			sigCache = lru.NewCache[ids.ID, []byte](100)
		} else {
			sigCache = &cache.Empty[ids.ID, []byte]{}
		}
		chainCtx := utilstest.NewTestConsensusContext(t)
		clk := &mockable.Clock{}
		validatorsManager, err := validators.NewManager(chainCtx, memdb.New(), clk)
		require.NoError(t, err)
		lock := &sync.RWMutex{}
		newLockedValidatorManager := validators.NewLockedValidatorReader(validatorsManager, lock)
		validatorsManager.StartTracking([]ids.NodeID{})
		warpSignerInterface := consensus.GetWarpSigner(consensusCtx)
		localWarpSigner := warpSignerInterface.(WarpSigner)
		warpBackend, err := NewBackend(
			networkID,
			chainID,
			localWarpSigner,
			warptest.EmptyBlockClient,
			newLockedValidatorManager,
			database,
			sigCache,
			nil,
		)
		require.NoError(t, err)
		warpSigner := consensus.GetWarpSigner(consensusCtx)
		handler := lp118.NewCachedHandler(sigCache, warpBackend, warpSigner)

		// sourceAddress nonZero
		protoBytes, _ := getUptimeMessageBytes([]byte{1, 2, 3}, ids.GenerateTestID(), 80)
		_, appErr := handler.AppRequest(context.Background(), ids.GenerateTestNodeID(), time.Time{}, protoBytes)
		require.ErrorIs(t, appErr, &AppError{Code: VerifyErrCode})
		require.Contains(t, appErr.Error(), "source address should be empty")

		// not existing validationID
		vID := ids.GenerateTestID()
		protoBytes, _ = getUptimeMessageBytes([]byte{}, vID, 80)
		_, appErr = handler.AppRequest(context.Background(), ids.GenerateTestNodeID(), time.Time{}, protoBytes)
		require.ErrorIs(t, appErr, &AppError{Code: VerifyErrCode})
		require.Contains(t, appErr.Error(), "failed to get validator")

		// uptime is less than requested (not connected)
		validationID := ids.GenerateTestID()
		nodeID := ids.GenerateTestNodeID()
		require.NoError(t, validatorsManager.AddValidator(stateinterfaces.Validator{
			ValidationID:   validationID,
			NodeID:         nodeID,
			Weight:         1,
			StartTimestamp: clk.Unix(),
			IsActive:       true,
			IsL1Validator:  true,
		}))
		protoBytes, _ = getUptimeMessageBytes([]byte{}, validationID, 80)
		_, appErr = handler.AppRequest(context.Background(), nodeID, time.Time{}, protoBytes)
		require.ErrorIs(t, appErr, &AppError{Code: VerifyErrCode})
		require.Contains(t, appErr.Error(), "current uptime 0 is less than queried uptime 80")

		// uptime is less than requested (not enough)
		require.NoError(t, validatorsManager.Connect(nodeID))
		clk.Set(clk.Time().Add(40 * time.Second))
		protoBytes, _ = getUptimeMessageBytes([]byte{}, validationID, 80)
		_, appErr = handler.AppRequest(context.Background(), nodeID, time.Time{}, protoBytes)
		require.ErrorIs(t, appErr, &AppError{Code: VerifyErrCode})
		require.Contains(t, appErr.Error(), "current uptime 40 is less than queried uptime 80")

		// valid uptime
		clk.Set(clk.Time().Add(40 * time.Second))
		protoBytes, msg := getUptimeMessageBytes([]byte{}, validationID, 80)
		responseBytes, appErr := handler.AppRequest(context.Background(), nodeID, time.Time{}, protoBytes)
		require.Nil(t, appErr)
		warpSigner := consensus.GetWarpSigner(consensusCtx)
		expectedSignature, err := warpSigner.Sign(msg.Bytes())
		require.NoError(t, err)
		response := &sdk.SignatureResponse{}
		require.NoError(t, proto.Unmarshal(responseBytes, response))
		require.Equal(t, expectedSignature[:], response.Signature)
	}
}
