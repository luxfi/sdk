// (c) 2025 Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package extras_test

import (
	"math/big"
	"testing"

	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/params/extras"
	"github.com/ethereum/go-ethereum/common"
	"github.com/luxfi/evm/commontype"
	"github.com/luxfi/evm/precompile/contracts/txallowlist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pointer[T any](v T) *T { return &v }

func TestChainConfigDescription(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config    *extras.ChainConfig
		wantRegex string
	}{
		"nil": {},
		"empty": {
			config: &extras.ChainConfig{},
			wantRegex: `Lux Upgrades \(timestamp based\)\:
 - EVM Timestamp: ( )+@nil( )+\(https:\/\/github\.com\/luxfi\/luxd\/releases\/tag\/v1\.10\.0\)
( - .+Timestamp: .+\n)+
Upgrade Config: {}
Fee Config: {}
Allow Fee Recipients: false
$`,
		},
		"set": {
			config: &extras.ChainConfig{
				NetworkUpgrades: extras.NetworkUpgrades{
					EVMTimestamp: pointer(uint64(1)),
					DurangoTimestamp:   pointer(uint64(2)),
					EtnaTimestamp:      pointer(uint64(3)),
					FortunaTimestamp:   pointer(uint64(4)),
				},
				FeeConfig: commontype.FeeConfig{
					GasLimit:                 big.NewInt(5),
					TargetBlockRate:          6,
					MinBaseFee:               big.NewInt(7),
					TargetGas:                big.NewInt(8),
					BaseFeeChangeDenominator: big.NewInt(9),
					MinBlockGasCost:          big.NewInt(10),
					MaxBlockGasCost:          big.NewInt(11),
					BlockGasCostStep:         big.NewInt(12),
				},
				AllowFeeRecipients: true,
				UpgradeConfig: extras.UpgradeConfig{
					NetworkUpgradeOverrides: &extras.NetworkUpgrades{
						EVMTimestamp: pointer(uint64(13)),
					},
					StateUpgrades: []extras.StateUpgrade{
						{
							BlockTimestamp: pointer(uint64(14)),
							StateUpgradeAccounts: map[common.Address]extras.StateUpgradeAccount{
								common.Address{15}: {
									Code: []byte{16},
								},
							},
						},
					},
				},
			},
			wantRegex: `Lux Upgrades \(timestamp based\)\:
 - EVM Timestamp: ( )+@1( )+\(https:\/\/github\.com\/luxfi\/luxd\/releases\/tag\/v1\.10\.0\)
( - .+Timestamp: .+\n)+
Upgrade Config: {"networkUpgradeOverrides":{"evmTimestamp":13},"stateUpgrades":\[{"blockTimestamp":14,"accounts":{"0x0f00000000000000000000000000000000000000":{"code":"0x10"}}}\]}
Fee Config: {"gasLimit":5,"targetBlockRate":6,"minBaseFee":7,"targetGas":8,"baseFeeChangeDenominator":9,"minBlockGasCost":10,"maxBlockGasCost":11,"blockGasCostStep":12}
Allow Fee Recipients: true
$`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.config.Description()
			assert.Regexp(t, test.wantRegex, got, "config description mismatch")
		})
	}
}

func TestChainConfigVerify(t *testing.T) {
	t.Parallel()

	validFeeConfig := commontype.FeeConfig{
		GasLimit:                 big.NewInt(1),
		TargetBlockRate:          1,
		MinBaseFee:               big.NewInt(1),
		TargetGas:                big.NewInt(1),
		BaseFeeChangeDenominator: big.NewInt(1),
		MinBlockGasCost:          big.NewInt(1),
		MaxBlockGasCost:          big.NewInt(1),
		BlockGasCostStep:         big.NewInt(1),
	}

	tests := map[string]struct {
		config   extras.ChainConfig
		errRegex string
	}{
		"invalid_feeconfig": {
			config: extras.ChainConfig{
				FeeConfig: commontype.FeeConfig{
					GasLimit: nil,
				},
			},
			errRegex: "^invalid fee config: ",
		},
		"invalid_precompile_upgrades": {
			// Also see precompile_config_test.go TestVerifyWithChainConfig* tests
			config: extras.ChainConfig{
				FeeConfig: validFeeConfig,
				UpgradeConfig: extras.UpgradeConfig{
					PrecompileUpgrades: []extras.PrecompileUpgrade{
						// same precompile cannot be configured twice for the same timestamp
						{Config: txallowlist.NewDisableConfig(pointer(uint64(1)))},
						{Config: txallowlist.NewDisableConfig(pointer(uint64(1)))},
					},
				},
			},
			errRegex: "^invalid precompile upgrades: ",
		},
		"invalid_state_upgrades": {
			config: extras.ChainConfig{
				FeeConfig: validFeeConfig,
				UpgradeConfig: extras.UpgradeConfig{
					StateUpgrades: []extras.StateUpgrade{
						{BlockTimestamp: nil},
					},
				},
			},
			errRegex: "^invalid state upgrades: ",
		},
		"invalid_network_upgrades": {
			config: extras.ChainConfig{
				FeeConfig: validFeeConfig,
				NetworkUpgrades: extras.NetworkUpgrades{
					EVMTimestamp: nil,
				},
				LuxContext: extras.LuxContext{ConsensusCtx: &interfaces.ChainContext{}},
			},
			errRegex: "^invalid network upgrades: ",
		},
		"valid": {
			config: extras.ChainConfig{
				FeeConfig: validFeeConfig,
				NetworkUpgrades: extras.NetworkUpgrades{
					EVMTimestamp: pointer(uint64(1)),
					DurangoTimestamp:   pointer(uint64(2)),
					EtnaTimestamp:      pointer(uint64(3)),
					FortunaTimestamp:   pointer(uint64(4)),
				},
				LuxContext: extras.LuxContext{ConsensusCtx: &interfaces.ChainContext{
					NetworkID: 1,
					ChainID: interfaces.ChainID{},
					NodeID: interfaces.NodeID{},
				}},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := test.config.Verify()
			if test.errRegex == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Regexp(t, test.errRegex, err.Error())
			}
		})
	}
}
