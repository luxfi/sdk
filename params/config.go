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
// Copyright 2016 The go-ethereum Authors
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
	"math/big"
	"time"

	ethparams "github.com/luxfi/geth/params"
	"github.com/luxfi/evm/params/extras"
	"github.com/luxfi/evm/utils"
)

// Guarantees extras initialisation before a call to [params.ChainConfig.Rules].
// Disabled: libevm integration removed
// var _ = gethInit()

// Local constants to replace upgrade package constants
var (
	// InitiallyActiveTime represents the Unix epoch (time 0)
	InitiallyActiveTime = time.Unix(0, 0)
)

var (
	// SubnetEVMDefaultConfig is the default configuration
	// without any network upgrades.
	SubnetEVMDefaultChainConfig = WithExtra(
		&ChainConfig{
			ChainID: DefaultChainID,

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
		},
		extras.SubnetEVMDefaultChainConfig,
	)

	TestChainConfig = WithExtra(
		&ChainConfig{
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
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
			// TODO: Once upgrade API is stable, restore network-specific upgrade times
			// For now, use InitiallyActiveTime for test networks
			ShanghaiTime:        utils.TimeToNewUint64(InitiallyActiveTime),
			CancunTime:          utils.TimeToNewUint64(InitiallyActiveTime),
			BlobScheduleConfig: &ethparams.BlobScheduleConfig{
				Cancun: &ethparams.BlobConfig{
					Target:         3,
					Max:            6,
					UpdateFraction: 3338477,
				},
			},
		},
		extras.TestChainConfig,
	)

	TestPreSubnetEVMChainConfig = WithExtra(
		&ChainConfig{
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
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
		},
		extras.TestPreSubnetEVMChainConfig,
	)

	TestSubnetEVMChainConfig = WithExtra(
		&ChainConfig{
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
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
		},
		extras.TestSubnetEVMChainConfig,
	)

	TestDurangoChainConfig = WithExtra(
		&ChainConfig{
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
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
			ShanghaiTime:        utils.TimeToNewUint64(InitiallyActiveTime),
		},
		extras.TestDurangoChainConfig,
	)

	TestEtnaChainConfig = WithExtra(
		&ChainConfig{
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
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
			ShanghaiTime:        utils.TimeToNewUint64(InitiallyActiveTime),
			CancunTime:          utils.TimeToNewUint64(InitiallyActiveTime),
			BlobScheduleConfig: &ethparams.BlobScheduleConfig{
				Cancun: &ethparams.BlobConfig{
					Target:         3,
					Max:            6,
					UpdateFraction: 3338477,
				},
			},
		},
		extras.TestEtnaChainConfig,
	)

	TestFortunaChainConfig = WithExtra(
		&ChainConfig{
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
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
			ShanghaiTime:        utils.TimeToNewUint64(InitiallyActiveTime),
			CancunTime:          utils.TimeToNewUint64(InitiallyActiveTime),
			BlobScheduleConfig: &ethparams.BlobScheduleConfig{
				Cancun: &ethparams.BlobConfig{
					Target:         3,
					Max:            6,
					UpdateFraction: 3338477,
				},
			},
		},
		extras.TestFortunaChainConfig,
	)

	TestGraniteChainConfig = WithExtra(
		&ChainConfig{
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
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
			ShanghaiTime:        utils.TimeToNewUint64(InitiallyActiveTime),
			CancunTime:          utils.TimeToNewUint64(InitiallyActiveTime),
			BlobScheduleConfig: &ethparams.BlobScheduleConfig{
				Cancun: &ethparams.BlobConfig{
					Target:         3,
					Max:            6,
					UpdateFraction: 3338477,
				},
			},
		},
		extras.TestGraniteChainConfig,
	)

	TestRules = TestChainConfig.Rules(new(big.Int), IsMergeTODO, 0)
)

// RulesAt returns the Rules for the given ChainConfig at the specified timestamp
// This is a helper that properly sets up the RulesExtra with precompile information
func RulesAt(c *ChainConfig, blockNum *big.Int, isMerge bool, timestamp uint64) Rules {
	rules := c.Rules(blockNum, isMerge, timestamp)
	// Store the context for GetRulesExtra to use
	SetRulesContext(&rules, c, timestamp)
	return rules
}

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig = ethparams.ChainConfig

// Rules wraps ChainConfig and is merely syntactic sugar or can be used for functions
// that do not have or require information about the block.
//
// Rules is a one time interface meaning that it shouldn't be used in between transition
// phases.
type Rules = ethparams.Rules

// ChainConfigJSON is a wrapper for ChainConfig that handles JSON marshaling/unmarshaling
// with extras fields. This is used when we need to unmarshal JSON that contains both
// standard ChainConfig fields and extras fields.
type ChainConfigJSON struct {
	*ChainConfig
}

// UnmarshalJSON unmarshals the JSON into the ChainConfig and handles extras fields
func (c *ChainConfigJSON) UnmarshalJSON(data []byte) error {
	// First unmarshal the standard ChainConfig fields
	c.ChainConfig = &ChainConfig{}
	if err := json.Unmarshal(data, c.ChainConfig); err != nil {
		return err
	}
	
	// Now unmarshal the extras fields
	extraFields := &extras.ChainConfig{}
	if err := json.Unmarshal(data, extraFields); err != nil {
		return err
	}
	
	// Set the extras using WithExtra
	WithExtra(c.ChainConfig, extraFields)
	return nil
}

// MarshalJSON marshals the ChainConfig with extras fields
func (c *ChainConfigJSON) MarshalJSON() ([]byte, error) {
	// Get the extras
	extra := GetExtra(c.ChainConfig)
	
	// Marshal the standard config
	configJSON, err := json.Marshal(c.ChainConfig)
	if err != nil {
		return nil, err
	}
	
	// Marshal the extras
	extraJSON, err := extra.MarshalJSON()
	if err != nil {
		return nil, err
	}
	
	// Merge the two JSON objects
	var configMap map[string]json.RawMessage
	if err := json.Unmarshal(configJSON, &configMap); err != nil {
		return nil, err
	}
	
	var extraMap map[string]json.RawMessage
	if err := json.Unmarshal(extraJSON, &extraMap); err != nil {
		return nil, err
	}
	
	// Merge extras into config
	for k, v := range extraMap {
		configMap[k] = v
	}
	
	return json.Marshal(configMap)
}
