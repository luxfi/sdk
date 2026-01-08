// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package validators

import (
	"context"
	"sync"

	validators "github.com/luxfi/consensus/validator"
	"github.com/luxfi/ids"
)

// NewLockedState creates a new locked validator state
func NewLockedState(lock sync.Locker, state validators.State) validators.State {
	return &lockedState{
		lock:  lock,
		State: state,
	}
}

type lockedState struct {
	lock sync.Locker
	validators.State
}

func (ls *lockedState) GetCurrentHeight(ctx context.Context) (uint64, error) {
	ls.lock.Lock()
	defer ls.lock.Unlock()
	return ls.State.GetCurrentHeight(ctx)
}

func (ls *lockedState) GetValidatorSet(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	ls.lock.Lock()
	defer ls.lock.Unlock()
	return ls.State.GetValidatorSet(ctx, height, subnetID)
}

func (ls *lockedState) GetCurrentValidators(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	ls.lock.Lock()
	defer ls.lock.Unlock()
	if state, ok := ls.State.(interface {
		GetCurrentValidators(context.Context, uint64, ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error)
	}); ok {
		return state.GetCurrentValidators(ctx, height, subnetID)
	}
	return ls.State.GetValidatorSet(ctx, height, subnetID)
}
