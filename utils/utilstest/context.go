// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utilstest

import (
	"context"
	"testing"

	consensuscontext "github.com/luxfi/consensus/context"
	consensustest "github.com/luxfi/consensus/test/helpers"
	validators "github.com/luxfi/consensus/validator"
	"github.com/luxfi/consensus/validator/validatorstest"
	"github.com/luxfi/ids"
)

// SubnetEVMTestChainID is a evm specific chain ID for testing
var SubnetEVMTestChainID = ids.GenerateTestID()

// testValidatorState wraps validatorstest.State to implement consensuscontext.ValidatorState
type testValidatorState struct {
	*validatorstest.State
}

func (t *testValidatorState) GetCurrentHeight(ctx context.Context) (uint64, error) {
	// Call GetCurrentHeightF if available, otherwise return 0
	if t.State != nil && t.State.GetCurrentHeightF != nil {
		return t.State.GetCurrentHeightF(ctx)
	}
	return 0, nil
}

func (t *testValidatorState) GetMinimumHeight(ctx context.Context) (uint64, error) {
	// Return 0 for test purposes - minimum height for testing
	return 0, nil
}

func (t *testValidatorState) GetValidatorSet(height uint64, subnetID ids.ID) (map[ids.NodeID]uint64, error) {
	// Delegate to the underlying State's GetValidatorSetF and convert output
	if t.State != nil && t.State.GetValidatorSetF != nil {
		// GetValidatorSetF returns map[ids.NodeID]*validators.GetValidatorOutput
		// Convert to map[ids.NodeID]uint64 (just weights)
		fullOutput, err := t.State.GetValidatorSetF(context.Background(), height, subnetID)
		if err != nil {
			return nil, err
		}
		result := make(map[ids.NodeID]uint64, len(fullOutput))
		for nodeID, output := range fullOutput {
			result[nodeID] = output.Weight
		}
		return result, nil
	}
	return make(map[ids.NodeID]uint64), nil
}

// chainIDToSubnetID maps chain IDs to their subnet IDs for testing
var chainIDToSubnetID = make(map[ids.ID]ids.ID)

// SetChainSubnetMapping registers a chain ID to subnet ID mapping for tests
func SetChainSubnetMapping(chainID, subnetID ids.ID) {
	chainIDToSubnetID[chainID] = subnetID
}

// ClearChainSubnetMapping clears the chain to subnet mapping
func ClearChainSubnetMapping() {
	chainIDToSubnetID = make(map[ids.ID]ids.ID)
}

func (t *testValidatorState) GetSubnetID(chainID ids.ID) (ids.ID, error) {
	// Check the global mapping first
	if subnetID, ok := chainIDToSubnetID[chainID]; ok {
		return subnetID, nil
	}
	// Default to empty (primary network) for any chain
	return ids.Empty, nil
}

func (t *testValidatorState) GetChainID(subnetID ids.ID) (ids.ID, error) {
	return ids.Empty, nil // TODO: Fix GetChainID
}

// GetValidatorSetWithOutput implements the ValidatorOutputGetter interface
// This returns the full validator output including public keys
func (t *testValidatorState) GetValidatorSetWithOutput(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	// Delegate to the underlying State's GetValidatorSetF
	if t.State != nil && t.State.GetValidatorSetF != nil {
		return t.State.GetValidatorSetF(ctx, height, subnetID)
	}
	return make(map[ids.NodeID]*validators.GetValidatorOutput), nil
}

func (t *testValidatorState) GetNetID(chainID ids.ID) (ids.ID, error) {
	return ids.Empty, nil // TODO: Fix GetNetID
}

// @TODO: This should eventually be replaced by a more robust solution, or alternatively, the presence of nil
// validator states shouldn't be depended upon by tests
func NewTestValidatorState() consensuscontext.ValidatorState {
	state := &validatorstest.State{
		GetCurrentHeightF: func(context.Context) (uint64, error) {
			return 0, nil
		},
		// GetSubnetIDF: func(chainID ids.ID) (ids.ID, error) { // TODO: Fix GetSubnetIDF field
		// 	// For testing, all chains belong to the primary network
		// 	if chainID == constants.PlatformChainID || chainID == SubnetEVMTestChainID {
		// 		return constants.PrimaryNetworkID, nil
		// 	}
		// 	// Default to primary network for any test chain
		// 	return constants.PrimaryNetworkID, nil
		// },
		GetValidatorSetF: func(context.Context, uint64, ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
			return map[ids.NodeID]*validators.GetValidatorOutput{}, nil
		},
	}

	return &testValidatorState{State: state}
}

// NewTestValidatorStateFromBase creates a testValidatorState that wraps an existing validatorstest.State
// This is useful when you need to use a specific validatorstest.State with custom functions
// but still implement the consensuscontext.ValidatorState interface.
func NewTestValidatorStateFromBase(baseState *validatorstest.State) consensuscontext.ValidatorState {
	return &testValidatorState{State: baseState}
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
	// Create a standard context and add the consensus context to it
	ctx := context.Background()
	ctx = consensuscontext.WithContext(ctx, consensusCtx)
	// Add validator state to the context
	return consensuscontext.WithValidatorState(ctx, NewTestValidatorState())
}

// NewTestConsensusContextWithChainID returns a context.Context with validator state properly configured for testing
// with a specific chain ID. This is provided for backward compatibility when a specific chain ID is needed.
func NewTestConsensusContextWithChainID(t testing.TB, chainID ids.ID) context.Context {
	consensusCtx := consensustest.Context(t, chainID)
	// Create a standard context and add the consensus context to it
	ctx := context.Background()
	ctx = consensuscontext.WithContext(ctx, consensusCtx)
	// Add validator state to the context
	return consensuscontext.WithValidatorState(ctx, NewTestValidatorState())
}
