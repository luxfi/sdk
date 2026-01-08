// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validatorstest

import (
	"context"

	"github.com/luxfi/ids"

	consensusvalidators "github.com/luxfi/consensus/validator"
	vmvalidators "github.com/luxfi/sdk/node/vms/platformvm/validators"
)

var Manager vmvalidators.Manager = manager{}

type manager struct{}

func (manager) GetMinimumHeight(context.Context) (uint64, error) {
	return 0, nil
}

func (manager) GetCurrentHeight(context.Context) (uint64, error) {
	return 0, nil
}

func (manager) GetNetID(context.Context, ids.ID) (ids.ID, error) {
	return ids.Empty, nil
}

func (manager) GetValidatorSet(context.Context, uint64, ids.ID) (map[ids.NodeID]*consensusvalidators.GetValidatorOutput, error) {
	return nil, nil
}

func (manager) GetCurrentValidators(context.Context, uint64, ids.ID) (map[ids.NodeID]*consensusvalidators.GetValidatorOutput, error) {
	return nil, nil
}

func (manager) OnAcceptedBlockID(ids.ID) {}

func (manager) GetCurrentValidatorSet(context.Context, ids.ID) (map[ids.ID]*consensusvalidators.GetValidatorOutput, uint64, error) {
	return nil, 0, nil
}

func (manager) GetWarpValidatorSet(context.Context, uint64, ids.ID) (*consensusvalidators.WarpSet, error) {
	return nil, nil
}

func (manager) GetWarpValidatorSets(context.Context, []uint64, []ids.ID) (map[ids.ID]map[uint64]*consensusvalidators.WarpSet, error) {
	return nil, nil
}
