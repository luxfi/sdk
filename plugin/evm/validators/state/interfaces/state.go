// Copyright (C) 2019-2024, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package interfaces

import (
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/utils"
)

type StateReader interface {
	// GetValidator returns the validator data for the given validation ID
	GetValidator(vID interfaces.ID) (Validator, error)
	// GetValidationIDs returns the validation IDs in the state
	GetValidationIDs() interfaces.Set[interfaces.ID]
	// GetNodeIDs returns the validator node IDs in the state
	GetNodeIDs() interfaces.Set[interfaces.NodeID]
	// GetValidationID returns the validation ID for the given node ID
	GetValidationID(nodeID interfaces.NodeID) (interfaces.ID, error)
}

type State interface {
	uptime.State
	StateReader
	// AddValidator adds a new validator to the state
	AddValidator(vdr Validator) error
	// UpdateValidator updates the validator in the state
	UpdateValidator(vdr Validator) error
	// DeleteValidator deletes the validator from the state
	DeleteValidator(vID interfaces.ID) error
	// WriteState writes the validator state to the disk
	WriteState() error
	// RegisterListener registers a listener to the state
	RegisterListener(StateCallbackListener)
}

// StateCallbackListener is a listener for the validator state
type StateCallbackListener interface {
	// OnValidatorAdded is called when a new validator is added
	OnValidatorAdded(vID interfaces.ID, nodeID interfaces.NodeID, startTime uint64, isActive bool)
	// OnValidatorRemoved is called when a validator is removed
	OnValidatorRemoved(vID interfaces.ID, nodeID interfaces.NodeID)
	// OnValidatorStatusUpdated is called when a validator status is updated
	OnValidatorStatusUpdated(vID interfaces.ID, nodeID interfaces.NodeID, isActive bool)
}

type Validator struct {
	ValidationID   interfaces.ID     `json:"validationID"`
	NodeID         interfaces.NodeID `json:"nodeID"`
	Weight         uint64     `json:"weight"`
	StartTimestamp uint64     `json:"startTimestamp"`
	IsActive       bool       `json:"isActive"`
	IsL1Validator  bool       `json:"isL1Validator"`
}
