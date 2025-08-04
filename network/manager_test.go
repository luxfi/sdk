// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package network

import (
	"context"
	"testing"

	"github.com/luxfi/log"
	"github.com/luxfi/sdk/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkManager_CreateNetwork(t *testing.T) {
	tests := []struct {
		name        string
		params      *NetworkParams
		wantErr     bool
		expectedErr string
	}{
		{
			name: "successful local network creation",
			params: &NetworkParams{
				Name:             "test-local",
				Type:             NetworkTypeLocal,
				NumNodes:         5,
				EnableMonitoring: false,
			},
			wantErr: false,
		},
		{
			name: "successful testnet network creation",
			params: &NetworkParams{
				Name:             "test-testnet",
				Type:             NetworkTypeTestnet,
				NumNodes:         3,
				EnableMonitoring: true,
			},
			wantErr: false,
		},
		{
			name: "invalid network type",
			params: &NetworkParams{
				Name:     "test-invalid",
				Type:     NetworkType("invalid"),
				NumNodes: 5,
			},
			wantErr:     true,
			expectedErr: "unsupported network type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.NewNoOpLogger()
			nm, err := NewNetworkManager(&config.NetworkConfig{}, logger)
			require.NoError(t, err)

			ctx := context.Background()
			network, err := nm.CreateNetwork(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != "" {
					assert.Contains(t, err.Error(), tt.expectedErr)
				}
				assert.Nil(t, network)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, network)
				assert.Equal(t, tt.params.Name, network.Name)
				assert.Equal(t, tt.params.Type, network.Type)
				assert.Equal(t, NetworkStatusRunning, network.Status)
				assert.NotEmpty(t, network.ID)
			}
		})
	}
}

func TestNetworkManager_StartNetwork(t *testing.T) {
	logger := log.NewNoOpLogger()
	nm, err := NewNetworkManager(&config.NetworkConfig{}, logger)
	require.NoError(t, err)
	ctx := context.Background()

	// Create a test network first
	params := &NetworkParams{
		Name:     "start-test",
		Type:     NetworkTypeLocal,
		NumNodes: 3,
	}
	network, err := nm.CreateNetwork(ctx, params)
	require.NoError(t, err)

	// Test starting the network
	err = nm.StartNetwork(ctx, network.ID)
	assert.NoError(t, err)

	// Verify network status
	updatedNetwork, err := nm.GetNetwork(network.ID)
	assert.NoError(t, err)
	assert.Equal(t, NetworkStatusRunning, updatedNetwork.Status)
}

func TestNetworkManager_StopNetwork(t *testing.T) {
	logger := log.NewNoOpLogger()
	nm, err := NewNetworkManager(&config.NetworkConfig{}, logger)
	require.NoError(t, err)
	ctx := context.Background()

	// Create and start a test network
	params := &NetworkParams{
		Name:     "stop-test",
		Type:     NetworkTypeLocal,
		NumNodes: 3,
	}
	network, err := nm.CreateNetwork(ctx, params)
	require.NoError(t, err)

	err = nm.StartNetwork(ctx, network.ID)
	require.NoError(t, err)

	// Test stopping the network
	err = nm.StopNetwork(ctx, network.ID)
	assert.NoError(t, err)

	// Verify network status
	updatedNetwork, err := nm.GetNetwork(network.ID)
	assert.NoError(t, err)
	assert.Equal(t, NetworkStatusStopped, updatedNetwork.Status)
}

func TestNetworkManager_DeleteNetwork(t *testing.T) {
	logger := log.NewNoOpLogger()
	nm, err := NewNetworkManager(&config.NetworkConfig{}, logger)
	require.NoError(t, err)
	ctx := context.Background()

	// Create a test network
	params := &NetworkParams{
		Name:     "delete-test",
		Type:     NetworkTypeLocal,
		NumNodes: 3,
	}
	network, err := nm.CreateNetwork(ctx, params)
	require.NoError(t, err)

	// Test deleting the network
	err = nm.DeleteNetwork(ctx, network.ID)
	assert.NoError(t, err)

	// Verify network is deleted
	_, err = nm.GetNetwork(network.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestNetworkManager_GetNodeStatus(t *testing.T) {
	logger := log.NewNoOpLogger()
	nm, err := NewNetworkManager(&config.NetworkConfig{}, logger)
	require.NoError(t, err)
	ctx := context.Background()

	// Create a test network
	params := &NetworkParams{
		Name:     "status-test",
		Type:     NetworkTypeLocal,
		NumNodes: 3,
	}
	network, err := nm.CreateNetwork(ctx, params)
	require.NoError(t, err)

	// Test getting node status
	if len(network.Nodes) > 0 {
		nodeID := network.Nodes[0].ID
		status, err := nm.GetNodeStatus(ctx, network.ID, nodeID)
		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, NodeStatusHealthy, *status)
	}
}

func TestNetworkManager_ListNetworks(t *testing.T) {
	logger := log.NewNoOpLogger()
	nm, err := NewNetworkManager(&config.NetworkConfig{}, logger)
	require.NoError(t, err)
	ctx := context.Background()

	// Test empty list initially
	list := nm.ListNetworks()
	initialCount := len(list)

	// Create multiple test networks
	networks := []string{"network1", "network2", "network3"}
	for _, name := range networks {
		params := &NetworkParams{
			Name:     name,
			Type:     NetworkTypeLocal,
			NumNodes: 3,
		}
		_, err := nm.CreateNetwork(ctx, params)
		require.NoError(t, err)
	}

	// Test listing networks
	list = nm.ListNetworks()
	assert.Equal(t, initialCount+3, len(list))

	// Verify all created networks are in the list
	names := make(map[string]bool)
	for _, n := range list {
		names[n.Name] = true
	}
	for _, expectedName := range networks {
		assert.True(t, names[expectedName], "Network %s should be in the list", expectedName)
	}
}
