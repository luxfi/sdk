// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"math/rand"
	"testing"
	"time"

	"github.com/luxfi/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/luxfi/database"
	"github.com/luxfi/database/memdb"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/utils/iterator"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/components/gas"
	"github.com/luxfi/sdk/node/vms/platformvm/fx/fxmock"
	"github.com/luxfi/sdk/node/vms/platformvm/status"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
)

type nilStateGetter struct{}

func (nilStateGetter) GetState(ids.ID) (Chain, bool) {
	return nil, false
}

func TestDiffMissingState(t *testing.T) {
	parentID := ids.GenerateTestID()
	_, err := NewDiff(parentID, nilStateGetter{})
	require.ErrorIs(t, err, ErrMissingParentState)
}

func TestNewDiffOn(t *testing.T) {
	require := require.New(t)

	state := newTestState(t, memdb.New())

	d, err := NewDiffOn(state)
	require.NoError(err)

	assertChainsEqual(t, state, d)
}

func TestDiffFeeState(t *testing.T) {
	require := require.New(t)

	state := newTestState(t, memdb.New())

	d, err := NewDiffOn(state)
	require.NoError(err)

	initialFeeState := state.GetFeeState()
	newFeeState := gas.State{
		Capacity: initialFeeState.Capacity + 1,
		Excess:   initialFeeState.Excess + 1,
	}
	d.SetFeeState(newFeeState)
	require.Equal(newFeeState, d.GetFeeState())
	require.Equal(initialFeeState, state.GetFeeState())

	require.NoError(d.Apply(state))
	assertChainsEqual(t, state, d)
}

func TestDiffL1ValidatorExcess(t *testing.T) {
	require := require.New(t)

	state := newTestState(t, memdb.New())

	d, err := NewDiffOn(state)
	require.NoError(err)

	initialExcess := state.GetL1ValidatorExcess()
	newExcess := initialExcess + 1
	d.SetL1ValidatorExcess(newExcess)
	require.Equal(newExcess, d.GetL1ValidatorExcess())
	require.Equal(initialExcess, state.GetL1ValidatorExcess())

	require.NoError(d.Apply(state))
	assertChainsEqual(t, state, d)
}

func TestDiffAccruedFees(t *testing.T) {
	require := require.New(t)

	state := newTestState(t, memdb.New())

	d, err := NewDiffOn(state)
	require.NoError(err)

	initialAccruedFees := state.GetAccruedFees()
	newAccruedFees := initialAccruedFees + 1
	d.SetAccruedFees(newAccruedFees)
	require.Equal(newAccruedFees, d.GetAccruedFees())
	require.Equal(initialAccruedFees, state.GetAccruedFees())

	require.NoError(d.Apply(state))
	assertChainsEqual(t, state, d)
}

func TestDiffCurrentSupply(t *testing.T) {
	require := require.New(t)

	state := newTestState(t, memdb.New())

	d, err := NewDiffOn(state)
	require.NoError(err)

	initialCurrentSupply, err := d.GetCurrentSupply(constants.PrimaryNetworkID)
	require.NoError(err)

	newCurrentSupply := initialCurrentSupply + 1
	d.SetCurrentSupply(constants.PrimaryNetworkID, newCurrentSupply)

	returnedNewCurrentSupply, err := d.GetCurrentSupply(constants.PrimaryNetworkID)
	require.NoError(err)
	require.Equal(newCurrentSupply, returnedNewCurrentSupply)

	returnedBaseCurrentSupply, err := state.GetCurrentSupply(constants.PrimaryNetworkID)
	require.NoError(err)
	require.Equal(initialCurrentSupply, returnedBaseCurrentSupply)

	require.NoError(d.Apply(state))
	assertChainsEqual(t, state, d)
}

func TestDiffExpiry(t *testing.T) {
	type op struct {
		put   bool
		entry ExpiryEntry
	}
	tests := []struct {
		name            string
		initialExpiries []ExpiryEntry
		ops             []op
	}{
		{
			name: "empty noop",
		},
		{
			name: "insert",
			ops: []op{
				{
					put:   true,
					entry: ExpiryEntry{Timestamp: 1},
				},
			},
		},
		{
			name: "remove",
			initialExpiries: []ExpiryEntry{
				{Timestamp: 1},
			},
			ops: []op{
				{
					put:   false,
					entry: ExpiryEntry{Timestamp: 1},
				},
			},
		},
		{
			name: "add and immediately remove",
			ops: []op{
				{
					put:   true,
					entry: ExpiryEntry{Timestamp: 1},
				},
				{
					put:   false,
					entry: ExpiryEntry{Timestamp: 1},
				},
			},
		},
		{
			name: "add + remove + add",
			ops: []op{
				{
					put:   true,
					entry: ExpiryEntry{Timestamp: 1},
				},
				{
					put:   false,
					entry: ExpiryEntry{Timestamp: 1},
				},
				{
					put:   true,
					entry: ExpiryEntry{Timestamp: 1},
				},
			},
		},
		{
			name: "everything",
			initialExpiries: []ExpiryEntry{
				{Timestamp: 1},
				{Timestamp: 2},
				{Timestamp: 3},
			},
			ops: []op{
				{
					put:   false,
					entry: ExpiryEntry{Timestamp: 1},
				},
				{
					put:   false,
					entry: ExpiryEntry{Timestamp: 2},
				},
				{
					put:   true,
					entry: ExpiryEntry{Timestamp: 1},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)

			state := newTestState(t, memdb.New())
			for _, expiry := range test.initialExpiries {
				state.PutExpiry(expiry)
			}

			d, err := NewDiffOn(state)
			require.NoError(err)

			var (
				expectedExpiries   = set.Of(test.initialExpiries...)
				unexpectedExpiries = set.NewSet[ExpiryEntry](len(test.ops))
			)
			for _, op := range test.ops {
				if op.put {
					d.PutExpiry(op.entry)
					expectedExpiries.Add(op.entry)
					unexpectedExpiries.Remove(op.entry)
				} else {
					d.DeleteExpiry(op.entry)
					expectedExpiries.Remove(op.entry)
					unexpectedExpiries.Add(op.entry)
				}
			}

			// If expectedExpiries is empty, we want expectedExpiriesSlice to be
			// nil.
			var expectedExpiriesSlice []ExpiryEntry
			if expectedExpiries.Len() > 0 {
				expectedExpiriesSlice = expectedExpiries.List()
				utils.Sort(expectedExpiriesSlice)
			}

			verifyChain := func(chain Chain) {
				expiryIterator, err := chain.GetExpiryIterator()
				require.NoError(err)
				require.Equal(
					expectedExpiriesSlice,
					iterator.ToSlice(expiryIterator),
				)

				for expiry := range expectedExpiries {
					has, err := chain.HasExpiry(expiry)
					require.NoError(err)
					require.True(has)
				}
				for expiry := range unexpectedExpiries {
					has, err := chain.HasExpiry(expiry)
					require.NoError(err)
					require.False(has)
				}
			}

			verifyChain(d)
			require.NoError(d.Apply(state))
			verifyChain(state)
			assertChainsEqual(t, d, state)
		})
	}
}

func TestDiffL1ValidatorsErrors(t *testing.T) {
	l1Validator := L1Validator{
		ValidationID: ids.GenerateTestID(),
		ChainID:     ids.GenerateTestID(),
		NodeID:       ids.GenerateTestNodeID(),
		Weight:       1, // Not removed
	}

	tests := []struct {
		name                     string
		initialEndAccumulatedFee uint64
		l1Validator              L1Validator
		expectedErr              error
	}{
		{
			name:                     "mutate active constants",
			initialEndAccumulatedFee: 1,
			l1Validator: L1Validator{
				ValidationID: l1Validator.ValidationID,
				NodeID:       ids.GenerateTestNodeID(),
			},
			expectedErr: ErrMutatedL1Validator,
		},
		{
			name:                     "mutate inactive constants",
			initialEndAccumulatedFee: 0,
			l1Validator: L1Validator{
				ValidationID: l1Validator.ValidationID,
				NodeID:       ids.GenerateTestNodeID(),
			},
			expectedErr: ErrMutatedL1Validator,
		},
		{
			name:                     "conflicting legacy subnetID and nodeID pair",
			initialEndAccumulatedFee: 1,
			l1Validator: L1Validator{
				ValidationID: ids.GenerateTestID(),
				NodeID:       defaultValidatorNodeID,
			},
			expectedErr: ErrConflictingL1Validator,
		},
		{
			name:                     "duplicate active subnetID and nodeID pair",
			initialEndAccumulatedFee: 1,
			l1Validator: L1Validator{
				ValidationID: ids.GenerateTestID(),
				NodeID:       l1Validator.NodeID,
			},
			expectedErr: ErrDuplicateL1Validator,
		},
		{
			name:                     "duplicate inactive subnetID and nodeID pair",
			initialEndAccumulatedFee: 0,
			l1Validator: L1Validator{
				ValidationID: ids.GenerateTestID(),
				NodeID:       l1Validator.NodeID,
			},
			expectedErr: ErrDuplicateL1Validator,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)

			state := newTestState(t, memdb.New())

			require.NoError(state.PutCurrentValidator(&Staker{
				TxID:    ids.GenerateTestID(),
				ChainID: l1Validator.ChainID,
				NodeID:  defaultValidatorNodeID,
			}))

			l1Validator.EndAccumulatedFee = test.initialEndAccumulatedFee
			require.NoError(state.PutL1Validator(l1Validator))

			d, err := NewDiffOn(state)
			require.NoError(err)

			// Initialize subnetID, weight, and endAccumulatedFee as they are
			// constant among all tests.
			test.l1Validator.ChainID = l1Validator.ChainID
			test.l1Validator.Weight = 1                        // Not removed
			test.l1Validator.EndAccumulatedFee = rand.Uint64() //#nosec G404
			err = d.PutL1Validator(test.l1Validator)
			require.ErrorIs(err, test.expectedErr)

			// The invalid addition should not have modified the diff.
			assertChainsEqual(t, state, d)
		})
	}
}

func TestDiffCurrentValidator(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	state := NewMockState(ctrl)
	// Called in NewDiffOn
	state.EXPECT().GetTimestamp().Return(time.Now()).Times(1)
	state.EXPECT().GetFeeState().Return(gas.State{}).Times(1)
	state.EXPECT().GetL1ValidatorExcess().Return(gas.Gas(0)).Times(1)
	state.EXPECT().GetAccruedFees().Return(uint64(0)).Times(1)
	state.EXPECT().NumActiveL1Validators().Return(0).Times(1)

	d, err := NewDiffOn(state)
	require.NoError(err)

	// Put a current validator
	currentValidator := &Staker{
		TxID:   ids.GenerateTestID(),
		ChainID:  ids.GenerateTestID(),
		NodeID: ids.GenerateTestNodeID(),
	}
	require.NoError(d.PutCurrentValidator(currentValidator))

	// Assert that we get the current validator back
	gotCurrentValidator, err := d.GetCurrentValidator(currentValidator.ChainID, currentValidator.NodeID)
	require.NoError(err)
	require.Equal(currentValidator, gotCurrentValidator)

	// Delete the current validator
	d.DeleteCurrentValidator(currentValidator)

	// Make sure the deletion worked
	state.EXPECT().GetCurrentValidator(currentValidator.ChainID, currentValidator.NodeID).Return(nil, database.ErrNotFound).Times(1)
	_, err = d.GetCurrentValidator(currentValidator.ChainID, currentValidator.NodeID)
	require.ErrorIs(err, database.ErrNotFound)
}

func TestDiffPendingValidator(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	state := NewMockState(ctrl)
	// Called in NewDiffOn
	state.EXPECT().GetTimestamp().Return(time.Now()).Times(1)
	state.EXPECT().GetFeeState().Return(gas.State{}).Times(1)
	state.EXPECT().GetL1ValidatorExcess().Return(gas.Gas(0)).Times(1)
	state.EXPECT().GetAccruedFees().Return(uint64(0)).Times(1)
	state.EXPECT().NumActiveL1Validators().Return(0).Times(1)

	d, err := NewDiffOn(state)
	require.NoError(err)

	// Put a pending validator
	pendingValidator := &Staker{
		TxID:   ids.GenerateTestID(),
		ChainID:  ids.GenerateTestID(),
		NodeID: ids.GenerateTestNodeID(),
	}
	require.NoError(d.PutPendingValidator(pendingValidator))

	// Assert that we get the pending validator back
	gotPendingValidator, err := d.GetPendingValidator(pendingValidator.ChainID, pendingValidator.NodeID)
	require.NoError(err)
	require.Equal(pendingValidator, gotPendingValidator)

	// Delete the pending validator
	d.DeletePendingValidator(pendingValidator)

	// Make sure the deletion worked
	state.EXPECT().GetPendingValidator(pendingValidator.ChainID, pendingValidator.NodeID).Return(nil, database.ErrNotFound).Times(1)
	_, err = d.GetPendingValidator(pendingValidator.ChainID, pendingValidator.NodeID)
	require.ErrorIs(err, database.ErrNotFound)
}

func TestDiffCurrentDelegator(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	currentDelegator := &Staker{
		TxID:   ids.GenerateTestID(),
		ChainID:  ids.GenerateTestID(),
		NodeID: ids.GenerateTestNodeID(),
	}

	state := NewMockState(ctrl)
	// Called in NewDiffOn
	state.EXPECT().GetTimestamp().Return(time.Now()).Times(1)
	state.EXPECT().GetFeeState().Return(gas.State{}).Times(1)
	state.EXPECT().GetL1ValidatorExcess().Return(gas.Gas(0)).Times(1)
	state.EXPECT().GetAccruedFees().Return(uint64(0)).Times(1)
	state.EXPECT().NumActiveL1Validators().Return(0).Times(1)

	d, err := NewDiffOn(state)
	require.NoError(err)

	// Put a current delegator
	d.PutCurrentDelegator(currentDelegator)

	// Assert that we get the current delegator back
	// Mock iterator for [state] returns no delegators.
	state.EXPECT().GetCurrentDelegatorIterator(
		currentDelegator.ChainID,
		currentDelegator.NodeID,
	).Return(iterator.Empty[*Staker]{}, nil).Times(2)
	gotCurrentDelegatorIter, err := d.GetCurrentDelegatorIterator(currentDelegator.ChainID, currentDelegator.NodeID)
	require.NoError(err)
	// The iterator should have the 1 delegator we put in [d]
	require.True(gotCurrentDelegatorIter.Next())
	require.Equal(gotCurrentDelegatorIter.Value(), currentDelegator)

	// Delete the current delegator
	d.DeleteCurrentDelegator(currentDelegator)

	// Make sure the deletion worked.
	// The iterator should have no elements.
	gotCurrentDelegatorIter, err = d.GetCurrentDelegatorIterator(currentDelegator.ChainID, currentDelegator.NodeID)
	require.NoError(err)
	require.False(gotCurrentDelegatorIter.Next())
}

func TestDiffPendingDelegator(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	pendingDelegator := &Staker{
		TxID:   ids.GenerateTestID(),
		ChainID:  ids.GenerateTestID(),
		NodeID: ids.GenerateTestNodeID(),
	}

	state := NewMockState(ctrl)
	// Called in NewDiffOn
	state.EXPECT().GetTimestamp().Return(time.Now()).Times(1)
	state.EXPECT().GetFeeState().Return(gas.State{}).Times(1)
	state.EXPECT().GetL1ValidatorExcess().Return(gas.Gas(0)).Times(1)
	state.EXPECT().GetAccruedFees().Return(uint64(0)).Times(1)
	state.EXPECT().NumActiveL1Validators().Return(0).Times(1)

	d, err := NewDiffOn(state)
	require.NoError(err)

	// Put a pending delegator
	d.PutPendingDelegator(pendingDelegator)

	// Assert that we get the pending delegator back
	// Mock iterator for [state] returns no delegators.
	state.EXPECT().GetPendingDelegatorIterator(
		pendingDelegator.ChainID,
		pendingDelegator.NodeID,
	).Return(iterator.Empty[*Staker]{}, nil).Times(2)
	gotPendingDelegatorIter, err := d.GetPendingDelegatorIterator(pendingDelegator.ChainID, pendingDelegator.NodeID)
	require.NoError(err)
	// The iterator should have the 1 delegator we put in [d]
	require.True(gotPendingDelegatorIter.Next())
	require.Equal(gotPendingDelegatorIter.Value(), pendingDelegator)

	// Delete the pending delegator
	d.DeletePendingDelegator(pendingDelegator)

	// Make sure the deletion worked.
	// The iterator should have no elements.
	gotPendingDelegatorIter, err = d.GetPendingDelegatorIterator(pendingDelegator.ChainID, pendingDelegator.NodeID)
	require.NoError(err)
	require.False(gotPendingDelegatorIter.Next())
}

func TestDiffNet(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	state := newTestState(t, memdb.New())

	// Initialize parent with one subnet
	parentStateCreateNetTx := &txs.Tx{
		Unsigned: &txs.CreateSubnetTx{
			Owner: fxmock.NewOwner(ctrl),
		},
	}
	state.AddNet(parentStateCreateNetTx.ID())

	// Verify parent returns one subnet
	chainIDs, err := state.GetChainIDs()
	require.NoError(err)
	require.Equal(
		[]ids.ID{
			parentStateCreateNetTx.ID(),
		},
		chainIDs,
	)

	diff, err := NewDiffOn(state)
	require.NoError(err)

	// Put a subnet
	createNetTx := &txs.Tx{
		Unsigned: &txs.CreateSubnetTx{
			Owner: fxmock.NewOwner(ctrl),
		},
	}
	diff.AddNet(createNetTx.ID())

	// Apply diff to parent state
	require.NoError(diff.Apply(state))

	// Verify parent now returns two subnets
	chainIDs, err = state.GetChainIDs()
	require.NoError(err)
	require.Equal(
		[]ids.ID{
			parentStateCreateNetTx.ID(),
			createNetTx.ID(),
		},
		chainIDs,
	)
}

func TestDiffChain(t *testing.T) {
	require := require.New(t)

	state := newTestState(t, memdb.New())
	subnetID := ids.GenerateTestID()

	// Initialize parent with one chain
	parentStateCreateChainTx := &txs.Tx{
		Unsigned: &txs.CreateChainTx{
			ChainID: subnetID,
		},
	}
	state.AddChain(parentStateCreateChainTx)

	// Verify parent returns one chain
	chains, err := state.GetChains(subnetID)
	require.NoError(err)
	require.Equal(
		[]*txs.Tx{
			parentStateCreateChainTx,
		},
		chains,
	)

	diff, err := NewDiffOn(state)
	require.NoError(err)

	// Put a chain
	createChainTx := &txs.Tx{
		Unsigned: &txs.CreateChainTx{
			ChainID: subnetID, // note this is the same net as [parentStateCreateChainTx]
		},
	}
	diff.AddChain(createChainTx)

	// Apply diff to parent state
	require.NoError(diff.Apply(state))

	// Verify parent now returns two chains
	chains, err = state.GetChains(subnetID)
	require.NoError(err)
	require.Equal(
		[]*txs.Tx{
			parentStateCreateChainTx,
			createChainTx,
		},
		chains,
	)
}

func TestDiffTx(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	state := NewMockState(ctrl)
	// Called in NewDiffOn
	state.EXPECT().GetTimestamp().Return(time.Now()).Times(1)
	state.EXPECT().GetFeeState().Return(gas.State{}).Times(1)
	state.EXPECT().GetL1ValidatorExcess().Return(gas.Gas(0)).Times(1)
	state.EXPECT().GetAccruedFees().Return(uint64(0)).Times(1)
	state.EXPECT().NumActiveL1Validators().Return(0).Times(1)

	d, err := NewDiffOn(state)
	require.NoError(err)

	// Put a tx
	netID := ids.GenerateTestID()
	tx := &txs.Tx{
		Unsigned: &txs.CreateChainTx{
			ChainID: netID,
		},
	}
	tx.SetBytes(utils.RandomBytes(16), utils.RandomBytes(16))
	d.AddTx(tx, status.Committed)

	{
		// Assert that we get the tx back
		gotTx, gotStatus, err := d.GetTx(tx.ID())
		require.NoError(err)
		require.Equal(status.Committed, gotStatus)
		require.Equal(tx, gotTx)
	}

	{
		// Assert that we can get a tx from the parent state
		// [state] returns 1 tx.
		parentTx := &txs.Tx{
			Unsigned: &txs.CreateChainTx{
				ChainID: netID,
			},
		}
		parentTx.SetBytes(utils.RandomBytes(16), utils.RandomBytes(16))
		state.EXPECT().GetTx(parentTx.ID()).Return(parentTx, status.Committed, nil).Times(1)
		gotParentTx, gotStatus, err := d.GetTx(parentTx.ID())
		require.NoError(err)
		require.Equal(status.Committed, gotStatus)
		require.Equal(parentTx, gotParentTx)
	}
}

func TestDiffRewardUTXO(t *testing.T) {
	require := require.New(t)

	state := newTestState(t, memdb.New())

	// Initialize parent with one reward UTXO
	var (
		txID             = ids.GenerateTestID()
		parentRewardUTXO = &lux.UTXO{
			UTXOID: lux.UTXOID{
				TxID: txID,
			},
		}
	)
	state.AddRewardUTXO(txID, parentRewardUTXO)

	// Verify parent returns the reward UTXO
	rewardUTXOs, err := state.GetRewardUTXOs(txID)
	require.NoError(err)
	require.Equal(
		[]*lux.UTXO{
			parentRewardUTXO,
		},
		rewardUTXOs,
	)

	diff, err := NewDiffOn(state)
	require.NoError(err)

	// Put a reward UTXO
	rewardUTXO := &lux.UTXO{
		UTXOID: lux.UTXOID{TxID: txID},
	}
	diff.AddRewardUTXO(txID, rewardUTXO)

	// Apply diff to parent state
	require.NoError(diff.Apply(state))

	// Verify parent now returns two reward UTXOs
	rewardUTXOs, err = state.GetRewardUTXOs(txID)
	require.NoError(err)
	require.Equal(
		[]*lux.UTXO{
			parentRewardUTXO,
			rewardUTXO,
		},
		rewardUTXOs,
	)
}

func TestDiffUTXO(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	state := NewMockState(ctrl)
	// Called in NewDiffOn
	state.EXPECT().GetTimestamp().Return(time.Now()).Times(1)
	state.EXPECT().GetFeeState().Return(gas.State{}).Times(1)
	state.EXPECT().GetL1ValidatorExcess().Return(gas.Gas(0)).Times(1)
	state.EXPECT().GetAccruedFees().Return(uint64(0)).Times(1)
	state.EXPECT().NumActiveL1Validators().Return(0).Times(1)

	d, err := NewDiffOn(state)
	require.NoError(err)

	// Put a UTXO
	utxo := &lux.UTXO{
		UTXOID: lux.UTXOID{TxID: ids.GenerateTestID()},
	}
	d.AddUTXO(utxo)

	{
		// Assert that we get the UTXO back
		gotUTXO, err := d.GetUTXO(utxo.InputID())
		require.NoError(err)
		require.Equal(utxo, gotUTXO)
	}

	{
		// Assert that we can get a UTXO from the parent state
		// [state] returns 1 UTXO.
		parentUTXO := &lux.UTXO{
			UTXOID: lux.UTXOID{TxID: ids.GenerateTestID()},
		}
		state.EXPECT().GetUTXO(parentUTXO.InputID()).Return(parentUTXO, nil).Times(1)
		gotParentUTXO, err := d.GetUTXO(parentUTXO.InputID())
		require.NoError(err)
		require.Equal(parentUTXO, gotParentUTXO)
	}

	{
		// Delete the UTXO
		d.DeleteUTXO(utxo.InputID())

		// Make sure it's gone
		_, err = d.GetUTXO(utxo.InputID())
		require.ErrorIs(err, database.ErrNotFound)
	}
}

func assertChainsEqual(t *testing.T, expected, actual Chain) {
	require := require.New(t)

	t.Helper()

	expectedExpiryIterator, expectedErr := expected.GetExpiryIterator()
	actualExpiryIterator, actualErr := actual.GetExpiryIterator()
	require.Equal(expectedErr, actualErr)
	if expectedErr == nil {
		require.Equal(
			iterator.ToSlice(expectedExpiryIterator),
			iterator.ToSlice(actualExpiryIterator),
		)
	}

	expectedActiveL1ValidatorsIterator, expectedErr := expected.GetActiveL1ValidatorsIterator()
	actualActiveL1ValidatorsIterator, actualErr := actual.GetActiveL1ValidatorsIterator()
	require.Equal(expectedErr, actualErr)
	if expectedErr == nil {
		require.Equal(
			iterator.ToSlice(expectedActiveL1ValidatorsIterator),
			iterator.ToSlice(actualActiveL1ValidatorsIterator),
		)
	}

	require.Equal(expected.NumActiveL1Validators(), actual.NumActiveL1Validators())

	expectedCurrentStakerIterator, expectedErr := expected.GetCurrentStakerIterator()
	actualCurrentStakerIterator, actualErr := actual.GetCurrentStakerIterator()
	require.Equal(expectedErr, actualErr)
	if expectedErr == nil {
		require.Equal(
			iterator.ToSlice(expectedCurrentStakerIterator),
			iterator.ToSlice(actualCurrentStakerIterator),
		)
	}

	expectedPendingStakerIterator, expectedErr := expected.GetPendingStakerIterator()
	actualPendingStakerIterator, actualErr := actual.GetPendingStakerIterator()
	require.Equal(expectedErr, actualErr)
	if expectedErr == nil {
		require.Equal(
			iterator.ToSlice(expectedPendingStakerIterator),
			iterator.ToSlice(actualPendingStakerIterator),
		)
	}

	require.Equal(expected.GetTimestamp(), actual.GetTimestamp())
	require.Equal(expected.GetFeeState(), actual.GetFeeState())
	require.Equal(expected.GetL1ValidatorExcess(), actual.GetL1ValidatorExcess())
	require.Equal(expected.GetAccruedFees(), actual.GetAccruedFees())

	expectedCurrentSupply, err := expected.GetCurrentSupply(constants.PrimaryNetworkID)
	require.NoError(err)

	actualCurrentSupply, err := actual.GetCurrentSupply(constants.PrimaryNetworkID)
	require.NoError(err)

	require.Equal(expectedCurrentSupply, actualCurrentSupply)
}

func TestDiffNetOwner(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	state := newTestState(t, memdb.New())

	var (
		owner1 = fxmock.NewOwner(ctrl)
		owner2 = fxmock.NewOwner(ctrl)

		createNetTx = &txs.Tx{
			Unsigned: &txs.CreateSubnetTx{
				BaseTx: txs.BaseTx{},
				Owner:  owner1,
			},
		}

		netID = createNetTx.ID()
	)

	// Create net on base state
	owner, err := state.GetNetOwner(netID)
	require.ErrorIs(err, database.ErrNotFound)
	require.Nil(owner)

	state.AddNet(netID)
	state.SetNetOwner(netID, owner1)

	owner, err = state.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner1, owner)

	// Create diff and verify that subnet owner returns correctly
	d, err := NewDiffOn(state)
	require.NoError(err)

	owner, err = d.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner1, owner)

	// Transferring net ownership on diff should be reflected on diff not state
	d.SetNetOwner(netID, owner2)
	owner, err = d.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner2, owner)

	owner, err = state.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner1, owner)

	// State should reflect new net owner after diff is applied.
	require.NoError(d.Apply(state))

	owner, err = state.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner2, owner)
}

func TestDiffNetToL1Conversion(t *testing.T) {
	var (
		require            = require.New(t)
		state              = newTestState(t, memdb.New())
		subnetID           = ids.GenerateTestID()
		expectedConversion = NetToL1Conversion{
			ConversionID: ids.GenerateTestID(),
			ChainID:      ids.GenerateTestID(),
			Addr:         []byte{1, 2, 3, 4},
		}
	)

	actualConversion, err := state.GetNetToL1Conversion(subnetID)
	require.ErrorIs(err, database.ErrNotFound)
	require.Zero(actualConversion)

	d, err := NewDiffOn(state)
	require.NoError(err)

	actualConversion, err = d.GetNetToL1Conversion(subnetID)
	require.ErrorIs(err, database.ErrNotFound)
	require.Zero(actualConversion)

	// Setting a subnet conversion should be reflected on diff not state
	d.SetNetToL1Conversion(subnetID, expectedConversion)
	actualConversion, err = d.GetNetToL1Conversion(subnetID)
	require.NoError(err)
	require.Equal(expectedConversion, actualConversion)

	actualConversion, err = state.GetNetToL1Conversion(subnetID)
	require.ErrorIs(err, database.ErrNotFound)
	require.Zero(actualConversion)

	// State should reflect new subnet conversion after diff is applied
	require.NoError(d.Apply(state))
	actualConversion, err = state.GetNetToL1Conversion(subnetID)
	require.NoError(err)
	require.Equal(expectedConversion, actualConversion)
}

func TestDiffStacking(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	state := newTestState(t, memdb.New())

	var (
		owner1 = fxmock.NewOwner(ctrl)
		owner2 = fxmock.NewOwner(ctrl)
		owner3 = fxmock.NewOwner(ctrl)

		createNetTx = &txs.Tx{
			Unsigned: &txs.CreateSubnetTx{
				BaseTx: txs.BaseTx{},
				Owner:  owner1,
			},
		}

		netID = createNetTx.ID()
	)

	// Create net on base state
	owner, err := state.GetNetOwner(netID)
	require.ErrorIs(err, database.ErrNotFound)
	require.Nil(owner)

	state.AddNet(netID)
	state.SetNetOwner(netID, owner1)

	owner, err = state.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner1, owner)

	// Create first diff and verify that subnet owner returns correctly
	statesDiff, err := NewDiffOn(state)
	require.NoError(err)

	owner, err = statesDiff.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner1, owner)

	// Transferring net ownership on first diff should be reflected on first diff not state
	statesDiff.SetNetOwner(netID, owner2)
	owner, err = statesDiff.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner2, owner)

	owner, err = state.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner1, owner)

	// Create a second diff on first diff and verify that net owner returns correctly
	stackedDiff, err := NewDiffOn(statesDiff)
	require.NoError(err)
	owner, err = stackedDiff.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner2, owner)

	// Transfer ownership on stacked diff and verify it is only reflected on stacked diff
	stackedDiff.SetNetOwner(netID, owner3)
	owner, err = stackedDiff.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner3, owner)

	owner, err = statesDiff.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner2, owner)

	owner, err = state.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner1, owner)

	// Applying both diffs successively should work as expected.
	require.NoError(stackedDiff.Apply(statesDiff))

	owner, err = statesDiff.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner3, owner)

	owner, err = state.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner1, owner)

	require.NoError(statesDiff.Apply(state))

	owner, err = state.GetNetOwner(netID)
	require.NoError(err)
	require.Equal(owner3, owner)
}
