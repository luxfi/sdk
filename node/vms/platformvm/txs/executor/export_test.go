// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/upgrade/upgradetest"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/platformvm/genesis/genesistest"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

func TestNewExportTx(t *testing.T) {
	env := newEnvironment(t, upgradetest.Banff)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	tests := []struct {
		description        string
		destinationChainID ids.ID
		timestamp          time.Time
	}{
		{
			description:        "P->X export",
			destinationChainID: env.ctx.XChainID,
			timestamp:          genesistest.DefaultValidatorStartTime,
		},
		{
			description:        "P->C export",
			destinationChainID: env.ctx.CChainID,
			timestamp:          env.config.UpgradeConfig.ApricotPhase5Time,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			require := require.New(t)

			wallet := newWallet(t, env, walletConfig{})

			tx, err := wallet.IssueExportTx(
				tt.destinationChainID,
				[]*lux.TransferableOutput{{
					Asset: lux.Asset{ID: env.ctx.XAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: genesistest.DefaultInitialBalance - defaultTxFee,
						OutputOwners: secp256k1fx.OutputOwners{
							Threshold: 1,
							Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
						},
					},
				}},
			)
			require.NoError(err)

			stateDiff, err := state.NewDiff(lastAcceptedID, env)
			require.NoError(err)

			stateDiff.SetTimestamp(tt.timestamp)

			feeCalculator := state.PickFeeCalculator(env.config, stateDiff)
			_, _, _, err = StandardTx(
				&env.backend,
				feeCalculator,
				tx,
				stateDiff,
			)
			require.NoError(err)
		})
	}
}
