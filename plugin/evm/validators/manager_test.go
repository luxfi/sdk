// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validators

import (
	"testing"

	"github.com/luxfi/database/memdb"
	"github.com/luxfi/evm/plugin/evm/validators/state"
	"github.com/luxfi/evm/plugin/evm/validators/state/interfaces"
	"github.com/luxfi/ids"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	luxdvalidators "github.com/luxfi/consensus/validators"
)

func TestLoadNewValidators(t *testing.T) {
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
	tests := []struct {
		name                      string
		initialValidators         map[ids.ID]*luxdvalidators.GetValidatorOutput
		newValidators             map[ids.ID]*luxdvalidators.GetValidatorOutput
		registerMockListenerCalls func(*interfaces.MockStateCallbackListener)
		expectedLoadErr           error
	}{
		{
			name:                      "before empty/after empty",
			initialValidators:         map[ids.ID]*luxdvalidators.GetValidatorOutput{},
			newValidators:             map[ids.ID]*luxdvalidators.GetValidatorOutput{},
			registerMockListenerCalls: func(*interfaces.MockStateCallbackListener) {},
		},
		{
			name:              "before empty/after one",
			initialValidators: map[ids.ID]*luxdvalidators.GetValidatorOutput{},
			newValidators: map[ids.ID]*luxdvalidators.GetValidatorOutput{
				testValidationIDs[0]: {
					NodeID: testNodeIDs[0],
					Weight: 1,
				},
			},
			registerMockListenerCalls: func(mock *interfaces.MockStateCallbackListener) {
				mock.EXPECT().OnValidatorAdded(testValidationIDs[0], testNodeIDs[0], uint64(0), true).Times(1)
			},
		},
		{
			name: "before one/after empty",
			initialValidators: map[ids.ID]*luxdvalidators.GetValidatorOutput{
				testValidationIDs[0]: {
					NodeID: testNodeIDs[0],
					Weight: 1,
				},
			},
			newValidators: map[ids.ID]*luxdvalidators.GetValidatorOutput{},
			registerMockListenerCalls: func(mock *interfaces.MockStateCallbackListener) {
				// initial validator will trigger first
				mock.EXPECT().OnValidatorAdded(testValidationIDs[0], testNodeIDs[0], uint64(0), true).Times(1)
				// then it will be removed
				mock.EXPECT().OnValidatorRemoved(testValidationIDs[0], testNodeIDs[0]).Times(1)
			},
		},
		{
			name: "no change",
			initialValidators: map[ids.ID]*luxdvalidators.GetValidatorOutput{
				testValidationIDs[0]: {
					NodeID: testNodeIDs[0],
					Weight: 1,
				},
			},
			newValidators: map[ids.ID]*luxdvalidators.GetValidatorOutput{
				testValidationIDs[0]: {
					NodeID: testNodeIDs[0],
					Weight: 1,
				},
			},
			registerMockListenerCalls: func(mock *interfaces.MockStateCallbackListener) {
				mock.EXPECT().OnValidatorAdded(testValidationIDs[0], testNodeIDs[0], uint64(0), true).Times(1)
			},
		},
		{
			name: "status and weight change and new one",
			initialValidators: map[ids.ID]*luxdvalidators.GetValidatorOutput{
				testValidationIDs[0]: {
					NodeID: testNodeIDs[0],
					Weight: 1,
				},
			},
			newValidators: map[ids.ID]*luxdvalidators.GetValidatorOutput{
				testValidationIDs[0]: {
					NodeID: testNodeIDs[0],
					Weight: 2,
				},
				testValidationIDs[1]: {
					NodeID: testNodeIDs[1],
					Weight: 1,
				},
			},
			registerMockListenerCalls: func(mock *interfaces.MockStateCallbackListener) {
				// initial validator will trigger first
				mock.EXPECT().OnValidatorAdded(testValidationIDs[0], testNodeIDs[0], uint64(0), true).Times(1)
				// Weight change happens but status stays the same (true) since we default to active
				// new validator will be added
				mock.EXPECT().OnValidatorAdded(testValidationIDs[1], testNodeIDs[1], uint64(0), true).Times(1)
			},
		},
		{
			name: "renew validation ID",
			initialValidators: map[ids.ID]*luxdvalidators.GetValidatorOutput{
				testValidationIDs[0]: {
					NodeID: testNodeIDs[0],
					Weight: 1,
				},
			},
			newValidators: map[ids.ID]*luxdvalidators.GetValidatorOutput{
				testValidationIDs[1]: {
					NodeID: testNodeIDs[0],
					Weight: 1,
				},
			},
			registerMockListenerCalls: func(mock *interfaces.MockStateCallbackListener) {
				// initial validator will trigger first
				mock.EXPECT().OnValidatorAdded(testValidationIDs[0], testNodeIDs[0], uint64(0), true).Times(1)
				// then it will be removed
				mock.EXPECT().OnValidatorRemoved(testValidationIDs[0], testNodeIDs[0]).Times(1)
				// new validator will be added
				mock.EXPECT().OnValidatorAdded(testValidationIDs[1], testNodeIDs[0], uint64(0), true).Times(1)
			},
		},
		{
			name: "renew node ID",
			initialValidators: map[ids.ID]*luxdvalidators.GetValidatorOutput{
				testValidationIDs[0]: {
					NodeID: testNodeIDs[0],
					Weight: 1,
				},
			},
			newValidators: map[ids.ID]*luxdvalidators.GetValidatorOutput{
				testValidationIDs[0]: {
					NodeID: testNodeIDs[1],
					Weight: 1,
				},
			},
			expectedLoadErr: state.ErrImmutableField,
			registerMockListenerCalls: func(mock *interfaces.MockStateCallbackListener) {
				// initial validator will trigger first
				mock.EXPECT().OnValidatorAdded(testValidationIDs[0], testNodeIDs[0], uint64(0), true).Times(1)
				// then it won't be called since we don't track the node ID changes
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			require := require.New(tt)
			db := memdb.New()
			validatorState, err := state.NewState(db)
			require.NoError(err)

			// set initial validators
			for vID, validator := range test.initialValidators {
				require.NoError(validatorState.AddValidator(interfaces.Validator{
					ValidationID:   vID,
					NodeID:         validator.NodeID,
					Weight:         validator.Weight,
					StartTimestamp: 0,     // Default value since GetValidatorOutput doesn't have StartTime
					IsActive:       true,  // Default to active
					IsL1Validator:  false, // Default value
				}))
			}
			// enable mock listener
			ctrl := gomock.NewController(tt)
			mockListener := interfaces.NewMockStateCallbackListener(ctrl)
			test.registerMockListenerCalls(mockListener)

			validatorState.RegisterListener(mockListener)
			// load new validators
			err = loadValidators(validatorState, test.newValidators)
			if test.expectedLoadErr != nil {
				require.Error(err)
				return
			}
			require.NoError(err)
			// check if the state is as expected
			require.Equal(len(test.newValidators), validatorState.GetValidationIDs().Len())
			for vID, validator := range test.newValidators {
				v, err := validatorState.GetValidator(vID)
				require.NoError(err)
				require.Equal(validator.NodeID, v.NodeID)
				require.Equal(validator.Weight, v.Weight)
				// Default values are used for fields not in GetValidatorOutput
				require.Equal(uint64(0), v.StartTimestamp)
				require.Equal(true, v.IsActive)
				require.Equal(false, v.IsL1Validator)
			}
		})
	}
}
