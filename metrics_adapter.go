// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"github.com/prometheus/client_golang/prometheus"
	luxmetric "github.com/luxfi/metric"
)

// MetricsAdapter wraps a prometheus.Registry to implement luxmetric.Metrics
type MetricsAdapter struct {
	reg *prometheus.Registry
}

// NewMetricsAdapter creates a new adapter
func NewMetricsAdapter(reg *prometheus.Registry) luxmetric.Metrics {
	if reg == nil {
		return luxmetric.New("")
	}
	// Return a new metrics instance with the prometheus registry
	return luxmetric.NewWithRegistry("", reg)
}

// WrapMetricsRegistry wraps a prometheus.Registry as luxmetric.Metrics
func WrapMetricsRegistry(reg interface{}) luxmetric.Metrics {
	if reg == nil {
		return luxmetric.New("")
	}
	
	// If it's already a Metrics interface, return it
	if m, ok := reg.(luxmetric.Metrics); ok {
		return m
	}
	
	// If it's a prometheus.Registry, wrap it
	if promReg, ok := reg.(*prometheus.Registry); ok {
		return luxmetric.NewWithRegistry("", promReg)
	}
	
	// Default to a new metrics instance
	return luxmetric.New("")
}