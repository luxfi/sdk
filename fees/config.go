// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package fees

import (
	"math/big"

	"github.com/luxfi/evm/commontype"
)

// Preset fee configurations for different network usage scenarios
var (
	// LowThroughputConfig is optimized for low disk usage and throughput (1.5M gas/s)
	// This is the default C-Chain configuration
	LowThroughputConfig = commontype.FeeConfig{
		GasLimit:                 big.NewInt(8_000_000),
		MinBaseFee:               big.NewInt(25_000_000_000),
		TargetGas:                big.NewInt(15_000_000),
		BaseFeeChangeDenominator: big.NewInt(36),
		MinBlockGasCost:          big.NewInt(0),
		MaxBlockGasCost:          big.NewInt(1_000_000),
		TargetBlockRate:          2,
		BlockGasCostStep:         big.NewInt(200_000),
	}

	// MediumThroughputConfig is optimized for medium disk usage and throughput (2M gas/s)
	MediumThroughputConfig = commontype.FeeConfig{
		GasLimit:                 big.NewInt(10_000_000),
		MinBaseFee:               big.NewInt(25_000_000_000),
		TargetGas:                big.NewInt(20_000_000),
		BaseFeeChangeDenominator: big.NewInt(36),
		MinBlockGasCost:          big.NewInt(0),
		MaxBlockGasCost:          big.NewInt(1_000_000),
		TargetBlockRate:          2,
		BlockGasCostStep:         big.NewInt(200_000),
	}

	// HighThroughputConfig is optimized for high disk usage and throughput (5M gas/s)
	HighThroughputConfig = commontype.FeeConfig{
		GasLimit:                 big.NewInt(15_000_000),
		MinBaseFee:               big.NewInt(25_000_000_000),
		TargetGas:                big.NewInt(50_000_000),
		BaseFeeChangeDenominator: big.NewInt(36),
		MinBlockGasCost:          big.NewInt(0),
		MaxBlockGasCost:          big.NewInt(1_000_000),
		TargetBlockRate:          2,
		BlockGasCostStep:         big.NewInt(200_000),
	}

	// DefaultFeeConfig is the standard fee configuration
	DefaultFeeConfig = LowThroughputConfig
)

// FeeConfigBuilder helps construct custom fee configurations
type FeeConfigBuilder struct {
	config commontype.FeeConfig
}

// NewFeeConfigBuilder creates a new fee config builder with default values
func NewFeeConfigBuilder() *FeeConfigBuilder {
	return &FeeConfigBuilder{
		config: DefaultFeeConfig,
	}
}

// WithGasLimit sets the gas limit
func (b *FeeConfigBuilder) WithGasLimit(gasLimit *big.Int) *FeeConfigBuilder {
	b.config.GasLimit = gasLimit
	return b
}

// WithMinBaseFee sets the minimum base fee
func (b *FeeConfigBuilder) WithMinBaseFee(minBaseFee *big.Int) *FeeConfigBuilder {
	b.config.MinBaseFee = minBaseFee
	return b
}

// WithTargetGas sets the target gas
func (b *FeeConfigBuilder) WithTargetGas(targetGas *big.Int) *FeeConfigBuilder {
	b.config.TargetGas = targetGas
	return b
}

// WithBaseFeeChangeDenominator sets the base fee change denominator
func (b *FeeConfigBuilder) WithBaseFeeChangeDenominator(denominator *big.Int) *FeeConfigBuilder {
	b.config.BaseFeeChangeDenominator = denominator
	return b
}

// WithMinBlockGasCost sets the minimum block gas cost
func (b *FeeConfigBuilder) WithMinBlockGasCost(minBlockGasCost *big.Int) *FeeConfigBuilder {
	b.config.MinBlockGasCost = minBlockGasCost
	return b
}

// WithMaxBlockGasCost sets the maximum block gas cost
func (b *FeeConfigBuilder) WithMaxBlockGasCost(maxBlockGasCost *big.Int) *FeeConfigBuilder {
	b.config.MaxBlockGasCost = maxBlockGasCost
	return b
}

// WithTargetBlockRate sets the target block rate
func (b *FeeConfigBuilder) WithTargetBlockRate(targetBlockRate uint64) *FeeConfigBuilder {
	b.config.TargetBlockRate = targetBlockRate
	return b
}

// WithBlockGasCostStep sets the block gas cost step
func (b *FeeConfigBuilder) WithBlockGasCostStep(blockGasCostStep *big.Int) *FeeConfigBuilder {
	b.config.BlockGasCostStep = blockGasCostStep
	return b
}

// Build returns the constructed fee configuration
func (b *FeeConfigBuilder) Build() commontype.FeeConfig {
	return b.config
}

// GetFeeConfigForThroughput returns a fee configuration for the specified throughput level
func GetFeeConfigForThroughput(throughput string) commontype.FeeConfig {
	switch throughput {
	case "high":
		return HighThroughputConfig
	case "medium":
		return MediumThroughputConfig
	case "low":
		return LowThroughputConfig
	default:
		return DefaultFeeConfig
	}
}
