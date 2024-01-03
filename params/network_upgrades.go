// (c) 2022-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package params

import (
	"github.com/luxdefi/evm/utils"
)

var (
	LocalNetworkUpgrades = MandatoryNetworkUpgrades{
		EVMTimestamp: utils.NewUint64(0),
		DUpgradeTimestamp:  utils.NewUint64(0),
	}

	FujiNetworkUpgrades = MandatoryNetworkUpgrades{
		EVMTimestamp: utils.NewUint64(0),
		// DUpgradeTimestamp: utils.NewUint64(0), // TODO: Uncomment and set this to the correct value
	}

	MainnetNetworkUpgrades = MandatoryNetworkUpgrades{
		EVMTimestamp: utils.NewUint64(0),
		// DUpgradeTimestamp: utils.NewUint64(0), // TODO: Uncomment and set this to the correct value
	}

	UnitTestNetworkUpgrades = MandatoryNetworkUpgrades{
		EVMTimestamp: utils.NewUint64(0),
		DUpgradeTimestamp:  utils.NewUint64(0),
	}
)

// MandatoryNetworkUpgrades contains timestamps that enable mandatory network upgrades.
// These upgrades are mandatory, meaning that if a node does not upgrade by the
// specified timestamp, it will be unable to participate in consensus.
// Lux specific network upgrades are also included here.
type MandatoryNetworkUpgrades struct {
	// EVMTimestamp is a placeholder that activates Lux Upgrades prior to ApricotPhase6 (nil = no fork, 0 = already activated)
	EVMTimestamp *uint64 `json:"subnetEVMTimestamp,omitempty"`
	// DUpgrade activates the Shanghai Execution Spec Upgrade from Ethereum (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/shanghai.md#included-eips)
	// and Lux Warp Messaging. (nil = no fork, 0 = already activated)
	// Note: EIP-4895 is excluded since withdrawals are not relevant to the Lux C-Chain or Subnets running the EVM.
	DUpgradeTimestamp *uint64 `json:"dUpgradeTimestamp,omitempty"`
	// Cancun activates the Cancun upgrade from Ethereum. (nil = no fork, 0 = already activated)
	CancunTime *uint64 `json:"cancunTime,omitempty"`
}

func (m *MandatoryNetworkUpgrades) CheckMandatoryCompatible(newcfg *MandatoryNetworkUpgrades, time uint64) *ConfigCompatError {
	if isForkTimestampIncompatible(m.EVMTimestamp, newcfg.EVMTimestamp, time) {
		return newTimestampCompatError("EVM fork block timestamp", m.EVMTimestamp, newcfg.EVMTimestamp)
	}
	if isForkTimestampIncompatible(m.DUpgradeTimestamp, newcfg.DUpgradeTimestamp, time) {
		return newTimestampCompatError("DUpgrade fork block timestamp", m.DUpgradeTimestamp, newcfg.DUpgradeTimestamp)
	}
	if isForkTimestampIncompatible(m.CancunTime, newcfg.CancunTime, time) {
		return newTimestampCompatError("Cancun fork block timestamp", m.CancunTime, m.CancunTime)
	}
	return nil
}

func (m *MandatoryNetworkUpgrades) mandatoryForkOrder() []fork {
	return []fork{
		{name: "subnetEVMTimestamp", timestamp: m.EVMTimestamp},
		{name: "dUpgradeTimestamp", timestamp: m.DUpgradeTimestamp},
	}
}

// OptionalNetworkUpgrades includes overridable and optional EVM network upgrades.
// These can be specified in genesis and upgrade configs.
// Timestamps can be different for each subnet network.
// TODO: once we add the first optional upgrade here, we should uncomment TestVMUpgradeBytesOptionalNetworkUpgrades
type OptionalNetworkUpgrades struct{}

func (n *OptionalNetworkUpgrades) CheckOptionalCompatible(newcfg *OptionalNetworkUpgrades, time uint64) *ConfigCompatError {
	return nil
}

func (n *OptionalNetworkUpgrades) optionalForkOrder() []fork {
	return []fork{}
}
