// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"sort"
	"github.com/luxfi/metric"

	"fmt"
	"slices"
	"sync"


	"github.com/luxfi/sdk/utils"

	dto "github.com/prometheus/client_model/go"
)

// MultiGatherer extends the Gatherer interface by allowing additional gatherers
// to be registered.
type MultiGatherer interface {
	metric.Gatherer

	// Register adds the outputs of [gatherer] to the results of future calls to
	// Gather with the provided [name] added to the metrics.
	Register(name string, gatherer metric.Gatherer) error

	// Deregister removes the outputs of a gatherer with [name] from the results
	// of future calls to Gather. Returns true if a gatherer with [name] was
	// found.
	Deregister(name string) bool
}

type multiGatherer struct {
	lock      sync.RWMutex
	names     []string
	gatherers []metric.Gatherer
}

// NewMultiGatherer creates and returns a new MultiGatherer that applies
// prefixes to metric names. For a MultiGatherer without prefix support, use
// the multiGatherer struct directly.
func NewMultiGatherer() MultiGatherer {
	return NewPrefixGatherer()
}

func (g *multiGatherer) Gather() ([]*dto.MetricFamily, error) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	var allFamilies []*dto.MetricFamily
	for _, gatherer := range g.gatherers {
		families, err := gatherer.Gather()
		if err != nil {
			return allFamilies, err
		}
		allFamilies = append(allFamilies, families...)
	}

	// Sort metrics by name for consistent ordering
	sort.Slice(allFamilies, func(i, j int) bool {
		return *allFamilies[i].Name < *allFamilies[j].Name
	})

	return allFamilies, nil
}

// Register adds the outputs of gatherer to the results of future calls to
// Gather with the provided name added to the metrics.
func (g *multiGatherer) Register(name string, gatherer metric.Gatherer) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	if slices.Contains(g.names, name) {
		return fmt.Errorf("gatherer with name %q already registered", name)
	}

	g.register(name, gatherer)
	return nil
}

func (g *multiGatherer) register(name string, gatherer metric.Gatherer) {
	g.names = append(g.names, name)
	g.gatherers = append(g.gatherers, gatherer)
}

func (g *multiGatherer) Deregister(name string) bool {
	g.lock.Lock()
	defer g.lock.Unlock()

	index := slices.Index(g.names, name)
	if index == -1 {
		return false
	}

	g.names = utils.DeleteIndex(g.names, index)
	g.gatherers = utils.DeleteIndex(g.gatherers, index)
	return true
}

func MakeAndRegister(gatherer MultiGatherer, name string) (metric.Registry, error) {
	reg := metric.NewRegistry()
	if err := gatherer.Register(name, reg); err != nil {
		return nil, fmt.Errorf("couldn't register %q metrics: %w", name, err)
	}
	return reg, nil
}
