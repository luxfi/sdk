// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utilstest

import (
	"context"
	"testing"

	"github.com/luxfi/ids"
	"github.com/stretchr/testify/require"
)

func TestNewTestConsensusContext(t *testing.T) {
	// Test that NewTestConsensusContext creates a context with validator state
	consensusCtx := NewTestConsensusContext(t)
	require.NotNil(t, consensusCtx.ValidatorState)

	// Test that the validator state has the required functions
	validatorState := consensusCtx.ValidatorState
	require.NotNil(t, validatorState)

	// Test that we can call GetValidatorSetF without panicking
	validators, err := validatorState.GetValidatorSet(context.TODO(), 0, ids.Empty)
	require.NoError(t, err)
	require.NotNil(t, validators)

	// Test that we can call GetCurrentValidatorSetF without panicking
	currentValidators, height, err := validatorState.GetCurrentValidatorSet(context.TODO(), ids.Empty)
	require.NoError(t, err)
	require.NotNil(t, currentValidators)
	require.Equal(t, uint64(0), height)
}
