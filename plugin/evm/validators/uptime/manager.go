// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package uptime

import (
	"sync"
	"time"

	"github.com/luxfi/consensus/validator/uptime"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/timer/mockable"
)

// nodeUptime tracks uptime information for a single node
type nodeUptime struct {
	// accumulatedUptime is the total uptime accumulated during tracking periods
	accumulatedUptime time.Duration
	// lastConnectedTime is when the node was last connected (zero if disconnected)
	lastConnectedTime time.Time
	// isTracking indicates if we're actively tracking this node
	isTracking bool
	// trackingStartTime is when we started tracking this node
	trackingStartTime time.Time
	// preTrackingUptime is the "optimistic" uptime credited before tracking began
	// This is the time from creation until StartTracking was called
	preTrackingUptime time.Duration
	// creationTime is when this node's uptime tracking was first initialized
	creationTime time.Time
	// loadedFromState indicates if we've loaded previous uptime from state
	loadedFromState bool
}

// Manager tracks validator uptime and connection status
type Manager struct {
	lock      sync.RWMutex
	state     uptime.State
	clock     *mockable.Clock
	connected set.Set[ids.NodeID]
	// uptimes tracks uptime information per node
	uptimes map[ids.NodeID]*nodeUptime
	// defaultNetID is used for state operations
	defaultNetID ids.ID
}

// NewManager creates a new uptime manager
func NewManager(state uptime.State, clock interface{}) *Manager {
	clk, ok := clock.(*mockable.Clock)
	if !ok {
		// Default clock if not provided or wrong type
		clk = &mockable.Clock{}
		clk.Set(time.Now())
	}
	return &Manager{
		state:        state,
		clock:        clk,
		connected:    make(set.Set[ids.NodeID]),
		uptimes:      make(map[ids.NodeID]*nodeUptime),
		defaultNetID: ids.Empty,
	}
}

// getOrCreateUptime returns the uptime entry for a node, creating if needed
// It also loads any previously persisted uptime from state
func (m *Manager) getOrCreateUptime(nodeID ids.NodeID) *nodeUptime {
	nu, ok := m.uptimes[nodeID]
	if !ok {
		nu = &nodeUptime{
			creationTime: m.clock.Time(),
		}
		m.uptimes[nodeID] = nu
	}

	// Load previous uptime from state if not already loaded
	if !nu.loadedFromState && m.state != nil {
		previousUptime, _, err := m.state.GetUptime(nodeID, m.defaultNetID)
		if err == nil && previousUptime > 0 {
			// Add previously persisted uptime to accumulated
			nu.accumulatedUptime = previousUptime
		}
		nu.loadedFromState = true
	}

	return nu
}

// Connect marks a validator as connected and begins counting uptime if tracking
func (m *Manager) Connect(nodeID ids.NodeID) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	// Track this node was connected (for pausable manager to know)
	m.connected.Add(nodeID)
	nu := m.getOrCreateUptime(nodeID)

	// Only start accumulating uptime if we're tracking and not already connected
	if nu.isTracking && nu.lastConnectedTime.IsZero() {
		nu.lastConnectedTime = m.clock.Time()
	}

	return nil
}

// Disconnect marks a validator as disconnected and finalizes uptime if tracking
func (m *Manager) Disconnect(nodeID ids.NodeID) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.connected.Remove(nodeID)
	nu := m.getOrCreateUptime(nodeID)

	// Accumulate uptime if we were tracking and had a valid connection time
	if nu.isTracking && !nu.lastConnectedTime.IsZero() {
		nu.accumulatedUptime += m.clock.Time().Sub(nu.lastConnectedTime)
	}
	nu.lastConnectedTime = time.Time{}

	return nil
}

// IsConnected returns whether a validator is connected
func (m *Manager) IsConnected(nodeID ids.NodeID) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.connected.Contains(nodeID)
}

// StartTracking starts tracking uptime for the given set of validators
// Any time before StartTracking is credited as "optimistic" uptime
func (m *Manager) StartTracking(nodeIDs []ids.NodeID) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	now := m.clock.Time()
	for _, nodeID := range nodeIDs {
		nu := m.getOrCreateUptime(nodeID)
		if nu.isTracking {
			continue
		}

		// Credit time since creation as "pre-tracking uptime"
		// This implements the optimistic assumption that node was up before tracking
		nu.preTrackingUptime += now.Sub(nu.creationTime)
		// Reset creation time for next stop/start cycle
		nu.creationTime = now

		nu.isTracking = true
		nu.trackingStartTime = now

		// If already connected, start counting uptime from now
		if m.connected.Contains(nodeID) {
			nu.lastConnectedTime = now
		}
	}
	return nil
}

// StopTracking stops tracking uptime for the given set of validators
// It persists the accumulated uptime to state
func (m *Manager) StopTracking(nodeIDs []ids.NodeID) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	now := m.clock.Time()
	for _, nodeID := range nodeIDs {
		nu := m.getOrCreateUptime(nodeID)
		if !nu.isTracking {
			continue
		}

		// Finalize uptime if connected
		if !nu.lastConnectedTime.IsZero() {
			nu.accumulatedUptime += now.Sub(nu.lastConnectedTime)
		}
		nu.lastConnectedTime = time.Time{}

		nu.isTracking = false
		// Reset creation time so next StartTracking credits time since now
		nu.creationTime = now

		// Persist uptime to state
		if m.state != nil {
			totalUptime := nu.preTrackingUptime + nu.accumulatedUptime
			_ = m.state.SetUptime(nodeID, m.defaultNetID, totalUptime, now)
		}
	}
	return nil
}

// CalculateUptime returns the uptime duration and total tracking duration for a node
func (m *Manager) CalculateUptime(nodeID ids.NodeID, netID ids.ID) (time.Duration, time.Duration, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	nu, ok := m.uptimes[nodeID]
	if !ok {
		// Try to load from state
		if m.state != nil {
			previousUptime, _, err := m.state.GetUptime(nodeID, netID)
			if err == nil && previousUptime > 0 {
				return previousUptime, previousUptime, nil
			}
		}
		return 0, 0, nil
	}

	now := m.clock.Time()

	// Start with pre-tracking (optimistic) uptime
	uptimeDuration := nu.preTrackingUptime + nu.accumulatedUptime

	// Add current connection time if connected and tracking
	if nu.isTracking && !nu.lastConnectedTime.IsZero() {
		uptimeDuration += now.Sub(nu.lastConnectedTime)
	}

	// If not tracking, add time since stop tracking (optimistic for non-tracked period)
	if !nu.isTracking {
		uptimeDuration += now.Sub(nu.creationTime)
	}

	// Calculate total duration (should equal uptime for optimistic periods + tracking period)
	totalDuration := uptimeDuration

	return uptimeDuration, totalDuration, nil
}

// CalculateUptimePercent returns the uptime percentage for a node
func (m *Manager) CalculateUptimePercent(nodeID ids.NodeID, subnetID ids.ID) (float64, error) {
	uptime, total, err := m.CalculateUptime(nodeID, subnetID)
	if err != nil {
		return 0, err
	}
	if total == 0 {
		return 1.0, nil // No tracking duration, assume 100%
	}
	return float64(uptime) / float64(total), nil
}

// CalculateUptimePercentFrom returns the uptime percentage since a given time
func (m *Manager) CalculateUptimePercentFrom(nodeID ids.NodeID, subnetID ids.ID, _ time.Time) (float64, error) {
	// For now, just use the same calculation
	return m.CalculateUptimePercent(nodeID, subnetID)
}

// SetCalculator is a no-op for compatibility with uptime.Calculator interface
func (m *Manager) SetCalculator(_ ids.ID, _ uptime.Calculator) error {
	return nil
}

// EnsureExists ensures a nodeUptime entry exists for the given node
// This is used to track nodes even before they connect
func (m *Manager) EnsureExists(nodeID ids.NodeID) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.getOrCreateUptime(nodeID)
}
