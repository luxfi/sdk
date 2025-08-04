// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package sdk

import (
	"context"
	"fmt"
	"math/big"

	"github.com/luxfi/sdk/blockchain"
	"github.com/luxfi/sdk/config"
	"github.com/luxfi/log"
	"github.com/luxfi/sdk/network"
)

// LuxSDK is the main SDK interface providing comprehensive blockchain development capabilities
type LuxSDK struct {
	networkManager    *network.NetworkManager
	blockchainBuilder *blockchain.Builder
	config           *config.Config
	logger           log.Logger
}

// New creates a new instance of the Lux SDK
func New(cfg *config.Config) (*LuxSDK, error) {
	if cfg == nil {
		cfg = config.Default()
	}

	// Create logger using lux log package
	// Use the global logger with SDK context
	logger := log.New("sdk")

	// Initialize network manager
	networkManager, err := network.NewNetworkManager(cfg.Network, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create network manager: %w", err)
	}

	// Initialize blockchain builder
	blockchainBuilder := blockchain.NewBuilder(logger)

	return &LuxSDK{
		networkManager:    networkManager,
		blockchainBuilder: blockchainBuilder,
		config:           cfg,
		logger:           logger,
	}, nil
}

// Networks returns the network manager for network operations
func (sdk *LuxSDK) Networks() *network.NetworkManager {
	return sdk.networkManager
}

// Blockchains returns the blockchain builder for blockchain operations
func (sdk *LuxSDK) Blockchains() *blockchain.Builder {
	return sdk.blockchainBuilder
}

// LaunchNetwork launches a network using the network manager
func (sdk *LuxSDK) LaunchNetwork(ctx context.Context, networkType string, numNodes int) (*network.Network, error) {
	params := &network.NetworkParams{
		Name:     networkType,
		Type:     network.NetworkType(networkType),
		NumNodes: numNodes,
	}
	return sdk.networkManager.CreateNetwork(ctx, params)
}

// CreateAndDeployBlockchain creates and deploys a blockchain
func (sdk *LuxSDK) CreateAndDeployBlockchain(ctx context.Context, params *BlockchainParams) (*blockchain.Blockchain, error) {
	// Create blockchain configuration
	createParams := &blockchain.CreateParams{
		Name:    params.Name,
		Type:    params.Type,
		VMType:  params.VMType,
		ChainID: params.ChainID,
		Genesis: params.Genesis,
	}

	// Create blockchain
	bc, err := sdk.blockchainBuilder.CreateBlockchain(ctx, createParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create blockchain: %w", err)
	}

	// Deploy blockchain to network if specified
	if params.Network != nil {
		if err := sdk.blockchainBuilder.Deploy(ctx, bc, params.Network); err != nil {
			return nil, fmt.Errorf("failed to deploy blockchain: %w", err)
		}
	}

	return bc, nil
}

// BlockchainParams defines parameters for creating and deploying a blockchain
type BlockchainParams struct {
	Name    string
	Type    blockchain.BlockchainType
	VMType  blockchain.VMType
	ChainID *big.Int
	Genesis []byte
	Network *network.Network
}

// NodeInfo contains information about a node
type NodeInfo struct {
	NodeID      string
	Version     string
	NetworkID   uint32
	NetworkName string
}