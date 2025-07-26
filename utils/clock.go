// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"sync"
	"time"
)

// MockableClock implements interfaces.MockableTimer
type MockableClock struct {
	mu   sync.RWMutex
	time time.Time
}

// NewMockableClock creates a new mockable clock
func NewMockableClock() *MockableClock {
	return &MockableClock{
		time: time.Now(),
	}
}

// Time returns the current time
func (c *MockableClock) Time() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.time.IsZero() {
		return time.Now()
	}
	return c.time
}

// Set sets the current time
func (c *MockableClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.time = t
}

// Advance advances the time by duration
func (c *MockableClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.time.IsZero() {
		c.time = time.Now()
	}
	c.time = c.time.Add(d)
}