// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"encoding/binary"
	"math/big"
	"sync"
	"testing"
	"time"

	consensuscore "github.com/luxfi/consensus/core"
	"github.com/luxfi/database/memdb"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"

	"github.com/luxfi/p2p"
	"github.com/luxfi/p2p/gossip"
	"github.com/luxfi/node/proto/pb/sdk"

	"github.com/luxfi/consensus/utils/set"
	agoUtils "github.com/luxfi/node/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/proto"

	"github.com/luxfi/crypto"
	"github.com/luxfi/evm/plugin/evm/config"
	"github.com/luxfi/geth/core/types"
)

func TestEthTxGossip(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	// Use createTestConsensusContext to get properly configured ValidatorState
	consensusCtx := createTestConsensusContext(t)

	sentResponse := make(chan []byte, 1)
	responseSender := &TestSender{
		T: t,
		SendAppResponseF: func(ctx context.Context, nodeID ids.NodeID, requestID uint32, response []byte) error {
			sentResponse <- response
			return nil
		},
	}

	vm := &VM{}

	require.NoError(vm.Initialize(
		ctx,
		consensusCtx,
		memdb.New(),
		[]byte(toGenesisJSON(forkToChainConfig["Latest"])),
		nil,
		nil,
		nil,
		nil, // fxs parameter
		responseSender,
	))

	// Set up gossip infrastructure before entering normal operations
	requestingNodeID := ids.GenerateTestNodeID()
	setupGossipInfrastructure(t, vm, requestingNodeID)

	require.NoError(vm.SetState(ctx, uint32(consensuscore.VMNormalOp)))

	defer func() {
		require.NoError(vm.Shutdown(ctx))
	}()

	// sender for the peer requesting gossip from [vm]
	sentAppRequest := make(chan []byte, 1)
	peerSender := &TestSender{
		T: t,
		SendAppRequestF: func(ctx context.Context, nodeSet set.Set[ids.NodeID], requestID uint32, request []byte) error {
			sentAppRequest <- request
			return nil
		},
	}

	network, err := p2p.NewNetwork(log.NewNoOpLogger(), peerSender, prometheus.NewRegistry(), "")
	require.NoError(err)
	client := network.NewClient(TxGossipHandlerID)

	// we only accept gossip requests from validators
	require.NoError(vm.Network.Connected(ctx, requestingNodeID, nil))

	// Ask the VM for any new transactions. We should get nothing at first.
	emptyBloomFilter, err := gossip.NewBloomFilter(prometheus.NewRegistry(), "", config.TxGossipBloomMinTargetElements, config.TxGossipBloomTargetFalsePositiveRate, config.TxGossipBloomResetFalsePositiveRate)
	require.NoError(err)
	emptyBloomFilterBytes, _ := emptyBloomFilter.Marshal()
	request := &sdk.PullGossipRequest{
		Filter: emptyBloomFilterBytes,
		Salt:   agoUtils.RandomBytes(32),
	}

	requestBytes, err := proto.Marshal(request)
	require.NoError(err)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	onResponse := func(_ context.Context, nodeID ids.NodeID, responseBytes []byte, err error) {
		require.NoError(err)

		response := &sdk.PullGossipResponse{}
		require.NoError(proto.Unmarshal(responseBytes, response))
		require.Empty(response.Gossip)
		wg.Done()
	}
	// Use requestingNodeID for the request since vm.ctx doesn't have NodeID
	nodeSet := set.Set[ids.NodeID]{}
	nodeSet.Add(requestingNodeID)
	require.NoError(client.AppRequest(ctx, nodeSet, requestBytes, onResponse))
	require.NoError(vm.AppRequest(ctx, requestingNodeID, 1, time.Time{}, <-sentAppRequest))
	// Use requestingNodeID for the response
	require.NoError(network.AppResponse(ctx, requestingNodeID, 1, <-sentResponse))
	wg.Wait()

	// Issue a tx to the VM
	address := testEthAddrs[0]
	tx := types.NewTransaction(0, address, big.NewInt(10), 21000, big.NewInt(testMinGasPrice), nil)
	// Convert secp256k1 key to ECDSA for signing
	keyBytes := testKeys[0].Bytes()
	privKey, err := crypto.ToECDSA(keyBytes)
	require.NoError(err)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(vm.chainConfig.ChainID), privKey)
	require.NoError(err)

	errs := vm.txPool.Add([]*types.Transaction{signedTx}, true, true)
	require.Len(errs, 1)
	require.Nil(errs[0])

	// wait so we aren't throttled by the vm
	time.Sleep(5 * time.Second)

	marshaller := GossipEthTxMarshaller{}
	// Ask the VM for new transactions. We should get the newly issued tx.
	wg.Add(1)
	onResponse = func(_ context.Context, nodeID ids.NodeID, responseBytes []byte, err error) {
		require.NoError(err)

		response := &sdk.PullGossipResponse{}
		require.NoError(proto.Unmarshal(responseBytes, response))
		require.Len(response.Gossip, 1)

		gotTx, err := marshaller.UnmarshalGossip(response.Gossip[0])
		require.NoError(err)
		require.Equal(signedTx.Hash(), gotTx.Tx.Hash())

		wg.Done()
	}
	nodeSet2 := set.Set[ids.NodeID]{}
	nodeSet2.Add(consensusCtx.NodeID)
	require.NoError(client.AppRequest(ctx, nodeSet2, requestBytes, onResponse))
	require.NoError(vm.AppRequest(ctx, requestingNodeID, 3, time.Time{}, <-sentAppRequest))
	require.NoError(network.AppResponse(ctx, consensusCtx.NodeID, 3, <-sentResponse))
	wg.Wait()
}

// Tests that a tx is gossiped when it is issued
func TestEthTxPushGossipOutbound(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	// Use createTestConsensusContext to get properly configured ValidatorState
	consensusCtx := createTestConsensusContext(t)

	sender := &TestSender{
		T:             t,
		SentAppGossip: make(chan []byte, 1),
	}

	vm := &VM{
		ethTxPullGossiper: gossip.NoOpGossiper{},
	}

	require.NoError(vm.Initialize(
		ctx,
		consensusCtx,
		memdb.New(),
		[]byte(toGenesisJSON(forkToChainConfig["Latest"])),
		nil,
		nil,
		nil,
		nil, // fxs parameter
		sender,
	))

	// Set up gossip infrastructure and push gossiper
	testNodeID := ids.GenerateTestNodeID()
	setupGossipInfrastructure(t, vm, testNodeID)
	setupPushGossiper(t, vm, sender)

	require.NoError(vm.SetState(ctx, uint32(consensuscore.VMNormalOp)))

	defer func() {
		require.NoError(vm.Shutdown(ctx))
	}()

	address := testEthAddrs[0]
	tx := types.NewTransaction(0, address, big.NewInt(10), 21000, big.NewInt(testMinGasPrice), nil)
	// Convert secp256k1 key to ECDSA for signing
	keyBytes := testKeys[0].Bytes()
	privKey, err := crypto.ToECDSA(keyBytes)
	require.NoError(err)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(vm.chainConfig.ChainID), privKey)
	require.NoError(err)

	// issue a tx
	require.NoError(vm.txPool.Add([]*types.Transaction{signedTx}, true, true)[0])
	vm.ethTxPushGossiper.Get().Add(&GossipEthTx{signedTx})

	sent := <-sender.SentAppGossip
	got := &sdk.PushGossip{}

	// we should get a message that has the protocol prefix and the gossip
	// message
	require.Equal(byte(TxGossipHandlerID), sent[0])
	require.NoError(proto.Unmarshal(sent[1:], got))

	marshaller := GossipEthTxMarshaller{}
	require.Len(got.Gossip, 1)
	gossipedTx, err := marshaller.UnmarshalGossip(got.Gossip[0])
	require.NoError(err)
	require.Equal(ids.ID(signedTx.Hash()), gossipedTx.GossipID())
}

// Tests that a gossiped tx is added to the mempool and forwarded
func TestEthTxPushGossipInbound(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	// Use createTestConsensusContext to get properly configured ValidatorState
	consensusCtx := createTestConsensusContext(t)

	sender := &TestSender{T: t}
	vm := &VM{
		ethTxPullGossiper: gossip.NoOpGossiper{},
	}

	require.NoError(vm.Initialize(
		ctx,
		consensusCtx,
		memdb.New(),
		[]byte(toGenesisJSON(forkToChainConfig["Latest"])),
		nil,
		nil,
		nil,
		nil, // fxs parameter
		sender,
	))

	// Set up gossip infrastructure
	testNodeID := ids.GenerateTestNodeID()
	setupGossipInfrastructure(t, vm, testNodeID)

	require.NoError(vm.SetState(ctx, uint32(consensuscore.VMNormalOp)))

	defer func() {
		require.NoError(vm.Shutdown(ctx))
	}()

	address := testEthAddrs[0]
	tx := types.NewTransaction(0, address, big.NewInt(10), 21000, big.NewInt(testMinGasPrice), nil)
	// Convert secp256k1 key to ECDSA for signing
	keyBytes := testKeys[0].Bytes()
	privKey, err := crypto.ToECDSA(keyBytes)
	require.NoError(err)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(vm.chainConfig.ChainID), privKey)
	require.NoError(err)

	marshaller := GossipEthTxMarshaller{}
	gossipedTx := &GossipEthTx{
		Tx: signedTx,
	}
	gossipedTxBytes, err := marshaller.MarshalGossip(gossipedTx)
	require.NoError(err)

	inboundGossip := &sdk.PushGossip{
		Gossip: [][]byte{gossipedTxBytes},
	}

	inboundGossipBytes, err := proto.Marshal(inboundGossip)
	require.NoError(err)

	inboundGossipMsg := append(binary.AppendUvarint(nil, TxGossipHandlerID), inboundGossipBytes...)
	require.NoError(vm.AppGossip(ctx, ids.EmptyNodeID, inboundGossipMsg))

	require.True(vm.txPool.Has(signedTx.Hash()))
}
