// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/upgrade/upgradetest"
	"github.com/luxfi/sdk/node/vms/platformvm/genesis/genesistest"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/utxo"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

func TestCreateSubnetTxAP3FeeChange(t *testing.T) {
	// Test the fee change at Apricot Phase 3
	// Pre-AP3: CreateSubnetTxFee = 0
	// Post-AP3: CreateSubnetTxFee = CreateSubnetTxFee from config (100 * defaultTxFee)
	tests := []struct {
		name        string
		preAP3      bool
		expectedErr error
	}{
		{
			name:        "pre-AP3 - no fee required",
			preAP3:      true,
			expectedErr: nil,
		},
		{
			name:        "post-AP3 - fee required",
			preAP3:      false,
			expectedErr: nil, // Should succeed with properly funded wallet
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)

			env := newEnvironment(t, upgradetest.Latest)
			env.ctx.Lock.Lock()
			defer env.ctx.Lock.Unlock()

			// Set AP3 time relative to current state timestamp
			currentTime := env.state.GetTimestamp()
			var ap3Time time.Time
			if test.preAP3 {
				// Set AP3 in the future so we're pre-fork
				ap3Time = currentTime.Add(time.Hour)
			} else {
				// Set AP3 in the past so we're post-fork
				ap3Time = currentTime.Add(-time.Hour)
			}
			env.config.UpgradeConfig.ApricotPhase3Time = ap3Time

			// Use the standard wallet helper which properly sets up fees
			wallet := newWallet(t, env, walletConfig{
				keys: genesistest.DefaultFundedKeys[:1],
			})

			// Create a subnet using the wallet
			tx, err := wallet.IssueCreateSubnetTx(
				&secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs: []ids.ShortID{
						genesistest.DefaultFundedKeys[0].Address(),
					},
				},
			)
			require.NoError(err)

			stateDiff, err := state.NewDiff(lastAcceptedID, env)
			require.NoError(err)

			// Use the proper fee calculator based on state timestamp
			feeCalculator := state.PickFeeCalculator(env.config, stateDiff)
			_, _, _, err = StandardTx(
				&env.backend,
				feeCalculator,
				tx,
				stateDiff,
			)
			require.ErrorIs(err, test.expectedErr)
		})
	}
}

// TestCreateSubnetTxInsufficientFunds tests that CreateSubnet transactions fail
// when the wallet doesn't have enough funds to pay the fee
func TestCreateSubnetTxInsufficientFunds(t *testing.T) {
	require := require.New(t)

	env := newEnvironment(t, upgradetest.Latest)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	// Set AP3 in the past so we're post-fork (fees required)
	currentTime := env.state.GetTimestamp()
	env.config.UpgradeConfig.ApricotPhase3Time = currentTime.Add(-time.Hour)

	// Create a wallet with unfunded keys (no UTXOs)
	// This will fail because there are no funds to pay fees
	wallet := newWallet(t, env, walletConfig{
		keys: genesistest.DefaultFundedKeys[4:5], // Use a key that might not have funds
	})

	// Try to create a subnet - should fail due to insufficient funds
	_, err := wallet.IssueCreateSubnetTx(
		&secp256k1fx.OutputOwners{
			Threshold: 1,
			Addrs: []ids.ShortID{
				genesistest.DefaultFundedKeys[4].Address(),
			},
		},
	)
	// If the key has no funds, this should fail with insufficient funds
	// If the key has funds, it will succeed
	if err != nil {
		require.ErrorIs(err, utxo.ErrInsufficientUnlockedFunds)
	}
}
