// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	"github.com/luxfi/upgrade/upgradetest"
	hashing "github.com/luxfi/crypto/hash"
	"github.com/luxfi/sdk/node/vms/platformvm/genesis/genesistest"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/status"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

func TestProposalTxExecuteAddDelegator(t *testing.T) {
	dummyHeight := uint64(1)
	rewardsOwner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
	}
	nodeID := genesistest.DefaultNodeIDs[0]

	newValidatorID := ids.GenerateTestNodeID()
	newValidatorStartTime := uint64(genesistest.DefaultValidatorStartTime.Add(5 * time.Second).Unix())
	newValidatorEndTime := uint64(genesistest.DefaultValidatorEndTime.Add(-5 * time.Second).Unix())

	// [addMinStakeValidator] adds a new validator to the primary network's
	// pending validator set with the minimum staking amount
	addMinStakeValidator := func(env *environment) {
		require := require.New(t)

		wallet := newWallet(t, env, walletConfig{
			keys: genesistest.DefaultFundedKeys[:1],
		})

		tx, err := wallet.IssueAddValidatorTx(
			&txs.Validator{
				NodeID: newValidatorID,
				Start:  newValidatorStartTime,
				End:    newValidatorEndTime,
				Wght:   env.config.MinValidatorStake,
			},
			rewardsOwner,
			reward.PercentDenominator,
		)
		require.NoError(err)

		addValTx := tx.Unsigned.(*txs.AddValidatorTx)
		staker, err := state.NewCurrentStaker(
			tx.ID(),
			addValTx,
			addValTx.StartTime(),
			0,
		)
		require.NoError(err)

		require.NoError(env.state.PutCurrentValidator(staker))
		env.state.AddTx(tx, status.Committed)
		env.state.SetHeight(dummyHeight)
		require.NoError(env.state.Commit())
	}

	// [addMaxStakeValidator] adds a new validator to the primary network's
	// pending validator set with the maximum staking amount
	addMaxStakeValidator := func(env *environment) {
		require := require.New(t)

		wallet := newWallet(t, env, walletConfig{
			keys: genesistest.DefaultFundedKeys[:1],
		})

		tx, err := wallet.IssueAddValidatorTx(
			&txs.Validator{
				NodeID: newValidatorID,
				Start:  newValidatorStartTime,
				End:    newValidatorEndTime,
				Wght:   env.config.MaxValidatorStake,
			},
			rewardsOwner,
			reward.PercentDenominator,
		)
		require.NoError(err)

		addValTx := tx.Unsigned.(*txs.AddValidatorTx)
		staker, err := state.NewCurrentStaker(
			tx.ID(),
			addValTx,
			addValTx.StartTime(),
			0,
		)
		require.NoError(err)

		require.NoError(env.state.PutCurrentValidator(staker))
		env.state.AddTx(tx, status.Committed)
		env.state.SetHeight(dummyHeight)
		require.NoError(env.state.Commit())
	}

	env := newEnvironment(t, upgradetest.ApricotPhase5)
	currentTimestamp := env.state.GetTimestamp()

	type test struct {
		description string
		stakeAmount uint64
		startTime   uint64
		endTime     uint64
		nodeID      ids.NodeID
		feeKeys     []*secp256k1.PrivateKey
		setup       func(*environment)
		AP3Time     time.Time
		expectedErr error
	}

	tests := []test{
		{
			description: "validator stops validating earlier than delegator",
			stakeAmount: env.config.MinDelegatorStake,
			startTime:   genesistest.DefaultValidatorStartTimeUnix + 1,
			endTime:     genesistest.DefaultValidatorEndTimeUnix + 1,
			nodeID:      nodeID,
			feeKeys:     []*secp256k1.PrivateKey{genesistest.DefaultFundedKeys[0]},
			setup:       nil,
			AP3Time:     genesistest.DefaultValidatorStartTime,
			expectedErr: ErrPeriodMismatch,
		},
		{
			description: "validator not in the current or pending validator sets",
			stakeAmount: env.config.MinDelegatorStake,
			startTime:   uint64(genesistest.DefaultValidatorStartTime.Add(5 * time.Second).Unix()),
			endTime:     uint64(genesistest.DefaultValidatorEndTime.Add(-5 * time.Second).Unix()),
			nodeID:      newValidatorID,
			feeKeys:     []*secp256k1.PrivateKey{genesistest.DefaultFundedKeys[0]},
			setup:       nil,
			AP3Time:     genesistest.DefaultValidatorStartTime,
			expectedErr: database.ErrNotFound,
		},
		{
			description: "delegator starts before validator",
			stakeAmount: env.config.MinDelegatorStake,
			startTime:   newValidatorStartTime - 1, // start validating net before primary network
			endTime:     newValidatorEndTime,
			nodeID:      newValidatorID,
			feeKeys:     []*secp256k1.PrivateKey{genesistest.DefaultFundedKeys[0]},
			setup:       addMinStakeValidator,
			AP3Time:     genesistest.DefaultValidatorStartTime,
			expectedErr: ErrPeriodMismatch,
		},
		{
			description: "delegator stops before validator",
			stakeAmount: env.config.MinDelegatorStake,
			startTime:   newValidatorStartTime,
			endTime:     newValidatorEndTime + 1, // stop validating net after stopping validating primary network
			nodeID:      newValidatorID,
			feeKeys:     []*secp256k1.PrivateKey{genesistest.DefaultFundedKeys[0]},
			setup:       addMinStakeValidator,
			AP3Time:     genesistest.DefaultValidatorStartTime,
			expectedErr: ErrPeriodMismatch,
		},
		{
			description: "valid",
			stakeAmount: env.config.MinDelegatorStake,
			startTime:   newValidatorStartTime, // same start time as for primary network
			endTime:     newValidatorEndTime,   // same end time as for primary network
			nodeID:      newValidatorID,
			feeKeys:     []*secp256k1.PrivateKey{genesistest.DefaultFundedKeys[0]},
			setup:       addMinStakeValidator,
			AP3Time:     genesistest.DefaultValidatorStartTime,
			expectedErr: nil,
		},
		{
			description: "starts delegating at current timestamp",
			stakeAmount: env.config.MinDelegatorStake,
			startTime:   uint64(currentTimestamp.Unix()),
			endTime:     genesistest.DefaultValidatorEndTimeUnix,
			nodeID:      nodeID,
			feeKeys:     []*secp256k1.PrivateKey{genesistest.DefaultFundedKeys[0]},
			setup:       nil,
			AP3Time:     genesistest.DefaultValidatorStartTime,
			expectedErr: ErrTimestampNotBeforeStartTime,
		},
		{
			description: "tx fee paying key has no funds",
			stakeAmount: env.config.MinDelegatorStake,
			startTime:   genesistest.DefaultValidatorStartTimeUnix + 1,
			endTime:     genesistest.DefaultValidatorEndTimeUnix,
			nodeID:      nodeID,
			feeKeys:     []*secp256k1.PrivateKey{genesistest.DefaultFundedKeys[1]},
			setup: func(env *environment) { // Remove all UTXOs owned by keys[1]
				utxoIDs, err := env.state.UTXOIDs(
					genesistest.DefaultFundedKeys[1].Address().Bytes(),
					ids.Empty,
					math.MaxInt32)
				require.NoError(t, err)

				for _, utxoID := range utxoIDs {
					env.state.DeleteUTXO(utxoID)
				}
				env.state.SetHeight(dummyHeight)
				require.NoError(t, env.state.Commit())
			},
			AP3Time:     genesistest.DefaultValidatorStartTime,
			expectedErr: ErrFlowCheckFailed,
		},
		{
			description: "over delegation before AP3",
			stakeAmount: env.config.MinDelegatorStake,
			startTime:   newValidatorStartTime, // same start time as for primary network
			endTime:     newValidatorEndTime,   // same end time as for primary network
			nodeID:      newValidatorID,
			feeKeys:     []*secp256k1.PrivateKey{genesistest.DefaultFundedKeys[0]},
			setup:       addMaxStakeValidator,
			AP3Time:     genesistest.DefaultValidatorEndTime,
			expectedErr: nil,
		},
		{
			description: "over delegation after AP3",
			stakeAmount: env.config.MinDelegatorStake,
			startTime:   newValidatorStartTime, // same start time as for primary network
			endTime:     newValidatorEndTime,   // same end time as for primary network
			nodeID:      newValidatorID,
			feeKeys:     []*secp256k1.PrivateKey{genesistest.DefaultFundedKeys[0]},
			setup:       addMaxStakeValidator,
			AP3Time:     genesistest.DefaultValidatorStartTime,
			expectedErr: ErrOverDelegated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			require := require.New(t)
			env := newEnvironment(t, upgradetest.ApricotPhase5)
			env.config.UpgradeConfig.ApricotPhase3Time = tt.AP3Time

			wallet := newWallet(t, env, walletConfig{
				keys: tt.feeKeys,
			})

			tx, err := wallet.IssueAddDelegatorTx(
				&txs.Validator{
					NodeID: tt.nodeID,
					Start:  tt.startTime,
					End:    tt.endTime,
					Wght:   tt.stakeAmount,
				},
				rewardsOwner,
			)
			require.NoError(err)

			if tt.setup != nil {
				tt.setup(env)
			}

			onCommitState, err := state.NewDiff(lastAcceptedID, env)
			require.NoError(err)

			onAbortState, err := state.NewDiff(lastAcceptedID, env)
			require.NoError(err)

			feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
			err = ProposalTx(
				&env.backend,
				feeCalculator,
				tx,
				onCommitState,
				onAbortState,
			)
			require.ErrorIs(err, tt.expectedErr)
		})
	}
}

func TestProposalTxExecuteAddNetValidator(t *testing.T) {
	require := require.New(t)
	env := newEnvironment(t, upgradetest.ApricotPhase5)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	nodeID := genesistest.DefaultNodeIDs[0]
	subnetID := testNet1.ID()
	{
		// Case: Proposed validator currently validating primary network
		// but stops validating net after stops validating primary network
		// (note that keys[0] is a genesis validator)
		wallet := newWallet(t, env, walletConfig{
			netIDs: []ids.ID{subnetID},
		})
		tx, err := wallet.IssueAddChainValidatorTx(
			&txs.ChainValidator{
				Validator: txs.Validator{
					NodeID: nodeID,
					Start:  genesistest.DefaultValidatorStartTimeUnix + 1,
					End:    genesistest.DefaultValidatorEndTimeUnix + 1,
					Wght:   genesistest.DefaultValidatorWeight,
				},
				Chain: subnetID,
			},
		)
		require.NoError(err)

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
		err = ProposalTx(
			&env.backend,
			feeCalculator,
			tx,
			onCommitState,
			onAbortState,
		)
		require.ErrorIs(err, ErrPeriodMismatch)
	}

	{
		// Case: Proposed validator currently validating primary network
		// and proposed net validation period is subset of
		// primary network validation period
		// (note that keys[0] is a genesis validator)
		wallet := newWallet(t, env, walletConfig{
			netIDs: []ids.ID{subnetID},
		})
		tx, err := wallet.IssueAddChainValidatorTx(
			&txs.ChainValidator{
				Validator: txs.Validator{
					NodeID: nodeID,
					Start:  genesistest.DefaultValidatorStartTimeUnix + 1,
					End:    genesistest.DefaultValidatorEndTimeUnix,
					Wght:   genesistest.DefaultValidatorWeight,
				},
				Chain: subnetID,
			},
		)
		require.NoError(err)

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
		require.NoError(ProposalTx(
			&env.backend,
			feeCalculator,
			tx,
			onCommitState,
			onAbortState,
		))
	}

	// Add a validator to pending validator set of primary network
	// Starts validating primary network 10 seconds after genesis
	pendingDSValidatorID := ids.GenerateTestNodeID()
	dsStartTime := genesistest.DefaultValidatorStartTime.Add(10 * time.Second)
	dsEndTime := dsStartTime.Add(5 * defaultMinStakingDuration)

	wallet := newWallet(t, env, walletConfig{
		keys: genesistest.DefaultFundedKeys[:1],
	})
	addDSTx, err := wallet.IssueAddValidatorTx(
		&txs.Validator{
			NodeID: pendingDSValidatorID,
			Start:  uint64(dsStartTime.Unix()),
			End:    uint64(dsEndTime.Unix()),
			Wght:   env.config.MinValidatorStake,
		},
		&secp256k1fx.OutputOwners{
			Threshold: 1,
			Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
		},
		reward.PercentDenominator,
	)
	require.NoError(err)

	{
		// Case: Proposed validator isn't in pending or current validator sets
		wallet := newWallet(t, env, walletConfig{
			netIDs: []ids.ID{subnetID},
		})
		tx, err := wallet.IssueAddChainValidatorTx(
			&txs.ChainValidator{
				Validator: txs.Validator{
					NodeID: pendingDSValidatorID,
					Start:  uint64(dsStartTime.Unix()), // start validating net before primary network
					End:    uint64(dsEndTime.Unix()),
					Wght:   genesistest.DefaultValidatorWeight,
				},
				Chain: subnetID,
			},
		)
		require.NoError(err)

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
		err = ProposalTx(
			&env.backend,
			feeCalculator,
			tx,
			onCommitState,
			onAbortState,
		)
		require.ErrorIs(err, ErrNotValidator)
	}

	addValTx := addDSTx.Unsigned.(*txs.AddValidatorTx)
	staker, err := state.NewCurrentStaker(
		addDSTx.ID(),
		addValTx,
		addValTx.StartTime(),
		0,
	)
	require.NoError(err)

	require.NoError(env.state.PutCurrentValidator(staker))
	env.state.AddTx(addDSTx, status.Committed)
	dummyHeight := uint64(1)
	env.state.SetHeight(dummyHeight)
	require.NoError(env.state.Commit())

	// Node with ID key.Address() now a pending validator for primary network

	{
		// Case: Proposed validator is pending validator of primary network
		// but starts validating subnet before primary network
		wallet := newWallet(t, env, walletConfig{
			netIDs: []ids.ID{subnetID},
		})
		tx, err := wallet.IssueAddChainValidatorTx(
			&txs.ChainValidator{
				Validator: txs.Validator{
					NodeID: pendingDSValidatorID,
					Start:  uint64(dsStartTime.Unix()) - 1, // start validating net before primary network
					End:    uint64(dsEndTime.Unix()),
					Wght:   genesistest.DefaultValidatorWeight,
				},
				Chain: subnetID,
			},
		)
		require.NoError(err)

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
		err = ProposalTx(
			&env.backend,
			feeCalculator,
			tx,
			onCommitState,
			onAbortState,
		)
		require.ErrorIs(err, ErrPeriodMismatch)
	}

	{
		// Case: Proposed validator is pending validator of primary network
		// but stops validating subnet after primary network
		wallet := newWallet(t, env, walletConfig{
			netIDs: []ids.ID{subnetID},
		})
		tx, err := wallet.IssueAddChainValidatorTx(
			&txs.ChainValidator{
				Validator: txs.Validator{
					NodeID: pendingDSValidatorID,
					Start:  uint64(dsStartTime.Unix()),
					End:    uint64(dsEndTime.Unix()) + 1, // stop validating subnet after stopping validating primary network
					Wght:   genesistest.DefaultValidatorWeight,
				},
				Chain: subnetID,
			},
		)
		require.NoError(err)

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
		err = ProposalTx(
			&env.backend,
			feeCalculator,
			tx,
			onCommitState,
			onAbortState,
		)
		require.ErrorIs(err, ErrPeriodMismatch)
	}

	{
		// Case: Proposed validator is pending validator of primary network and
		// period validating subnet is subset of time validating primary network
		wallet := newWallet(t, env, walletConfig{
			netIDs: []ids.ID{subnetID},
		})
		tx, err := wallet.IssueAddChainValidatorTx(
			&txs.ChainValidator{
				Validator: txs.Validator{
					NodeID: pendingDSValidatorID,
					Start:  uint64(dsStartTime.Unix()), // same start time as for primary network
					End:    uint64(dsEndTime.Unix()),   // same end time as for primary network
					Wght:   genesistest.DefaultValidatorWeight,
				},
				Chain: subnetID,
			},
		)
		require.NoError(err)

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
		require.NoError(ProposalTx(
			&env.backend,
			feeCalculator,
			tx,
			onCommitState,
			onAbortState,
		))
	}

	// Case: Proposed validator start validating at/before current timestamp
	// First, advance the timestamp
	newTimestamp := genesistest.DefaultValidatorStartTime.Add(2 * time.Second)
	env.state.SetTimestamp(newTimestamp)

	{
		wallet := newWallet(t, env, walletConfig{
			netIDs: []ids.ID{subnetID},
		})
		tx, err := wallet.IssueAddChainValidatorTx(
			&txs.ChainValidator{
				Validator: txs.Validator{
					NodeID: nodeID,
					Start:  uint64(newTimestamp.Unix()),
					End:    uint64(newTimestamp.Add(defaultMinStakingDuration).Unix()),
					Wght:   genesistest.DefaultValidatorWeight,
				},
				Chain: subnetID,
			},
		)
		require.NoError(err)

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
		err = ProposalTx(
			&env.backend,
			feeCalculator,
			tx,
			onCommitState,
			onAbortState,
		)
		require.ErrorIs(err, ErrTimestampNotBeforeStartTime)
	}

	// reset the timestamp
	env.state.SetTimestamp(genesistest.DefaultValidatorStartTime)

	// Case: Proposed validator already validating the subnet
	// First, add validator as validator of subnet
	wallet = newWallet(t, env, walletConfig{
		netIDs: []ids.ID{subnetID},
	})
	subnetTx, err := wallet.IssueAddChainValidatorTx(
		&txs.ChainValidator{
			Validator: txs.Validator{
				NodeID: nodeID,
				Start:  genesistest.DefaultValidatorStartTimeUnix,
				End:    genesistest.DefaultValidatorEndTimeUnix,
				Wght:   genesistest.DefaultValidatorWeight,
			},
			Chain: subnetID,
		},
	)
	require.NoError(err)

	addNetValTx := subnetTx.Unsigned.(*txs.AddChainValidatorTx)
	staker, err = state.NewCurrentStaker(
		subnetTx.ID(),
		addNetValTx,
		addNetValTx.StartTime(),
		0,
	)
	require.NoError(err)

	require.NoError(env.state.PutCurrentValidator(staker))
	env.state.AddTx(subnetTx, status.Committed)
	env.state.SetHeight(dummyHeight)
	require.NoError(env.state.Commit())

	{
		// Node with ID nodeIDKey.Address() now validating subnet with ID testNet1.ID
		wallet = newWallet(t, env, walletConfig{
			netIDs: []ids.ID{subnetID},
		})
		duplicateNetTx, err := wallet.IssueAddChainValidatorTx(
			&txs.ChainValidator{
				Validator: txs.Validator{
					NodeID: nodeID,
					Start:  genesistest.DefaultValidatorStartTimeUnix + 1,
					End:    genesistest.DefaultValidatorEndTimeUnix,
					Wght:   genesistest.DefaultValidatorWeight,
				},
				Chain: subnetID,
			},
		)
		require.NoError(err)

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
		err = ProposalTx(
			&env.backend,
			feeCalculator,
			duplicateNetTx,
			onCommitState,
			onAbortState,
		)
		require.ErrorIs(err, ErrDuplicateValidator)
	}

	env.state.DeleteCurrentValidator(staker)
	env.state.SetHeight(dummyHeight)
	require.NoError(env.state.Commit())

	{
		// Case: Too few signatures
		wallet = newWallet(t, env, walletConfig{
			netIDs: []ids.ID{subnetID},
		})
		tx, err := wallet.IssueAddChainValidatorTx(
			&txs.ChainValidator{
				Validator: txs.Validator{
					NodeID: nodeID,
					Start:  genesistest.DefaultValidatorStartTimeUnix + 1,
					End:    uint64(genesistest.DefaultValidatorStartTime.Add(defaultMinStakingDuration).Unix()) + 1,
					Wght:   genesistest.DefaultValidatorWeight,
				},
				Chain: subnetID,
			},
		)
		require.NoError(err)

		// Remove a signature
		addNetValidatorTx := tx.Unsigned.(*txs.AddChainValidatorTx)
		input := addNetValidatorTx.ChainAuth.(*secp256k1fx.Input)
		input.SigIndices = input.SigIndices[1:]
		// This tx was syntactically verified when it was created...pretend it wasn't so we don't use cache
		addNetValidatorTx.SyntacticallyVerified = false

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
		err = ProposalTx(
			&env.backend,
			feeCalculator,
			tx,
			onCommitState,
			onAbortState,
		)
		require.ErrorIs(err, errUnauthorizedModification)
	}

	{
		// Case: Control Signature from invalid key (keys[3] is not a control key)
		wallet = newWallet(t, env, walletConfig{
			netIDs: []ids.ID{subnetID},
		})
		tx, err := wallet.IssueAddChainValidatorTx(
			&txs.ChainValidator{
				Validator: txs.Validator{
					NodeID: nodeID,
					Start:  genesistest.DefaultValidatorStartTimeUnix + 1,
					End:    uint64(genesistest.DefaultValidatorStartTime.Add(defaultMinStakingDuration).Unix()) + 1,
					Wght:   genesistest.DefaultValidatorWeight,
				},
				Chain: subnetID,
			},
		)
		require.NoError(err)

		// Replace a valid signature with one from keys[3]
		// The subnet authorization credential is the LAST credential (not first)
		sig, err := genesistest.DefaultFundedKeys[3].SignHash(hashing.ComputeHash256(tx.Unsigned.Bytes()))
		require.NoError(err)
		subnetAuthCredIdx := len(tx.Creds) - 1
		copy(tx.Creds[subnetAuthCredIdx].(*secp256k1fx.Credential).Sigs[0][:], sig)

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
		err = ProposalTx(
			&env.backend,
			feeCalculator,
			tx,
			onCommitState,
			onAbortState,
		)
		require.ErrorIs(err, errUnauthorizedModification)
	}

	{
		// Case: Proposed validator in pending validator set for subnet
		// First, add validator to pending validator set of subnet
		wallet = newWallet(t, env, walletConfig{
			netIDs: []ids.ID{subnetID},
		})
		tx, err := wallet.IssueAddChainValidatorTx(
			&txs.ChainValidator{
				Validator: txs.Validator{
					NodeID: nodeID,
					Start:  genesistest.DefaultValidatorStartTimeUnix + 1,
					End:    uint64(genesistest.DefaultValidatorStartTime.Add(defaultMinStakingDuration).Unix()) + 1,
					Wght:   genesistest.DefaultValidatorWeight,
				},
				Chain: subnetID,
			},
		)
		require.NoError(err)

		addNetValTx := subnetTx.Unsigned.(*txs.AddChainValidatorTx)
		staker, err = state.NewCurrentStaker(
			subnetTx.ID(),
			addNetValTx,
			addNetValTx.StartTime(),
			0,
		)
		require.NoError(err)

		require.NoError(env.state.PutCurrentValidator(staker))
		env.state.AddTx(tx, status.Committed)
		env.state.SetHeight(dummyHeight)
		require.NoError(env.state.Commit())

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
		err = ProposalTx(
			&env.backend,
			feeCalculator,
			tx,
			onCommitState,
			onAbortState,
		)
		require.ErrorIs(err, ErrDuplicateValidator)
	}
}

func TestProposalTxExecuteAddValidator(t *testing.T) {
	require := require.New(t)
	env := newEnvironment(t, upgradetest.ApricotPhase5)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	nodeID := ids.GenerateTestNodeID()
	chainTime := env.state.GetTimestamp()
	rewardsOwner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
	}

	{
		// Case: Validator's start time too early
		wallet := newWallet(t, env, walletConfig{})
		tx, err := wallet.IssueAddValidatorTx(
			&txs.Validator{
				NodeID: nodeID,
				Start:  uint64(chainTime.Unix()),
				End:    genesistest.DefaultValidatorEndTimeUnix,
				Wght:   env.config.MinValidatorStake,
			},
			rewardsOwner,
			reward.PercentDenominator,
		)
		require.NoError(err)

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
		err = ProposalTx(
			&env.backend,
			feeCalculator,
			tx,
			onCommitState,
			onAbortState,
		)
		require.ErrorIs(err, ErrTimestampNotBeforeStartTime)
	}

	{
		nodeID := genesistest.DefaultNodeIDs[0]

		// Case: Validator already validating primary network
		wallet := newWallet(t, env, walletConfig{})
		tx, err := wallet.IssueAddValidatorTx(
			&txs.Validator{
				NodeID: nodeID,
				Start:  genesistest.DefaultValidatorStartTimeUnix + 1,
				End:    genesistest.DefaultValidatorEndTimeUnix,
				Wght:   env.config.MinValidatorStake,
			},
			rewardsOwner,
			reward.PercentDenominator,
		)
		require.NoError(err)

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
		err = ProposalTx(
			&env.backend,
			feeCalculator,
			tx,
			onCommitState,
			onAbortState,
		)
		require.ErrorIs(err, ErrAlreadyValidator)
	}

	{
		// Case: Validator in pending validator set of primary network
		startTime := genesistest.DefaultValidatorStartTime.Add(1 * time.Second)
		wallet := newWallet(t, env, walletConfig{})
		tx, err := wallet.IssueAddValidatorTx(
			&txs.Validator{
				NodeID: nodeID,
				Start:  uint64(startTime.Unix()),
				End:    uint64(startTime.Add(defaultMinStakingDuration).Unix()),
				Wght:   env.config.MinValidatorStake,
			},
			rewardsOwner,
			reward.PercentDenominator,
		)
		require.NoError(err)

		addValTx := tx.Unsigned.(*txs.AddValidatorTx)
		staker, err := state.NewCurrentStaker(
			tx.ID(),
			addValTx,
			addValTx.StartTime(),
			0,
		)
		require.NoError(err)

		require.NoError(env.state.PutPendingValidator(staker))
		env.state.AddTx(tx, status.Committed)
		dummyHeight := uint64(1)
		env.state.SetHeight(dummyHeight)
		require.NoError(env.state.Commit())

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
		err = ProposalTx(
			&env.backend,
			feeCalculator,
			tx,
			onCommitState,
			onAbortState,
		)
		require.ErrorIs(err, ErrAlreadyValidator)
	}

	{
		// Case: Validator doesn't have enough tokens to cover stake amount
		wallet := newWallet(t, env, walletConfig{
			keys: genesistest.DefaultFundedKeys[:1],
		})
		tx, err := wallet.IssueAddValidatorTx(
			&txs.Validator{
				NodeID: ids.GenerateTestNodeID(),
				Start:  genesistest.DefaultValidatorStartTimeUnix + 1,
				End:    genesistest.DefaultValidatorEndTimeUnix,
				Wght:   env.config.MinValidatorStake,
			},
			rewardsOwner,
			reward.PercentDenominator,
		)
		require.NoError(err)

		// Remove all UTXOs owned by preFundedKeys[0]
		utxoIDs, err := env.state.UTXOIDs(genesistest.DefaultFundedKeys[0].Address().Bytes(), ids.Empty, math.MaxInt32)
		require.NoError(err)

		for _, utxoID := range utxoIDs {
			env.state.DeleteUTXO(utxoID)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		require.NoError(err)

		feeCalculator := state.PickFeeCalculator(env.config, onCommitState)
		err = ProposalTx(
			&env.backend,
			feeCalculator,
			tx,
			onCommitState,
			onAbortState,
		)
		require.ErrorIs(err, ErrFlowCheckFailed)
	}
}
