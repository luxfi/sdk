// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package fees

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetFeeConfigForThroughput(t *testing.T) {
	tests := []struct {
		name              string
		throughput        string
		expectedGasLimit  *big.Int
		expectedTargetGas *big.Int
	}{
		{
			name:              "high throughput",
			throughput:        "high",
			expectedGasLimit:  big.NewInt(15_000_000),
			expectedTargetGas: big.NewInt(50_000_000),
		},
		{
			name:              "medium throughput",
			throughput:        "medium",
			expectedGasLimit:  big.NewInt(10_000_000),
			expectedTargetGas: big.NewInt(20_000_000),
		},
		{
			name:              "low throughput",
			throughput:        "low",
			expectedGasLimit:  big.NewInt(8_000_000),
			expectedTargetGas: big.NewInt(15_000_000),
		},
		{
			name:              "unknown throughput defaults to low",
			throughput:        "unknown",
			expectedGasLimit:  big.NewInt(8_000_000),
			expectedTargetGas: big.NewInt(15_000_000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := GetFeeConfigForThroughput(tt.throughput)
			require.Equal(t, tt.expectedGasLimit, config.GasLimit)
			require.Equal(t, tt.expectedTargetGas, config.TargetGas)
		})
	}
}

func TestFeeConfigBuilder(t *testing.T) {
	t.Run("default builder", func(t *testing.T) {
		builder := NewFeeConfigBuilder()
		config := builder.Build()

		// Should have default values
		require.Equal(t, DefaultFeeConfig.GasLimit, config.GasLimit)
		require.Equal(t, DefaultFeeConfig.MinBaseFee, config.MinBaseFee)
		require.Equal(t, DefaultFeeConfig.TargetGas, config.TargetGas)
	})

	t.Run("custom builder", func(t *testing.T) {
		customGasLimit := big.NewInt(20_000_000)
		customMinBaseFee := big.NewInt(30_000_000_000)
		customTargetGas := big.NewInt(25_000_000)
		customDenominator := big.NewInt(48)
		customMinBlockGas := big.NewInt(100)
		customMaxBlockGas := big.NewInt(2_000_000)
		customBlockRate := uint64(3)
		customGasStep := big.NewInt(300_000)

		config := NewFeeConfigBuilder().
			WithGasLimit(customGasLimit).
			WithMinBaseFee(customMinBaseFee).
			WithTargetGas(customTargetGas).
			WithBaseFeeChangeDenominator(customDenominator).
			WithMinBlockGasCost(customMinBlockGas).
			WithMaxBlockGasCost(customMaxBlockGas).
			WithTargetBlockRate(customBlockRate).
			WithBlockGasCostStep(customGasStep).
			Build()

		require.Equal(t, customGasLimit, config.GasLimit)
		require.Equal(t, customMinBaseFee, config.MinBaseFee)
		require.Equal(t, customTargetGas, config.TargetGas)
		require.Equal(t, customDenominator, config.BaseFeeChangeDenominator)
		require.Equal(t, customMinBlockGas, config.MinBlockGasCost)
		require.Equal(t, customMaxBlockGas, config.MaxBlockGasCost)
		require.Equal(t, customBlockRate, config.TargetBlockRate)
		require.Equal(t, customGasStep, config.BlockGasCostStep)
	})

	t.Run("partial builder", func(t *testing.T) {
		customGasLimit := big.NewInt(12_000_000)
		customTargetGas := big.NewInt(30_000_000)

		config := NewFeeConfigBuilder().
			WithGasLimit(customGasLimit).
			WithTargetGas(customTargetGas).
			Build()

		// Custom values should be set
		require.Equal(t, customGasLimit, config.GasLimit)
		require.Equal(t, customTargetGas, config.TargetGas)

		// Other values should remain default
		require.Equal(t, DefaultFeeConfig.MinBaseFee, config.MinBaseFee)
		require.Equal(t, DefaultFeeConfig.BaseFeeChangeDenominator, config.BaseFeeChangeDenominator)
	})
}

func TestPresetConfigurations(t *testing.T) {
	t.Run("low throughput config", func(t *testing.T) {
		require.Equal(t, big.NewInt(8_000_000), LowThroughputConfig.GasLimit)
		require.Equal(t, big.NewInt(15_000_000), LowThroughputConfig.TargetGas)
		require.Equal(t, uint64(2), LowThroughputConfig.TargetBlockRate)
	})

	t.Run("medium throughput config", func(t *testing.T) {
		require.Equal(t, big.NewInt(10_000_000), MediumThroughputConfig.GasLimit)
		require.Equal(t, big.NewInt(20_000_000), MediumThroughputConfig.TargetGas)
		require.Equal(t, uint64(2), MediumThroughputConfig.TargetBlockRate)
	})

	t.Run("high throughput config", func(t *testing.T) {
		require.Equal(t, big.NewInt(15_000_000), HighThroughputConfig.GasLimit)
		require.Equal(t, big.NewInt(50_000_000), HighThroughputConfig.TargetGas)
		require.Equal(t, uint64(2), HighThroughputConfig.TargetBlockRate)
	})

	t.Run("default config is low throughput", func(t *testing.T) {
		require.Equal(t, LowThroughputConfig, DefaultFeeConfig)
	})
}
