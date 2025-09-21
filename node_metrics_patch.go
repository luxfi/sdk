// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build !no_patch
// +build !no_patch

package evm

import (
	"github.com/luxfi/database/meterdb"
	"github.com/luxfi/database"
	luxmetric "github.com/luxfi/metric"
	"github.com/prometheus/client_golang/prometheus"
)

// PatchedMeterDBNew creates a new meterdb with metrics compatibility fix
func PatchedMeterDBNew(reg interface{}, db database.Database) (*meterdb.Database, error) {
	var metrics luxmetric.Metrics
	
	switch v := reg.(type) {
	case luxmetric.Metrics:
		metrics = v
	case *prometheus.Registry:
		// Convert prometheus.Registry to luxmetric.Metrics
		metrics = luxmetric.NewWithRegistry("", v)
	default:
		// Fallback to a new metrics instance
		metrics = luxmetric.New("")
	}
	
	return meterdb.New(metrics, db)
}

func init() {
	// This init function doesn't do anything but ensures this file is included
	// The actual patching would need to happen at build time or through replace directives
}