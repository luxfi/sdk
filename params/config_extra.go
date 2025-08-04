// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package params

import (
	"encoding/json"
	"errors"
	"math/big"
	"sync"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/node/upgrade"
	"github.com/luxfi/evm/params/extras"
	"github.com/luxfi/evm/utils"
)

const (
	maxJSONLen = 64 * 1024 * 1024 // 64MB

	// TODO: Value to pass to geth's Rules by default where the appropriate
	// context is not available in the lux code. (similar to context.TODO())
	IsMergeTODO = true
)

var (
	DefaultChainID   = big.NewInt(43214)
	DefaultFeeConfig = extras.DefaultFeeConfig

	// Use the constants from extras package
	initiallyActive       = uint64(extras.InitiallyActiveTime.Unix())
	unscheduledActivation = uint64(extras.UnscheduledActivationTime.Unix())

	// Simple map-based replacement for libevm payloads system
	chainConfigExtras = make(map[*ChainConfig]*extras.ChainConfig)
	chainConfigMutex  sync.RWMutex
)

// RulesExtra represents extra EVM rules - part of libevm integration
type RulesExtra struct {
	IsSubnetEVM bool
	// TODO: Add other fields as needed
}

// IsPrecompileEnabled checks if a precompile is enabled
func (r RulesExtra) IsPrecompileEnabled(addr common.Address) bool {
	// TODO: Implement proper precompile checking
	return false
}

// SetEthUpgrades enables Ethereum network upgrades using the same time as
// the Lux network upgrade that enables them.
//
// TODO: Prior to Cancun, Lux upgrades are referenced inline in the
// code in place of their Ethereum counterparts. The original Ethereum names
// should be restored for maintainability.
func SetEthUpgrades(c *ChainConfig) error {
	if c.HomesteadBlock == nil {
		c.HomesteadBlock = big.NewInt(0)
	}
	if c.EIP150Block == nil {
		c.EIP150Block = big.NewInt(0)
	}
	if c.EIP155Block == nil {
		c.EIP155Block = big.NewInt(0)
	}
	if c.EIP158Block == nil {
		c.EIP158Block = big.NewInt(0)
	}
	if c.ByzantiumBlock == nil {
		c.ByzantiumBlock = big.NewInt(0)
	}
	if c.ConstantinopleBlock == nil {
		c.ConstantinopleBlock = big.NewInt(0)
	}
	if c.PetersburgBlock == nil {
		c.PetersburgBlock = big.NewInt(0)
	}
	if c.IstanbulBlock == nil {
		c.IstanbulBlock = big.NewInt(0)
	}
	if c.MuirGlacierBlock == nil {
		c.MuirGlacierBlock = big.NewInt(0)
	}
	if c.BerlinBlock == nil {
		c.BerlinBlock = big.NewInt(0)
	}
	if c.LondonBlock == nil {
		c.LondonBlock = big.NewInt(0)
	}

	extra := GetExtra(c)
	// We only mark Eth upgrades as enabled if we have marked them as scheduled.
	if durango := extra.DurangoTimestamp; durango != nil && *durango < unscheduledActivation {
		c.ShanghaiTime = utils.NewUint64(*durango)
	}

	if etna := extra.EtnaTimestamp; etna != nil && *etna < unscheduledActivation {
		c.CancunTime = utils.NewUint64(*etna)
	}
	return nil
}

func GetExtra(c *ChainConfig) *extras.ChainConfig {
	chainConfigMutex.RLock()
	ex, ok := chainConfigExtras[c]
	chainConfigMutex.RUnlock()
	
	if !ok || ex == nil {
		chainConfigMutex.Lock()
		// Double-check after acquiring write lock
		ex, ok = chainConfigExtras[c]
		if !ok || ex == nil {
			ex = &extras.ChainConfig{}
			chainConfigExtras[c] = ex
		}
		chainConfigMutex.Unlock()
	}
	return ex
}

func Copy(c *ChainConfig) ChainConfig {
	cpy := *c
	extraCpy := *GetExtra(c)
	return *WithExtra(&cpy, &extraCpy)
}

// WithExtra sets the extra payload on `c` and returns the modified argument.
func WithExtra(c *ChainConfig, extra *extras.ChainConfig) *ChainConfig {
	chainConfigMutex.Lock()
	chainConfigExtras[c] = extra
	chainConfigMutex.Unlock()
	return c
}

type ChainConfigWithUpgradesJSON struct {
	ChainConfig
	UpgradeConfig extras.UpgradeConfig `json:"upgrades,omitempty"`
}

// MarshalJSON implements json.Marshaler. This is a workaround for the fact that
// the embedded ChainConfig struct has a MarshalJSON method, which prevents
// the default JSON marshalling from working for UpgradeConfig.
// TODO: consider removing this method by allowing external tag for the embedded
// ChainConfig struct.
func (cu ChainConfigWithUpgradesJSON) MarshalJSON() ([]byte, error) {
	// embed the ChainConfig struct into the response
	chainConfigJSON, err := json.Marshal(&cu.ChainConfig)
	if err != nil {
		return nil, err
	}
	if len(chainConfigJSON) > maxJSONLen {
		return nil, errors.New("value too large")
	}

	type upgrades struct {
		UpgradeConfig extras.UpgradeConfig `json:"upgrades"`
	}

	upgradeJSON, err := json.Marshal(upgrades{cu.UpgradeConfig})
	if err != nil {
		return nil, err
	}
	if len(upgradeJSON) > maxJSONLen {
		return nil, errors.New("value too large")
	}

	// merge the two JSON objects
	mergedJSON := make([]byte, 0, len(chainConfigJSON)+len(upgradeJSON)+1)
	mergedJSON = append(mergedJSON, chainConfigJSON[:len(chainConfigJSON)-1]...)
	mergedJSON = append(mergedJSON, ',')
	mergedJSON = append(mergedJSON, upgradeJSON[1:]...)
	return mergedJSON, nil
}

func (cu *ChainConfigWithUpgradesJSON) UnmarshalJSON(input []byte) error {
	var cc ChainConfig
	if err := json.Unmarshal(input, &cc); err != nil {
		return err
	}

	type upgrades struct {
		UpgradeConfig extras.UpgradeConfig `json:"upgrades"`
	}

	var u upgrades
	if err := json.Unmarshal(input, &u); err != nil {
		return err
	}
	cu.ChainConfig = cc
	cu.UpgradeConfig = u.UpgradeConfig
	return nil
}

// ToWithUpgradesJSON converts the ChainConfig to ChainConfigWithUpgradesJSON with upgrades explicitly displayed.
// ChainConfig does not include upgrades in its JSON output.
// This is a workaround for showing upgrades in the JSON output.
func ToWithUpgradesJSON(c *ChainConfig) *ChainConfigWithUpgradesJSON {
	return &ChainConfigWithUpgradesJSON{
		ChainConfig:   *c,
		UpgradeConfig: GetExtra(c).UpgradeConfig,
	}
}

func SetNetworkUpgradeDefaults(c *ChainConfig) {
	// TODO: NetworkUpgrades field not available in current consensus.Context
	// GetExtra(c).NetworkUpgrades.SetDefaults(GetExtra(c).SnowCtx.NetworkUpgrades)
	// For now, set empty defaults with empty upgrade config
	emptyUpgradeConfig := upgrade.Config{}
	GetExtra(c).NetworkUpgrades.SetDefaults(emptyUpgradeConfig)
}

// GetRulesExtra stub - was part of libevm integration
func GetRulesExtra(rules Rules) RulesExtra {
	// Return default RulesExtra
	return RulesExtra{
		IsSubnetEVM: true, // Default to true for SubnetEVM
	}
}
