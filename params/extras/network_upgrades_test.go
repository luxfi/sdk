// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package extras

import (
	"testing"

	"github.com/luxfi/node/upgrade"
	"github.com/luxfi/node/upgrade/upgradetest"
	"github.com/luxfi/node/utils/constants"
	"github.com/luxfi/evm/utils"
	"github.com/stretchr/testify/require"
)

func TestNetworkUpgradesEqual(t *testing.T) {
	testcases := []struct {
		name      string
		upgrades1 *NetworkUpgrades
		upgrades2 *NetworkUpgrades
		expected  bool
	}{
		{
			name: "EqualNetworkUpgrades",
			upgrades1: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(2),
			},
			upgrades2: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(2),
			},
			expected: true,
		},
		{
			name: "NotEqualNetworkUpgrades",
			upgrades1: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(2),
			},
			upgrades2: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(3),
			},
			expected: false,
		},
		{
			name: "NilNetworkUpgrades",
			upgrades1: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(2),
			},
			upgrades2: nil,
			expected:  false,
		},
		{
			name: "NilNetworkUpgrade",
			upgrades1: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(2),
			},
			upgrades2: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   nil,
			},
			expected: false,
		},
	}
	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.upgrades1.Equal(test.upgrades2))
		})
	}
}

func TestCheckNetworkUpgradesCompatible(t *testing.T) {
	testcases := []struct {
		name      string
		upgrades1 *NetworkUpgrades
		upgrades2 *NetworkUpgrades
		time      uint64
		valid     bool
	}{
		{
			name: "Compatible_same_NetworkUpgrades",
			upgrades1: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(2),
			},
			upgrades2: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(2),
			},
			time:  1,
			valid: true,
		},
		{
			name: "Compatible_different_NetworkUpgrades",
			upgrades1: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(2),
			},
			upgrades2: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(3),
			},
			time:  1,
			valid: true,
		},
		{
			name: "Compatible_nil_NetworkUpgrades",
			upgrades1: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(2),
			},
			upgrades2: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   nil,
			},
			time:  1,
			valid: true,
		},
		{
			name: "Incompatible_rewinded_NetworkUpgrades",
			upgrades1: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(2),
			},
			upgrades2: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(1),
			},
			time:  1,
			valid: false,
		},
		{
			name: "Incompatible_fastforward_NetworkUpgrades",
			upgrades1: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(2),
			},
			upgrades2: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(3),
			},
			time:  4,
			valid: false,
		},
		{
			name: "Incompatible_nil_NetworkUpgrades",
			upgrades1: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(2),
			},
			upgrades2: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   nil,
			},
			time:  2,
			valid: false,
		},
		{
			name: "Incompatible_fastforward_nil_NetworkUpgrades",
			upgrades1: func() *NetworkUpgrades {
				upgrades := GetNetworkUpgrades(upgrade.Testnet)
				return &upgrades
			}(),
			upgrades2: func() *NetworkUpgrades {
				upgrades := GetNetworkUpgrades(upgrade.Testnet)
				upgrades.EtnaTimestamp = nil
				return &upgrades
			}(),
			time:  uint64(upgrade.Testnet.EtnaTime.Unix()),
			valid: false,
		},
		{
			name: "Compatible_Fortuna_fastforward_nil_NetworkUpgrades",
			upgrades1: func() *NetworkUpgrades {
				upgrades := GetNetworkUpgrades(upgrade.Testnet)
				return &upgrades
			}(),
			upgrades2: func() *NetworkUpgrades {
				upgrades := GetNetworkUpgrades(upgrade.Testnet)
				upgrades.FortunaTimestamp = nil
				return &upgrades
			}(),
			time:  uint64(upgrade.Testnet.FortunaTime.Unix()),
			valid: true,
		},
	}
	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			err := test.upgrades1.checkNetworkUpgradesCompatible(test.upgrades2, test.time)
			if test.valid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func TestVerifyNetworkUpgrades(t *testing.T) {
	testcases := []struct {
		name          string
		upgrades      *NetworkUpgrades
		luxdUpgrades upgrade.Config
		valid         bool
	}{
		{
			name: "ValidNetworkUpgrades_for_latest_network",
			upgrades: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(0),
				DurangoTimestamp:   utils.NewUint64(1607144400),
				EtnaTimestamp:      utils.NewUint64(1607144400),
			},
			luxdUpgrades: upgradetest.GetConfig(upgradetest.Latest),
			valid:         true,
		},
		{
			name: "Invalid_Durango_nil_upgrade",
			upgrades: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   nil,
			},
			luxdUpgrades: upgrade.Mainnet,
			valid:         false,
		},
		{
			name: "Invalid_Subnet-EVM_non-zero",
			upgrades: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(2),
			},
			luxdUpgrades: upgrade.Mainnet,
			valid:         false,
		},
		{
			name: "Invalid_Durango_before_default_upgrade",
			upgrades: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(0),
				DurangoTimestamp:   utils.NewUint64(1),
			},
			luxdUpgrades: upgrade.Mainnet,
			valid:         false,
		},
		{
			name: "Invalid_Mainnet_Durango_reconfigured_to_Testnet",
			upgrades: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(0),
				DurangoTimestamp:   utils.TimeToNewUint64(upgrade.GetConfig(constants.TestnetID).DurangoTime),
			},
			luxdUpgrades: upgrade.Mainnet,
			valid:         false,
		},
		{
			name: "Valid_Testnet_Durango_reconfigured_to_Mainnet",
			upgrades: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(0),
				DurangoTimestamp:   utils.TimeToNewUint64(upgrade.GetConfig(constants.MainnetID).DurangoTime),
			},
			luxdUpgrades: upgrade.Testnet,
			valid:         false,
		},
		{
			name: "Invalid_Etna_nil",
			upgrades: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(0),
				DurangoTimestamp:   utils.TimeToNewUint64(upgrade.Mainnet.DurangoTime),
				EtnaTimestamp:      nil,
			},
			luxdUpgrades: upgrade.Mainnet,
			valid:         false,
		},
		{
			name: "Invalid_Etna_before_Durango",
			upgrades: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(0),
				DurangoTimestamp:   utils.TimeToNewUint64(upgrade.Mainnet.DurangoTime),
				EtnaTimestamp:      utils.TimeToNewUint64(upgrade.Mainnet.DurangoTime.Add(-1)),
			},
			luxdUpgrades: upgrade.Mainnet,
			valid:         false,
		},
		{
			name: "Valid_Fortuna_nil",
			upgrades: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(0),
				DurangoTimestamp:   utils.TimeToNewUint64(upgrade.Testnet.DurangoTime),
				EtnaTimestamp:      utils.TimeToNewUint64(upgrade.Testnet.EtnaTime),
				FortunaTimestamp:   nil,
			},
			luxdUpgrades: upgrade.Testnet,
			valid:         true,
		},
	}
	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			err := test.upgrades.verifyNetworkUpgrades(test.luxdUpgrades)
			if test.valid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func TestForkOrder(t *testing.T) {
	testcases := []struct {
		name        string
		upgrades    *NetworkUpgrades
		expectedErr bool
	}{
		{
			name: "ValidNetworkUpgrades",
			upgrades: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(0),
				DurangoTimestamp:   utils.NewUint64(2),
			},
			expectedErr: false,
		},
		{
			name: "Invalid order",
			upgrades: &NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(1),
				DurangoTimestamp:   utils.NewUint64(0),
			},
			expectedErr: true,
		},
	}
	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			err := checkForks(test.upgrades.forkOrder(), false)
			if test.expectedErr {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}
		})
	}
}
