// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utilstest

import (
	"context"
	"testing"

	"github.com/luxfi/consensus"
	"github.com/luxfi/ids"
	"github.com/stretchr/testify/require"
)

func TestNewTestConsensusContext(t *testing.T) {
	// Test that NewTestConsensusContext creates a context with validator state
	consensusCtx := NewTestConsensusContext(t)

	// Extract validator state from context using consensus.GetValidatorState
	validatorState := consensus.GetValidatorState(consensusCtx)
	require.NotNil(t, validatorState)

	// Test that we can call GetValidatorSet without panicking
	validators, err := validatorState.GetValidatorSet(0, ids.Empty)
	require.NoError(t, err)
	require.NotNil(t, validators)

	// Test GetCurrentHeight
	height, err := validatorState.GetCurrentHeight(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(0), height)

	// Test GetSubnetID
	subnetID, err := validatorState.GetSubnetID(SubnetEVMTestChainID)
	require.NoError(t, err)
	require.Equal(t, ids.Empty, subnetID) // Default returns empty ID
}
