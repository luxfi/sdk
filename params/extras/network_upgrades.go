// (c) 2022, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package extras

import (
	"fmt"
	"reflect"

	"github.com/luxfi/node/upgrade"
	"github.com/luxfi/evm/utils"
	gethparams "github.com/luxfi/geth/params"
)

var errCannotBeNil = fmt.Errorf("timestamp cannot be nil")

// newTimestampCompatError creates a new timestamp compatibility error
func newTimestampCompatError(what string, storedtime, newtime *uint64) *gethparams.ConfigCompatError {
	var rew *uint64
	switch {
	case storedtime == nil:
		rew = newtime
	case newtime == nil || *storedtime < *newtime:
		rew = storedtime
	default:
		rew = newtime
	}
	var rewTime uint64
	if rew != nil {
		rewTime = *rew
	}
	return &gethparams.ConfigCompatError{
		What:         what,
		StoredTime:   storedtime,
		NewTime:      newtime,
		RewindToTime: rewTime,
	}
}

// NetworkUpgrades contains timestamps that enable network upgrades.
// Lux specific network upgrades are also included here.
// (nil = no fork, 0 = already activated)
type NetworkUpgrades struct {
	// EVMTimestamp is a placeholder that activates Lux Upgrades prior to ApricotPhase6
	EVMTimestamp *uint64 `json:"evmTimestamp,omitempty"`
	// Durango activates the Shanghai Execution Spec Upgrade from Ethereum (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/shanghai.md#included-eips)
	// and Lux Warp Messaging.
	// Note: EIP-4895 is excluded since withdrawals are not relevant to the Lux C-Chain or Subnets running the EVM.
	DurangoTimestamp *uint64 `json:"durangoTimestamp,omitempty"`
	// Placeholder for EtnaTimestamp
	EtnaTimestamp *uint64 `json:"etnaTimestamp,omitempty"`
	// Fortuna has no effect on EVM by itself, but is included for completeness.
	FortunaTimestamp *uint64 `json:"fortunaTimestamp,omitempty"`
	// Granite is a placeholder for the next upgrade.
	GraniteTimestamp *uint64 `json:"graniteTimestamp,omitempty"`
}

func (n *NetworkUpgrades) Equal(other *NetworkUpgrades) bool {
	return reflect.DeepEqual(n, other)
}

func (n *NetworkUpgrades) checkNetworkUpgradesCompatible(newcfg *NetworkUpgrades, time uint64) *gethparams.ConfigCompatError {
	if isForkTimestampIncompatible(n.EVMTimestamp, newcfg.EVMTimestamp, time) {
		return newTimestampCompatError("EVM fork block timestamp", n.EVMTimestamp, newcfg.EVMTimestamp)
	}
	if isForkTimestampIncompatible(n.DurangoTimestamp, newcfg.DurangoTimestamp, time) {
		return newTimestampCompatError("Durango fork block timestamp", n.DurangoTimestamp, newcfg.DurangoTimestamp)
	}
	if isForkTimestampIncompatible(n.EtnaTimestamp, newcfg.EtnaTimestamp, time) {
		return newTimestampCompatError("Etna fork block timestamp", n.EtnaTimestamp, newcfg.EtnaTimestamp)
	}
	if isForkTimestampIncompatible(n.FortunaTimestamp, newcfg.FortunaTimestamp, time) {
		return newTimestampCompatError("Fortuna fork block timestamp", n.FortunaTimestamp, newcfg.FortunaTimestamp)
	}
	if isForkTimestampIncompatible(n.GraniteTimestamp, newcfg.GraniteTimestamp, time) {
		return newTimestampCompatError("Granite fork block timestamp", n.GraniteTimestamp, newcfg.GraniteTimestamp)
	}

	return nil
}

func (n *NetworkUpgrades) forkOrder() []fork {
	return []fork{
		{name: "evmTimestamp", timestamp: n.EVMTimestamp},
		{name: "durangoTimestamp", timestamp: n.DurangoTimestamp},
		{name: "etnaTimestamp", timestamp: n.EtnaTimestamp},
		{name: "fortunaTimestamp", timestamp: n.FortunaTimestamp, optional: true},
		{name: "graniteTimestamp", timestamp: n.GraniteTimestamp},
	}
}

// SetDefaults sets the default values for the network upgrades.
// This overrides deactivating the network upgrade by providing a timestamp of nil value.
func (n *NetworkUpgrades) SetDefaults(agoUpgrades upgrade.Config) {
	defaults := getDefaultNetworkUpgrades(agoUpgrades)
	// If the network upgrade is not set, set it to the default value.
	// If the network upgrade is set to 0, we also treat it as nil and set it default.
	// This is because in prior versions, upgrades were not modifiable and were directly set to their default values.
	// Most of the tools and configurations just provide these as 0, so it is safer to treat 0 as nil and set to default
	// to prevent premature activations of the network upgrades for live networks.
	if n.EVMTimestamp == nil || *n.EVMTimestamp == 0 {
		n.EVMTimestamp = defaults.EVMTimestamp
	}
	if n.DurangoTimestamp == nil || *n.DurangoTimestamp == 0 {
		n.DurangoTimestamp = defaults.DurangoTimestamp
	}
	if n.EtnaTimestamp == nil || *n.EtnaTimestamp == 0 {
		n.EtnaTimestamp = defaults.EtnaTimestamp
	}
	if n.FortunaTimestamp == nil || *n.FortunaTimestamp == 0 {
		n.FortunaTimestamp = defaults.FortunaTimestamp
	}
}

// verifyNetworkUpgrades checks that the network upgrades are well formed.
func (n *NetworkUpgrades) verifyNetworkUpgrades(agoUpgrades upgrade.Config) error {
	defaults := getDefaultNetworkUpgrades(agoUpgrades)
	if err := verifyWithDefault(n.EVMTimestamp, defaults.EVMTimestamp); err != nil {
		return fmt.Errorf("EVM fork block timestamp is invalid: %w", err)
	}
	if err := verifyWithDefault(n.DurangoTimestamp, defaults.DurangoTimestamp); err != nil {
		return fmt.Errorf("Durango fork block timestamp is invalid: %w", err)
	}
	if err := verifyWithDefault(n.EtnaTimestamp, defaults.EtnaTimestamp); err != nil {
		return fmt.Errorf("Etna fork block timestamp is invalid: %w", err)
	}
	if err := verifyWithDefault(n.FortunaTimestamp, defaults.FortunaTimestamp); err != nil {
		return fmt.Errorf("Fortuna fork block timestamp is invalid: %w", err)
	}
	if err := verifyWithDefault(n.GraniteTimestamp, defaults.GraniteTimestamp); err != nil {
		return fmt.Errorf("Granite fork block timestamp is invalid: %w", err)
	}
	return nil
}

func (n *NetworkUpgrades) Override(o *NetworkUpgrades) {
	if o.EVMTimestamp != nil {
		n.EVMTimestamp = o.EVMTimestamp
	}
	if o.DurangoTimestamp != nil {
		n.DurangoTimestamp = o.DurangoTimestamp
	}
	if o.EtnaTimestamp != nil {
		n.EtnaTimestamp = o.EtnaTimestamp
	}
	if o.FortunaTimestamp != nil {
		n.FortunaTimestamp = o.FortunaTimestamp
	}
	if o.GraniteTimestamp != nil {
		n.GraniteTimestamp = o.GraniteTimestamp
	}
}

// IsEVM returns whether [time] represents a block
// with a timestamp after the EVM upgrade time.
func (n NetworkUpgrades) IsEVM(time uint64) bool {
	return isTimestampForked(n.EVMTimestamp, time)
}

// IsDurango returns whether [time] represents a block
// with a timestamp after the Durango upgrade time.
func (n NetworkUpgrades) IsDurango(time uint64) bool {
	return isTimestampForked(n.DurangoTimestamp, time)
}

// IsEtna returns whether [time] represents a block
// with a timestamp after the Etna upgrade time.
func (n NetworkUpgrades) IsEtna(time uint64) bool {
	return isTimestampForked(n.EtnaTimestamp, time)
}

// IsFortuna returns whether [time] represents a block
// with a timestamp after the Fortuna upgrade time.
func (n *NetworkUpgrades) IsFortuna(time uint64) bool {
	return isTimestampForked(n.FortunaTimestamp, time)
}

// IsGranite returns whether [time] represents a block
// with a timestamp after the Granite upgrade time.
func (n *NetworkUpgrades) IsGranite(time uint64) bool {
	return isTimestampForked(n.GraniteTimestamp, time)
}

func (n *NetworkUpgrades) Description() string {
	var banner string
	banner += fmt.Sprintf(" - EVM Timestamp:          @%-10v (https://github.com/luxfi/node/releases/tag/v1.10.0)\n", ptrToString(n.EVMTimestamp))
	banner += fmt.Sprintf(" - Durango Timestamp:            @%-10v (https://github.com/luxfi/node/releases/tag/v1.11.0)\n", ptrToString(n.DurangoTimestamp))
	banner += fmt.Sprintf(" - Etna Timestamp:               @%-10v (https://github.com/luxfi/node/releases/tag/v1.12.0)\n", ptrToString(n.EtnaTimestamp))
	banner += fmt.Sprintf(" - Fortuna Timestamp:            @%-10v (https://github.com/luxfi/node/releases/tag/v1.13.0)\n", ptrToString(n.FortunaTimestamp))
	banner += fmt.Sprintf(" - Granite Timestamp:            @%-10v (https://github.com/luxfi/node/releases/tag/v1.14.0)\n", ptrToString(n.GraniteTimestamp))
	return banner
}

type LuxRules struct {
	IsEVM bool
	IsDurango   bool
	IsEtna      bool
	IsFortuna   bool
	IsGranite   bool
}

func (n *NetworkUpgrades) GetLuxRules(time uint64) LuxRules {
	return LuxRules{
		IsEVM: n.IsEVM(time),
		IsDurango:   n.IsDurango(time),
		IsEtna:      n.IsEtna(time),
		IsFortuna:   n.IsFortuna(time),
		IsGranite:   n.IsGranite(time),
	}
}

// getDefaultNetworkUpgrades returns the network upgrades for the specified luxd upgrades.
// Nil values are used to indicate optional upgrades.
func getDefaultNetworkUpgrades(agoUpgrade upgrade.Config) NetworkUpgrades {
	return NetworkUpgrades{
		EVMTimestamp: utils.NewUint64(0),
		DurangoTimestamp:   utils.TimeToNewUint64(agoUpgrade.DurangoTime),
		EtnaTimestamp:      utils.TimeToNewUint64(agoUpgrade.EtnaTime),
		FortunaTimestamp:   nil, // Fortuna is optional and has no effect on EVM
		GraniteTimestamp:   nil, // Granite is optional and has no effect on EVM
	}
}

// verifyWithDefault checks that the provided timestamp is greater than or equal to the default timestamp.
func verifyWithDefault(configTimestamp *uint64, defaultTimestamp *uint64) error {
	if defaultTimestamp == nil {
		return nil
	}

	if configTimestamp == nil {
		return errCannotBeNil
	}

	if *configTimestamp < *defaultTimestamp {
		return fmt.Errorf("provided timestamp (%d) must be greater than or equal to the default timestamp (%d)", *configTimestamp, *defaultTimestamp)
	}
	return nil
}

func ptrToString(val *uint64) string {
	if val == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *val)
}
