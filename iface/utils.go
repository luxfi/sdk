// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package iface

import "time"

// Cacher is a generic cache interface
type Cacher[K comparable, V any] interface {
	Put(key K, value V)
	Get(key K) (V, bool)
	Evict(key K)
	Flush()
	Len() int
}

// MockableTimer is an interface for a mockable clock
type MockableTimer interface {
	Time() time.Time
	Set(time time.Time)
	Advance(duration time.Duration)
}

// GenericSet is a generic set interface
type GenericSet[T comparable] interface {
	Add(items ...T)
	Remove(item T)
	Contains(item T) bool
	Clear()
	Len() int
	List() []T
}