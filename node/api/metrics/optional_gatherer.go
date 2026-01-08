// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package metrics

import (
	"errors"
	"fmt"
	"github.com/luxfi/metric"
	"sync"

	dto "github.com/prometheus/client_model/go"
)

var errReregisterGatherer = errors.New("attempted to register a gatherer when one is already registered")

var _ OptionalGatherer = (*optionalGatherer)(nil)

// OptionalGatherer extends the Gatherer interface by allowing the optional
// registration of a single gatherer. If no gatherer is registered, Gather will
// return no metrics and no error. If a gatherer is registered, Gather will
// return the results of calling Gather on the provided gatherer.
type OptionalGatherer interface {
	metric.Gatherer

	// Register the provided gatherer. If a gatherer was previously registered,
	// an error will be returned.
	Register(gatherer metric.Gatherer) error
}

type optionalGatherer struct {
	lock     sync.RWMutex
	gatherer metric.Gatherer
}

func NewOptionalGatherer() OptionalGatherer {
	return &optionalGatherer{}
}

func (g *optionalGatherer) Gather() ([]*dto.MetricFamily, error) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	if g.gatherer == nil {
		return nil, nil
	}
	return g.gatherer.Gather()
}

func (g *optionalGatherer) Register(gatherer metric.Gatherer) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	if g.gatherer != nil {
		return fmt.Errorf("%w; existing: %#v; new: %#v",
			errReregisterGatherer,
			g.gatherer,
			gatherer,
		)
	}
	g.gatherer = gatherer
	return nil
}
