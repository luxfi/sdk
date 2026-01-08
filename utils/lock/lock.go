// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package lock

import "sync"

// State represents a lockable state
type State struct {
	lock sync.RWMutex
}

// Acquire acquires the write lock
func (s *State) Acquire() {
	s.lock.Lock()
}

// Release releases the write lock
func (s *State) Release() {
	s.lock.Unlock()
}

// RAcquire acquires the read lock
func (s *State) RAcquire() {
	s.lock.RLock()
}

// RRelease releases the read lock
func (s *State) RRelease() {
	s.lock.RUnlock()
}

// Lock returns the underlying lock
func (s *State) Lock() sync.Locker {
	return &s.lock
}

// RLock returns the underlying read lock
func (s *State) RLock() sync.Locker {
	return s.lock.RLocker()
}
