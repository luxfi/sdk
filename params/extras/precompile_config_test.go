// (c) 2022 Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package extras_test

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/luxfi/evm/commontype"
	"github.com/luxfi/evm/params/extras"
	"github.com/luxfi/evm/precompile/contracts/deployerallowlist"
	"github.com/luxfi/evm/precompile/contracts/feemanager"
	"github.com/luxfi/evm/precompile/contracts/nativeminter"
	"github.com/luxfi/evm/precompile/contracts/rewardmanager"
	"github.com/luxfi/evm/precompile/contracts/txallowlist"
	"github.com/luxfi/evm/utils"
	"github.com/luxfi/geth/common"
	"github.com/stretchr/testify/require"
)

func TestVerifyWithChainConfig(t *testing.T) {
	admins := []common.Address{{1}}
	copy := *extras.TestChainConfig
	config := &copy
	config.GenesisPrecompiles = extras.Precompiles{
		txallowlist.ConfigKey: txallowlist.NewConfig(utils.NewUint64(2), nil, nil, nil),
	}
	config.PrecompileUpgrades = []extras.PrecompileUpgrade{
		{
			// disable TxAllowList at timestamp 4
			txallowlist.NewDisableConfig(utils.NewUint64(4)),
		},
		{
			// re-enable TxAllowList at timestamp 5
			txallowlist.NewConfig(utils.NewUint64(5), admins, nil, nil),
		},
	}

	// check this config is valid
	err := config.Verify()
	require.NoError(t, err)

	// same precompile cannot be configured twice for the same timestamp
	badConfig := *config
	badConfig.PrecompileUpgrades = append(
		badConfig.PrecompileUpgrades,
		extras.PrecompileUpgrade{
			Config: txallowlist.NewDisableConfig(utils.NewUint64(5)),
		},
	)
	err = badConfig.Verify()
	require.ErrorContains(t, err, "config block timestamp (5) <= previous timestamp (5) of same key")

	// cannot enable a precompile without disabling it first.
	badConfig = *config
	badConfig.PrecompileUpgrades = append(
		badConfig.PrecompileUpgrades,
		extras.PrecompileUpgrade{
			Config: txallowlist.NewConfig(utils.NewUint64(5), admins, nil, nil),
		},
	)
	err = badConfig.Verify()
	require.ErrorContains(t, err, "disable should be [true]")
}

func TestVerifyWithChainConfigAtNilTimestamp(t *testing.T) {
	admins := []common.Address{{0}}
	copy := *extras.TestChainConfig
	config := &copy
	config.PrecompileUpgrades = []extras.PrecompileUpgrade{
		// this does NOT enable the precompile, so it should be upgradeable.
		{Config: txallowlist.NewConfig(nil, nil, nil, nil)},
	}
	require.False(t, config.IsPrecompileEnabled(txallowlist.ContractAddress, 0)) // check the precompile is not enabled.
	config.PrecompileUpgrades = []extras.PrecompileUpgrade{
		{
			// enable TxAllowList at timestamp 5
			Config: txallowlist.NewConfig(utils.NewUint64(5), admins, nil, nil),
		},
	}

	// check this config is valid
	require.NoError(t, config.Verify())
}

func TestVerifyPrecompileUpgrades(t *testing.T) {
	admins := []common.Address{{1}}
	tests := []struct {
		name          string
		upgrades      []extras.PrecompileUpgrade
		expectedError string
	}{
		{
			name: "enable and disable tx allow list",
			upgrades: []extras.PrecompileUpgrade{
				{
					Config: txallowlist.NewConfig(utils.NewUint64(1), admins, nil, nil),
				},
				{
					Config: txallowlist.NewDisableConfig(utils.NewUint64(2)),
				},
			},
			expectedError: "",
		},
		{
			name: "invalid allow list config in tx allowlist",
			upgrades: []extras.PrecompileUpgrade{
				{
					Config: txallowlist.NewConfig(utils.NewUint64(1), admins, nil, nil),
				},
				{
					Config: txallowlist.NewDisableConfig(utils.NewUint64(2)),
				},
				{
					Config: txallowlist.NewConfig(utils.NewUint64(3), admins, admins, admins),
				},
			},
			expectedError: "cannot set address",
		},
		{
			name: "invalid initial fee manager config",
			upgrades: []extras.PrecompileUpgrade{
				{
					Config: feemanager.NewConfig(utils.NewUint64(3), admins, nil, nil,
						func() *commontype.FeeConfig {
							feeConfig := extras.DefaultFeeConfig
							feeConfig.GasLimit = big.NewInt(-1)
							return &feeConfig
						}()),
				},
			},
			expectedError: "gasLimit = -1 cannot be less than or equal to 0",
		},
		{
			name: "invalid initial fee manager config gas limit 0",
			upgrades: []extras.PrecompileUpgrade{
				{
					Config: feemanager.NewConfig(utils.NewUint64(3), admins, nil, nil,
						func() *commontype.FeeConfig {
							feeConfig := extras.DefaultFeeConfig
							feeConfig.GasLimit = common.Big0
							return &feeConfig
						}()),
				},
			},
			expectedError: "gasLimit = 0 cannot be less than or equal to 0",
		},
		{
			name: "different upgrades are allowed to configure same timestamp for different precompiles",
			upgrades: []extras.PrecompileUpgrade{
				{
					Config: txallowlist.NewConfig(utils.NewUint64(1), admins, nil, nil),
				},
				{
					Config: feemanager.NewConfig(utils.NewUint64(1), admins, nil, nil, nil),
				},
			},
			expectedError: "",
		},
		{
			name: "different upgrades must be monotonically increasing",
			upgrades: []extras.PrecompileUpgrade{
				{
					Config: txallowlist.NewConfig(utils.NewUint64(2), admins, nil, nil),
				},
				{
					Config: feemanager.NewConfig(utils.NewUint64(1), admins, nil, nil, nil),
				},
			},
			expectedError: "config block timestamp (1) < previous timestamp (2)",
		},
		{
			name: "upgrades with same keys are not allowed to configure same timestamp for same precompiles",
			upgrades: []extras.PrecompileUpgrade{
				{
					Config: txallowlist.NewConfig(utils.NewUint64(1), admins, nil, nil),
				},
				{
					Config: txallowlist.NewDisableConfig(utils.NewUint64(1)),
				},
			},
			expectedError: "config block timestamp (1) <= previous timestamp (1) of same key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			copy := *extras.TestChainConfig
			config := &copy
			config.PrecompileUpgrades = tt.upgrades

			err := config.Verify()
			if tt.expectedError == "" {
				require.NoError(err)
			} else {
				require.ErrorContains(err, tt.expectedError)
			}
		})
	}
}

func TestVerifyPrecompiles(t *testing.T) {
	admins := []common.Address{{1}}
	tests := []struct {
		name          string
		precompiles   extras.Precompiles
		expectedError string
	}{
		{
			name: "invalid allow list config in tx allowlist",
			precompiles: extras.Precompiles{
				txallowlist.ConfigKey: txallowlist.NewConfig(utils.NewUint64(3), admins, admins, admins),
			},
			expectedError: "cannot set address",
		},
		{
			name: "invalid initial fee manager config",
			precompiles: extras.Precompiles{
				feemanager.ConfigKey: feemanager.NewConfig(utils.NewUint64(3), admins, nil, nil,
					func() *commontype.FeeConfig {
						feeConfig := extras.DefaultFeeConfig
						feeConfig.GasLimit = big.NewInt(-1)
						return &feeConfig
					}()),
			},
			expectedError: "gasLimit = -1 cannot be less than or equal to 0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			copy := *extras.TestChainConfig
			config := &copy
			config.GenesisPrecompiles = tt.precompiles

			err := config.Verify()
			if tt.expectedError == "" {
				require.NoError(err)
			} else {
				require.ErrorContains(err, tt.expectedError)
			}
		})
	}
}

func TestVerifyRequiresSortedTimestamps(t *testing.T) {
	admins := []common.Address{{1}}
	config := &extras.ChainConfig{
		FeeConfig: extras.DefaultFeeConfig,
	}
	config.PrecompileUpgrades = []extras.PrecompileUpgrade{
		{
			Config: txallowlist.NewConfig(utils.NewUint64(2), admins, nil, nil),
		},
		{
			Config: deployerallowlist.NewConfig(utils.NewUint64(1), admins, nil, nil),
		},
	}

	// block timestamps must be monotonically increasing, so this config is invalid
	err := config.Verify()
	require.ErrorContains(t, err, "config block timestamp (1) < previous timestamp (2)")
}

func TestGetPrecompileConfig(t *testing.T) {
	require := require.New(t)
	config := &extras.ChainConfig{}
	config.GenesisPrecompiles = extras.Precompiles{
		deployerallowlist.ConfigKey: deployerallowlist.NewConfig(utils.NewUint64(10), nil, nil, nil),
	}

	deployerConfig := config.GetActivePrecompileConfig(deployerallowlist.ContractAddress, 0)
	require.Nil(deployerConfig)

	deployerConfig = config.GetActivePrecompileConfig(deployerallowlist.ContractAddress, 10)
	require.NotNil(deployerConfig)

	deployerConfig = config.GetActivePrecompileConfig(deployerallowlist.ContractAddress, 11)
	require.NotNil(deployerConfig)

	txAllowListConfig := config.GetActivePrecompileConfig(txallowlist.ContractAddress, 0)
	require.Nil(txAllowListConfig)
}

func TestPrecompileUpgradeUnmarshalJSON(t *testing.T) {
	require := require.New(t)

	upgradeBytes := []byte(`
			{
				"precompileUpgrades": [
					{
						"rewardManagerConfig": {
							"blockTimestamp": 1671542573,
							"adminAddresses": [
								"0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
							],
							"initialRewardConfig": {
								"allowFeeRecipients": true
							}
						}
					},
					{
						"contractNativeMinterConfig": {
							"blockTimestamp": 1671543172,
							"disable": false
						}
					}
				]
			}
	`)

	var upgradeConfig extras.UpgradeConfig
	err := json.Unmarshal(upgradeBytes, &upgradeConfig)
	require.NoError(err)

	require.Len(upgradeConfig.PrecompileUpgrades, 2)

	rewardManagerConf := upgradeConfig.PrecompileUpgrades[0]
	require.Equal(rewardManagerConf.Key(), rewardmanager.ConfigKey)
	testRewardManagerConfig := rewardmanager.NewConfig(
		utils.NewUint64(1671542573),
		[]common.Address{common.HexToAddress("0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC")},
		nil,
		nil,
		&rewardmanager.InitialRewardConfig{
			AllowFeeRecipients: true,
		})
	require.True(rewardManagerConf.Equal(testRewardManagerConfig))

	nativeMinterConfig := upgradeConfig.PrecompileUpgrades[1]
	require.Equal(nativeMinterConfig.Key(), nativeminter.ConfigKey)
	expectedNativeMinterConfig := nativeminter.NewConfig(utils.NewUint64(1671543172), nil, nil, nil, nil)
	require.True(nativeMinterConfig.Equal(expectedNativeMinterConfig))

	// Marshal and unmarshal again and check that the result is the same
	upgradeBytes2, err := json.Marshal(upgradeConfig)
	require.NoError(err)
	var upgradeConfig2 extras.UpgradeConfig
	err = json.Unmarshal(upgradeBytes2, &upgradeConfig2)
	require.NoError(err)
	require.Equal(upgradeConfig, upgradeConfig2)
}
