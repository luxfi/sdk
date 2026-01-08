// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/database/memdb"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/genesis/builder"
	"github.com/luxfi/upgrade/upgradetest"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/utils/timer/mockable"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/sdk/node/vms/components/gas"
	"github.com/luxfi/sdk/node/vms/platformvm/config"
	"github.com/luxfi/sdk/node/vms/platformvm/genesis/genesistest"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"

	txfee "github.com/luxfi/sdk/node/vms/platformvm/txs/fee"
	validatorfee "github.com/luxfi/sdk/node/vms/platformvm/validators/fee"
)

func TestNextBlockTime(t *testing.T) {
	tests := []struct {
		name           string
		chainTime      time.Time
		now            time.Time
		expectedTime   time.Time
		expectedCapped bool
	}{
		{
			name:           "parent time is after now",
			chainTime:      genesistest.DefaultValidatorStartTime,
			now:            genesistest.DefaultValidatorStartTime.Add(-time.Second),
			expectedTime:   genesistest.DefaultValidatorStartTime,
			expectedCapped: false,
		},
		{
			name:           "parent time is before now",
			chainTime:      genesistest.DefaultValidatorStartTime,
			now:            genesistest.DefaultValidatorStartTime.Add(time.Second),
			expectedTime:   genesistest.DefaultValidatorStartTime.Add(time.Second),
			expectedCapped: false,
		},
		{
			name:           "now is at next staker change time",
			chainTime:      genesistest.DefaultValidatorStartTime,
			now:            genesistest.DefaultValidatorEndTime,
			expectedTime:   genesistest.DefaultValidatorEndTime,
			expectedCapped: true,
		},
		{
			name:           "now is after next staker change time",
			chainTime:      genesistest.DefaultValidatorStartTime,
			now:            genesistest.DefaultValidatorEndTime.Add(time.Second),
			expectedTime:   genesistest.DefaultValidatorEndTime,
			expectedCapped: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				require = require.New(t)
				s       = newTestState(t, memdb.New())
				clk     mockable.Clock
			)

			s.SetTimestamp(test.chainTime)
			clk.Set(test.now)

			actualTime, actualCapped, err := NextBlockTime(
				builder.LocalValidatorFeeConfig,
				s,
				&clk,
			)
			require.NoError(err)
			require.Equal(test.expectedTime.Local(), actualTime.Local())
			require.Equal(test.expectedCapped, actualCapped)
		})
	}
}

func TestGetNextStakerChangeTime(t *testing.T) {
	config := validatorfee.Config{
		Capacity:                 builder.LocalValidatorFeeConfig.Capacity,
		Target:                   builder.LocalValidatorFeeConfig.Target,
		MinPrice:                 gas.Price(2 * units.NanoLux), // Increase minimum price to test fractional seconds
		ExcessConversionConstant: builder.LocalValidatorFeeConfig.ExcessConversionConstant,
	}

	tests := []struct {
		name         string
		pending      []*Staker
		l1Validators []L1Validator
		maxTime      time.Time
		expected     time.Time
	}{
		{
			name:     "only current validators",
			maxTime:  mockable.MaxTime,
			expected: genesistest.DefaultValidatorEndTime,
		},
		{
			name: "current and pending validators",
			pending: []*Staker{
				{
					TxID:      ids.GenerateTestID(),
					NodeID:    ids.GenerateTestNodeID(),
					PublicKey: nil,
					ChainID:  constants.PrimaryNetworkID,
					Weight:    1,
					StartTime: genesistest.DefaultValidatorStartTime.Add(time.Second),
					EndTime:   genesistest.DefaultValidatorEndTime,
					NextTime:  genesistest.DefaultValidatorStartTime.Add(time.Second),
					Priority:  txs.PrimaryNetworkValidatorPendingPriority,
				},
			},
			maxTime:  mockable.MaxTime,
			expected: genesistest.DefaultValidatorStartTime.Add(time.Second),
		},
		{
			name: "L1 validator with less than 1 second of fees",
			l1Validators: []L1Validator{
				{
					ValidationID:      ids.GenerateTestID(),
					ChainID:          ids.GenerateTestID(),
					NodeID:            ids.GenerateTestNodeID(),
					Weight:            1,
					EndAccumulatedFee: 1, // This validator should be evicted in .5 seconds, which is rounded to 0.
				},
			},
			maxTime:  mockable.MaxTime,
			expected: genesistest.DefaultValidatorStartTime,
		},
		{
			name: "L1 validator with 1 second of fees",
			l1Validators: []L1Validator{
				{
					ValidationID:      ids.GenerateTestID(),
					ChainID:          ids.GenerateTestID(),
					NodeID:            ids.GenerateTestNodeID(),
					Weight:            1,
					EndAccumulatedFee: 2, // This validator should be evicted in 1 second.
				},
			},
			maxTime:  mockable.MaxTime,
			expected: genesistest.DefaultValidatorStartTime.Add(time.Second),
		},
		{
			name: "L1 validator with less than 2 seconds of fees",
			l1Validators: []L1Validator{
				{
					ValidationID:      ids.GenerateTestID(),
					ChainID:          ids.GenerateTestID(),
					NodeID:            ids.GenerateTestNodeID(),
					Weight:            1,
					EndAccumulatedFee: 3, // This validator should be evicted in 1.5 seconds, which is rounded to 1.
				},
			},
			maxTime:  mockable.MaxTime,
			expected: genesistest.DefaultValidatorStartTime.Add(time.Second),
		},
		{
			name: "current and L1 validator with high balance",
			l1Validators: []L1Validator{
				{
					ValidationID:      ids.GenerateTestID(),
					ChainID:          ids.GenerateTestID(),
					NodeID:            ids.GenerateTestNodeID(),
					Weight:            1,
					EndAccumulatedFee: 10 * units.Lux, // 10 LUX = 10M microLux, lasts ~58 days at 2 nanoLux/sec, well past the 28-day validator end time.
				},
			},
			maxTime:  mockable.MaxTime,
			expected: genesistest.DefaultValidatorEndTime,
		},
		{
			name:     "restricted timestamp",
			maxTime:  genesistest.DefaultValidatorStartTime,
			expected: genesistest.DefaultValidatorStartTime,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				require = require.New(t)
				s       = newTestState(t, memdb.New())
			)
			for _, staker := range test.pending {
				require.NoError(s.PutPendingValidator(staker))
			}
			for _, l1Validator := range test.l1Validators {
				require.NoError(s.PutL1Validator(l1Validator))
			}

			actual, err := GetNextStakerChangeTime(
				config,
				s,
				test.maxTime,
			)
			require.NoError(err)
			require.Equal(test.expected.Local(), actual.Local())
		})
	}
}

func TestPickFeeCalculator(t *testing.T) {
	dynamicFeeConfig := builder.LocalDynamicFeeConfig

	tests := []struct {
		fork     upgradetest.Fork
		expected txfee.Calculator
	}{
		{
			fork:     upgradetest.ApricotPhase2,
			expected: txfee.NewSimpleCalculator(0),
		},
		{
			fork:     upgradetest.ApricotPhase3,
			expected: txfee.NewSimpleCalculator(0),
		},
		{
			fork: upgradetest.Etna,
			expected: txfee.NewDynamicCalculator(
				dynamicFeeConfig.Weights,
				dynamicFeeConfig.MinPrice,
			),
		},
	}
	for _, test := range tests {
		t.Run(test.fork.String(), func(t *testing.T) {
			var (
				config = &config.Internal{
					DynamicFeeConfig: dynamicFeeConfig,
					UpgradeConfig:    upgradetest.GetConfig(test.fork),
				}
				s = newTestState(t, memdb.New())
			)
			actual := PickFeeCalculator(config, s)
			require.Equal(t, test.expected, actual)
		})
	}
}
