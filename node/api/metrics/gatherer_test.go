// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"github.com/luxfi/metric"

	dto "github.com/prometheus/client_model/go"
)

var counterOpts = metric.CounterOpts{
	Name: "counter",
	Help: "help",
}

type testGatherer struct {
	mfs []*dto.MetricFamily
	err error
}

func (g *testGatherer) Gather() ([]*dto.MetricFamily, error) {
	return g.mfs, g.err
}
