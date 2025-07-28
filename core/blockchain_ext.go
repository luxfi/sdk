// (c) 2024 Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package core

import "github.com/luxfi/geth/metrics"

// getOrOverrideAsRegisteredCounter searches for a metric already registered
// with `name`. If a metric is found and it is a [metrics.Counter], it is returned. If a
// metric is found and it is not a [metrics.Counter], it is unregistered and replaced with
// a new registered [metrics.Counter]. If no metric is found, a new [metrics.Counter] is constructed
// and registered.
//
// This is necessary for a metric defined in libevm with the same name but a
// different type to what we expect.
func getOrOverrideAsRegisteredCounter(name string, r metrics.Registry) *metrics.Counter {
	if r == nil {
		r = metrics.DefaultRegistry
	}

	// Try to get existing metric
	if existing := r.Get(name); existing != nil {
		// If it's already a counter, return it
		if c, ok := existing.(*metrics.Counter); ok {
			return c
		}
		// Otherwise unregister it
		r.Unregister(name)
	}
	
	// Register new counter
	return metrics.NewRegisteredCounter(name, r)
}
