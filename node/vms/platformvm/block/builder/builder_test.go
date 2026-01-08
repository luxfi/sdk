// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luxfi/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/upgrade/upgradetest"
	"github.com/luxfi/const"
	"github.com/luxfi/crypto/bls/signer/localsigner"
	"github.com/luxfi/sdk/utils/iterator"
	"github.com/luxfi/sdk/utils/timer/mockable"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/vm/chain"
	"github.com/luxfi/sdk/node/vms/platformvm/block"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
	"github.com/luxfi/sdk/node/vms/platformvm/signer"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"

	blockexecutor "github.com/luxfi/sdk/node/vms/platformvm/block/executor"
	txexecutor "github.com/luxfi/sdk/node/vms/platformvm/txs/executor"
)

func TestBuildBlockBasic(t *testing.T) {
	require := require.New(t)

	env := newEnvironment(t, upgradetest.Latest)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	subnetID := testNet1.ID()
	wallet := newWallet(t, env, walletConfig{
		netIDs: []ids.ID{subnetID},
	})

	// Create a valid transaction
	tx, err := wallet.IssueCreateChainTx(
		subnetID,
		nil,
		constants.XVMID,
		nil,
		"chain name",
	)
	require.NoError(err)

	// Issue the transaction
	env.ctx.Lock.Unlock()
	require.NoError(env.network.IssueTxFromRPC(tx))
	env.ctx.Lock.Lock()

	txID := tx.ID()
	_, ok := env.mempool.Get(txID)
	require.True(ok)

	// [BuildBlock] should build a block with the transaction
	blkIntf, err := env.Builder.BuildBlock(context.Background())
	require.NoError(err)

	require.IsType(&blockexecutor.Block{}, blkIntf)
	blk := blkIntf.(*blockexecutor.Block)
	require.Len(blk.Txs(), 1)
	require.Equal(txID, blk.Txs()[0].ID())

	// Mempool should not contain the transaction or have marked it as dropped
	_, ok = env.mempool.Get(txID)
	require.False(ok)
	require.NoError(env.mempool.GetDropReason(txID))
}

func TestBuildBlockDoesNotBuildWithEmptyMempool(t *testing.T) {
	require := require.New(t)

	env := newEnvironment(t, upgradetest.Latest)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	tx, exists := env.mempool.Peek()
	require.False(exists)
	require.Nil(tx)

	// [BuildBlock] should not build an empty block
	blk, err := env.Builder.BuildBlock(context.Background())
	require.ErrorIs(err, ErrNoPendingBlocks)
	require.Nil(blk)
}

func TestBuildBlockShouldReward(t *testing.T) {
	require := require.New(t)

	env := newEnvironment(t, upgradetest.Latest)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	wallet := newWallet(t, env, walletConfig{})

	var (
		now    = env.backend.Clk.Time()
		nodeID = ids.GenerateTestNodeID()

		defaultValidatorStake = 100 * units.MilliLux
		validatorStartTime    = now.Add(2 * txexecutor.SyncBound)
		validatorEndTime      = validatorStartTime.Add(360 * 24 * time.Hour)
	)

	sk, err := localsigner.New()
	require.NoError(err)
	pop, err := signer.NewProofOfPossession(sk)
	require.NoError(err)

	rewardOwners := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
	}

	// Create a valid [AddPermissionlessValidatorTx]
	tx, err := wallet.IssueAddPermissionlessValidatorTx(
		&txs.ChainValidator{
			Validator: txs.Validator{
				NodeID: nodeID,
				Start:  uint64(validatorStartTime.Unix()),
				End:    uint64(validatorEndTime.Unix()),
				Wght:   defaultValidatorStake,
			},
			Chain: constants.PrimaryNetworkID,
		},
		pop,
		env.ctx.XAssetID,
		rewardOwners,
		rewardOwners,
		reward.PercentDenominator,
	)
	require.NoError(err)

	// Issue the transaction
	env.ctx.Lock.Unlock()
	require.NoError(env.network.IssueTxFromRPC(tx))
	env.ctx.Lock.Lock()

	txID := tx.ID()
	_, ok := env.mempool.Get(txID)
	require.True(ok)

	// Build and accept a block with the tx
	blk, err := env.Builder.BuildBlock(context.Background())
	require.NoError(err)
	require.IsType(&block.BanffStandardBlock{}, blk.(*blockexecutor.Block).Block)
	require.Equal([]*txs.Tx{tx}, blk.(*blockexecutor.Block).Block.Txs())
	require.NoError(blk.Verify(context.Background()))
	require.NoError(blk.Accept(context.Background()))
	// Commit parent state to avoid "missing parent state" error
	require.NoError(env.state.Commit())
	env.blkManager.SetPreference(blk.ID())

	// Validator should now be current
	staker, err := env.state.GetCurrentValidator(constants.PrimaryNetworkID, nodeID)
	require.NoError(err)
	require.Equal(txID, staker.TxID)

	// Should be rewarded at the end of staking period
	env.backend.Clk.Set(validatorEndTime)

	for {
		iter, err := env.state.GetCurrentStakerIterator()
		require.NoError(err)
		require.True(iter.Next())
		staker := iter.Value()
		iter.Release()

		// Check that the right block was built
		blk, err := env.Builder.BuildBlock(context.Background())
		require.NoError(err)
		require.NoError(blk.Verify(context.Background()))
		require.IsType(&block.BanffProposalBlock{}, blk.(*blockexecutor.Block).Block)

		expectedTx, err := NewRewardValidatorTx(env.ctx, staker.TxID)
		require.NoError(err)
		require.Equal([]*txs.Tx{expectedTx}, blk.(*blockexecutor.Block).Block.Txs())

		// Proposal blocks require their commit/abort option to be accepted
		// Get the commit option and accept both proposal and commit blocks
		options, err := blk.(chain.OracleBlock).Options(context.Background())
		require.NoError(err)
		commitBlk := options[0]
		require.NoError(commitBlk.Verify(context.Background()))
		
		// Accept both proposal and commit blocks
		require.NoError(blk.Accept(context.Background()))
		require.NoError(commitBlk.Accept(context.Background()))
		
		// Commit state after accepting
		require.NoError(env.state.Commit())
		commitBlkID := commitBlk.ID()
		env.blkManager.SetPreference(commitBlkID)

		// Stop rewarding once our staker is rewarded
		if staker.TxID == txID {
			break
		}
	}

	// Staking rewards should have been issued
	rewardUTXOs, err := env.state.GetRewardUTXOs(txID)
	require.NoError(err)
	require.NotEmpty(rewardUTXOs)
}

func TestBuildBlockAdvanceTime(t *testing.T) {
	require := require.New(t)

	env := newEnvironment(t, upgradetest.Latest)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	var (
		now      = env.backend.Clk.Time()
		nextTime = now.Add(2 * txexecutor.SyncBound)
	)

	// Add a staker to [env.state]
	require.NoError(env.state.PutCurrentValidator(&state.Staker{
		NextTime: nextTime,
		Priority: txs.PrimaryNetworkValidatorCurrentPriority,
	}))

	// Advance wall clock to [nextTime]
	env.backend.Clk.Set(nextTime)

	// [BuildBlock] should build a block advancing the time to [NextTime]
	blkIntf, err := env.Builder.BuildBlock(context.Background())
	require.NoError(err)

	require.IsType(&blockexecutor.Block{}, blkIntf)
	blk := blkIntf.(*blockexecutor.Block)
	require.Empty(blk.Txs())
	require.IsType(&block.BanffStandardBlock{}, blk.Block)
	standardBlk := blk.Block.(*block.BanffStandardBlock)
	require.Equal(nextTime.Unix(), standardBlk.Timestamp().Unix())
}

func TestBuildBlockForceAdvanceTime(t *testing.T) {
	require := require.New(t)

	env := newEnvironment(t, upgradetest.Latest)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	subnetID := testNet1.ID()
	wallet := newWallet(t, env, walletConfig{
		netIDs: []ids.ID{subnetID},
	})

	// Create a valid transaction
	tx, err := wallet.IssueCreateChainTx(
		subnetID,
		nil,
		constants.XVMID,
		nil,
		"chain name",
	)
	require.NoError(err)

	// Issue the transaction
	env.ctx.Lock.Unlock()
	require.NoError(env.network.IssueTxFromRPC(tx))
	env.ctx.Lock.Lock()

	txID := tx.ID()
	_, ok := env.mempool.Get(txID)
	require.True(ok)

	var (
		now      = env.backend.Clk.Time()
		nextTime = now.Add(2 * txexecutor.SyncBound)
	)

	// Add a staker to [env.state]
	require.NoError(env.state.PutCurrentValidator(&state.Staker{
		NextTime: nextTime,
		Priority: txs.PrimaryNetworkValidatorCurrentPriority,
	}))

	// Advance wall clock to [nextTime] + [txexecutor.SyncBound]
	env.backend.Clk.Set(nextTime.Add(txexecutor.SyncBound))

	// [BuildBlock] should build a block advancing the time to [nextTime],
	// not the current wall clock.
	blkIntf, err := env.Builder.BuildBlock(context.Background())
	require.NoError(err)

	require.IsType(&blockexecutor.Block{}, blkIntf)
	blk := blkIntf.(*blockexecutor.Block)
	require.Equal([]*txs.Tx{tx}, blk.Txs())
	require.IsType(&block.BanffStandardBlock{}, blk.Block)
	standardBlk := blk.Block.(*block.BanffStandardBlock)
	require.Equal(nextTime.Unix(), standardBlk.Timestamp().Unix())
}

func TestBuildBlockInvalidStakingDurations(t *testing.T) {
	require := require.New(t)

	env := newEnvironment(t, upgradetest.Latest)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	// Post-Durango, [StartTime] is no longer validated. Staking durations are
	// based on the current chain timestamp and must be validated.
	env.config.UpgradeConfig.DurangoTime = time.Time{}

	wallet := newWallet(t, env, walletConfig{})

	var (
		now                   = env.backend.Clk.Time()
		defaultValidatorStake = 100 * units.MilliLux

		// Add a validator ending in [MaxStakeDuration]
		validatorEndTime = now.Add(env.config.MaxStakeDuration)
	)

	sk, err := localsigner.New()
	require.NoError(err)
	pop, err := signer.NewProofOfPossession(sk)
	require.NoError(err)

	rewardsOwner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
	}
	tx1, err := wallet.IssueAddPermissionlessValidatorTx(
		&txs.ChainValidator{
			Validator: txs.Validator{
				NodeID: ids.GenerateTestNodeID(),
				Start:  uint64(now.Unix()),
				End:    uint64(validatorEndTime.Unix()),
				Wght:   defaultValidatorStake,
			},
			Chain: constants.PrimaryNetworkID,
		},
		pop,
		env.ctx.XAssetID,
		rewardsOwner,
		rewardsOwner,
		reward.PercentDenominator,
	)
	require.NoError(err)
	require.NoError(env.mempool.Add(tx1))

	tx1ID := tx1.ID()
	_, ok := env.mempool.Get(tx1ID)
	require.True(ok)

	// Add a validator ending past [MaxStakeDuration]
	validator2EndTime := now.Add(env.config.MaxStakeDuration + time.Second)

	sk, err = localsigner.New()
	require.NoError(err)
	pop, err = signer.NewProofOfPossession(sk)
	require.NoError(err)

	tx2, err := wallet.IssueAddPermissionlessValidatorTx(
		&txs.ChainValidator{
			Validator: txs.Validator{
				NodeID: ids.GenerateTestNodeID(),
				Start:  uint64(now.Unix()),
				End:    uint64(validator2EndTime.Unix()),
				Wght:   defaultValidatorStake,
			},
			Chain: constants.PrimaryNetworkID,
		},
		pop,
		env.ctx.XAssetID,
		rewardsOwner,
		rewardsOwner,
		reward.PercentDenominator,
	)
	require.NoError(err)
	require.NoError(env.mempool.Add(tx2))

	tx2ID := tx2.ID()
	_, ok = env.mempool.Get(tx2ID)
	require.True(ok)

	// Only tx1 should be in a built block since [MaxStakeDuration] is satisfied.
	blkIntf, err := env.Builder.BuildBlock(context.Background())
	require.NoError(err)

	require.IsType(&blockexecutor.Block{}, blkIntf)
	blk := blkIntf.(*blockexecutor.Block)
	require.Len(blk.Txs(), 1)
	require.Equal(tx1ID, blk.Txs()[0].ID())

	// Mempool should have none of the txs
	_, ok = env.mempool.Get(tx1ID)
	require.False(ok)
	_, ok = env.mempool.Get(tx2ID)
	require.False(ok)

	// Only tx2 should be dropped
	require.NoError(env.mempool.GetDropReason(tx1ID))

	tx2DropReason := env.mempool.GetDropReason(tx2ID)
	require.ErrorIs(tx2DropReason, txexecutor.ErrStakeTooLong)
}

func TestPreviouslyDroppedTxsCannotBeReAddedToMempool(t *testing.T) {
	require := require.New(t)

	env := newEnvironment(t, upgradetest.Latest)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	subnetID := testNet1.ID()
	wallet := newWallet(t, env, walletConfig{
		netIDs: []ids.ID{subnetID},
	})

	// Create a valid transaction
	tx, err := wallet.IssueCreateChainTx(
		testNet1.ID(),
		nil,
		constants.XVMID,
		nil,
		"chain name",
	)
	require.NoError(err)

	// Transaction should not be marked as dropped before being added to the
	// mempool
	txID := tx.ID()
	require.NoError(env.mempool.GetDropReason(txID))

	// Mark the transaction as dropped
	errTestingDropped := errors.New("testing dropped")
	env.mempool.MarkDropped(txID, errTestingDropped)
	err = env.mempool.GetDropReason(txID)
	require.ErrorIs(err, errTestingDropped)

	// Issue the transaction
	env.ctx.Lock.Unlock()
	err = env.network.IssueTxFromRPC(tx)
	require.ErrorIs(err, errTestingDropped)
	env.ctx.Lock.Lock()
	_, ok := env.mempool.Get(txID)
	require.False(ok)

	// When issued again, the mempool should still be marked as dropped
	err = env.mempool.GetDropReason(txID)
	require.ErrorIs(err, errTestingDropped)
}

func TestNoErrorOnUnexpectedSetPreferenceDuringBootstrapping(t *testing.T) {
	env := newEnvironment(t, upgradetest.Latest)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	env.isBootstrapped.Set(false)
	env.blkManager.SetPreference(ids.GenerateTestID()) // should not panic
}

func TestGetNextStakerToReward(t *testing.T) {
	var (
		now  = time.Now()
		txID = ids.GenerateTestID()
	)

	type test struct {
		name                 string
		timestamp            time.Time
		stateF               func(*gomock.Controller) state.Chain
		expectedTxID         ids.ID
		expectedShouldReward bool
		expectedErr          error
	}

	tests := []test{
		{
			name:      "end of time",
			timestamp: mockable.MaxTime,
			stateF: func(ctrl *gomock.Controller) state.Chain {
				return state.NewMockChain(ctrl)
			},
			expectedErr: ErrEndOfTime,
		},
		{
			name:      "no stakers",
			timestamp: now,
			stateF: func(ctrl *gomock.Controller) state.Chain {
				s := state.NewMockChain(ctrl)
				s.EXPECT().GetCurrentStakerIterator().Return(iterator.Empty[*state.Staker]{}, nil)
				return s
			},
		},
		{
			name:      "expired net validator/delegator",
			timestamp: now,
			stateF: func(ctrl *gomock.Controller) state.Chain {
				s := state.NewMockChain(ctrl)
				s.EXPECT().GetCurrentStakerIterator().Return(
					iterator.FromSlice(
						&state.Staker{
							Priority: txs.NetPermissionedValidatorCurrentPriority,
							EndTime:  now,
						},
						&state.Staker{
							TxID:     txID,
							Priority: txs.NetPermissionlessDelegatorCurrentPriority,
							EndTime:  now,
						},
					),
					nil,
				)
				return s
			},
			expectedTxID:         txID,
			expectedShouldReward: true,
		},
		{
			name:      "expired primary network validator after net expired net validator",
			timestamp: now,
			stateF: func(ctrl *gomock.Controller) state.Chain {
				s := state.NewMockChain(ctrl)
				s.EXPECT().GetCurrentStakerIterator().Return(
					iterator.FromSlice(
						&state.Staker{
							Priority: txs.NetPermissionedValidatorCurrentPriority,
							EndTime:  now,
						},
						&state.Staker{
							TxID:     txID,
							Priority: txs.PrimaryNetworkValidatorCurrentPriority,
							EndTime:  now,
						},
					),
					nil,
				)
				return s
			},
			expectedTxID:         txID,
			expectedShouldReward: true,
		},
		{
			name:      "expired primary network delegator after net expired net validator",
			timestamp: now,
			stateF: func(ctrl *gomock.Controller) state.Chain {
				s := state.NewMockChain(ctrl)
				s.EXPECT().GetCurrentStakerIterator().Return(
					iterator.FromSlice(
						&state.Staker{
							Priority: txs.NetPermissionedValidatorCurrentPriority,
							EndTime:  now,
						},
						&state.Staker{
							TxID:     txID,
							Priority: txs.PrimaryNetworkDelegatorCurrentPriority,
							EndTime:  now,
						},
					),
					nil,
				)
				return s
			},
			expectedTxID:         txID,
			expectedShouldReward: true,
		},
		{
			name:      "non-expired primary network delegator",
			timestamp: now,
			stateF: func(ctrl *gomock.Controller) state.Chain {
				s := state.NewMockChain(ctrl)
				s.EXPECT().GetCurrentStakerIterator().Return(
					iterator.FromSlice(
						&state.Staker{
							TxID:     txID,
							Priority: txs.PrimaryNetworkDelegatorCurrentPriority,
							EndTime:  now.Add(time.Second),
						},
					),
					nil,
				)
				return s
			},
			expectedTxID:         txID,
			expectedShouldReward: false,
		},
		{
			name:      "non-expired primary network validator",
			timestamp: now,
			stateF: func(ctrl *gomock.Controller) state.Chain {
				s := state.NewMockChain(ctrl)
				s.EXPECT().GetCurrentStakerIterator().Return(
					iterator.FromSlice(
						&state.Staker{
							TxID:     txID,
							Priority: txs.PrimaryNetworkValidatorCurrentPriority,
							EndTime:  now.Add(time.Second),
						},
					),
					nil,
				)
				return s
			},
			expectedTxID:         txID,
			expectedShouldReward: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)

			state := tt.stateF(ctrl)
			txID, shouldReward, err := getNextStakerToReward(tt.timestamp, state)
			require.ErrorIs(err, tt.expectedErr)
			if tt.expectedErr != nil {
				return
			}
			require.Equal(tt.expectedTxID, txID)
			require.Equal(tt.expectedShouldReward, shouldReward)
		})
	}
}
