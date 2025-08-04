// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build integration
// +build integration

package tests

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk"
	"github.com/luxfi/sdk/blockchain"
	"github.com/luxfi/sdk/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_FullWorkflow tests the complete SDK workflow
func TestIntegration_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Initialize SDK
	luxSDK, err := sdk.New(
		sdk.WithLogLevel("debug"),
		sdk.WithDataDir(t.TempDir()),
	)
	require.NoError(t, err)
	defer luxSDK.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Step 1: Create a local network
	t.Log("Creating local network...")
	network, err := luxSDK.CreateNetwork(ctx, &network.NetworkParams{
		Name:          "integration-test",
		Type:          network.NetworkTypeLocal,
		NumNodes:      5,
		EnableStaking: true,
	})
	require.NoError(t, err)
	assert.Equal(t, network.NetworkStatusRunning, network.Status)

	// Wait for network to stabilize
	time.Sleep(10 * time.Second)

	// Step 2: Create and deploy an L1 blockchain
	t.Log("Creating L1 blockchain...")
	l1Chain, err := luxSDK.CreateL1(ctx, "test-l1", &blockchain.L1Params{
		VMType: blockchain.VMTypeEVM,
		ChainConfig: []byte(`{
			"feeConfig": {
				"gasLimit": 15000000,
				"minBaseFee": 25000000000
			}
		}`),
	})
	require.NoError(t, err)

	t.Log("Deploying L1 blockchain...")
	err = luxSDK.DeployBlockchain(ctx, l1Chain.ID, network.ID)
	require.NoError(t, err)

	// Step 3: Test network operations
	t.Run("NetworkOperations", func(t *testing.T) {
		// Add a new node
		node, err := luxSDK.networkManager.AddNode(ctx, network.ID, &network.NodeParams{
			Name:        "additional-node",
			Type:        network.NodeTypeValidator,
			StakeAmount: 2000,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, node.ID)

		// Check node status
		status, err := luxSDK.networkManager.GetNodeStatus(ctx, network.ID, node.ID)
		require.NoError(t, err)
		assert.NotNil(t, status)

		// List networks
		networks := luxSDK.ListNetworks()
		assert.GreaterOrEqual(t, len(networks), 1)
	})

	// Step 4: Test blockchain operations
	t.Run("BlockchainOperations", func(t *testing.T) {
		// Create L2
		l2Chain, err := luxSDK.CreateL2(ctx, "test-l2", &blockchain.L2Params{
			VMType:          blockchain.VMTypeEVM,
			SequencerType:   "centralized",
			DALayer:         "celestia",
			SettlementChain: l1Chain.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, blockchain.StatusCreated, l2Chain.Status)

		// Deploy L2
		err = luxSDK.DeployBlockchain(ctx, l2Chain.ID, network.ID)
		require.NoError(t, err)

		// Validate chain config
		err = luxSDK.ValidateChainConfig(l2Chain.ChainConfig)
		assert.NoError(t, err)
	})

	// Step 5: Test client operations
	t.Run("ClientOperations", func(t *testing.T) {
		endpoint := fmt.Sprintf("http://localhost:%d", 9650)
		client, err := luxSDK.NewClient(endpoint)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	// Step 6: Clean up
	t.Log("Cleaning up...")
	err = luxSDK.StopNetwork(ctx, network.ID)
	assert.NoError(t, err)

	err = luxSDK.DeleteNetwork(ctx, network.ID)
	assert.NoError(t, err)
}

// TestIntegration_ChainOperations tests chain-specific operations
func TestIntegration_ChainOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	endpoint := os.Getenv("LUX_ENDPOINT")
	if endpoint == "" {
		t.Skip("LUX_ENDPOINT not set, skipping chain operations test")
	}

	// This test requires a running Lux node
	ctx := context.Background()

	// Initialize chain manager
	wallet, err := wallet.New("test-wallet")
	require.NoError(t, err)

	chainManager, err := chain.NewChainManager(endpoint, wallet, log.NewNoop())
	require.NoError(t, err)

	t.Run("PChainOperations", func(t *testing.T) {
		// Get validators
		validators, err := chainManager.GetValidators(ctx)
		require.NoError(t, err)
		assert.NotNil(t, validators)
		t.Logf("Current validators: %d", len(validators.Current))

		// Get minimum stake
		minStake, err := chainManager.P().GetMinStake(ctx)
		require.NoError(t, err)
		assert.NotNil(t, minStake)
		t.Logf("Min validator stake: %s", minStake.MinValidatorStake)

		// Get P-Chain height
		height, err := chainManager.P().GetHeight(ctx)
		require.NoError(t, err)
		assert.Greater(t, height, uint64(0))
		t.Logf("P-Chain height: %d", height)
	})

	t.Run("XChainOperations", func(t *testing.T) {
		// Get X-Chain balance
		balance, err := chainManager.X().GetBalance(ctx, wallet.X().Address().String(), ids.Empty)
		require.NoError(t, err)
		assert.NotNil(t, balance)
		t.Logf("X-Chain LUX balance: %s", balance)

		// Get all balances
		allBalances, err := chainManager.X().GetAllBalances(ctx, wallet.X().Address().String())
		require.NoError(t, err)
		assert.NotNil(t, allBalances)
		t.Logf("Total assets on X-Chain: %d", len(allBalances))
	})

	t.Run("CChainOperations", func(t *testing.T) {
		// Get C-Chain balance
		cBalance, err := chainManager.C().GetBalance(ctx, wallet.C().Address().String())
		require.NoError(t, err)
		assert.NotNil(t, cBalance)
		t.Logf("C-Chain balance: %s", cBalance)

		// Get gas price
		gasPrice, err := chainManager.C().GetGasPrice(ctx)
		require.NoError(t, err)
		assert.NotNil(t, gasPrice)
		t.Logf("Current gas price: %s", gasPrice)

		// Get block number
		blockNumber, err := chainManager.C().GetBlockNumber(ctx)
		require.NoError(t, err)
		assert.Greater(t, blockNumber, uint64(0))
		t.Logf("C-Chain block number: %d", blockNumber)
	})

	t.Run("CrossChainBalance", func(t *testing.T) {
		// Get balance across all chains
		balances, err := chainManager.GetBalance(ctx, wallet.P().Address().String())
		require.NoError(t, err)
		assert.NotNil(t, balances)

		for chain, balance := range balances.Chains {
			if balance.LUX != nil {
				t.Logf("%s-Chain LUX balance: %s", chain, balance.LUX)
			}
			for assetID, amount := range balance.Assets {
				t.Logf("%s-Chain asset %s balance: %s", chain, assetID, amount)
			}
		}
	})
}

// TestIntegration_Performance tests SDK performance
func TestIntegration_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	luxSDK, err := sdk.New(
		sdk.WithLogLevel("error"), // Reduce logging for performance test
		sdk.WithDataDir(t.TempDir()),
	)
	require.NoError(t, err)
	defer luxSDK.Close()

	ctx := context.Background()

	t.Run("BlockchainCreationPerformance", func(t *testing.T) {
		start := time.Now()
		numBlockchains := 10

		for i := 0; i < numBlockchains; i++ {
			_, err := luxSDK.CreateBlockchain(ctx, &blockchain.CreateParams{
				Name:    fmt.Sprintf("perf-test-%d", i),
				Type:    blockchain.TypeL1,
				VMType:  blockchain.VMTypeEVM,
				ChainID: big.NewInt(int64(10000 + i)),
			})
			require.NoError(t, err)
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(numBlockchains)
		t.Logf("Created %d blockchains in %v (avg: %v per blockchain)", numBlockchains, elapsed, avgTime)

		// Performance assertion
		assert.Less(t, avgTime, 100*time.Millisecond, "Blockchain creation should be fast")
	})

	t.Run("ConcurrentOperations", func(t *testing.T) {
		// Test concurrent blockchain creation
		numGoroutines := 5
		numOpsPerGoroutine := 10
		errors := make(chan error, numGoroutines)

		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				for j := 0; j < numOpsPerGoroutine; j++ {
					_, err := luxSDK.CreateBlockchain(ctx, &blockchain.CreateParams{
						Name:    fmt.Sprintf("concurrent-%d-%d", goroutineID, j),
						Type:    blockchain.TypeL1,
						VMType:  blockchain.VMTypeEVM,
						ChainID: big.NewInt(int64(20000 + goroutineID*100 + j)),
					})
					if err != nil {
						errors <- err
						return
					}
				}
				errors <- nil
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			err := <-errors
			assert.NoError(t, err)
		}

		elapsed := time.Since(start)
		totalOps := numGoroutines * numOpsPerGoroutine
		opsPerSecond := float64(totalOps) / elapsed.Seconds()

		t.Logf("Completed %d concurrent operations in %v (%.2f ops/sec)", totalOps, elapsed, opsPerSecond)
		assert.Greater(t, opsPerSecond, 50.0, "Should handle at least 50 ops/sec")
	})
}

// TestIntegration_ErrorHandling tests error handling and recovery
func TestIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping error handling test in short mode")
	}

	luxSDK, err := sdk.New(
		sdk.WithLogLevel("debug"),
		sdk.WithDataDir(t.TempDir()),
	)
	require.NoError(t, err)
	defer luxSDK.Close()

	ctx := context.Background()

	t.Run("InvalidNetworkOperations", func(t *testing.T) {
		// Try to stop non-existent network
		err := luxSDK.StopNetwork(ctx, "non-existent-network")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Try to get non-existent network
		_, err = luxSDK.GetNetwork("non-existent-network")
		assert.Error(t, err)
	})

	t.Run("InvalidBlockchainOperations", func(t *testing.T) {
		// Try to deploy non-existent blockchain
		err := luxSDK.DeployBlockchain(ctx, "non-existent-blockchain", "some-network")
		assert.Error(t, err)

		// Try to validate invalid config
		err = luxSDK.ValidateChainConfig([]byte("invalid json"))
		assert.Error(t, err)
	})

	t.Run("RecoveryFromErrors", func(t *testing.T) {
		// Create a blockchain with invalid parameters
		_, err := luxSDK.CreateBlockchain(ctx, &blockchain.CreateParams{
			Name:   "", // Invalid empty name
			Type:   blockchain.TypeL1,
			VMType: blockchain.VMTypeEVM,
		})
		assert.Error(t, err)

		// SDK should still be functional after error
		blockchain, err := luxSDK.CreateBlockchain(ctx, &blockchain.CreateParams{
			Name:   "valid-blockchain",
			Type:   blockchain.TypeL1,
			VMType: blockchain.VMTypeEVM,
		})
		assert.NoError(t, err)
		assert.NotNil(t, blockchain)
	})
}
