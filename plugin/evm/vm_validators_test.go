// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"testing"
	"time"

	"github.com/luxfi/consensus"
	commonEng "github.com/luxfi/consensus/core"
	luxdvalidators "github.com/luxfi/consensus/validators"
	"github.com/luxfi/consensus/validators/validatorstest"
	"github.com/luxfi/database"
	"github.com/luxfi/evm/plugin/evm/validators"
	"github.com/luxfi/evm/utils/utilstest"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/upgrade/upgradetest"
	"github.com/luxfi/math/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatorState(t *testing.T) {
	require := require.New(t)
	ctx, dbManager, genesisBytes, _ := setupGenesis(t, upgradetest.Latest)

	vm := &VM{}

	appSender := &TestSender{T: t}
	appSender.CantSendAppGossip = true
	testNodeIDs := []ids.NodeID{
		ids.GenerateTestNodeID(),
		ids.GenerateTestNodeID(),
		ids.GenerateTestNodeID(),
	}
	testValidationIDs := []ids.ID{
		ids.GenerateTestID(),
		ids.GenerateTestID(),
		ids.GenerateTestID(),
	}
	validatorState := &validatorstest.State{
		GetCurrentValidatorSetF: func(ctx context.Context, subnetID ids.ID) (map[ids.ID]*luxdvalidators.GetCurrentValidatorOutput, uint64, error) {
			return map[ids.ID]*luxdvalidators.GetCurrentValidatorOutput{
				testValidationIDs[0]: {
					NodeID:    testNodeIDs[0],
					PublicKey: nil,
					Weight:    1,
				},
				testValidationIDs[1]: {
					NodeID:    testNodeIDs[1],
					PublicKey: nil,
					Weight:    1,
				},
				testValidationIDs[2]: {
					NodeID:    testNodeIDs[2],
					PublicKey: nil,
					Weight:    1,
				},
			}, 0, nil
		},
		GetCurrentHeightF: func(context.Context) (uint64, error) {
			return 0, nil
		},
		GetMinimumHeightF: func(context.Context) (uint64, error) {
			return 0, nil
		},
		GetValidatorSetF: func(context.Context, uint64, ids.ID) (map[ids.NodeID]*luxdvalidators.GetValidatorOutput, error) {
			return map[ids.NodeID]*luxdvalidators.GetValidatorOutput{}, nil
		},
		GetSubnetIDF: func(context.Context, ids.ID) (ids.ID, error) {
			return ids.Empty, nil
		},
	}
	// Create a wrapper that implements consensus.ValidatorState interface
	wrappedValidatorState := utilstest.NewTestValidatorStateFromBase(validatorState)
	ctx = consensus.WithValidatorState(ctx, wrappedValidatorState)
	appSender.SendAppGossipF = func(context.Context, set.Set[ids.NodeID], []byte) error { return nil }
	err := vm.Initialize(
		context.Background(),
		ctx,
		dbManager,
		genesisBytes,
		[]byte(""),
		[]byte(""),
		[]*commonEng.Fx{},
		appSender,
	)
	require.NoError(err, "error initializing GenesisVM")

	// Test case 1: state should not be populated until bootstrapped
	require.NoError(vm.SetState(context.Background(), consensus.Bootstrapping))
	require.Equal(0, vm.validatorsManager.GetValidationIDs().Len())
	_, _, err = vm.validatorsManager.CalculateUptime(testNodeIDs[0])
	require.ErrorIs(database.ErrNotFound, err)
	require.False(vm.validatorsManager.StartedTracking())

	// Test case 2: state should be populated after bootstrapped
	require.NoError(vm.SetState(context.Background(), consensus.NormalOp))
	require.Len(vm.validatorsManager.GetValidationIDs(), 3)
	_, _, err = vm.validatorsManager.CalculateUptime(testNodeIDs[0])
	require.NoError(err)
	require.True(vm.validatorsManager.StartedTracking())

	// Test case 3: restarting VM should not lose state
	vm.Shutdown(context.Background())
	// Shutdown should stop tracking
	require.False(vm.validatorsManager.StartedTracking())

	vm = &VM{}
	err = vm.Initialize(
		context.Background(),
		utilstest.NewTestConsensusContext(t), // this context does not have validators state, making VM to source it from the database
		dbManager,
		genesisBytes,
		[]byte(""),
		[]byte(""),
		[]*commonEng.Fx{},
		appSender,
	)
	require.NoError(err, "error initializing GenesisVM")
	require.Len(vm.validatorsManager.GetValidationIDs(), 3)
	_, _, err = vm.validatorsManager.CalculateUptime(testNodeIDs[0])
	require.NoError(err)
	require.False(vm.validatorsManager.StartedTracking())

	// Test case 4: new validators should be added to the state
	newValidationID := ids.GenerateTestID()
	newNodeID := ids.GenerateTestNodeID()
	testState := &validatorstest.State{
		GetCurrentValidatorSetF: func(ctx context.Context, subnetID ids.ID) (map[ids.ID]*luxdvalidators.GetCurrentValidatorOutput, uint64, error) {
			return map[ids.ID]*luxdvalidators.GetCurrentValidatorOutput{
				testValidationIDs[0]: {
					NodeID:    testNodeIDs[0],
					PublicKey: nil,
					Weight:    1,
				},
				testValidationIDs[1]: {
					NodeID:    testNodeIDs[1],
					PublicKey: nil,
					Weight:    1,
				},
				testValidationIDs[2]: {
					NodeID:    testNodeIDs[2],
					PublicKey: nil,
					Weight:    1,
				},
				newValidationID: {
					NodeID:    newNodeID,
					PublicKey: nil,
					Weight:    1,
				},
			}, 0, nil
		},
	}
	// set VM as bootstrapped
	require.NoError(vm.SetState(context.Background(), consensus.Bootstrapping))
	require.NoError(vm.SetState(context.Background(), consensus.NormalOp))

	// Update the VM's context with the new validator state
	wrappedTestState := utilstest.NewTestValidatorStateFromBase(testState)
	vm.ctx = consensus.WithValidatorState(vm.ctx, wrappedTestState)

	// new validator should be added to the state eventually after SyncFrequency
	require.EventuallyWithT(func(c *assert.CollectT) {
		vm.vmLock.Lock()
		defer vm.vmLock.Unlock()
		assert.Len(c, vm.validatorsManager.GetNodeIDs(), 4)
		newValidator, err := vm.validatorsManager.GetValidator(newValidationID)
		assert.NoError(c, err)
		assert.Equal(c, newNodeID, newValidator.NodeID)
	}, validators.SyncFrequency*2, 5*time.Second)
}
