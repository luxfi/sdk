// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package health

import "github.com/luxfi/metric"

type healthMetrics struct {
	// failingChecks keeps track of the number of check failing
	failingChecks metric.GaugeVec
}

func newMetrics(namespace string, registry metric.Registry) (*healthMetrics, error) {
	metricsInstance := metric.NewWithRegistry(namespace, registry)

	m := &healthMetrics{
		failingChecks: metricsInstance.NewGaugeVec(
			"checks_failing",
			"number of currently failing health checks",
			[]string{"tag"},
		),
	}
	m.failingChecks.WithLabelValues(AllTag).Set(0)
	m.failingChecks.WithLabelValues(ApplicationTag).Set(0)
	return m, nil
}
