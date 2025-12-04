// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validators

import (
	"context"
	"testing"

	validators "github.com/luxfi/consensus/validator"
	"github.com/luxfi/constants"
	"github.com/luxfi/ids"
	"github.com/stretchr/testify/require"
)

// testValidatorState is a mock implementation of validators.State for testing
type testValidatorState struct {
	getValidatorSet      func(context.Context, uint64, ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error)
	getCurrentValidators func(context.Context, uint64, ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error)
	getCurrentHeight     func(context.Context) (uint64, error)
}

func (t *testValidatorState) GetValidatorSet(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	if t.getValidatorSet != nil {
		return t.getValidatorSet(ctx, height, subnetID)
	}
	return nil, nil
}

func (t *testValidatorState) GetCurrentValidators(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	if t.getCurrentValidators != nil {
		return t.getCurrentValidators(ctx, height, subnetID)
	}
	return nil, nil
}

func (t *testValidatorState) GetCurrentHeight(ctx context.Context) (uint64, error) {
	if t.getCurrentHeight != nil {
		return t.getCurrentHeight(ctx)
	}
	return 0, nil
}

func (t *testValidatorState) GetWarpValidatorSet(ctx context.Context, height uint64, netID ids.ID) (*validators.WarpSet, error) {
	return &validators.WarpSet{
		Height:     height,
		Validators: make(map[ids.NodeID]*validators.WarpValidator),
	}, nil
}

func (t *testValidatorState) GetWarpValidatorSets(ctx context.Context, heights []uint64, netIDs []ids.ID) (map[ids.ID]map[uint64]*validators.WarpSet, error) {
	result := make(map[ids.ID]map[uint64]*validators.WarpSet)
	for _, netID := range netIDs {
		result[netID] = make(map[uint64]*validators.WarpSet)
		for _, height := range heights {
			result[netID][height] = &validators.WarpSet{
				Height:     height,
				Validators: make(map[ids.NodeID]*validators.WarpValidator),
			}
		}
	}
	return result, nil
}

func TestGetValidatorSetPrimaryNetwork(t *testing.T) {
	require := require.New(t)

	mySubnetID := ids.GenerateTestID()
	otherSubnetID := ids.GenerateTestID()
	myChainID := ids.GenerateTestID()

	// Create a mock state with the necessary functions
	mockState := &testValidatorState{
		getValidatorSet: func(_ context.Context, _ uint64, subnetID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
			// Return empty validator set for any subnet
			return make(map[ids.NodeID]*validators.GetValidatorOutput), nil
		},
		getCurrentValidators: func(_ context.Context, _ uint64, subnetID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
			// Return empty validator set for any subnet
			return make(map[ids.NodeID]*validators.GetValidatorOutput), nil
		},
		getCurrentHeight: func(_ context.Context) (uint64, error) {
			return 0, nil
		},
	}

	state := NewState(mockState, mySubnetID, myChainID, false)

	// Test that requesting my validator set returns my validator set
	output, err := state.GetValidatorSet(context.Background(), 10, mySubnetID)
	require.NoError(err)
	require.Len(output, 0)

	// Test that requesting the Primary Network validator set overrides and returns my validator set
	output, err = state.GetValidatorSet(context.Background(), 10, constants.PrimaryNetworkID)
	require.NoError(err)
	require.Len(output, 0)

	// Test that requesting other validator set returns that validator set
	output, err = state.GetValidatorSet(context.Background(), 10, otherSubnetID)
	require.NoError(err)
	require.Len(output, 0)
}
