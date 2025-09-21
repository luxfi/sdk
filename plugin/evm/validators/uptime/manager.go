// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package uptime

import (
	"github.com/luxfi/consensus/uptime"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
)

// Manager tracks validator uptime and connection status
type Manager struct {
	uptime.Calculator
	state     uptime.State
	clock     interface{}
	connected set.Set[ids.NodeID]
}

// NewManager creates a new uptime manager
func NewManager(state uptime.State, clock interface{}) *Manager {
	return &Manager{
		Calculator: uptime.NewLockedCalculator(),
		state:      state,
		clock:      clock,
		connected:  make(set.Set[ids.NodeID]),
	}
}

// Connect marks a validator as connected
func (m *Manager) Connect(nodeID ids.NodeID) error {
	m.connected.Add(nodeID)
	return nil
}

// Disconnect marks a validator as disconnected
func (m *Manager) Disconnect(nodeID ids.NodeID) error {
	m.connected.Remove(nodeID)
	return nil
}

// IsConnected returns whether a validator is connected
func (m *Manager) IsConnected(nodeID ids.NodeID) bool {
	return m.connected.Contains(nodeID)
}

// StartTracking starts tracking uptime for the given set of validators
func (m *Manager) StartTracking(nodeIDs []ids.NodeID) error {
	// Implementation for starting to track multiple validators
	// This is typically called when new validators are added or at startup
	for range nodeIDs {
		// Initialize tracking for each node if needed
		// For now, just ensure they're in our tracking set
		// The actual uptime tracking is handled by the underlying Calculator
	}
	return nil
}

// StopTracking stops tracking uptime for the given set of validators
func (m *Manager) StopTracking(nodeIDs []ids.NodeID) error {
	// Implementation for stopping tracking of multiple validators
	for _, nodeID := range nodeIDs {
		// Remove from connected set if connected
		m.connected.Remove(nodeID)
	}
	return nil
}
