// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package bloom

import "github.com/luxfi/metric"

// Metrics is a collection of commonly useful metrics when using a long-lived
// bloom filter.
type Metrics struct {
	Count      metric.Gauge
	NumHashes  metric.Gauge
	NumEntries metric.Gauge
	MaxCount   metric.Gauge
	ResetCount metric.Counter
}

func NewMetrics(
	namespace string,
	registry metric.Registry,
) (*Metrics, error) {
	metricsInstance := metric.NewWithRegistry(namespace, registry)

	m := &Metrics{
		Count: metricsInstance.NewGauge(
			"count",
			"Number of additions that have been performed to the bloom",
		),
		NumHashes: metricsInstance.NewGauge(
			"hashes",
			"Number of hashes in the bloom",
		),
		NumEntries: metricsInstance.NewGauge(
			"entries",
			"Number of bytes allocated to slots in the bloom",
		),
		MaxCount: metricsInstance.NewGauge(
			"max_count",
			"Maximum number of additions that should be performed to the bloom before resetting",
		),
		ResetCount: metricsInstance.NewCounter(
			"reset_count",
			"Number times the bloom has been reset",
		),
	}
	return m, nil
}

// Reset the metrics to align with the provided bloom filter and max count.
func (m *Metrics) Reset(newFilter *Filter, maxCount int) {
	m.Count.Set(float64(newFilter.Count()))
	m.NumHashes.Set(float64(len(newFilter.hashSeeds)))
	m.NumEntries.Set(float64(len(newFilter.entries)))
	m.MaxCount.Set(float64(maxCount))
	m.ResetCount.Inc()
}
