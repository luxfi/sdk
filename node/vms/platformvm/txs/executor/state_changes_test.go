// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/crypto/bls/signer/localsigner"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/genesis/builder"
	"github.com/luxfi/upgrade/upgradetest"
	"github.com/luxfi/sdk/utils/iterator"
	"github.com/luxfi/sdk/utils/timer/mockable"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/sdk/node/vms/components/gas"
	"github.com/luxfi/sdk/node/vms/platformvm/config"
	"github.com/luxfi/sdk/node/vms/platformvm/genesis/genesistest"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/state/statetest"
	"github.com/luxfi/sdk/node/vms/platformvm/validators/fee"
)

func TestAdvanceTimeTo_UpdatesFeeState(t *testing.T) {
	const (
		secondsToAdvance  = 3
		durationToAdvance = secondsToAdvance * time.Second
	)

	feeConfig := gas.Config{
		MaxCapacity:     1000,
		MaxPerSecond:    100,
		TargetPerSecond: 50,
	}

	tests := []struct {
		name          string
		fork          upgradetest.Fork
		initialState  gas.State
		expectedState gas.State
	}{
		{
			name:          "Pre-Etna",
			fork:          upgradetest.Durango,
			initialState:  gas.State{},
			expectedState: gas.State{}, // Pre-Etna, fee state should not change
		},
		{
			name: "Etna with no usage",
			initialState: gas.State{
				Capacity: feeConfig.MaxCapacity,
				Excess:   0,
			},
			expectedState: gas.State{
				Capacity: feeConfig.MaxCapacity,
				Excess:   0,
			},
		},
		{
			name: "Etna with usage",
			fork: upgradetest.Etna,
			initialState: gas.State{
				Capacity: 1,
				Excess:   10_000,
			},
			expectedState: gas.State{
				Capacity: min(gas.Gas(1).AddPerSecond(feeConfig.MaxPerSecond, secondsToAdvance), feeConfig.MaxCapacity),
				Excess:   gas.Gas(10_000).SubPerSecond(feeConfig.TargetPerSecond, secondsToAdvance),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				require = require.New(t)

				s        = statetest.New(t, statetest.Config{})
				nextTime = s.GetTimestamp().Add(durationToAdvance)
			)

			// Ensure the invariant that [nextTime <= nextStakerChangeTime] on
			// AdvanceTimeTo is maintained.
			nextStakerChangeTime, err := state.GetNextStakerChangeTime(
				builder.LocalValidatorFeeConfig,
				s,
				mockable.MaxTime,
			)
			require.NoError(err)
			require.False(nextTime.After(nextStakerChangeTime))

			s.SetFeeState(test.initialState)

			validatorsModified, err := AdvanceTimeTo(
				&Backend{
					Config: &config.Internal{
						DynamicFeeConfig: feeConfig,
						UpgradeConfig:    upgradetest.GetConfig(test.fork),
					},
				},
				s,
				nextTime,
			)
			require.NoError(err)
			require.False(validatorsModified)
			require.Equal(test.expectedState, s.GetFeeState())
			require.Equal(nextTime, s.GetTimestamp())
		})
	}
}

func TestAdvanceTimeTo_RemovesStaleExpiries(t *testing.T) {
	var (
		currentTime = genesistest.DefaultValidatorStartTime
		newTime     = currentTime.Add(3 * time.Second)
		newTimeUnix = uint64(newTime.Unix())

		unexpiredTime         = newTimeUnix + 1
		expiredTime           = newTimeUnix
		previouslyExpiredTime = newTimeUnix - 1
		validationID          = ids.GenerateTestID()
	)

	tests := []struct {
		name             string
		initialExpiries  []state.ExpiryEntry
		expectedExpiries []state.ExpiryEntry
	}{
		{
			name: "no expiries",
		},
		{
			name: "unexpired expiry",
			initialExpiries: []state.ExpiryEntry{
				{
					Timestamp:    unexpiredTime,
					ValidationID: validationID,
				},
			},
			expectedExpiries: []state.ExpiryEntry{
				{
					Timestamp:    unexpiredTime,
					ValidationID: validationID,
				},
			},
		},
		{
			name: "unexpired expiry at new time",
			initialExpiries: []state.ExpiryEntry{
				{
					Timestamp:    expiredTime,
					ValidationID: ids.GenerateTestID(),
				},
			},
		},
		{
			name: "unexpired expiry at previous time",
			initialExpiries: []state.ExpiryEntry{
				{
					Timestamp:    previouslyExpiredTime,
					ValidationID: ids.GenerateTestID(),
				},
			},
		},
		{
			name: "limit expiries removed",
			initialExpiries: []state.ExpiryEntry{
				{
					Timestamp:    expiredTime,
					ValidationID: ids.GenerateTestID(),
				},
				{
					Timestamp:    unexpiredTime,
					ValidationID: validationID,
				},
			},
			expectedExpiries: []state.ExpiryEntry{
				{
					Timestamp:    unexpiredTime,
					ValidationID: validationID,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				require = require.New(t)
				s       = statetest.New(t, statetest.Config{})
			)

			// Ensure the invariant that [newTime <= nextStakerChangeTime] on
			// AdvanceTimeTo is maintained.
			nextStakerChangeTime, err := state.GetNextStakerChangeTime(
				builder.LocalValidatorFeeConfig,
				s,
				mockable.MaxTime,
			)
			require.NoError(err)
			require.False(newTime.After(nextStakerChangeTime))

			for _, expiry := range test.initialExpiries {
				s.PutExpiry(expiry)
			}

			validatorsModified, err := AdvanceTimeTo(
				&Backend{
					Config: &config.Internal{
						UpgradeConfig: upgradetest.GetConfig(upgradetest.Latest),
					},
				},
				s,
				newTime,
			)
			require.NoError(err)
			require.False(validatorsModified)

			expiryIterator, err := s.GetExpiryIterator()
			require.NoError(err)
			require.Equal(
				test.expectedExpiries,
				iterator.ToSlice(expiryIterator),
			)
		})
	}
}

func TestAdvanceTimeTo_UpdateL1Validators(t *testing.T) {
	sk, err := localsigner.New()
	require.NoError(t, err)

	const (
		secondsToAdvance = 3
		timeToAdvance    = secondsToAdvance * time.Second
	)

	var (
		pk      = sk.PublicKey()
		pkBytes = bls.PublicKeyToUncompressedBytes(pk)

		validatorFeeConfig = fee.Config{
			Capacity:                 builder.LocalValidatorFeeConfig.Capacity,
			Target:                   1,
			MinPrice:                 builder.LocalValidatorFeeConfig.MinPrice,
			ExcessConversionConstant: builder.LocalValidatorFeeConfig.ExcessConversionConstant,
		}

		newL1Validator = func(endAccumulatedFee uint64) state.L1Validator {
			return state.L1Validator{
				ValidationID:      ids.GenerateTestID(),
				ChainID:           ids.GenerateTestID(),
				NodeID:            ids.GenerateTestNodeID(),
				PublicKey:         pkBytes,
				Weight:            1,
				EndAccumulatedFee: endAccumulatedFee,
			}
		}

		// Calculate the cost for 3 seconds based on validator count.
		// This ensures validators are evicted at exactly 3 seconds (satisfying invariant).
		costForValidators = func(numValidators int) uint64 {
			return fee.State{Current: gas.Gas(numValidators), Excess: 0}.CostOf(validatorFeeConfig, secondsToAdvance)
		}

		// Very high balance for validators that should NOT be evicted
		keeperBalance = uint64(1000 * units.Lux)

		currentTime = genesistest.DefaultValidatorStartTime
		newTime     = currentTime.Add(timeToAdvance)

		config = config.Internal{
			ValidatorFeeConfig: validatorFeeConfig,
			UpgradeConfig:      upgradetest.GetConfig(upgradetest.Latest),
		}
	)

	tests := []struct {
		name                    string
		numEvict                int  // number of validators to evict
		numKeep                 int  // number of validators to keep
		expectedModified        bool
		expectedExcess          gas.Gas
	}{
		{
			name:             "no L1 validators",
			numEvict:         0,
			numKeep:          0,
			expectedModified: false,
			expectedExcess:   0,
		},
		{
			name:             "evicted one",
			numEvict:         1,
			numKeep:          0,
			expectedModified: true,
			expectedExcess:   0,
		},
		{
			name:             "evicted all",
			numEvict:         2,
			numKeep:          0,
			expectedModified: true,
			expectedExcess:   3,
		},
		{
			name:             "evicted 2 of 3",
			numEvict:         2,
			numKeep:          1,
			expectedModified: true,
			expectedExcess:   6,
		},
		{
			name:             "no evictions",
			numEvict:         0,
			numKeep:          1,
			expectedModified: false,
			expectedExcess:   0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				require = require.New(t)
				s       = statetest.New(t, statetest.Config{})
			)

			// Calculate cost based on total number of validators
			totalValidators := test.numEvict + test.numKeep
			evictCost := costForValidators(max(totalValidators, 1))

			// Create and add validators to evict (with exact 3-second balance)
			var expectedL1Validators []state.L1Validator
			for i := 0; i < test.numEvict; i++ {
				v := newL1Validator(evictCost)
				require.NoError(s.PutL1Validator(v))
			}

			// Create and add validators to keep (with very high balance)
			for i := 0; i < test.numKeep; i++ {
				v := newL1Validator(keeperBalance)
				require.NoError(s.PutL1Validator(v))
				expectedL1Validators = append(expectedL1Validators, v)
			}

			// Ensure the invariant that [newTime <= nextStakerChangeTime] on
			// AdvanceTimeTo is maintained.
			nextStakerChangeTime, err := state.GetNextStakerChangeTime(
				config.ValidatorFeeConfig,
				s,
				mockable.MaxTime,
			)
			require.NoError(err)
			require.False(newTime.After(nextStakerChangeTime))

			validatorsModified, err := AdvanceTimeTo(
				&Backend{
					Config: &config,
				},
				s,
				newTime,
			)
			require.NoError(err)
			require.Equal(test.expectedModified, validatorsModified)

			activeL1Validators, err := s.GetActiveL1ValidatorsIterator()
			require.NoError(err)
			require.Equal(
				expectedL1Validators,
				iterator.ToSlice(activeL1Validators),
			)

			require.Equal(test.expectedExcess, s.GetL1ValidatorExcess())
			// Accrued fees = cost for the number of validators over the time period
			// When Current > Target, fee rate increases due to excess
			expectedAccruedFees := costForValidators(max(totalValidators, 1))
			require.Equal(expectedAccruedFees, s.GetAccruedFees())
		})
	}
}
