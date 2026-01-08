// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package resource

import (
	"errors"

	"github.com/luxfi/metric"
)

type metricsImpl struct {
	cpuCyclesGauge       metric.Gauge
	diskReadsGauge       metric.Gauge
	diskReadBytesGauge   metric.Gauge
	diskWritesGauge      metric.Gauge
	diskWriteBytesGauge  metric.Gauge
}

func newMetrics(registerer metric.Registerer) (*metricsImpl, error) {
	m := &metricsImpl{
		cpuCyclesGauge: metric.NewGauge("num_cpu_cycles", "Total number of CPU cycles"),
		diskReadsGauge: metric.NewGauge("num_disk_reads", "Total number of disk reads"),
		diskReadBytesGauge: metric.NewGauge("num_disk_read_bytes", "Total number of disk read bytes"),
		diskWritesGauge: metric.NewGauge("num_disk_writes", "Total number of disk writes"),
		diskWriteBytesGauge: metric.NewGauge("num_disk_write_bytes", "Total number of disk write bytes"),
	}
	return m, nil
}
