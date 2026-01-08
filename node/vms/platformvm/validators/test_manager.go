// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package validators

import (
	"context"

	consensusset "github.com/luxfi/consensus/utils/set"
	validators "github.com/luxfi/consensus/validator"
	"github.com/luxfi/ids"
)

var TestManager Manager = testManager{}

type testManager struct{}

func (testManager) GetMinimumHeight(context.Context) (uint64, error) {
	return 0, nil
}

func (testManager) GetCurrentHeight(context.Context) (uint64, error) {
	return 0, nil
}

func (testManager) GetNetID(context.Context, ids.ID) (ids.ID, error) {
	return ids.Empty, nil
}

func (testManager) GetValidatorSet(ctx context.Context, height uint64, netID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	return nil, nil
}

func (testManager) GetCurrentValidators(ctx context.Context, height uint64, netID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	return nil, nil
}

// func (testManager) GetCurrentValidatorSet(context.Context, ids.ID) (map[ids.ID]*validators.GetCurrentValidatorOutput, uint64, error) {
// 	return nil, 0, nil
// }

func (testManager) OnAcceptedBlockID(ids.ID) {}

// AddStaker implements validators.Manager interface
func (testManager) AddStaker(ids.ID, ids.NodeID, []byte, ids.ID, uint64) error {
	return nil
}

// AddWeight implements validators.Manager interface
func (testManager) AddWeight(ids.ID, ids.NodeID, uint64) error {
	return nil
}

// RemoveWeight implements validators.Manager interface
func (testManager) RemoveWeight(ids.ID, ids.NodeID, uint64) error {
	return nil
}

// GetWeight implements validators.Manager interface
func (testManager) GetWeight(ids.ID, ids.NodeID) uint64 {
	return 0
}

// SubsetWeight implements validators.Manager interface
func (testManager) SubsetWeight(ids.ID, consensusset.Set[ids.NodeID]) (uint64, error) {
	return 0, nil
}

// TotalWeight implements validators.Manager interface
func (testManager) TotalWeight(ids.ID) (uint64, error) {
	return 0, nil
}

// GetValidator implements validators.Manager interface
func (testManager) GetValidator(ids.ID, ids.NodeID) (*validators.GetValidatorOutput, bool) {
	return nil, false
}

// GetValidatorIDs implements validators.Manager interface
func (testManager) GetValidatorIDs(ids.ID) []ids.NodeID {
	return nil
}

// Count implements validators.Manager interface
func (testManager) Count(ids.ID) int {
	return 0
}

// NumValidators implements validators.Manager interface
func (testManager) NumValidators(ids.ID) int {
	return 0
}

// RegisterSetCallbackListener implements validators.Manager interface
func (testManager) RegisterSetCallbackListener(ids.ID, validators.SetCallbackListener) {}

// RegisterWeightCallbackListener removed - doesn't exist in consensus

// GetValidators implements validators.Manager interface
func (testManager) GetValidators(ids.ID) (validators.Set, error) {
	return nil, nil
}

// TotalLight implements validators.Manager interface
func (testManager) TotalLight(ids.ID) (uint64, error) {
	return 0, nil
}

// String implements validators.Manager interface
func (testManager) String() string {
	return "test_manager"
}

// GetWarpValidatorSet implements validators.Manager interface
func (testManager) GetWarpValidatorSet(context.Context, uint64, ids.ID) (*validators.WarpSet, error) {
	return &validators.WarpSet{
		Height:     0,
		Validators: make(map[ids.NodeID]*validators.WarpValidator),
	}, nil
}

// GetWarpValidatorSets implements validators.Manager interface
func (testManager) GetWarpValidatorSets(context.Context, []uint64, []ids.ID) (map[ids.ID]map[uint64]*validators.WarpSet, error) {
	return make(map[ids.ID]map[uint64]*validators.WarpSet), nil
}
