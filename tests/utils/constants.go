// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import "time"

const (
	// Timeout to boot the Luxd node
	BootLuxNodeTimeout = 5 * time.Minute

	// Timeout for the health API to check the Luxd is ready
	HealthCheckTimeout = 5 * time.Second

	DefaultLocalNodeURI = "http://127.0.0.1:9630"
)
