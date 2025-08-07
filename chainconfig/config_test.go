// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chainconfig

import (
	"math/big"
	"testing"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/sdk/fees"
	"github.com/stretchr/testify/require"
)

func TestChainConfigBuilder(t *testing.T) {
	t.Run("default builder", func(t *testing.T) {
		builder := NewChainConfigBuilder()
		config := builder.Build()

		require.NotNil(t, config)
		require.NotNil(t, config.ChainID)
		require.Equal(t, big.NewInt(99999), config.ChainID)
		require.Nil(t, builder.GetFeeConfig())
		require.False(t, builder.GetAllowFeeRecipients())
	})

	t.Run("custom chain ID", func(t *testing.T) {
		chainID := big.NewInt(12345)
		config := NewChainConfigBuilder().
			WithChainID(chainID).
			Build()

		require.Equal(t, chainID, config.ChainID)
	})

	t.Run("with fee config", func(t *testing.T) {
		builder := NewChainConfigBuilder().
			WithFeeConfig(fees.HighThroughputConfig)
		builder.Build()

		require.Equal(t, fees.HighThroughputConfig, builder.GetFeeConfig())
	})

	t.Run("with allow fee recipients", func(t *testing.T) {
		builder := NewChainConfigBuilder().
			WithAllowFeeRecipients(true)
		builder.Build()

		require.True(t, builder.GetAllowFeeRecipients())
	})

	t.Run("with precompile", func(t *testing.T) {
		address := common.HexToAddress("0x0100000000000000000000000000000000000000")
		precompileConfig := map[string]interface{}{
			"enabled": true,
		}

		builder := NewChainConfigBuilder().
			WithPrecompile(address, precompileConfig)
		builder.Build()

		precompiles := builder.GetPrecompiles()
		require.NotNil(t, precompiles)
		require.Contains(t, precompiles, address.Hex())
		require.Equal(t, precompileConfig, precompiles[address.Hex()])
	})

	t.Run("with network upgrade", func(t *testing.T) {
		upgradeName := "testUpgrade"
		upgradeTime := big.NewInt(1000000)

		builder := NewChainConfigBuilder().
			WithNetworkUpgrade(upgradeName, upgradeTime)
		builder.Build()

		upgrades := builder.GetNetworkUpgrades()
		require.NotNil(t, upgrades)
		require.Contains(t, upgrades, upgradeName)
		require.Equal(t, upgradeTime, upgrades[upgradeName])
	})
}

func TestPresetConfigurations(t *testing.T) {
	chainID := big.NewInt(43114)

	t.Run("mainnet config", func(t *testing.T) {
		config := MainnetChainConfig(chainID)

		require.Equal(t, chainID, config.ChainID)
	})

	t.Run("testnet config", func(t *testing.T) {
		config := TestnetChainConfig(chainID)

		require.Equal(t, chainID, config.ChainID)
	})

	t.Run("local config", func(t *testing.T) {
		config := LocalChainConfig(chainID)

		require.Equal(t, chainID, config.ChainID)
	})
}

func TestGetPresetChainConfig(t *testing.T) {
	chainID := big.NewInt(99999)

	t.Run("mainnet preset", func(t *testing.T) {
		config := GetPresetChainConfig("mainnet", chainID)
		require.Equal(t, chainID, config.ChainID)
	})

	t.Run("testnet preset", func(t *testing.T) {
		config := GetPresetChainConfig("testnet", chainID)
		require.Equal(t, chainID, config.ChainID)
	})

	t.Run("local preset", func(t *testing.T) {
		config := GetPresetChainConfig("local", chainID)
		require.Equal(t, chainID, config.ChainID)
	})

	t.Run("unknown preset returns default", func(t *testing.T) {
		config := GetPresetChainConfig("unknown", chainID)
		require.Equal(t, big.NewInt(99999), config.ChainID) // Default chain ID
	})
}
