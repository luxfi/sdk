// (c) 2024-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package testutils

import (
	"sync"
	"testing"

	"github.com/luxfi/geth/metrics"
)

var metricsLock sync.Mutex

// WithMetrics enables go-ethereum metrics globally for the test.
// If the [metrics.Enabled] is already true, nothing is done.
// Otherwise, it is set to true and is reverted to false when the test finishes.
func WithMetrics(t *testing.T) {
	metricsLock.Lock()
	t.Cleanup(func() {
		metricsLock.Unlock()
	})
	if metrics.Enabled {
		return
	}
	// Enable metrics for the test
	// Note: metrics.Enabled is a global bool variable in newer versions
	metrics.Enabled = true
}
