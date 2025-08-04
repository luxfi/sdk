// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/luxfi/log"
	"github.com/luxfi/sdk/internal/evm"
	"github.com/luxfi/sdk/internal/types"
	"github.com/luxfi/sdk/network"
)

// Builder handles blockchain creation and deployment
type Builder struct {
	logger      log.Logger
	blockchains map[string]*Blockchain
}

// Blockchain represents a Lux blockchain
type Blockchain struct {
	ID          string
	Name        string
	Type        BlockchainType
	VMType      VMType
	ChainID     types.ID
	Genesis     []byte
	ChainConfig []byte
	Status      BlockchainStatus
	CreatedAt   time.Time
	DeployedAt  *time.Time
	NetworkID   string
}

// BlockchainType defines the type of blockchain
type BlockchainType string

const (
	TypeL1 BlockchainType = "L1"
	TypeL2 BlockchainType = "L2"
	TypeL3 BlockchainType = "L3"
)

// VMType defines the virtual machine type
type VMType string

const (
	VMTypeEVM        VMType = "evm"
	VMTypeWASM       VMType = "wasm"
	VMTypeCustom     VMType = "custom"
	VMTypeTokenVM    VMType = "tokenvm"
	VMTypeMorpheusVM VMType = "morpheusvm"
)

// BlockchainStatus defines the status of a blockchain
type BlockchainStatus string

const (
	StatusCreated   BlockchainStatus = "created"
	StatusDeploying BlockchainStatus = "deploying"
	StatusDeployed  BlockchainStatus = "deployed"
	StatusRunning   BlockchainStatus = "running"
	StatusStopped   BlockchainStatus = "stopped"
	StatusError     BlockchainStatus = "error"
)

// NewBuilder creates a new blockchain builder
func NewBuilder(logger log.Logger) *Builder {
	return &Builder{
		logger:      logger,
		blockchains: make(map[string]*Blockchain),
	}
}

// CreateBlockchain creates a new blockchain
func (b *Builder) CreateBlockchain(ctx context.Context, params *CreateParams) (*Blockchain, error) {
	b.logger.Info("creating blockchain", "name", params.Name, "type", params.Type, "vm", params.VMType)

	// Generate chain ID
	chainID := types.GenerateTestID()

	// Create genesis based on VM type
	genesis, err := b.createGenesis(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create genesis: %w", err)
	}

	// Create chain configuration
	chainConfig, err := b.createChainConfig(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create chain config: %w", err)
	}

	// Create blockchain object
	blockchain := &Blockchain{
		ID:          types.GenerateTestID().String(),
		Name:        params.Name,
		Type:        params.Type,
		VMType:      params.VMType,
		ChainID:     chainID,
		Genesis:     genesis,
		ChainConfig: chainConfig,
		Status:      StatusCreated,
		CreatedAt:   time.Now(),
	}

	b.blockchains[blockchain.ID] = blockchain
	return blockchain, nil
}

// Deploy deploys a blockchain to a network
func (b *Builder) Deploy(ctx context.Context, blockchain *Blockchain, network *network.Network) error {
	b.logger.Info("deploying blockchain", "blockchain", blockchain.Name, "network", network.Name)

	blockchain.Status = StatusDeploying

	// Deploy based on blockchain type
	switch blockchain.Type {
	case TypeL1:
		if err := b.deployL1(ctx, blockchain, network); err != nil {
			blockchain.Status = StatusError
			return err
		}
	case TypeL2:
		if err := b.deployL2(ctx, blockchain, network); err != nil {
			blockchain.Status = StatusError
			return err
		}
	case TypeL3:
		if err := b.deployL3(ctx, blockchain, network); err != nil {
			blockchain.Status = StatusError
			return err
		}
	default:
		blockchain.Status = StatusError
		return fmt.Errorf("unsupported blockchain type: %s", blockchain.Type)
	}

	now := time.Now()
	blockchain.DeployedAt = &now
	blockchain.NetworkID = network.ID
	blockchain.Status = StatusDeployed

	return nil
}

// GetBlockchain returns a blockchain by ID
func (b *Builder) GetBlockchain(blockchainID string) (*Blockchain, error) {
	blockchain, ok := b.blockchains[blockchainID]
	if !ok {
		return nil, fmt.Errorf("blockchain %s not found", blockchainID)
	}
	return blockchain, nil
}

// ListBlockchains returns all blockchains
func (b *Builder) ListBlockchains() []*Blockchain {
	blockchains := make([]*Blockchain, 0, len(b.blockchains))
	for _, blockchain := range b.blockchains {
		blockchains = append(blockchains, blockchain)
	}
	return blockchains
}

// GenerateGenesis generates a genesis file for a blockchain
func (b *Builder) GenerateGenesis(params *GenesisParams) ([]byte, error) {
	switch params.VMType {
	case VMTypeEVM:
		return b.generateEVMGenesis(params)
	case VMTypeWASM:
		return b.generateWASMGenesis(params)
	case VMTypeTokenVM:
		return b.generateTokenVMGenesis(params)
	default:
		return nil, fmt.Errorf("unsupported VM type: %s", params.VMType)
	}
}

// ValidateConfig validates a chain configuration
func (b *Builder) ValidateConfig(config []byte) error {
	// Parse and validate configuration
	var cfg map[string]interface{}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid configuration format: %w", err)
	}

	// Validate required fields
	requiredFields := []string{"chainId", "consensus", "vm"}
	for _, field := range requiredFields {
		if _, ok := cfg[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	return nil
}

// createGenesis creates genesis based on VM type
func (b *Builder) createGenesis(params *CreateParams) ([]byte, error) {
	if params.Genesis != nil {
		return params.Genesis, nil
	}

	// Generate default genesis based on VM type
	genesisParams := &GenesisParams{
		VMType:        params.VMType,
		ChainID:       params.ChainID,
		Allocations:   params.Allocations,
		ValidatorSet:  params.ValidatorSet,
		InitialSupply: params.InitialSupply,
	}

	return b.GenerateGenesis(genesisParams)
}

// createChainConfig creates chain configuration
func (b *Builder) createChainConfig(params *CreateParams) ([]byte, error) {
	if params.ChainConfig != nil {
		return params.ChainConfig, nil
	}

	// Create default configuration
	config := map[string]interface{}{
		"chainId": params.ChainID,
		"consensus": map[string]interface{}{
			"type": "lux",
			"parameters": map[string]interface{}{
				"k":            21,
				"alpha":        13,
				"beta":         8,
				"maxBlockTime": "10s",
				"minBlockTime": "1s",
			},
		},
		"vm": map[string]interface{}{
			"type":   string(params.VMType),
			"config": params.VMConfig,
		},
		"network": map[string]interface{}{
			"minStake":         2000,
			"maxStake":         3000000,
			"minDelegation":    25,
			"minDelegationFee": 2,
		},
	}

	return json.Marshal(config)
}

// deployL1 deploys an L1 blockchain
func (b *Builder) deployL1(ctx context.Context, blockchain *Blockchain, network *network.Network) error {
	// L1 deployment logic
	b.logger.Info("deploying L1 blockchain", "chain", blockchain.Name)

	// TODO: Implement actual L1 deployment using netrunner
	// This would involve:
	// 1. Creating subnet
	// 2. Adding validators
	// 3. Creating blockchain in subnet
	// 4. Starting blockchain

	return nil
}

// deployL2 deploys an L2 blockchain
func (b *Builder) deployL2(ctx context.Context, blockchain *Blockchain, network *network.Network) error {
	// L2 deployment logic
	b.logger.Info("deploying L2 blockchain", "chain", blockchain.Name)

	// TODO: Implement L2 deployment
	// This would involve:
	// 1. Setting up sequencer
	// 2. Configuring DA layer
	// 3. Setting up bridge contracts
	// 4. Starting L2 chain

	return nil
}

// deployL3 deploys an L3 blockchain
func (b *Builder) deployL3(ctx context.Context, blockchain *Blockchain, network *network.Network) error {
	// L3 deployment logic
	b.logger.Info("deploying L3 blockchain", "chain", blockchain.Name)

	// TODO: Implement L3 deployment
	// This would involve:
	// 1. Connecting to L2
	// 2. Deploying app-specific contracts
	// 3. Starting L3 chain

	return nil
}

// generateEVMGenesis generates EVM genesis
func (b *Builder) generateEVMGenesis(params *GenesisParams) ([]byte, error) {
	// Convert allocations to evm.GenesisAccount
	evmAlloc := make(map[common.Address]evm.GenesisAccount)
	for addr, account := range params.Allocations {
		evmAlloc[addr] = evm.GenesisAccount{
			Balance: account.Balance,
			Code:    account.Code,
			Storage: account.Storage,
		}
	}

	genesis := evm.Genesis{
		Config: &evm.ChainConfig{
			ChainID: params.ChainID,
		},
		Alloc:     evmAlloc,
		Timestamp: uint64(time.Now().Unix()),
		GasLimit:  8000000,
	}

	return json.Marshal(genesis)
}

// generateWASMGenesis generates WASM genesis
func (b *Builder) generateWASMGenesis(params *GenesisParams) ([]byte, error) {
	// TODO: Implement WASM genesis generation
	return json.Marshal(map[string]interface{}{
		"chainID": params.ChainID,
		"vmType":  "wasm",
	})
}

// generateTokenVMGenesis generates TokenVM genesis
func (b *Builder) generateTokenVMGenesis(params *GenesisParams) ([]byte, error) {
	// TODO: Implement TokenVM genesis generation
	return json.Marshal(map[string]interface{}{
		"chainID": params.ChainID,
		"vmType":  "tokenvm",
		"supply":  params.InitialSupply,
	})
}

// CreateParams defines parameters for creating a blockchain
type CreateParams struct {
	Name          string
	Type          BlockchainType
	VMType        VMType
	ChainID       *big.Int
	Genesis       []byte
	ChainConfig   []byte
	VMConfig      map[string]interface{}
	Allocations   map[common.Address]GenesisAccount
	ValidatorSet  []Validator
	InitialSupply *big.Int
	L2Config      *L2Config
	L3Config      *L3Config
}

// L1Params defines parameters for L1 creation
type L1Params struct {
	VMType      VMType
	Genesis     []byte
	ChainConfig []byte
}

// L2Params defines parameters for L2 creation
type L2Params struct {
	VMType          VMType
	Genesis         []byte
	ChainConfig     []byte
	SequencerType   string
	DALayer         string
	SettlementChain string
}

// L3Params defines parameters for L3 creation
type L3Params struct {
	VMType      VMType
	Genesis     []byte
	ChainConfig []byte
	L2Chain     string
	AppType     string
	AppConfig   map[string]interface{}
}

// L2Config defines L2-specific configuration
type L2Config struct {
	SequencerType   string
	DALayer         string
	SettlementChain string
	BridgeContract  string
}

// L3Config defines L3-specific configuration
type L3Config struct {
	L2Chain   string
	AppType   string
	AppConfig map[string]interface{}
}

// GenesisParams defines parameters for genesis generation
type GenesisParams struct {
	VMType        VMType
	ChainID       *big.Int
	Allocations   map[common.Address]GenesisAccount
	ValidatorSet  []Validator
	InitialSupply *big.Int
}
