// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/upgrade/upgradetest"
	"github.com/luxfi/const"
	hashing "github.com/luxfi/crypto/hash"
	"github.com/luxfi/sdk/node/vms/platformvm/genesis/genesistest"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

// Ensure Execute fails when there are not enough control sigs
func TestCreateChainTxInsufficientControlSigs(t *testing.T) {
	require := require.New(t)
	env := newEnvironment(t, upgradetest.Banff)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	subnetID := testNet1.ID()
	wallet := newWallet(t, env, walletConfig{
		netIDs: []ids.ID{subnetID},
	})

	tx, err := wallet.IssueCreateChainTx(
		subnetID,
		nil,
		constants.XVMID,
		nil,
		"chain name",
	)
	require.NoError(err)

	// Remove a signature from the subnet authorization (last credential)
	// The last credential is for subnet authorization, earlier ones are for input UTXOs
	lastCredIdx := len(tx.Creds) - 1
	tx.Creds[lastCredIdx].(*secp256k1fx.Credential).Sigs = tx.Creds[lastCredIdx].(*secp256k1fx.Credential).Sigs[1:]

	stateDiff, err := state.NewDiff(lastAcceptedID, env)
	require.NoError(err)

	feeCalculator := state.PickFeeCalculator(env.config, stateDiff)
	_, _, _, err = StandardTx(
		&env.backend,
		feeCalculator,
		tx,
		stateDiff,
	)
	require.ErrorIs(err, errUnauthorizedModification)
}

// Ensure Execute fails when an incorrect control signature is given
func TestCreateChainTxWrongControlSig(t *testing.T) {
	require := require.New(t)
	env := newEnvironment(t, upgradetest.Banff)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	subnetID := testNet1.ID()
	wallet := newWallet(t, env, walletConfig{
		netIDs: []ids.ID{subnetID},
	})

	tx, err := wallet.IssueCreateChainTx(
		subnetID,
		nil,
		constants.XVMID,
		nil,
		"chain name",
	)
	require.NoError(err)

	// Generate new, random key to sign tx with
	key, err := secp256k1.NewPrivateKey()
	require.NoError(err)

	// Replace a valid signature with one from another key
	// Modify the subnet authorization credential (last credential)
	sig, err := key.SignHash(hashing.ComputeHash256(tx.Unsigned.Bytes()))
	require.NoError(err)
	lastCredIdx := len(tx.Creds) - 1
	copy(tx.Creds[lastCredIdx].(*secp256k1fx.Credential).Sigs[0][:], sig)

	stateDiff, err := state.NewDiff(lastAcceptedID, env)
	require.NoError(err)

	feeCalculator := state.PickFeeCalculator(env.config, stateDiff)
	_, _, _, err = StandardTx(
		&env.backend,
		feeCalculator,
		tx,
		stateDiff,
	)
	require.ErrorIs(err, errUnauthorizedModification)
}

// Ensure Execute fails when the Net the blockchain specifies as
// its validator set doesn't exist
func TestCreateChainTxNoSuchNet(t *testing.T) {
	require := require.New(t)
	env := newEnvironment(t, upgradetest.Banff)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	subnetID := testNet1.ID()
	wallet := newWallet(t, env, walletConfig{
		netIDs: []ids.ID{subnetID},
	})

	tx, err := wallet.IssueCreateChainTx(
		subnetID,
		nil,
		constants.XVMID,
		nil,
		"chain name",
	)
	require.NoError(err)

	tx.Unsigned.(*txs.CreateChainTx).ChainID = ids.GenerateTestID()

	stateDiff, err := state.NewDiff(lastAcceptedID, env)
	require.NoError(err)

	builderDiff, err := state.NewDiffOn(stateDiff)
	require.NoError(err)

	feeCalculator := state.PickFeeCalculator(env.config, builderDiff)
	_, _, _, err = StandardTx(
		&env.backend,
		feeCalculator,
		tx,
		stateDiff,
	)
	require.ErrorIs(err, database.ErrNotFound)
}

// Ensure valid tx passes semanticVerify
func TestCreateChainTxValid(t *testing.T) {
	require := require.New(t)
	env := newEnvironment(t, upgradetest.Banff)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	subnetID := testNet1.ID()
	wallet := newWallet(t, env, walletConfig{
		netIDs: []ids.ID{subnetID},
	})

	tx, err := wallet.IssueCreateChainTx(
		subnetID,
		nil,
		constants.XVMID,
		nil,
		"chain name",
	)
	require.NoError(err)

	stateDiff, err := state.NewDiff(lastAcceptedID, env)
	require.NoError(err)

	builderDiff, err := state.NewDiffOn(stateDiff)
	require.NoError(err)

	feeCalculator := state.PickFeeCalculator(env.config, builderDiff)
	_, _, _, err = StandardTx(
		&env.backend,
		feeCalculator,
		tx,
		stateDiff,
	)
	require.NoError(err)
}

func TestCreateChainTxAP3FeeChange(t *testing.T) {
	ap3Time := genesistest.DefaultValidatorStartTime.Add(time.Hour)
	tests := []struct {
		name          string
		time          time.Time
		fee           uint64
		expectedError error
	}{
		{
			name:          "pre-fork - correctly priced",
			time:          genesistest.DefaultValidatorStartTime,
			fee:           0,
			expectedError: nil,
		},
		{
			name:          "post-fork - correctly priced",
			time:          ap3Time,
			fee:           100 * defaultTxFee,
			expectedError: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)

			env := newEnvironment(t, upgradetest.Banff)
			env.config.UpgradeConfig.ApricotPhase3Time = ap3Time

			addrs := set.NewSet[ids.ShortID](len(genesistest.DefaultFundedKeys))
			for _, key := range genesistest.DefaultFundedKeys {
				addrs.Add(key.Address())
			}

			env.state.SetTimestamp(test.time) // to duly set fee

			config := *env.config
			subnetID := testNet1.ID()
			wallet := newWallet(t, env, walletConfig{
				config:    &config,
				netIDs: []ids.ID{subnetID},
			})

			tx, err := wallet.IssueCreateChainTx(
				subnetID,
				nil,
				ids.GenerateTestID(),
				nil,
				"",
			)
			require.NoError(err)

			stateDiff, err := state.NewDiff(lastAcceptedID, env)
			require.NoError(err)

			stateDiff.SetTimestamp(test.time)

			feeCalculator := state.PickFeeCalculator(env.config, stateDiff)
			_, _, _, err = StandardTx(
				&env.backend,
				feeCalculator,
				tx,
				stateDiff,
			)
			require.ErrorIs(err, test.expectedError)
		})
	}
}

func TestEtnaCreateChainTxInvalidWithManagedNet(t *testing.T) {
	require := require.New(t)
	env := newEnvironment(t, upgradetest.Etna)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	subnetID := testNet1.ID()
	wallet := newWallet(t, env, walletConfig{
		netIDs: []ids.ID{subnetID},
	})

	tx, err := wallet.IssueCreateChainTx(
		subnetID,
		nil,
		constants.XVMID,
		nil,
		"chain name",
	)
	require.NoError(err)

	stateDiff, err := state.NewDiff(lastAcceptedID, env)
	require.NoError(err)

	builderDiff, err := state.NewDiffOn(stateDiff)
	require.NoError(err)

	stateDiff.SetNetToL1Conversion(
		subnetID,
		state.NetToL1Conversion{
			ConversionID: ids.GenerateTestID(),
			ChainID:      ids.GenerateTestID(),
			Addr:         []byte("address"),
		},
	)

	feeCalculator := state.PickFeeCalculator(env.config, builderDiff)
	_, _, _, err = StandardTx(
		&env.backend,
		feeCalculator,
		tx,
		stateDiff,
	)
	require.ErrorIs(err, errIsImmutable)
}
