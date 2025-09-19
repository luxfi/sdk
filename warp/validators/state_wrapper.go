// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validators

import (
	"context"

	"github.com/luxfi/consensus"
	"github.com/luxfi/consensus/validators"
	"github.com/luxfi/ids"
)

// ConsensusStateWrapper wraps consensus.ValidatorState to implement validators.State
type ConsensusStateWrapper struct {
	vs consensus.ValidatorState
}

// NewConsensusStateWrapper creates a new wrapper
func NewConsensusStateWrapper(vs consensus.ValidatorState) *ConsensusStateWrapper {
	return &ConsensusStateWrapper{vs: vs}
}

// GetCurrentHeight implements validators.State
func (w *ConsensusStateWrapper) GetCurrentHeight(ctx context.Context) (uint64, error) {
	// consensus.ValidatorState doesn't use context for GetCurrentHeight
	return w.vs.GetCurrentHeight()
}

// GetValidatorSet implements validators.State
func (w *ConsensusStateWrapper) GetValidatorSet(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	// consensus.ValidatorState returns a simpler map, need to convert
	simpleMap, err := w.vs.GetValidatorSet(height, subnetID)
	if err != nil {
		return nil, err
	}

	// Convert map[ids.NodeID]uint64 to map[ids.NodeID]*validators.GetValidatorOutput
	result := make(map[ids.NodeID]*validators.GetValidatorOutput)
	for nodeID, weight := range simpleMap {
		result[nodeID] = &validators.GetValidatorOutput{
			NodeID: nodeID,
			Weight: weight,
		}
	}
	return result, nil
}

// GetMinimumHeight implements validators.State
func (w *ConsensusStateWrapper) GetMinimumHeight(ctx context.Context) (uint64, error) {
	// consensus.ValidatorState doesn't have GetMinimumHeight, return 0
	return 0, nil
}

// GetSubnetID implements validators.State
func (w *ConsensusStateWrapper) GetSubnetID(ctx context.Context, chainID ids.ID) (ids.ID, error) {
	// consensus.ValidatorState doesn't have GetSubnetID, return empty
	return ids.Empty, nil
}

// GetCurrentValidatorSet implements validators.State
func (w *ConsensusStateWrapper) GetCurrentValidatorSet(ctx context.Context, subnetID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	// Get current height and use GetValidatorSet
	height, err := w.vs.GetCurrentHeight()
	if err != nil {
		return nil, err
	}
	return w.GetValidatorSet(ctx, height, subnetID)
}
