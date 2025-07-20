// (c) 2025 Hanzo Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package prometheus

import "github.com/luxfi/geth/metrics"

var _ Registry = (*metrics.StandardRegistry)(nil)

type Registry interface {
	// Call the given function for each registered metric.
	Each(func(string, any))
	// Get the metric by the given name or nil if none is registered.
	Get(string) any
}
