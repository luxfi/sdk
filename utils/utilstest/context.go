// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utilstest

import (
	"context"
	"testing"

	"github.com/luxfi/ids"
	"github.com/luxfi/consensus"
	"github.com/luxfi/consensus/consensustest"
	"github.com/luxfi/consensus/validators"
	"github.com/luxfi/consensus/validators/validatorstest"
	"github.com/luxfi/node/utils/constants"
)

// SubnetEVMTestChainID is a evm specific chain ID for testing
var SubnetEVMTestChainID = ids.GenerateTestID()

// testValidatorState wraps validatorstest.State to implement consensus.ValidatorState
type testValidatorState struct {
	*validatorstest.State
}

func (t *testValidatorState) GetCurrentHeight() (uint64, error) {
	return t.State.GetCurrentHeight(context.Background())
}

func (t *testValidatorState) GetMinimumHeight(ctx context.Context) (uint64, error) {
	return t.State.GetMinimumHeight(ctx)
}

func (t *testValidatorState) GetValidatorSet(height uint64, subnetID ids.ID) (map[ids.NodeID]uint64, error) {
	// Get the validator set with GetValidatorOutput
	validators, err := t.State.GetValidatorSet(context.Background(), height, subnetID)
	if err != nil {
		return nil, err
	}
	
	// Convert to map[ids.NodeID]uint64 for consensus interface
	result := make(map[ids.NodeID]uint64, len(validators))
	for nodeID, output := range validators {
		result[nodeID] = output.Weight
	}
	return result, nil
}

func (t *testValidatorState) GetSubnetID(chainID ids.ID) (ids.ID, error) {
	return t.State.GetSubnetID(context.Background(), chainID)
}

// @TODO: This should eventually be replaced by a more robust solution, or alternatively, the presence of nil
// validator states shouldn't be depended upon by tests
func NewTestValidatorState() consensus.ValidatorState {
	state := &validatorstest.State{
		GetCurrentHeightF: func(context.Context) (uint64, error) {
			return 0, nil
		},
		GetSubnetIDF: func(_ context.Context, chainID ids.ID) (ids.ID, error) {
			// For testing, all chains belong to the primary network
			if chainID == constants.PlatformChainID || chainID == SubnetEVMTestChainID {
				return constants.PrimaryNetworkID, nil
			}
			// Default to primary network for any test chain
			return constants.PrimaryNetworkID, nil
		},
		GetValidatorSetF: func(context.Context, uint64, ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
			return map[ids.NodeID]*validators.GetValidatorOutput{}, nil
		},
		GetCurrentValidatorSetF: func(context.Context, ids.ID) (map[ids.ID]*validators.GetCurrentValidatorOutput, uint64, error) {
			return map[ids.ID]*validators.GetCurrentValidatorOutput{}, 0, nil
		},
	}
	
	return &testValidatorState{State: state}
}

// NewTestConsensusContext returns a context.Context with validator state properly configured for testing.
// This wraps consensustest.Context and sets the validator state to avoid the missing GetValidatorSetF issue.
//
// Usage example:
//
//	// Instead of:
//	// consensusCtx := utilstest.NewTestConsensusContext(t, consensustest.CChainID)
//	// validatorState := utils.NewTestValidatorState()
//	// consensusCtx.ValidatorState = validatorState
//
//	// Use:
//	consensusCtx := utils.NewTestConsensusContext(t)
//
// This function ensures that the consensus context has a properly configured validator state
// that includes the GetValidatorSetF function, which is required by many tests.
func NewTestConsensusContext(t testing.TB) context.Context {
	consensusCtx := consensustest.Context(t, SubnetEVMTestChainID)
	// Use consensus.WithValidatorState to add validator state to context
	consensusCtx = consensus.WithValidatorState(consensusCtx, NewTestValidatorState())
	return consensusCtx
}

// NewTestConsensusContextWithChainID returns a context.Context with validator state properly configured for testing
// with a specific chain ID. This is provided for backward compatibility when a specific chain ID is needed.
func NewTestConsensusContextWithChainID(t testing.TB, chainID ids.ID) context.Context {
	consensusCtx := consensustest.Context(t, chainID)
	// Use consensus.WithValidatorState to add validator state to context
	consensusCtx = consensus.WithValidatorState(consensusCtx, NewTestValidatorState())
	return consensusCtx
}
