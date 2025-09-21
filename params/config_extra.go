// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package params

import (
	"encoding/json"
	"errors"
	"math/big"
	"sync"

	"github.com/luxfi/evm/params/extras"
	"github.com/luxfi/evm/precompile/modules"
	"github.com/luxfi/evm/precompile/precompileconfig"
	"github.com/luxfi/evm/utils"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/node/upgrade"
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

	// Last Rules context - workaround since Rules is passed by value
	// This assumes single-threaded test execution
	lastRulesContext struct {
		chainConfig *ChainConfig
		timestamp   uint64
	}
	lastRulesContextMutex sync.RWMutex
)

// getPrecompileAddress returns the address for a precompile config
func getPrecompileAddress(config precompileconfig.Config) common.Address {
	// Get all registered modules
	for _, module := range modules.RegisteredModules() {
		// Match by config key
		if module.ConfigKey == config.Key() {
			return module.Address
		}
	}
	return common.Address{}
}

// RulesExtra represents extra EVM rules - part of libevm integration
type RulesExtra struct {
	IsSubnetEVM bool
	IsDurango   bool
	IsEtna      bool
	IsFortuna   bool
	IsGranite   bool

	// Fields for predicate support
	PredicatersExist bool
	Predicaters      map[common.Address]precompileconfig.Predicater

	// LuxRules for header verification
	LuxRules extras.LuxRules

	// Active precompiles for this rule set
	activePrecompiles map[common.Address]bool
}

// IsPrecompileEnabled checks if a precompile is enabled
func (r RulesExtra) IsPrecompileEnabled(addr common.Address) bool {
	if r.activePrecompiles == nil {
		return false
	}
	return r.activePrecompiles[addr]
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

// SetExtra updates the ChainConfig extra payload
func SetExtra(c *ChainConfig, extra *extras.ChainConfig) {
	chainConfigMutex.Lock()
	chainConfigExtras[c] = extra
	chainConfigMutex.Unlock()
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
func (cu *ChainConfigWithUpgradesJSON) MarshalJSON() ([]byte, error) {
	// First get the extras
	extra := GetExtra(&cu.ChainConfig)
	// fmt.Printf("DEBUG ChainConfigWithUpgradesJSON.MarshalJSON: extra=%+v\n", extra)
	// fmt.Printf("DEBUG extra.FeeConfig: %+v\n", extra.FeeConfig)

	// Marshal the base ChainConfig
	baseJSON, err := json.Marshal(&cu.ChainConfig)
	if err != nil {
		return nil, err
	}

	// Marshal the extras
	extraJSON, err := extra.MarshalJSON()
	if err != nil {
		return nil, err
	}

	// Debug
	// fmt.Printf("DEBUG: Base JSON: %s\n", string(baseJSON))
	// fmt.Printf("DEBUG: Extra JSON: %s\n", string(extraJSON))

	// Parse both as maps to merge
	var baseMap map[string]json.RawMessage
	if err := json.Unmarshal(baseJSON, &baseMap); err != nil {
		return nil, err
	}

	var extraMap map[string]json.RawMessage
	if err := json.Unmarshal(extraJSON, &extraMap); err != nil {
		return nil, err
	}

	// Merge extras into base - overwrite any conflicting keys with extras values
	for k, v := range extraMap {
		baseMap[k] = v
	}

	// Now marshal the merged config
	chainConfigJSON, err := json.Marshal(baseMap)
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
	// TODO: NetworkUpgrades field not available in current context.Context
	// GetExtra(c).NetworkUpgrades.SetDefaults(GetExtra(c).ConsensusCtx.NetworkUpgrades)
	// For now, set empty defaults with empty upgrade config
	emptyUpgradeConfig := upgrade.Config{}
	GetExtra(c).SetDefaults(emptyUpgradeConfig)
}

// SetRulesContext associates a Rules instance with its ChainConfig and timestamp
func SetRulesContext(r *Rules, c *ChainConfig, timestamp uint64) {
	lastRulesContextMutex.Lock()
	lastRulesContext.chainConfig = c
	lastRulesContext.timestamp = timestamp
	lastRulesContextMutex.Unlock()
}

// GetRulesExtra returns the RulesExtra for the given Rules
func GetRulesExtra(rules Rules) RulesExtra {
	// Use the last stored context - this is a workaround since Rules is passed by value
	lastRulesContextMutex.RLock()
	chainConfig := lastRulesContext.chainConfig
	timestamp := lastRulesContext.timestamp
	lastRulesContextMutex.RUnlock()

	if chainConfig == nil {
		// No context found, return default RulesExtra
		rulesExtra := RulesExtra{
			IsSubnetEVM:      true,
			IsDurango:        true,
			IsEtna:           true,
			IsFortuna:        true,
			IsGranite:        true,
			PredicatersExist: false,
			Predicaters:      make(map[common.Address]precompileconfig.Predicater),
		}
		rulesExtra.LuxRules = extras.LuxRules{
			IsSubnetEVM: true,
			IsDurango:   true,
			IsEtna:      true,
			IsFortuna:   true,
			IsGranite:   true,
		}
		rulesExtra.activePrecompiles = make(map[common.Address]bool)
		return rulesExtra
	}

	// Get the extras rules for the chain config and timestamp
	extrasRules := GetExtrasRules(rules, chainConfig, timestamp)

	// Build RulesExtra from extras.Rules
	rulesExtra := RulesExtra{
		IsSubnetEVM:       extrasRules.IsSubnetEVM,
		IsDurango:         extrasRules.IsDurango,
		IsEtna:            extrasRules.IsEtna,
		IsFortuna:         extrasRules.IsFortuna,
		IsGranite:         extrasRules.IsGranite,
		PredicatersExist:  len(extrasRules.Predicaters) > 0,
		Predicaters:       extrasRules.Predicaters,
		LuxRules:          extrasRules.LuxRules,
		activePrecompiles: make(map[common.Address]bool),
	}

	// Populate active precompiles
	for addr, config := range extrasRules.Precompiles {
		if !config.IsDisabled() {
			rulesExtra.activePrecompiles[addr] = true
		}
	}

	return rulesExtra
}

// GetExtrasRules returns the extras.Rules for the given params.Rules and timestamp
func GetExtrasRules(ethRules Rules, c *ChainConfig, timestamp uint64) *extras.Rules {
	if c == nil {
		return &extras.Rules{
			LuxRules:            extras.LuxRules{},
			Precompiles:         make(map[common.Address]precompileconfig.Config),
			Predicaters:         make(map[common.Address]precompileconfig.Predicater),
			AccepterPrecompiles: make(map[common.Address]precompileconfig.Accepter),
		}
	}

	extra := GetExtra(c)
	luxRules := extra.GetLuxRules(timestamp)

	// Build extras.Rules
	rules := &extras.Rules{
		LuxRules:            luxRules,
		Precompiles:         make(map[common.Address]precompileconfig.Config),
		Predicaters:         make(map[common.Address]precompileconfig.Predicater),
		AccepterPrecompiles: make(map[common.Address]precompileconfig.Accepter),
	}

	// Add active precompiles based on upgrades
	for _, upgrade := range extra.PrecompileUpgrades {
		if upgrade.Timestamp() != nil && *upgrade.Timestamp() <= timestamp {
			// Get address from the registry based on the config
			address := getPrecompileAddress(upgrade.Config)
			if address == (common.Address{}) {
				continue // Skip if no address found
			}

			if upgrade.IsDisabled() {
				delete(rules.Precompiles, address)
				delete(rules.Predicaters, address)
				delete(rules.AccepterPrecompiles, address)
			} else {
				rules.Precompiles[address] = upgrade.Config
				if predicater, ok := upgrade.Config.(precompileconfig.Predicater); ok {
					rules.Predicaters[address] = predicater
				}
				if accepter, ok := upgrade.Config.(precompileconfig.Accepter); ok {
					rules.AccepterPrecompiles[address] = accepter
				}
			}
		}
	}

	// Add genesis precompiles if at genesis
	if timestamp == 0 {
		for key, config := range extra.GenesisPrecompiles {
			if !config.IsDisabled() {
				// Get address from the key
				module, ok := modules.GetPrecompileModule(key)
				if !ok {
					continue // Skip unknown precompiles
				}
				address := module.Address

				rules.Precompiles[address] = config
				if predicater, ok := config.(precompileconfig.Predicater); ok {
					rules.Predicaters[address] = predicater
				}
				if accepter, ok := config.(precompileconfig.Accepter); ok {
					rules.AccepterPrecompiles[address] = accepter
				}
			}
		}
	}

	return rules
}
