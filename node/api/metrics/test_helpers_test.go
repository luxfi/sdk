// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package metrics

import (
	"context"

	dto "github.com/prometheus/client_model/go"
)

type testGathererWithContext struct {
	mfs []*dto.MetricFamily
}

func (g *testGathererWithContext) Gather(context.Context) ([]*dto.MetricFamily, error) {
	return g.mfs, nil
}
