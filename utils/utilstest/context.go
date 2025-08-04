// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utilstest

import (
	"context"
	"errors"
	"testing"

	"github.com/luxfi/ids"
	"github.com/luxfi/node/consensus"
	"github.com/luxfi/node/consensus/consensustest"
	"github.com/luxfi/node/consensus/validators"
	"github.com/luxfi/node/consensus/validators/validatorstest"
	"github.com/luxfi/node/utils/constants"
)

// SubnetEVMTestChainID is a evm specific chain ID for testing
var SubnetEVMTestChainID = ids.GenerateTestID()

// @TODO: This should eventually be replaced by a more robust solution, or alternatively, the presence of nil
// validator states shouldn't be depended upon by tests
func NewTestValidatorState() *validatorstest.State {
	return &validatorstest.State{
		GetCurrentHeightF: func(context.Context) (uint64, error) {
			return 0, nil
		},
		GetSubnetIDF: func(_ context.Context, chainID ids.ID) (ids.ID, error) {
			subnetID, ok := map[ids.ID]ids.ID{
				constants.PlatformChainID: constants.PrimaryNetworkID,
				consensustest.XChainID:         constants.PrimaryNetworkID,
				consensustest.CChainID:         constants.PrimaryNetworkID,
				SubnetEVMTestChainID:      constants.PrimaryNetworkID,
			}[chainID]
			if !ok {
				return ids.Empty, errors.New("unknown chain")
			}
			return subnetID, nil
		},
		GetValidatorSetF: func(context.Context, uint64, ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
			return map[ids.NodeID]*validators.GetValidatorOutput{}, nil
		},
		GetCurrentValidatorSetF: func(context.Context, ids.ID) (map[ids.ID]*validators.GetCurrentValidatorOutput, uint64, error) {
			return map[ids.ID]*validators.GetCurrentValidatorOutput{}, 0, nil
		},
	}
}

// NewTestConsensusContext returns a consensus.Context with validator state properly configured for testing.
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
func NewTestConsensusContext(t testing.TB) *consensus.Context {
	consensusCtx := consensustest.Context(t, SubnetEVMTestChainID)
	consensusCtx.ValidatorState = NewTestValidatorState()
	return consensusCtx
}

// NewTestConsensusContextWithChainID returns a consensus.Context with validator state properly configured for testing
// with a specific chain ID. This is provided for backward compatibility when a specific chain ID is needed.
func NewTestConsensusContextWithChainID(t testing.TB, chainID ids.ID) *consensus.Context {
	consensusCtx := consensustest.Context(t, chainID)
	consensusCtx.ValidatorState = NewTestValidatorState()
	return consensusCtx
}
