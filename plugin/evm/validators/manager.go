// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validators

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/consensus"
	luxuptime "github.com/luxfi/node/consensus/uptime"
	luxvalidators "github.com/luxfi/node/consensus/validators"
	"github.com/luxfi/node/utils/timer/mockable"
	validators "github.com/luxfi/evm/plugin/evm/validators/state"
	stateinterfaces "github.com/luxfi/evm/plugin/evm/validators/state/interfaces"
	"github.com/luxfi/evm/plugin/evm/validators/uptime"
	uptimeinterfaces "github.com/luxfi/evm/plugin/evm/validators/uptime/interfaces"

	"github.com/luxfi/geth/log"
)

const (
	SyncFrequency = 1 * time.Minute
)

type manager struct {
	chainCtx *consensus.Context
	stateinterfaces.State
	uptimeinterfaces.PausableManager
}

// NewManager returns a new validator manager
// that manages the validator state and the uptime manager.
// Manager is not thread safe and should be used with the VM locked.
func NewManager(
	ctx *consensus.Context,
	db database.Database,
	clock *mockable.Clock,
) (*manager, error) {
	validatorState, err := validators.NewState(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize validator state: %w", err)
	}

	// Initialize uptime manager
	uptimeManager := uptime.NewPausableManager(luxuptime.NewManager(validatorState, clock))
	validatorState.RegisterListener(uptimeManager)

	return &manager{
		chainCtx:        ctx,
		State:           validatorState,
		PausableManager: uptimeManager,
	}, nil
}

// Initialize initializes the validator manager
// by syncing the validator state with the current validator set
// and starting the uptime tracking.
func (m *manager) Initialize(ctx context.Context) error {
	// sync validators first
	if err := m.sync(ctx); err != nil {
		return fmt.Errorf("failed to update validators: %w", err)
	}
	vdrIDs := m.GetNodeIDs().List()
	// Then start tracking with updated validators
	// StartTracking initializes the uptime tracking with the known validators
	// and update their uptime to account for the time we were being offline.
	if err := m.StartTracking(vdrIDs); err != nil {
		return fmt.Errorf("failed to start tracking uptime: %w", err)
	}
	return nil
}

// Shutdown stops the uptime tracking and writes the validator state to the database.
func (m *manager) Shutdown() error {
	vdrIDs := m.GetNodeIDs().List()
	if err := m.StopTracking(vdrIDs); err != nil {
		return fmt.Errorf("failed to stop tracking uptime: %w", err)
	}
	if err := m.WriteState(); err != nil {
		return fmt.Errorf("failed to write validator: %w", err)
	}
	return nil
}

// DispatchSync starts the sync process
// DispatchSync holds the given lock while performing the sync.
func (m *manager) DispatchSync(ctx context.Context, lock sync.Locker) {
	ticker := time.NewTicker(SyncFrequency)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lock.Lock()
			if err := m.sync(ctx); err != nil {
				log.Error("failed to sync validators", "error", err)
			}
			lock.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

// sync synchronizes the validator state with the current validator set
// and writes the state to the database.
// sync is not safe to call concurrently and should be called with the VM locked.
func (m *manager) sync(ctx context.Context) error {
	now := time.Now()
	log.Debug("performing validator sync")
	// get current validator set
	currentValidatorSet, _, err := m.chainCtx.ValidatorState.GetCurrentValidatorSet(ctx, m.chainCtx.SubnetID)
	if err != nil {
		return fmt.Errorf("failed to get current validator set: %w", err)
	}

	// load the current validator set into the validator state
	if err := loadValidators(m.State, currentValidatorSet); err != nil {
		return fmt.Errorf("failed to load current validators: %w", err)
	}

	// write validators to the database
	if err := m.State.WriteState(); err != nil {
		return fmt.Errorf("failed to write validator state: %w", err)
	}

	// TODO: add metrics
	log.Debug("validator sync complete", "duration", time.Since(now))
	return nil
}

// loadValidators loads the [validators] into the validator state [validatorState]
func loadValidators(validatorState stateinterfaces.State, newValidators map[ids.ID]*luxvalidators.GetCurrentValidatorOutput) error {
	currentValidationIDs := validatorState.GetValidationIDs()
	// first check if we need to delete any existing validators
	for vID := range currentValidationIDs {
		// if the validator is not in the new set of validators
		// delete the validator
		if _, exists := newValidators[vID]; !exists {
			validatorState.DeleteValidator(vID)
		}
	}

	// then load the new validators
	for newVID, newVdr := range newValidators {
		currentVdr := stateinterfaces.Validator{
			ValidationID:   newVID,
			NodeID:         newVdr.NodeID,
			Weight:         newVdr.Weight,
			StartTimestamp: newVdr.StartTime,
			IsActive:       newVdr.IsActive,
			IsL1Validator:  newVdr.IsL1Validator,
		}
		if currentValidationIDs.Contains(newVID) {
			if err := validatorState.UpdateValidator(currentVdr); err != nil {
				return err
			}
		} else {
			if err := validatorState.AddValidator(currentVdr); err != nil {
				return err
			}
		}
	}
	return nil
}
