// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package resource

import (
	"errors"

	"github.com/luxfi/metric"
)

type metricsImpl struct {
	numCPUCycles       metric.GaugeVec
	numDiskReads       metric.GaugeVec
	numDiskReadBytes   metric.GaugeVec
	numDiskWrites      metric.GaugeVec
	numDiskWritesBytes metric.GaugeVec
}

func newMetrics(registerer metric.Registerer) (*metricsImpl, error) {
	m := &metricsImpl{
		numCPUCycles: metric.NewGaugeVec(
			metric.GaugeOpts{
				Name: "num_cpu_cycles",
				Help: "Total number of CPU cycles",
			},
			[]string{"processID"},
		),
		numDiskReads: metric.NewGaugeVec(
			metric.GaugeOpts{
				Name: "num_disk_reads",
				Help: "Total number of disk reads",
			},
			[]string{"processID"},
		),
		numDiskReadBytes: metric.NewGaugeVec(
			metric.GaugeOpts{
				Name: "num_disk_read_bytes",
				Help: "Total number of disk read bytes",
			},
			[]string{"processID"},
		),
		numDiskWrites: metric.NewGaugeVec(
			metric.GaugeOpts{
				Name: "num_disk_writes",
				Help: "Total number of disk writes",
			},
			[]string{"processID"},
		),
		numDiskWritesBytes: metric.NewGaugeVec(
			metric.GaugeOpts{
				Name: "num_disk_write_bytes",
				Help: "Total number of disk write bytes",
			},
			[]string{"processID"},
		),
	}
	err := errors.Join(
	// registerer.Register(m.numCPUCycles),
	// registerer.Register(m.numDiskReads),
	// registerer.Register(m.numDiskReadBytes),
	// registerer.Register(m.numDiskWrites),
	// registerer.Register(m.numDiskWritesBytes),
	)
	return m, err
}
