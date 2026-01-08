// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/luxfi/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	"github.com/luxfi/upgrade/upgradetest"
	"github.com/luxfi/const"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/sdk/utils/iterator"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/components/gas"
	"github.com/luxfi/sdk/node/vms/platformvm/block"
	"github.com/luxfi/sdk/node/vms/platformvm/genesis/genesistest"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/status"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/platformvm/txs/executor"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

func TestApricotStandardBlockTimeVerification(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	env := newEnvironment(t, ctrl, upgradetest.ApricotPhase5)

	// setup and store parent block
	// it's a standard block for simplicity
	parentHeight := uint64(2022)

	apricotParentBlk, err := block.NewApricotStandardBlock(
		ids.Empty, // does not matter
		parentHeight,
		nil, // txs do not matter in this test
	)
	require.NoError(err)
	parentID := apricotParentBlk.ID()

	// store parent block, with relevant quantities
	onParentAccept := state.NewMockDiff(ctrl)
	env.blkManager.(*manager).blkIDToState[parentID] = &blockState{
		statelessBlock: apricotParentBlk,
		onAcceptState:  onParentAccept,
	}
	env.blkManager.(*manager).lastAccepted = parentID

	chainTime := env.clk.Time().Truncate(time.Second)
	onParentAccept.EXPECT().GetTimestamp().Return(chainTime).AnyTimes()
	onParentAccept.EXPECT().GetFeeState().Return(gas.State{}).AnyTimes()
	onParentAccept.EXPECT().GetL1ValidatorExcess().Return(gas.Gas(0)).AnyTimes()
	onParentAccept.EXPECT().GetAccruedFees().Return(uint64(0)).AnyTimes()
	onParentAccept.EXPECT().NumActiveL1Validators().Return(0).AnyTimes()
	onParentAccept.EXPECT().GetActiveL1ValidatorsIterator().Return(&iterator.Empty[state.L1Validator]{}, nil).AnyTimes()

	// wrong height
	apricotChildBlk, err := block.NewApricotStandardBlock(
		apricotParentBlk.ID(),
		apricotParentBlk.Height(),
		nil, // txs nulled to simplify test
	)
	require.NoError(err)
	blk := env.blkManager.NewBlock(apricotChildBlk)
	err = blk.Verify(context.Background())
	require.ErrorIs(err, errIncorrectBlockHeight)

	// valid height
	apricotChildBlk, err = block.NewApricotStandardBlock(
		apricotParentBlk.ID(),
		apricotParentBlk.Height()+1,
		nil, // txs nulled to simplify test
	)
	require.NoError(err)
	blk = env.blkManager.NewBlock(apricotChildBlk)
	require.NoError(blk.Verify(context.Background()))
}

func TestBanffStandardBlockTimeVerification(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	env := newEnvironment(t, ctrl, upgradetest.Banff)
	now := env.clk.Time()
	env.clk.Set(now)

	// setup and store parent block
	// it's a standard block for simplicity
	parentTime := now
	parentHeight := uint64(2022)

	banffParentBlk, err := block.NewBanffStandardBlock(
		parentTime,
		ids.Empty, // does not matter
		parentHeight,
		nil, // txs do not matter in this test
	)
	require.NoError(err)
	parentID := banffParentBlk.ID()

	// store parent block, with relevant quantities
	onParentAccept := state.NewMockDiff(ctrl)
	chainTime := env.clk.Time().Truncate(time.Second)
	env.blkManager.(*manager).blkIDToState[parentID] = &blockState{
		statelessBlock: banffParentBlk,
		onAcceptState:  onParentAccept,
		timestamp:      chainTime,
	}
	env.blkManager.(*manager).lastAccepted = parentID

	nextStakerTime := chainTime.Add(executor.SyncBound).Add(-1 * time.Second)

	// store just once current staker to mark next staker time.
	onParentAccept.EXPECT().GetCurrentStakerIterator().DoAndReturn(func() (iterator.Iterator[*state.Staker], error) {
		return iterator.FromSlice(
			&state.Staker{
				NextTime: nextStakerTime,
				Priority: txs.PrimaryNetworkValidatorCurrentPriority,
			},
		), nil
	}).AnyTimes()

	onParentAccept.EXPECT().GetPendingStakerIterator().Return(iterator.Empty[*state.Staker]{}, nil).AnyTimes()
	onParentAccept.EXPECT().GetActiveL1ValidatorsIterator().Return(iterator.Empty[state.L1Validator]{}, nil).AnyTimes()
	onParentAccept.EXPECT().GetExpiryIterator().Return(iterator.Empty[state.ExpiryEntry]{}, nil).AnyTimes()

	onParentAccept.EXPECT().GetTimestamp().Return(chainTime).AnyTimes()
	onParentAccept.EXPECT().GetFeeState().Return(gas.State{}).AnyTimes()
	onParentAccept.EXPECT().GetL1ValidatorExcess().Return(gas.Gas(0)).AnyTimes()
	onParentAccept.EXPECT().GetAccruedFees().Return(uint64(0)).AnyTimes()
	onParentAccept.EXPECT().NumActiveL1Validators().Return(0).AnyTimes()

	txID := ids.GenerateTestID()
	utxo := &lux.UTXO{
		UTXOID: lux.UTXOID{
			TxID: txID,
		},
		Asset: lux.Asset{
			ID: env.ctx.XAssetID,
		},
		Out: &secp256k1fx.TransferOutput{
			Amt: 1,
		},
	}
	utxoID := utxo.InputID()
	onParentAccept.EXPECT().GetUTXO(utxoID).Return(utxo, nil).AnyTimes()

	// Create the tx
	utx := &txs.CreateSubnetTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    env.ctx.NetworkID,
			BlockchainID: env.ctx.ChainID,
			Ins: []*lux.TransferableInput{{
				UTXOID: utxo.UTXOID,
				Asset:  utxo.Asset,
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
		}},
		Owner: &secp256k1fx.OutputOwners{},
	}
	tx := &txs.Tx{Unsigned: utx}
	require.NoError(tx.Sign(txs.Codec, [][]*secp256k1.PrivateKey{{}}))

	{
		// wrong version
		banffChildBlk, err := block.NewApricotStandardBlock(
			banffParentBlk.ID(),
			banffParentBlk.Height()+1,
			[]*txs.Tx{tx},
		)
		require.NoError(err)
		block := env.blkManager.NewBlock(banffChildBlk)
		err = block.Verify(context.Background())
		require.ErrorIs(err, errApricotBlockIssuedAfterFork)
	}

	{
		// wrong height
		childTimestamp := parentTime.Add(time.Second)
		banffChildBlk, err := block.NewBanffStandardBlock(
			childTimestamp,
			banffParentBlk.ID(),
			banffParentBlk.Height(),
			[]*txs.Tx{tx},
		)
		require.NoError(err)
		block := env.blkManager.NewBlock(banffChildBlk)
		err = block.Verify(context.Background())
		require.ErrorIs(err, errIncorrectBlockHeight)
	}

	{
		// wrong timestamp, earlier than parent
		childTimestamp := parentTime.Add(-1 * time.Second)
		banffChildBlk, err := block.NewBanffStandardBlock(
			childTimestamp,
			banffParentBlk.ID(),
			banffParentBlk.Height()+1,
			[]*txs.Tx{tx},
		)
		require.NoError(err)
		block := env.blkManager.NewBlock(banffChildBlk)
		err = block.Verify(context.Background())
		require.ErrorIs(err, executor.ErrChildBlockEarlierThanParent)
	}

	{
		// wrong timestamp, violated synchrony bound
		initClkTime := env.clk.Time()
		env.clk.Set(parentTime.Add(-executor.SyncBound))
		banffChildBlk, err := block.NewBanffStandardBlock(
			parentTime.Add(time.Second),
			banffParentBlk.ID(),
			banffParentBlk.Height()+1,
			[]*txs.Tx{tx},
		)
		require.NoError(err)
		block := env.blkManager.NewBlock(banffChildBlk)
		err = block.Verify(context.Background())
		require.ErrorIs(err, executor.ErrChildBlockBeyondSyncBound)
		env.clk.Set(initClkTime)
	}

	{
		// wrong timestamp, skipped staker set change event
		childTimestamp := nextStakerTime.Add(time.Second)
		banffChildBlk, err := block.NewBanffStandardBlock(
			childTimestamp,
			banffParentBlk.ID(),
			banffParentBlk.Height()+1,
			[]*txs.Tx{tx},
		)
		require.NoError(err)
		block := env.blkManager.NewBlock(banffChildBlk)
		err = block.Verify(context.Background())
		require.ErrorIs(err, executor.ErrChildBlockAfterStakerChangeTime)
	}

	{
		// no state changes
		childTimestamp := parentTime
		banffChildBlk, err := block.NewBanffStandardBlock(
			childTimestamp,
			banffParentBlk.ID(),
			banffParentBlk.Height()+1,
			nil,
		)
		require.NoError(err)
		block := env.blkManager.NewBlock(banffChildBlk)
		err = block.Verify(context.Background())
		require.ErrorIs(err, ErrStandardBlockWithoutChanges)
	}

	{
		// valid block, same timestamp as parent block
		childTimestamp := parentTime
		banffChildBlk, err := block.NewBanffStandardBlock(
			childTimestamp,
			banffParentBlk.ID(),
			banffParentBlk.Height()+1,
			[]*txs.Tx{tx},
		)
		require.NoError(err)
		block := env.blkManager.NewBlock(banffChildBlk)
		require.NoError(block.Verify(context.Background()))
	}

	{
		// valid
		childTimestamp := nextStakerTime
		banffChildBlk, err := block.NewBanffStandardBlock(
			childTimestamp,
			banffParentBlk.ID(),
			banffParentBlk.Height()+1,
			[]*txs.Tx{tx},
		)
		require.NoError(err)
		block := env.blkManager.NewBlock(banffChildBlk)
		require.NoError(block.Verify(context.Background()))
	}
}

func TestBanffStandardBlockUpdatePrimaryNetworkStakers(t *testing.T) {
	require := require.New(t)

	env := newEnvironment(t, nil, upgradetest.Banff)

	// Case: Timestamp is after next validator start time
	// Add a pending validator
	pendingValidatorStartTime := genesistest.DefaultValidatorStartTime.Add(1 * time.Second)
	pendingValidatorEndTime := pendingValidatorStartTime.Add(defaultMinStakingDuration)
	nodeID := ids.GenerateTestNodeID()
	rewardAddress := ids.GenerateTestShortID()
	addPendingValidatorTx := addPendingValidator(
		t,
		env,
		pendingValidatorStartTime,
		pendingValidatorEndTime,
		nodeID,
		rewardAddress,
		[]*secp256k1.PrivateKey{genesistest.DefaultFundedKeys[0]},
	)

	// build standard block moving ahead chain time
	preferredID := env.state.GetLastAccepted()
	parentBlk, err := env.state.GetStatelessBlock(preferredID)
	require.NoError(err)
	statelessStandardBlock, err := block.NewBanffStandardBlock(
		pendingValidatorStartTime,
		parentBlk.ID(),
		parentBlk.Height()+1,
		nil, // txs nulled to simplify test
	)
	require.NoError(err)
	block := env.blkManager.NewBlock(statelessStandardBlock)

	// update staker set
	require.NoError(block.Verify(context.Background()))

	// tests
	blkStateMap := env.blkManager.(*manager).blkIDToState
	updatedState := blkStateMap[block.ID()].onAcceptState
	currentValidator, err := updatedState.GetCurrentValidator(constants.PrimaryNetworkID, nodeID)
	require.NoError(err)
	require.Equal(addPendingValidatorTx.ID(), currentValidator.TxID)
	require.Equal(uint64(1), currentValidator.PotentialReward) // With 6 decimal precision, small stake rounds to 1 μLUX

	_, err = updatedState.GetPendingValidator(constants.PrimaryNetworkID, nodeID)
	require.ErrorIs(err, database.ErrNotFound)

	// Test VM validators
	require.NoError(block.Accept(context.Background()))
	_, ok := env.config.Validators.GetValidator(constants.PrimaryNetworkID, nodeID)
	require.True(ok)
}

// Ensure semantic verification updates the current and pending staker sets correctly.
// Namely, it should add pending stakers whose start time is at or before the timestamp.
// It will not remove primary network stakers; that happens in rewardTxs.
func TestBanffStandardBlockUpdateStakers(t *testing.T) {
	// Chronological order (not in scale):
	// Staker1:    |----------------------------------------------------------|
	// Staker2:        |------------------------|
	// Staker3:            |------------------------|
	// Staker3sub:             |----------------|
	// Staker4:            |------------------------|
	// Staker5:                                 |--------------------|

	// In this test multiple stakers may join and leave the staker set at the same time.
	// The order in which they do it is asserted; the order may depend on the staker.TxID,
	// which in turns depend on every feature of the transaction creating the staker.
	// So in this test we avoid ids.GenerateTestNodeID, in favour of ids.BuildTestNodeID
	// so that TxID does not depend on the order we run tests.
	staker1 := staker{
		nodeID:        ids.BuildTestNodeID([]byte{0xf1}),
		rewardAddress: ids.ShortID{0xf1},
		startTime:     genesistest.DefaultValidatorStartTime.Add(1 * time.Minute),
		endTime:       genesistest.DefaultValidatorStartTime.Add(10 * defaultMinStakingDuration).Add(1 * time.Minute),
	}
	staker2 := staker{
		nodeID:        ids.BuildTestNodeID([]byte{0xf2}),
		rewardAddress: ids.ShortID{0xf2},
		startTime:     staker1.startTime.Add(1 * time.Minute),
		endTime:       staker1.startTime.Add(1 * time.Minute).Add(defaultMinStakingDuration),
	}
	staker3 := staker{
		nodeID:        ids.BuildTestNodeID([]byte{0xf3}),
		rewardAddress: ids.ShortID{0xf3},
		startTime:     staker2.startTime.Add(1 * time.Minute),
		endTime:       staker2.endTime.Add(1 * time.Minute),
	}
	staker3Sub := staker{
		nodeID:        ids.BuildTestNodeID([]byte{0xf3}),
		rewardAddress: ids.ShortID{0xff},
		startTime:     staker3.startTime.Add(1 * time.Minute),
		endTime:       staker3.endTime.Add(-1 * time.Minute),
	}
	staker4 := staker{
		nodeID:        ids.BuildTestNodeID([]byte{0xf4}),
		rewardAddress: ids.ShortID{0xf4},
		startTime:     staker3.startTime,
		endTime:       staker3.endTime,
	}
	staker5 := staker{
		nodeID:        ids.BuildTestNodeID([]byte{0xf5}),
		rewardAddress: ids.ShortID{0xf5},
		startTime:     staker2.endTime,
		endTime:       staker2.endTime.Add(defaultMinStakingDuration),
	}

	tests := []test{
		{
			description:   "advance time to staker 1 start with subnet",
			stakers:       []staker{staker1, staker2, staker3, staker4, staker5},
			subnetStakers: []staker{staker1},
			advanceTimeTo: []time.Time{staker1.startTime},
			expectedStakers: map[ids.NodeID]stakerStatus{
				staker1.nodeID: current,
				staker2.nodeID: pending,
				staker3.nodeID: pending,
				staker4.nodeID: pending,
				staker5.nodeID: pending,
			},
			expectedNetStakers: map[ids.NodeID]stakerStatus{
				staker1.nodeID: current,
				staker2.nodeID: pending,
				staker3.nodeID: pending,
				staker4.nodeID: pending,
				staker5.nodeID: pending,
			},
		},
		{
			description:   "advance time to the staker2 start",
			stakers:       []staker{staker1, staker2, staker3, staker4, staker5},
			advanceTimeTo: []time.Time{staker1.startTime, staker2.startTime},
			expectedStakers: map[ids.NodeID]stakerStatus{
				staker1.nodeID: current,
				staker2.nodeID: current,
				staker3.nodeID: pending,
				staker4.nodeID: pending,
				staker5.nodeID: pending,
			},
		},
		{
			description:   "staker3 should validate only primary network",
			stakers:       []staker{staker1, staker2, staker3, staker4, staker5},
			subnetStakers: []staker{staker1, staker2, staker3Sub, staker4, staker5},
			advanceTimeTo: []time.Time{staker1.startTime, staker2.startTime, staker3.startTime},
			expectedStakers: map[ids.NodeID]stakerStatus{
				staker1.nodeID: current,
				staker2.nodeID: current,
				staker3.nodeID: current,
				staker4.nodeID: current,
				staker5.nodeID: pending,
			},
			expectedNetStakers: map[ids.NodeID]stakerStatus{
				staker1.nodeID:    current,
				staker2.nodeID:    current,
				staker3Sub.nodeID: pending,
				staker4.nodeID:    current,
				staker5.nodeID:    pending,
			},
		},
		{
			description:   "advance time to staker3 start with subnet",
			stakers:       []staker{staker1, staker2, staker3, staker4, staker5},
			subnetStakers: []staker{staker1, staker2, staker3Sub, staker4, staker5},
			advanceTimeTo: []time.Time{staker1.startTime, staker2.startTime, staker3.startTime, staker3Sub.startTime},
			expectedStakers: map[ids.NodeID]stakerStatus{
				staker1.nodeID: current,
				staker2.nodeID: current,
				staker3.nodeID: current,
				staker4.nodeID: current,
				staker5.nodeID: pending,
			},
			expectedNetStakers: map[ids.NodeID]stakerStatus{
				staker1.nodeID: current,
				staker2.nodeID: current,
				staker3.nodeID: current,
				staker4.nodeID: current,
				staker5.nodeID: pending,
			},
		},
		{
			description:   "advance time to staker5 start",
			stakers:       []staker{staker1, staker2, staker3, staker4, staker5},
			advanceTimeTo: []time.Time{staker1.startTime, staker2.startTime, staker3.startTime, staker5.startTime},
			expectedStakers: map[ids.NodeID]stakerStatus{
				staker1.nodeID: current,

				// Staker2's end time matches staker5's start time, so typically
				// the block builder would produce a ProposalBlock to remove
				// staker2 when advancing the time. However, it is valid to only
				// advance the time with a StandardBlock and not remove staker2,
				// which is what this test does.
				staker2.nodeID: current,
				staker3.nodeID: current,
				staker4.nodeID: current,
				staker5.nodeID: current,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			require := require.New(t)
			env := newEnvironment(t, nil, upgradetest.Banff)

			netID := testNet1.ID()
			env.config.TrackedChains.Add(netID)

			for _, staker := range test.stakers {
				addPendingValidator(
					t,
					env,
					staker.startTime,
					staker.endTime,
					staker.nodeID,
					staker.rewardAddress,
					[]*secp256k1.PrivateKey{genesistest.DefaultFundedKeys[0]},
				)
			}

			for _, staker := range test.subnetStakers {
				wallet := newWallet(t, env, walletConfig{
					subnetIDs: []ids.ID{netID},
				})

				tx, err := wallet.IssueAddChainValidatorTx(
					&txs.ChainValidator{
						Validator: txs.Validator{
							NodeID: staker.nodeID,
							Start:  uint64(staker.startTime.Unix()),
							End:    uint64(staker.endTime.Unix()),
							Wght:   10,
						},
						Chain: netID,
					},
				)
				require.NoError(err)

				staker, err := state.NewPendingStaker(
					tx.ID(),
					tx.Unsigned.(*txs.AddChainValidatorTx),
				)
				require.NoError(err)

				require.NoError(env.state.PutPendingValidator(staker))
				env.state.AddTx(tx, status.Committed)
			}
			env.state.SetHeight( /*dummyHeight*/ 1)
			require.NoError(env.state.Commit())

			for _, newTime := range test.advanceTimeTo {
				env.clk.Set(newTime)

				// build standard block moving ahead chain time
				preferredID := env.state.GetLastAccepted()
				parentBlk, err := env.state.GetStatelessBlock(preferredID)
				require.NoError(err)
				statelessStandardBlock, err := block.NewBanffStandardBlock(
					newTime,
					parentBlk.ID(),
					parentBlk.Height()+1,
					nil, // txs nulled to simplify test
				)
				block := env.blkManager.NewBlock(statelessStandardBlock)

				require.NoError(err)

				// update staker set
				require.NoError(block.Verify(context.Background()))
				require.NoError(block.Accept(context.Background()))

				// Commit state after accepting blocks so they persist for next iteration
				require.NoError(env.state.Commit())
			}

			for stakerNodeID, status := range test.expectedStakers {
				switch status {
				case pending:
					_, err := env.state.GetPendingValidator(constants.PrimaryNetworkID, stakerNodeID)
					require.NoError(err)
					_, ok := env.config.Validators.GetValidator(constants.PrimaryNetworkID, stakerNodeID)
					require.False(ok)
				case current:
					_, err := env.state.GetCurrentValidator(constants.PrimaryNetworkID, stakerNodeID)
					require.NoError(err)
					_, ok := env.config.Validators.GetValidator(constants.PrimaryNetworkID, stakerNodeID)
					require.True(ok)
				}
			}

			for stakerNodeID, status := range test.expectedNetStakers {
				switch status {
				case pending:
					_, ok := env.config.Validators.GetValidator(netID, stakerNodeID)
					require.False(ok)
				case current:
					_, ok := env.config.Validators.GetValidator(netID, stakerNodeID)
					require.True(ok)
				}
			}
		})
	}
}

// Regression test for https://github.com/luxfi/sdk/node/pull/584
// that ensures it fixes a bug where net validators are not removed
// when timestamp is advanced and there is a pending staker whose start time
// is after the new timestamp
func TestBanffStandardBlockRemoveNetValidator(t *testing.T) {
	require := require.New(t)
	env := newEnvironment(t, nil, upgradetest.Banff)

	netID := testNet1.ID()
	env.config.TrackedChains.Add(netID)

	wallet := newWallet(t, env, walletConfig{
		subnetIDs: []ids.ID{netID},
	})

	// Add a subnet validator to the staker set
	subnetValidatorNodeID := genesistest.DefaultNodeIDs[0]
	subnetVdr1EndTime := genesistest.DefaultValidatorStartTime.Add(defaultMinStakingDuration)
	tx, err := wallet.IssueAddChainValidatorTx(
		&txs.ChainValidator{
			Validator: txs.Validator{
				NodeID: subnetValidatorNodeID,
				Start:  genesistest.DefaultValidatorStartTimeUnix,
				End:    uint64(subnetVdr1EndTime.Unix()),
				Wght:   1,
			},
			Chain: netID,
		},
	)
	require.NoError(err)

	addNetValTx := tx.Unsigned.(*txs.AddChainValidatorTx)
	staker, err := state.NewCurrentStaker(
		tx.ID(),
		addNetValTx,
		addNetValTx.StartTime(),
		0,
	)
	require.NoError(err)

	require.NoError(env.state.PutCurrentValidator(staker))
	env.state.AddTx(tx, status.Committed)
	require.NoError(env.state.Commit())

	// The above validator is now part of the staking set

	// Queue a staker that joins the staker set after the above validator leaves
	subnetVdr2NodeID := genesistest.DefaultNodeIDs[1]
	tx, err = wallet.IssueAddChainValidatorTx(
		&txs.ChainValidator{
			Validator: txs.Validator{
				NodeID: subnetVdr2NodeID,
				Start:  uint64(subnetVdr1EndTime.Add(time.Second).Unix()),
				End:    uint64(subnetVdr1EndTime.Add(time.Second).Add(defaultMinStakingDuration).Unix()),
				Wght:   1,
			},
			Chain: netID,
		},
	)
	require.NoError(err)

	staker, err = state.NewPendingStaker(
		tx.ID(),
		tx.Unsigned.(*txs.AddChainValidatorTx),
	)
	require.NoError(err)

	require.NoError(env.state.PutPendingValidator(staker))
	env.state.AddTx(tx, status.Committed)
	require.NoError(env.state.Commit())

	// The above validator is now in the pending staker set

	// Advance time to the first staker's end time.
	env.clk.Set(subnetVdr1EndTime)
	// build standard block moving ahead chain time
	preferredID := env.state.GetLastAccepted()
	parentBlk, err := env.state.GetStatelessBlock(preferredID)
	require.NoError(err)
	statelessStandardBlock, err := block.NewBanffStandardBlock(
		subnetVdr1EndTime,
		parentBlk.ID(),
		parentBlk.Height()+1,
		nil, // txs nulled to simplify test
	)
	require.NoError(err)
	block := env.blkManager.NewBlock(statelessStandardBlock)

	// update staker set
	require.NoError(block.Verify(context.Background()))

	blkStateMap := env.blkManager.(*manager).blkIDToState
	updatedState := blkStateMap[block.ID()].onAcceptState
	_, err = updatedState.GetCurrentValidator(netID, subnetValidatorNodeID)
	require.ErrorIs(err, database.ErrNotFound)

	// Check VM Validators are removed successfully
	require.NoError(block.Accept(context.Background()))
	_, ok := env.config.Validators.GetValidator(netID, subnetVdr2NodeID)
	require.False(ok)
	_, ok = env.config.Validators.GetValidator(netID, subnetValidatorNodeID)
	require.False(ok)
}

func TestBanffStandardBlockTrackedNet(t *testing.T) {
	for _, tracked := range []bool{true, false} {
		t.Run(fmt.Sprintf("tracked %t", tracked), func(t *testing.T) {
			require := require.New(t)
			env := newEnvironment(t, nil, upgradetest.Banff)

			netID := testNet1.ID()
			if tracked {
				env.config.TrackedChains.Add(netID)
			}

			wallet := newWallet(t, env, walletConfig{
				subnetIDs: []ids.ID{netID},
			})

			// Add a subnet validator to the staker set
			subnetValidatorNodeID := genesistest.DefaultNodeIDs[0]
			subnetVdr1StartTime := genesistest.DefaultValidatorStartTime.Add(1 * time.Minute)
			subnetVdr1EndTime := genesistest.DefaultValidatorStartTime.Add(10 * defaultMinStakingDuration).Add(1 * time.Minute)
			tx, err := wallet.IssueAddChainValidatorTx(
				&txs.ChainValidator{
					Validator: txs.Validator{
						NodeID: subnetValidatorNodeID,
						Start:  uint64(subnetVdr1StartTime.Unix()),
						End:    uint64(subnetVdr1EndTime.Unix()),
						Wght:   1,
					},
					Chain: netID,
				},
			)
			require.NoError(err)

			staker, err := state.NewPendingStaker(
				tx.ID(),
				tx.Unsigned.(*txs.AddChainValidatorTx),
			)
			require.NoError(err)

			require.NoError(env.state.PutPendingValidator(staker))
			env.state.AddTx(tx, status.Committed)
			require.NoError(env.state.Commit())

			// Advance time to the staker's start time.
			env.clk.Set(subnetVdr1StartTime)

			// build standard block moving ahead chain time
			preferredID := env.state.GetLastAccepted()
			parentBlk, err := env.state.GetStatelessBlock(preferredID)
			require.NoError(err)
			statelessStandardBlock, err := block.NewBanffStandardBlock(
				subnetVdr1StartTime,
				parentBlk.ID(),
				parentBlk.Height()+1,
				nil, // txs nulled to simplify test
			)
			require.NoError(err)
			block := env.blkManager.NewBlock(statelessStandardBlock)

			// update staker set
			require.NoError(block.Verify(context.Background()))
			require.NoError(block.Accept(context.Background()))
			_, ok := env.config.Validators.GetValidator(netID, subnetValidatorNodeID)
			require.True(ok)
		})
	}
}

func TestBanffStandardBlockDelegatorStakerWeight(t *testing.T) {
	require := require.New(t)
	env := newEnvironment(t, nil, upgradetest.Banff)

	// Case: Timestamp is after next validator start time
	// Add a pending validator
	pendingValidatorStartTime := genesistest.DefaultValidatorStartTime.Add(1 * time.Second)
	pendingValidatorEndTime := pendingValidatorStartTime.Add(defaultMaxStakingDuration)
	nodeID := ids.GenerateTestNodeID()
	rewardAddress := ids.GenerateTestShortID()
	addPendingValidator(
		t,
		env,
		pendingValidatorStartTime,
		pendingValidatorEndTime,
		nodeID,
		rewardAddress,
		[]*secp256k1.PrivateKey{genesistest.DefaultFundedKeys[0]},
	)

	// build standard block moving ahead chain time
	preferredID := env.state.GetLastAccepted()
	parentBlk, err := env.state.GetStatelessBlock(preferredID)
	require.NoError(err)
	statelessStandardBlock, err := block.NewBanffStandardBlock(
		pendingValidatorStartTime,
		parentBlk.ID(),
		parentBlk.Height()+1,
		nil, // txs nulled to simplify test
	)
	require.NoError(err)
	blk := env.blkManager.NewBlock(statelessStandardBlock)
	require.NoError(blk.Verify(context.Background()))
	require.NoError(blk.Accept(context.Background()))
	require.NoError(env.state.Commit())

	// Test validator weight before delegation
	vdrWeight := env.config.Validators.GetWeight(constants.PrimaryNetworkID, nodeID)
	require.Equal(env.config.MinValidatorStake, vdrWeight)

	wallet := newWallet(t, env, walletConfig{})

	// Add delegator
	pendingDelegatorStartTime := pendingValidatorStartTime.Add(1 * time.Second)
	pendingDelegatorEndTime := pendingDelegatorStartTime.Add(1 * time.Second)

	addDelegatorTx, err := wallet.IssueAddDelegatorTx(
		&txs.Validator{
			NodeID: nodeID,
			Start:  uint64(pendingDelegatorStartTime.Unix()),
			End:    uint64(pendingDelegatorEndTime.Unix()),
			Wght:   env.config.MinDelegatorStake,
		},
		&secp256k1fx.OutputOwners{
			Threshold: 1,
			Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
		},
	)
	require.NoError(err)

	staker, err := state.NewPendingStaker(
		addDelegatorTx.ID(),
		addDelegatorTx.Unsigned.(*txs.AddDelegatorTx),
	)
	require.NoError(err)

	env.state.PutPendingDelegator(staker)
	env.state.AddTx(addDelegatorTx, status.Committed)
	env.state.SetHeight( /*dummyHeight*/ uint64(1))
	require.NoError(env.state.Commit())

	// Advance Time
	preferredID = env.state.GetLastAccepted()
	parentBlk, err = env.state.GetStatelessBlock(preferredID)
	require.NoError(err)
	statelessStandardBlock, err = block.NewBanffStandardBlock(
		pendingDelegatorStartTime,
		parentBlk.ID(),
		parentBlk.Height()+1,
		nil, // txs nulled to simplify test
	)
	require.NoError(err)
	blk = env.blkManager.NewBlock(statelessStandardBlock)
	require.NoError(blk.Verify(context.Background()))
	require.NoError(blk.Accept(context.Background()))
	require.NoError(env.state.Commit())

	// Test validator weight after delegation
	vdrWeight = env.config.Validators.GetWeight(constants.PrimaryNetworkID, nodeID)
	require.Equal(env.config.MinDelegatorStake+env.config.MinValidatorStake, vdrWeight)
}
