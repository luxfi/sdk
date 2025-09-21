// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
//
// This file is a derived work, based on the go-ethereum library whose original
// notices appear below.
//
// It is distributed under a license compatible with the licensing terms of the
// original code from which it is derived.
//
// Much love to the original authors for their work.
// **********
// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

import (
	"encoding/json"
	"math"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/luxfi/evm/params/extras"
	"github.com/luxfi/evm/precompile/contracts/nativeminter"
	"github.com/luxfi/evm/precompile/contracts/rewardmanager"
	"github.com/luxfi/evm/precompile/contracts/txallowlist"
	"github.com/luxfi/evm/utils"
	"github.com/luxfi/geth/common"
	ethparams "github.com/luxfi/geth/params"
	"github.com/stretchr/testify/require"
)

func TestCheckCompatible(t *testing.T) {
	// Test Lux-specific CheckCompatible functionality
	// This replaces the skipped geth test with Lux-specific logic

	// First test basic compatibility with TestChainConfig
	err := TestChainConfig.CheckCompatible(TestChainConfig, 0, 0)
	if err != nil {
		t.Errorf("TestChainConfig should be compatible with itself, got: %v", err)
	}

	type test struct {
		stored, new   *ChainConfig
		headBlock     uint64
		headTimestamp uint64
		wantErr       *ethparams.ConfigCompatError
	}
	tests := []test{
		{stored: TestChainConfig, new: TestChainConfig, headBlock: 0, headTimestamp: 0, wantErr: nil},
		{stored: TestChainConfig, new: TestChainConfig, headBlock: 0, headTimestamp: uint64(time.Now().Unix()), wantErr: nil},
		{stored: TestChainConfig, new: TestChainConfig, headBlock: 100, wantErr: nil},
		{
			stored:        &ChainConfig{EIP150Block: big.NewInt(10)},
			new:           &ChainConfig{EIP150Block: big.NewInt(20)},
			headBlock:     9,
			headTimestamp: 90,
			wantErr:       nil,
		},
		{
			stored:        TestChainConfig,
			new:           &ChainConfig{HomesteadBlock: nil},
			headBlock:     3,
			headTimestamp: 30,
			wantErr: &ethparams.ConfigCompatError{
				What:          "Homestead fork block",
				StoredBlock:   big.NewInt(0),
				NewBlock:      nil,
				RewindToBlock: 0,
			},
		},
		{
			stored:        TestChainConfig,
			new:           &ChainConfig{HomesteadBlock: big.NewInt(1)},
			headBlock:     3,
			headTimestamp: 30,
			wantErr: &ethparams.ConfigCompatError{
				What:          "Homestead fork block",
				StoredBlock:   big.NewInt(0),
				NewBlock:      big.NewInt(1),
				RewindToBlock: 0,
			},
		},
		{
			stored:        &ChainConfig{HomesteadBlock: big.NewInt(30), EIP150Block: big.NewInt(10)},
			new:           &ChainConfig{HomesteadBlock: big.NewInt(25), EIP150Block: big.NewInt(20)},
			headBlock:     25,
			headTimestamp: 250,
			wantErr: &ethparams.ConfigCompatError{
				What:          "EIP150 fork block",
				StoredBlock:   big.NewInt(10),
				NewBlock:      big.NewInt(20),
				RewindToBlock: 9,
			},
		},
		{
			stored:        &ChainConfig{ConstantinopleBlock: big.NewInt(30)},
			new:           &ChainConfig{ConstantinopleBlock: big.NewInt(30), PetersburgBlock: big.NewInt(30)},
			headBlock:     40,
			headTimestamp: 400,
			wantErr:       nil,
		},
		{
			stored:        &ChainConfig{ConstantinopleBlock: big.NewInt(30)},
			new:           &ChainConfig{ConstantinopleBlock: big.NewInt(30), PetersburgBlock: big.NewInt(31)},
			headBlock:     40,
			headTimestamp: 400,
			wantErr: &ethparams.ConfigCompatError{
				What:          "Petersburg fork block",
				StoredBlock:   nil,
				NewBlock:      big.NewInt(31),
				RewindToBlock: 30,
			},
		},
		{
			stored:        TestChainConfig,
			new:           TestPreSubnetEVMChainConfig,
			headBlock:     0,
			headTimestamp: 0,
			wantErr: &ethparams.ConfigCompatError{
				What:         "SubnetEVM fork block timestamp",
				StoredTime:   utils.NewUint64(0),
				NewTime:      GetExtra(TestPreSubnetEVMChainConfig).SubnetEVMTimestamp,
				RewindToTime: 0,
			},
		},
		{
			stored:        TestChainConfig,
			new:           TestPreSubnetEVMChainConfig,
			headBlock:     10,
			headTimestamp: 100,
			wantErr: &ethparams.ConfigCompatError{
				What:         "SubnetEVM fork block timestamp",
				StoredTime:   utils.NewUint64(0),
				NewTime:      GetExtra(TestPreSubnetEVMChainConfig).SubnetEVMTimestamp,
				RewindToTime: 0,
			},
		},
	}

	for _, test := range tests {
		err := test.stored.CheckCompatible(test.new, test.headBlock, test.headTimestamp)
		if !reflect.DeepEqual(err, test.wantErr) {
			t.Errorf("error mismatch:\nstored: %v\nnew: %v\nblockHeight: %v\nerr: %v\nwant: %v", test.stored, test.new, test.headBlock, err, test.wantErr)
		}
	}
}

func TestConfigRules(t *testing.T) {
	c := WithExtra(
		&ChainConfig{},
		&extras.ChainConfig{
			NetworkUpgrades: extras.NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(500),
			},
		},
	)

	// Get the extras configuration
	extra := GetExtra(c)

	var stamp uint64
	// Test that SubnetEVM is not active at timestamp 0
	if extra.IsSubnetEVM(stamp) {
		t.Errorf("expected timestamp %v to not be SubnetEVM", stamp)
	}

	stamp = 500
	// Test that SubnetEVM is active at timestamp 500
	if !extra.IsSubnetEVM(stamp) {
		t.Errorf("expected timestamp %v to be SubnetEVM", stamp)
	}

	stamp = math.MaxInt64
	// Test that SubnetEVM is active at max timestamp
	if !extra.IsSubnetEVM(stamp) {
		t.Errorf("expected timestamp %v to be SubnetEVM", stamp)
	}
}

func TestConfigUnmarshalJSON(t *testing.T) {
	require := require.New(t)

	testRewardManagerConfig := rewardmanager.NewConfig(
		utils.NewUint64(1671542573),
		[]common.Address{common.HexToAddress("0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC")},
		nil,
		nil,
		&rewardmanager.InitialRewardConfig{
			AllowFeeRecipients: true,
		})

	testContractNativeMinterConfig := nativeminter.NewConfig(
		utils.NewUint64(0),
		[]common.Address{common.HexToAddress("0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC")},
		nil,
		nil,
		nil,
	)

	config := []byte(`
	{
		"chainId": 43214,
		"allowFeeRecipients": true,
		"rewardManagerConfig": {
			"blockTimestamp": 1671542573,
			"adminAddresses": [
				"0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
			],
			"initialRewardConfig": {
				"allowFeeRecipients": true
			}
		},
		"contractNativeMinterConfig": {
			"blockTimestamp": 0,
			"adminAddresses": [
				"0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
			]
		}
	}
	`)
	cJSON := ChainConfigJSON{}
	err := json.Unmarshal(config, &cJSON)
	require.NoError(err)
	c := cJSON.ChainConfig // Use the pointer directly

	require.Equal(big.NewInt(43214), c.ChainID)
	require.Equal(true, GetExtra(c).AllowFeeRecipients)

	rewardManagerConfig, ok := GetExtra(c).GenesisPrecompiles[rewardmanager.ConfigKey]
	require.True(ok)
	require.Equal(rewardManagerConfig.Key(), rewardmanager.ConfigKey)
	require.True(rewardManagerConfig.Equal(testRewardManagerConfig))

	nativeMinterConfig := GetExtra(c).GenesisPrecompiles[nativeminter.ConfigKey]
	require.Equal(nativeMinterConfig.Key(), nativeminter.ConfigKey)
	require.True(nativeMinterConfig.Equal(testContractNativeMinterConfig))

	// Marshal and unmarshal again and check that the result is the same
	cJSONMarshal := ChainConfigJSON{ChainConfig: c}
	marshaled, err := json.Marshal(&cJSONMarshal)
	require.NoError(err)
	c2JSON := ChainConfigJSON{}
	err = json.Unmarshal(marshaled, &c2JSON)
	require.NoError(err)
	c2 := c2JSON.ChainConfig
	require.Equal(*c, *c2)
	// Also check that extras are preserved
	require.Equal(GetExtra(c).AllowFeeRecipients, GetExtra(c2).AllowFeeRecipients)
}

func TestActivePrecompiles(t *testing.T) {
	config := WithExtra(
		&ChainConfig{},
		&extras.ChainConfig{
			UpgradeConfig: extras.UpgradeConfig{
				PrecompileUpgrades: []extras.PrecompileUpgrade{
					{
						Config: nativeminter.NewConfig(utils.NewUint64(0), nil, nil, nil, nil), // enable at genesis
					},
					{
						Config: nativeminter.NewDisableConfig(utils.NewUint64(1)), // disable at timestamp 1
					},
				},
			},
		},
	)

	// Create rules for timestamp 0 and 1
	// Rules(blockNum *big.Int, isMerge bool, timestamp uint64)
	rules0 := config.Rules(&big.Int{}, false, 0)
	rules1 := config.Rules(&big.Int{}, false, 1)

	require.True(t, GetRulesExtra(rules0).IsPrecompileEnabled(nativeminter.Module.Address))

	require.False(t, GetRulesExtra(rules1).IsPrecompileEnabled(nativeminter.Module.Address))
}

func TestExtrasMarshaling(t *testing.T) {
	// Test that extras.ChainConfig marshals correctly
	extra := &extras.ChainConfig{
		FeeConfig:          DefaultFeeConfig,
		AllowFeeRecipients: false,
		NetworkUpgrades: extras.NetworkUpgrades{
			SubnetEVMTimestamp: utils.NewUint64(0),
			DurangoTimestamp:   utils.NewUint64(0),
		},
		GenesisPrecompiles: extras.Precompiles{},
	}

	result, err := json.Marshal(extra)
	require.NoError(t, err)
	t.Logf("Extras marshaled: %s", string(result))

	// Check that it contains expected fields
	var decoded map[string]interface{}
	err = json.Unmarshal(result, &decoded)
	require.NoError(t, err)

	require.Contains(t, decoded, "feeConfig")
	require.Contains(t, decoded, "subnetEVMTimestamp")
	require.Contains(t, decoded, "durangoTimestamp")

	feeConfig := decoded["feeConfig"].(map[string]interface{})
	require.Equal(t, float64(8000000), feeConfig["gasLimit"])
}

func TestChainConfigMarshalWithUpgrades(t *testing.T) {
	// Create ChainConfig with extras
	chainConfig := &ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
	}

	extraConfig := &extras.ChainConfig{
		FeeConfig:          DefaultFeeConfig,
		AllowFeeRecipients: false,
		NetworkUpgrades: extras.NetworkUpgrades{
			SubnetEVMTimestamp: utils.NewUint64(0),
			DurangoTimestamp:   utils.NewUint64(0),
		},
		GenesisPrecompiles: extras.Precompiles{},
	}

	// Set the extras
	WithExtra(chainConfig, extraConfig)

	config := ChainConfigWithUpgradesJSON{
		ChainConfig: *chainConfig,
		UpgradeConfig: extras.UpgradeConfig{
			PrecompileUpgrades: []extras.PrecompileUpgrade{
				{
					Config: txallowlist.NewConfig(utils.NewUint64(100), nil, nil, nil),
				},
			},
		},
	}

	// Need to re-set the extras for the copied ChainConfig
	WithExtra(&config.ChainConfig, extraConfig)

	result, err := json.Marshal(&config)
	require.NoError(t, err)
	expectedJSON := `{
		"chainId": 1,
		"feeConfig": {
			"gasLimit": 8000000,
			"targetBlockRate": 2,
			"minBaseFee": 25000000000,
			"targetGas": 15000000,
			"baseFeeChangeDenominator": 36,
			"minBlockGasCost": 0,
			"maxBlockGasCost": 1000000,
			"blockGasCostStep": 200000
		},
		"homesteadBlock": 0,
		"eip150Block": 0,
		"eip155Block": 0,
		"eip158Block": 0,
		"byzantiumBlock": 0,
		"constantinopleBlock": 0,
		"petersburgBlock": 0,
		"istanbulBlock": 0,
		"muirGlacierBlock": 0,
		"depositContractAddress": "0x0000000000000000000000000000000000000000",
		"subnetEVMTimestamp": 0,
		"durangoTimestamp": 0,
		"upgrades": {
			"precompileUpgrades": [
				{
					"txAllowListConfig": {
						"blockTimestamp": 100
					}
				}
			]
		}
	}`
	require.JSONEq(t, expectedJSON, string(result))

	var unmarshalled ChainConfigWithUpgradesJSON
	err = json.Unmarshal(result, &unmarshalled)
	require.NoError(t, err)
	require.Equal(t, config, unmarshalled)
}
