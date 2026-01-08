// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"errors"
	"fmt"
	"github.com/luxfi/metric"
	utilmetric "github.com/luxfi/sdk/utils/metric"

	"google.golang.org/protobuf/proto"
)

var (
	_ MultiGatherer = (*prefixGatherer)(nil)

	errOverlappingNamespaces = errors.New("prefix could create overlapping namespaces")
)

// NewPrefixGatherer returns a new MultiGatherer that merges metrics by adding a
// prefix to their names.
func NewPrefixGatherer() MultiGatherer {
	return &prefixGatherer{}
}

type prefixGatherer struct {
	multiGatherer
}

func (g *prefixGatherer) Register(prefix string, gatherer metric.Gatherer) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	for _, existingPrefix := range g.names {
		if eitherIsPrefix(prefix, existingPrefix) {
			return fmt.Errorf("%w: %q conflicts with %q",
				errOverlappingNamespaces,
				prefix,
				existingPrefix,
			)
		}
	}

	prefixPtr := new(string)
	*prefixPtr = prefix
	g.register(
		prefix,
		&prefixedGatherer{
			prefix:    prefix,
			prefixPtr: prefixPtr,
			gatherer:  gatherer,
		},
	)
	return nil
}

func (g *prefixGatherer) Deregister(prefix string) bool {
	g.lock.Lock()
	defer g.lock.Unlock()

	for i, existingPrefix := range g.names {
		if existingPrefix == prefix {
			// Remove the gatherer and prefix
			g.names = append(g.names[:i], g.names[i+1:]...)
			g.gatherers = append(g.gatherers[:i], g.gatherers[i+1:]...)
			return true
		}
	}
	return false
}

type prefixedGatherer struct {
	prefix      string
	prefixPtr   *string // Cached pointer for when suffix is empty
	gatherer    metric.Gatherer
}

func (g *prefixedGatherer) Gather() ([]*metric.MetricFamily, error) {
	// Gather returns partially filled metrics in the case of an error. So, it
	// is expected to still return the metrics in the case an error is returned.
	metricFamilies, err := g.gatherer.Gather()
	for _, metricFamily := range metricFamilies {
		originalName := metricFamily.GetName()
		if originalName == "" {
			// When the original name is empty, just use the prefix pointer
			metricFamily.Name = g.prefixPtr
		} else {
			metricFamily.Name = proto.String(utilmetric.AppendNamespace(
				g.prefix,
				originalName,
			))
		}
	}
	return metricFamilies, err
}

// eitherIsPrefix returns true if either [a] is a prefix of [b] or [b] is a
// prefix of [a].
//
// This function accounts for the usage of the namespace boundary, so "hello" is
// not considered a prefix of "helloworld". However, "hello" is considered a
// prefix of "hello_world".
func eitherIsPrefix(a, b string) bool {
	if len(a) > len(b) {
		a, b = b, a
	}
	return a == b[:len(a)] && // a is a prefix of b
		(len(a) == 0 || // a is empty
			len(a) == len(b) || // a is equal to b
			b[len(a)] == utilmetric.NamespaceSeparatorByte) // a ends at a namespace boundary of b
}
