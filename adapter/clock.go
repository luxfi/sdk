// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package adapter

import (
	"time"

	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
)

// ClockAdapter adapts node's interfaces.MockableTimer to interfaces.MockableTimer
type ClockAdapter struct {
	clock *interfaces.MockableTimer
}

// NewClockAdapter creates a new clock adapter
func NewClockAdapter(clock *interfaces.MockableTimer) interfaces.MockableTimer {
	return &ClockAdapter{clock: clock}
}

// NewClock creates a new mockable clock
func NewClock() interfaces.MockableTimer {
	return &ClockAdapter{clock: &interfaces.MockableTimer{}}
}

// Time returns the current time
func (c *ClockAdapter) Time() time.Time {
	return c.clock.Time()
}

// Set sets the current time
func (c *ClockAdapter) Set(t time.Time) {
	c.clock.Set(t)
}

// Advance advances the time by duration
func (c *ClockAdapter) Advance(d time.Duration) {
	c.clock.Advance(d)
}