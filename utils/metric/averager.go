// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utilmetric

import (
	"errors"

	metric "github.com/luxfi/metric"

	"github.com/luxfi/vm/utils/wrappers"
)

var ErrFailedRegistering = errors.New("failed registering metric")

type Averager interface {
	Observe(float64)
}

type averager struct {
	count metric.Counter
	sum   metric.Gauge
}

func NewAverager(name, desc string, registry metric.Registry) (Averager, error) {
	errs := wrappers.Errs{}
	a := NewAveragerWithErrs(name, desc, registry, &errs)
	return a, errs.Err
}

func NewAveragerWithErrs(name, desc string, registry metric.Registry, errs *wrappers.Errs) Averager {
	metricsInstance := metric.NewWithRegistry("", registry)

	a := averager{
		count: metricsInstance.NewCounter(
			AppendNamespace(name, "count"),
			"Total # of observations of "+desc,
		),
		sum: metricsInstance.NewGauge(
			AppendNamespace(name, "sum"),
			"Sum of "+desc,
		),
	}

	return &a
}

func (a *averager) Observe(v float64) {
	a.count.Inc()
	a.sum.Add(v)
}
