// Copyright (C) 2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package sdk

import (
	"context"
	"fmt"

	"github.com/luxfi/sdk/blockchain"
	"github.com/luxfi/sdk/config"
	"github.com/luxfi/sdk/integration"
	"github.com/luxfi/sdk/internal/logging"
	"github.com/luxfi/sdk/network"
	"github.com/luxfi/sdk/vm"
)

// LuxSDK is the main SDK interface providing comprehensive blockchain development capabilities
type LuxSDK struct {
	networkManager    *network.NetworkManager
	blockchainBuilder *blockchain.Builder
	vmManager         *vm.Manager
	config           *config.Config
	logger           logging.Logger
	
	// Integrations with other Lux components
	netrunner *integration.NetrunnerIntegration
	cli       *integration.CLIIntegration
	node      *integration.NodeIntegration
}

// New creates a new instance of the Lux SDK
func New(cfg *config.Config) (*LuxSDK, error) {
	if cfg == nil {
		cfg = config.Default()
	}

	logger := logging.NewLogger(cfg.LogLevel)

	// Initialize network manager
	networkManager, err := network.NewNetworkManager(cfg.Network, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create network manager: %w", err)
	}

	// Initialize blockchain builder
	blockchainBuilder := blockchain.NewBuilder(logger)

	// Initialize VM manager
	vmManager := vm.NewManager(logger)

	// Initialize integrations (optional)
	var netrunnerInt *integration.NetrunnerIntegration
	var cliInt *integration.CLIIntegration
	var nodeInt *integration.NodeIntegration

	// Try to initialize netrunner integration
	if netrunnerInt, err = integration.NewNetrunnerIntegration(logger); err != nil {
		logger.Warn("netrunner integration not available", "error", err)
	}

	// Try to initialize CLI integration
	if cliInt, err = integration.NewCLIIntegration(logger); err != nil {
		logger.Warn("CLI integration not available", "error", err)
	}

	// Try to initialize node integration if endpoint is configured
	if cfg.NodeEndpoint != "" {
		if nodeInt, err = integration.NewNodeIntegration(logger, cfg.NodeEndpoint); err != nil {
			logger.Warn("node integration not available", "error", err)
		}
	}

	return &LuxSDK{
		networkManager:    networkManager,
		blockchainBuilder: blockchainBuilder,
		vmManager:        vmManager,
		config:           cfg,
		logger:           logger,
		netrunner:        netrunnerInt,
		cli:              cliInt,
		node:             nodeInt,
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

// VMs returns the VM manager for VM operations
func (sdk *LuxSDK) VMs() *vm.Manager {
	return sdk.vmManager
}

// LaunchNetwork launches a network using the best available method
func (sdk *LuxSDK) LaunchNetwork(ctx context.Context, networkType string, numNodes int) (*network.Network, error) {
	// Try CLI first (most user-friendly)
	if sdk.cli != nil {
		if err := sdk.cli.LaunchNetwork(ctx, networkType); err == nil {
			return &network.Network{
				Name:   networkType,
				Type:   network.NetworkType(networkType),
				Status: network.NetworkStatusRunning,
			}, nil
		}
	}

	// Try netrunner (more control)
	if sdk.netrunner != nil {
		tmpnet, err := sdk.netrunner.CreateNetwork(ctx, networkType, numNodes)
		if err == nil {
			return &network.Network{
				Name:   tmpnet.Name,
				Type:   network.NetworkType(networkType),
				Status: network.NetworkStatusRunning,
			}, nil
		}
	}

	// Fall back to SDK's built-in network manager
	params := &network.NetworkParams{
		Name:     networkType,
		Type:     network.NetworkType(networkType),
		NumNodes: numNodes,
	}
	return sdk.networkManager.CreateNetwork(ctx, params)
}

// CreateAndDeployBlockchain creates and deploys a blockchain using the best available method
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

	// Try to deploy using CLI if available
	if sdk.cli != nil && params.Network != nil {
		if err := sdk.cli.DeployBlockchain(ctx, bc.Name, params.Network.Name); err == nil {
			bc.Status = blockchain.StatusDeployed
			return bc, nil
		}
	}

	// Try to deploy using netrunner if available
	if sdk.netrunner != nil && params.Network != nil {
		spec := &integration.BlockchainSpec{
			Name:        bc.Name,
			VMType:      string(bc.VMType),
			Genesis:     bc.Genesis,
			ChainConfig: bc.ChainConfig,
		}
		if err := sdk.netrunner.DeployBlockchain(ctx, params.Network.ID, spec); err == nil {
			bc.Status = blockchain.StatusDeployed
			return bc, nil
		}
	}

	// Fall back to SDK's built-in deployment
	if params.Network != nil {
		if err := sdk.blockchainBuilder.Deploy(ctx, bc, params.Network); err != nil {
			return nil, fmt.Errorf("failed to deploy blockchain: %w", err)
		}
	}

	return bc, nil
}

// GetNodeInfo returns information about the connected node
func (sdk *LuxSDK) GetNodeInfo(ctx context.Context) (*NodeInfo, error) {
	if sdk.node == nil {
		return nil, fmt.Errorf("node integration not available")
	}

	info, err := sdk.node.GetNodeInfo(ctx)
	if err != nil {
		return nil, err
	}

	return &NodeInfo{
		NodeID:      info.NodeID.String(),
		Version:     info.Version,
		NetworkID:   info.NetworkID,
		NetworkName: info.NetworkName,
	}, nil
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