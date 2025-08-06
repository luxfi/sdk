// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package blockchain

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/log"
	"github.com/luxfi/sdk/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder_CreateBlockchain(t *testing.T) {
	logger := log.NewNoOpLogger()
	builder := NewBuilder(logger)

	tests := []struct {
		name    string
		params  *CreateParams
		wantErr bool
	}{
		{
			name: "create EVM L1",
			params: &CreateParams{
				Name:    "test-evm-l1",
				Type:    TypeL1,
				VMType:  VMTypeEVM,
				ChainID: big.NewInt(12345),
				Allocations: map[common.Address]GenesisAccount{
					common.HexToAddress("0x123"): {
						Balance: big.NewInt(1000000),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "create WASM L2",
			params: &CreateParams{
				Name:    "test-wasm-l2",
				Type:    TypeL2,
				VMType:  VMTypeWASM,
				ChainID: big.NewInt(23456),
				L2Config: &L2Config{
					SequencerType:   "centralized",
					DALayer:         "celestia",
					SettlementChain: "chain-123",
				},
			},
			wantErr: false,
		},
		{
			name: "create TokenVM L3",
			params: &CreateParams{
				Name:          "test-token-l3",
				Type:          TypeL3,
				VMType:        VMTypeTokenVM,
				ChainID:       big.NewInt(34567),
				InitialSupply: big.NewInt(1000000000),
				L3Config: &L3Config{
					L2Chain: "l2-chain-456",
					AppType: "defi",
					AppConfig: map[string]interface{}{
						"dex":     true,
						"lending": true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "with custom genesis",
			params: &CreateParams{
				Name:    "test-custom-genesis",
				Type:    TypeL1,
				VMType:  VMTypeEVM,
				Genesis: []byte(`{"chainId": 99999}`),
			},
			wantErr: false,
		},
		{
			name: "with custom chain config",
			params: &CreateParams{
				Name:   "test-custom-config",
				Type:   TypeL1,
				VMType: VMTypeEVM,
				ChainConfig: []byte(`{
					"chainId": 88888,
					"consensus": {
						"type": "lux",
						"parameters": {
							"k": 15
						}
					}
				}`),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			blockchain, err := builder.CreateBlockchain(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, blockchain)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, blockchain)
				assert.Equal(t, tt.params.Name, blockchain.Name)
				assert.Equal(t, tt.params.Type, blockchain.Type)
				assert.Equal(t, tt.params.VMType, blockchain.VMType)
				assert.Equal(t, StatusCreated, blockchain.Status)
				assert.NotEmpty(t, blockchain.ID)
				assert.NotNil(t, blockchain.Genesis)
				assert.NotNil(t, blockchain.ChainConfig)

				// Verify blockchain is stored
				stored, err := builder.GetBlockchain(blockchain.ID)
				assert.NoError(t, err)
				assert.Equal(t, blockchain, stored)
			}
		})
	}
}

func TestBuilder_Deploy(t *testing.T) {
	logger := log.NewNoOpLogger()
	builder := NewBuilder(logger)
	ctx := context.Background()

	// Create a test blockchain
	blockchain, err := builder.CreateBlockchain(ctx, &CreateParams{
		Name:    "deploy-test",
		Type:    TypeL1,
		VMType:  VMTypeEVM,
		ChainID: big.NewInt(55555),
	})
	require.NoError(t, err)

	// Create a test network
	testNetwork := &network.Network{
		ID:     "test-network",
		Name:   "Test Network",
		Type:   network.NetworkTypeLocal,
		Status: network.NetworkStatusRunning,
	}

	t.Run("deploy L1", func(t *testing.T) {
		err := builder.Deploy(ctx, blockchain, testNetwork)
		// Since actual deployment is not implemented, we expect no error
		// but the blockchain should be marked as deployed
		assert.NoError(t, err)
		assert.Equal(t, StatusDeployed, blockchain.Status)
		assert.NotNil(t, blockchain.DeployedAt)
		assert.Equal(t, testNetwork.ID, blockchain.NetworkID)
	})

	t.Run("deploy L2", func(t *testing.T) {
		l2Blockchain, err := builder.CreateBlockchain(ctx, &CreateParams{
			Name:   "l2-deploy-test",
			Type:   TypeL2,
			VMType: VMTypeEVM,
			L2Config: &L2Config{
				SequencerType: "centralized",
			},
		})
		require.NoError(t, err)

		err = builder.Deploy(ctx, l2Blockchain, testNetwork)
		assert.NoError(t, err)
		assert.Equal(t, StatusDeployed, l2Blockchain.Status)
	})

	t.Run("deploy L3", func(t *testing.T) {
		l3Blockchain, err := builder.CreateBlockchain(ctx, &CreateParams{
			Name:   "l3-deploy-test",
			Type:   TypeL3,
			VMType: VMTypeWASM,
			L3Config: &L3Config{
				L2Chain: "l2-chain",
				AppType: "gaming",
			},
		})
		require.NoError(t, err)

		err = builder.Deploy(ctx, l3Blockchain, testNetwork)
		assert.NoError(t, err)
		assert.Equal(t, StatusDeployed, l3Blockchain.Status)
	})

	t.Run("deploy error recovery", func(t *testing.T) {
		errorBlockchain, err := builder.CreateBlockchain(ctx, &CreateParams{
			Name:   "error-test",
			Type:   BlockchainType("invalid"),
			VMType: VMTypeEVM,
		})
		require.NoError(t, err)

		err = builder.Deploy(ctx, errorBlockchain, testNetwork)
		assert.Error(t, err)
		assert.Equal(t, StatusError, errorBlockchain.Status)
	})
}

func TestBuilder_GenerateGenesis(t *testing.T) {
	logger := log.NewNoOpLogger()
	builder := NewBuilder(logger)

	tests := []struct {
		name    string
		params  *GenesisParams
		wantErr bool
		check   func(t *testing.T, genesis []byte)
	}{
		{
			name: "EVM genesis",
			params: &GenesisParams{
				VMType:  VMTypeEVM,
				ChainID: big.NewInt(12345),
				Allocations: map[common.Address]GenesisAccount{
					common.HexToAddress("0xabc"): {
						Balance: big.NewInt(1000000),
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, genesis []byte) {
				var g map[string]interface{}
				err := json.Unmarshal(genesis, &g)
				assert.NoError(t, err)

				config, ok := g["config"].(map[string]interface{})
				assert.True(t, ok)

				chainID, ok := config["chainId"].(float64)
				assert.True(t, ok)
				assert.Equal(t, float64(12345), chainID)
			},
		},
		{
			name: "WASM genesis",
			params: &GenesisParams{
				VMType:  VMTypeWASM,
				ChainID: big.NewInt(23456),
			},
			wantErr: false,
			check: func(t *testing.T, genesis []byte) {
				var g map[string]interface{}
				err := json.Unmarshal(genesis, &g)
				assert.NoError(t, err)
				assert.Equal(t, "wasm", g["vmType"])
			},
		},
		{
			name: "TokenVM genesis",
			params: &GenesisParams{
				VMType:        VMTypeTokenVM,
				ChainID:       big.NewInt(34567),
				InitialSupply: big.NewInt(1000000000),
			},
			wantErr: false,
			check: func(t *testing.T, genesis []byte) {
				var g map[string]interface{}
				err := json.Unmarshal(genesis, &g)
				assert.NoError(t, err)
				assert.Equal(t, "tokenvm", g["vmType"])
				assert.NotNil(t, g["supply"])
			},
		},
		{
			name: "unsupported VM type",
			params: &GenesisParams{
				VMType: VMType("unsupported"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			genesis, err := builder.GenerateGenesis(tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, genesis)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, genesis)
				if tt.check != nil {
					tt.check(t, genesis)
				}
			}
		})
	}
}

func TestBuilder_ValidateConfig(t *testing.T) {
	logger := log.NewNoOpLogger()
	builder := NewBuilder(logger)

	tests := []struct {
		name    string
		config  []byte
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: []byte(`{
				"chainId": 12345,
				"consensus": {
					"type": "lux"
				},
				"vm": {
					"type": "evm"
				}
			}`),
			wantErr: false,
		},
		{
			name: "missing required field - chainId",
			config: []byte(`{
				"consensus": {
					"type": "lux"
				},
				"vm": {
					"type": "evm"
				}
			}`),
			wantErr: true,
			errMsg:  "missing required field: chainId",
		},
		{
			name: "missing required field - consensus",
			config: []byte(`{
				"chainId": 12345,
				"vm": {
					"type": "evm"
				}
			}`),
			wantErr: true,
			errMsg:  "missing required field: consensus",
		},
		{
			name: "missing required field - vm",
			config: []byte(`{
				"chainId": 12345,
				"consensus": {
					"type": "lux"
				}
			}`),
			wantErr: true,
			errMsg:  "missing required field: vm",
		},
		{
			name:    "invalid JSON",
			config:  []byte(`{invalid json`),
			wantErr: true,
			errMsg:  "invalid configuration format",
		},
		{
			name:    "empty config",
			config:  []byte(`{}`),
			wantErr: true,
			errMsg:  "missing required field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := builder.ValidateConfig(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuilder_ListBlockchains(t *testing.T) {
	logger := log.NewNoOpLogger()
	builder := NewBuilder(logger)
	ctx := context.Background()

	// Create multiple blockchains
	blockchains := []struct {
		name   string
		bcType BlockchainType
		vmType VMType
	}{
		{"chain-1", TypeL1, VMTypeEVM},
		{"chain-2", TypeL2, VMTypeWASM},
		{"chain-3", TypeL3, VMTypeTokenVM},
	}

	createdIDs := make(map[string]bool)

	for _, bc := range blockchains {
		blockchain, err := builder.CreateBlockchain(ctx, &CreateParams{
			Name:   bc.name,
			Type:   bc.bcType,
			VMType: bc.vmType,
		})
		require.NoError(t, err)
		createdIDs[blockchain.ID] = true
	}

	// List all blockchains
	list := builder.ListBlockchains()

	assert.Len(t, list, 3)

	// Verify all blockchains are present
	for _, bc := range list {
		assert.True(t, createdIDs[bc.ID])
	}
}

func TestBuilder_GetBlockchain(t *testing.T) {
	logger := log.NewNoOpLogger()
	builder := NewBuilder(logger)
	ctx := context.Background()

	// Create a test blockchain
	blockchain, err := builder.CreateBlockchain(ctx, &CreateParams{
		Name:   "get-test",
		Type:   TypeL1,
		VMType: VMTypeEVM,
	})
	require.NoError(t, err)

	t.Run("existing blockchain", func(t *testing.T) {
		result, err := builder.GetBlockchain(blockchain.ID)
		assert.NoError(t, err)
		assert.Equal(t, blockchain, result)
	})

	t.Run("non-existent blockchain", func(t *testing.T) {
		result, err := builder.GetBlockchain("non-existent")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestBuilder_CreateChainConfig(t *testing.T) {
	logger := log.NewNoOpLogger()
	builder := NewBuilder(logger)
	ctx := context.Background()

	// Test default chain config generation
	params := &CreateParams{
		Name:    "config-test",
		Type:    TypeL1,
		VMType:  VMTypeEVM,
		ChainID: big.NewInt(77777),
		VMConfig: map[string]interface{}{
			"gasLimit": 15000000,
		},
	}

	blockchain, err := builder.CreateBlockchain(ctx, params)
	require.NoError(t, err)

	// Verify the generated config
	var config map[string]interface{}
	err = json.Unmarshal(blockchain.ChainConfig, &config)
	assert.NoError(t, err)

	// Check chain ID
	chainID, ok := config["chainId"].(float64)
	assert.True(t, ok)
	assert.Equal(t, float64(77777), chainID)

	// Check consensus config
	consensus, ok := config["consensus"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "lux", consensus["type"])

	// Check VM config
	vm, ok := config["vm"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "evm", vm["type"])

	vmConfig, ok := vm["config"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(15000000), vmConfig["gasLimit"])
}
