// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mockable

import (
	"sync"
	"time"
)

// MaxTime was taken from https://stackoverflow.com/questions/25065055/what-is-the-maximum-time-time-in-go/32620397#32620397
var MaxTime = time.Unix(1<<63-62135596801, 0) // 0 is used because we drop the nano-seconds

// Clock acts as a thin wrapper around global time that allows for easy testing.
// It is safe for concurrent use.
type Clock struct {
	mu    sync.RWMutex
	faked bool
	time  time.Time
}

// Set the time on the clock
func (c *Clock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.faked = true
	c.time = t
}

// Sync this clock with global time
func (c *Clock) Sync() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.faked = false
}

// Time returns the time on this clock
func (c *Clock) Time() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.faked {
		return c.time
	}
	return time.Now()
}

// Time returns the unix time on this clock
func (c *Clock) UnixTime() time.Time {
	resTime := c.Time()
	return resTime.Truncate(time.Second)
}

// Unix returns the unix timestamp on this clock.
func (c *Clock) Unix() uint64 {
	unix := max(c.Time().Unix(), 0)
	return uint64(unix)
}
