// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chainconfig

import (
	"math/big"

	"github.com/luxfi/evm/params"
	"github.com/luxfi/geth/common"
)

// ChainConfigBuilder helps construct EVM chain configurations
type ChainConfigBuilder struct {
	config *params.ChainConfig
	// Additional fields for SubnetEVM specific features
	feeConfig          interface{}
	allowFeeRecipients bool
	precompiles        map[string]interface{}
	networkUpgrades    map[string]interface{}
}

// NewChainConfigBuilder creates a new chain config builder with subnet EVM defaults
func NewChainConfigBuilder() *ChainConfigBuilder {
	// Start with standard EVM configuration
	config := &params.ChainConfig{
		ChainID:             big.NewInt(99999), // Default chain ID
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
	}
	
	return &ChainConfigBuilder{
		config:          config,
		precompiles:     make(map[string]interface{}),
		networkUpgrades: make(map[string]interface{}),
	}
}

// WithChainID sets the chain ID
func (b *ChainConfigBuilder) WithChainID(chainID *big.Int) *ChainConfigBuilder {
	b.config.ChainID = chainID
	return b
}

// WithFeeConfig sets the fee configuration
func (b *ChainConfigBuilder) WithFeeConfig(feeConfig interface{}) *ChainConfigBuilder {
	b.feeConfig = feeConfig
	return b
}

// WithAllowFeeRecipients enables/disables fee recipients
func (b *ChainConfigBuilder) WithAllowFeeRecipients(allow bool) *ChainConfigBuilder {
	b.allowFeeRecipients = allow
	return b
}

// WithPrecompile adds a precompile configuration
func (b *ChainConfigBuilder) WithPrecompile(address common.Address, config interface{}) *ChainConfigBuilder {
	b.precompiles[address.Hex()] = config
	return b
}

// WithNetworkUpgrade adds a network upgrade configuration
func (b *ChainConfigBuilder) WithNetworkUpgrade(name string, timestamp *big.Int) *ChainConfigBuilder {
	b.networkUpgrades[name] = timestamp
	return b
}

// Build returns the constructed chain configuration
func (b *ChainConfigBuilder) Build() *params.ChainConfig {
	return b.config
}

// GetFeeConfig returns the fee configuration
func (b *ChainConfigBuilder) GetFeeConfig() interface{} {
	return b.feeConfig
}

// GetAllowFeeRecipients returns the allow fee recipients setting
func (b *ChainConfigBuilder) GetAllowFeeRecipients() bool {
	return b.allowFeeRecipients
}

// GetPrecompiles returns the precompile configurations
func (b *ChainConfigBuilder) GetPrecompiles() map[string]interface{} {
	return b.precompiles
}

// GetNetworkUpgrades returns the network upgrade configurations
func (b *ChainConfigBuilder) GetNetworkUpgrades() map[string]interface{} {
	return b.networkUpgrades
}

// DefaultChainConfig returns the default SubnetEVM chain configuration
func DefaultChainConfig() *params.ChainConfig {
	return NewChainConfigBuilder().Build()
}

// MainnetChainConfig returns a chain configuration suitable for mainnet
func MainnetChainConfig(chainID *big.Int) *params.ChainConfig {
	return NewChainConfigBuilder().
		WithChainID(chainID).
		Build()
}

// TestnetChainConfig returns a chain configuration suitable for testnet
func TestnetChainConfig(chainID *big.Int) *params.ChainConfig {
	return NewChainConfigBuilder().
		WithChainID(chainID).
		Build()
}

// LocalChainConfig returns a chain configuration suitable for local development
func LocalChainConfig(chainID *big.Int) *params.ChainConfig {
	return NewChainConfigBuilder().
		WithChainID(chainID).
		Build()
}

// ChainConfigPresets provides preset configurations for common scenarios
var ChainConfigPresets = map[string]func(*big.Int) *params.ChainConfig{
	"mainnet": MainnetChainConfig,
	"testnet": TestnetChainConfig,
	"local":   LocalChainConfig,
}

// GetPresetChainConfig returns a preset chain configuration by name
func GetPresetChainConfig(preset string, chainID *big.Int) *params.ChainConfig {
	if configFunc, ok := ChainConfigPresets[preset]; ok {
		return configFunc(chainID)
	}
	return DefaultChainConfig()
}