// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"testing"

	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/node/network/p2p"
	"github.com/luxfi/node/network/p2p/gossip"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	gossipHandler "github.com/luxfi/evm/plugin/evm/gossip"
	"github.com/luxfi/evm/plugin/evm/config"
)

// mockValidatorSet implements p2p.ValidatorSet for testing
type mockValidatorSet struct {
	validators map[ids.NodeID]bool
}

func newMockValidatorSet() *mockValidatorSet {
	return &mockValidatorSet{
		validators: make(map[ids.NodeID]bool),
	}
}

func (m *mockValidatorSet) Has(ctx context.Context, nodeID ids.NodeID) bool {
	return m.validators[nodeID]
}

func (m *mockValidatorSet) AddValidator(nodeID ids.NodeID) {
	m.validators[nodeID] = true
}

// mockValidatorSubset implements p2p.ValidatorSubset for testing
type mockValidatorSubset struct {
	validators []ids.NodeID
}

func (m *mockValidatorSubset) Top(ctx context.Context, percentage float64) []ids.NodeID {
	if len(m.validators) == 0 {
		return nil
	}
	// Return top percentage of validators
	count := int(float64(len(m.validators)) * percentage)
	if count < 1 {
		count = 1
	}
	if count > len(m.validators) {
		count = len(m.validators)
	}
	return m.validators[:count]
}

// mockNodeSampler implements p2p.NodeSampler for testing
type mockNodeSampler struct {
	validators []ids.NodeID
}

func (m *mockNodeSampler) Sample(ctx context.Context, limit int) []ids.NodeID {
	if limit > len(m.validators) {
		limit = len(m.validators)
	}
	if limit <= 0 {
		return nil
	}
	return m.validators[:limit]
}

// testP2PValidators wraps a mock validator set for testing gossip infrastructure
type testP2PValidators struct {
	*mockValidatorSet
	*mockValidatorSubset
	*mockNodeSampler
}

// newTestP2PValidators creates a mock p2p validators for testing
func newTestP2PValidators(nodeIDs ...ids.NodeID) *testP2PValidators {
	mvs := newMockValidatorSet()
	for _, nodeID := range nodeIDs {
		mvs.AddValidator(nodeID)
	}
	return &testP2PValidators{
		mockValidatorSet:    mvs,
		mockValidatorSubset: &mockValidatorSubset{validators: nodeIDs},
		mockNodeSampler:     &mockNodeSampler{validators: nodeIDs},
	}
}

// setupGossipInfrastructure initializes the gossip infrastructure for a VM in test mode
// This must be called after VM.Initialize but before VM.SetState(VMNormalOp)
func setupGossipInfrastructure(t *testing.T, vm *VM, testNodeID ids.NodeID) {
	require := require.New(t)
	ctx := context.Background()

	// Create mock p2p validators with the test node
	mockValidators := newTestP2PValidators(testNodeID)

	// Create a mock p2p.Validators by using p2p.NewValidators with a mock state
	// Since we can't easily create a real p2p.Validators without full network,
	// we need to manually set up the gossip handlers

	// Create the gossip eth tx pool
	ethTxPool, err := NewGossipEthTxPool(vm.txPool, prometheus.NewRegistry())
	require.NoError(err)

	// Start the subscription in a goroutine
	go ethTxPool.Subscribe(ctx)

	// Create gossip metrics
	ethTxGossipMetrics, err := gossip.NewMetrics(vm.sdkMetrics, ethTxGossipNamespace)
	require.NoError(err)

	// Create gossip marshaller
	ethTxGossipMarshaller := GossipEthTxMarshaller{}

	// Create the gossip handler
	// Note: We need to use a compatible p2p.ValidatorSet implementation
	handler, err := gossipHandler.NewTxGossipHandler[*GossipEthTx](
		log.NewNoOpLogger(),
		ethTxGossipMarshaller,
		ethTxPool,
		ethTxGossipMetrics,
		config.TxGossipTargetMessageSize,
		config.TxGossipThrottlingPeriod,
		float64(config.TxGossipThrottlingLimit),
		mockValidators,
		vm.sdkMetrics,
		ethTxGossipNamespace,
	)
	require.NoError(err)
	vm.ethTxGossipHandler = handler

	// NOTE: Don't register the handler here - let onNormalOperationsStarted() do it
	// when SetState(VMNormalOp) is called. The handler will be registered there
	// because vm.ethTxGossipHandler is now set.

	// Create a mock push gossiper that stores gossip messages
	// This is used for outbound gossip testing
	vm.ethTxPushGossiper.Set(nil) // Will be created in tests that need it
}

// setupPushGossiper sets up the push gossiper for outbound gossip tests
func setupPushGossiper(t *testing.T, vm *VM, sender *TestSender) {
	require := require.New(t)

	// Create gossip metrics
	ethTxGossipMetrics, err := gossip.NewMetrics(prometheus.NewRegistry(), ethTxGossipNamespace)
	require.NoError(err)

	// Create gossip marshaller
	ethTxGossipMarshaller := GossipEthTxMarshaller{}

	// Create the gossip eth tx pool
	ethTxPool, err := NewGossipEthTxPool(vm.txPool, prometheus.NewRegistry())
	require.NoError(err)

	// Create a mock p2p.Validators
	testNodeID := ids.GenerateTestNodeID()
	mockValidators := newTestP2PValidators(testNodeID)

	// Create a p2p network for the client
	network, err := p2p.NewNetwork(log.NewNoOpLogger(), sender, prometheus.NewRegistry(), "")
	require.NoError(err)

	// Create the gossip client
	ethTxGossipClient := network.NewClient(TxGossipHandlerID)

	// Create push gossip parameters
	pushGossipParams := gossip.BranchingFactor{
		StakePercentage: vm.config.PushGossipPercentStake,
		Validators:      vm.config.PushGossipNumValidators,
		Peers:           vm.config.PushGossipNumPeers,
	}
	pushRegossipParams := gossip.BranchingFactor{
		Validators: vm.config.PushRegossipNumValidators,
		Peers:      vm.config.PushRegossipNumPeers,
	}

	// Create the push gossiper
	pushGossiper, err := gossip.NewPushGossiper[*GossipEthTx](
		ethTxGossipMarshaller,
		ethTxPool,
		mockValidators,
		ethTxGossipClient,
		ethTxGossipMetrics,
		pushGossipParams,
		pushRegossipParams,
		config.PushGossipDiscardedElements,
		config.TxGossipTargetMessageSize,
		vm.config.RegossipFrequency.Duration,
	)
	require.NoError(err)

	vm.ethTxPushGossiper.Set(pushGossiper)
}

// setupPushGossiperWithLoop sets up the push gossiper and starts the gossip loop
// This is for tests that use newVM() which doesn't start the gossip loop automatically
func setupPushGossiperWithLoop(t *testing.T, vm *VM, sender *TestSender) context.CancelFunc {
	require := require.New(t)

	// Create gossip metrics
	ethTxGossipMetrics, err := gossip.NewMetrics(prometheus.NewRegistry(), ethTxGossipNamespace)
	require.NoError(err)

	// Create gossip marshaller
	ethTxGossipMarshaller := GossipEthTxMarshaller{}

	// Create the gossip eth tx pool
	ethTxPool, err := NewGossipEthTxPool(vm.txPool, prometheus.NewRegistry())
	require.NoError(err)

	// Start the subscription in a goroutine
	ctx, cancel := context.WithCancel(context.Background())
	go ethTxPool.Subscribe(ctx)

	// Create a mock p2p.Validators
	testNodeID := ids.GenerateTestNodeID()
	mockValidators := newTestP2PValidators(testNodeID)

	// Create a p2p network for the client
	network, err := p2p.NewNetwork(log.NewNoOpLogger(), sender, prometheus.NewRegistry(), "")
	require.NoError(err)

	// Create the gossip client
	ethTxGossipClient := network.NewClient(TxGossipHandlerID)

	// Create push gossip parameters
	pushGossipParams := gossip.BranchingFactor{
		StakePercentage: vm.config.PushGossipPercentStake,
		Validators:      vm.config.PushGossipNumValidators,
		Peers:           vm.config.PushGossipNumPeers,
	}
	pushRegossipParams := gossip.BranchingFactor{
		Validators: vm.config.PushRegossipNumValidators,
		Peers:      vm.config.PushRegossipNumPeers,
	}

	// Create the push gossiper
	pushGossiper, err := gossip.NewPushGossiper[*GossipEthTx](
		ethTxGossipMarshaller,
		ethTxPool,
		mockValidators,
		ethTxGossipClient,
		ethTxGossipMetrics,
		pushGossipParams,
		pushRegossipParams,
		config.PushGossipDiscardedElements,
		config.TxGossipTargetMessageSize,
		vm.config.RegossipFrequency.Duration,
	)
	require.NoError(err)

	vm.ethTxPushGossiper.Set(pushGossiper)

	// Start the gossip loop in a goroutine
	go gossip.Every(ctx, log.NewNoOpLogger(), pushGossiper, vm.config.PushGossipFrequency.Duration)

	return cancel
}
