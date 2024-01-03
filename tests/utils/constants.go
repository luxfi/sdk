// Copyright (C) 2021-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import "time"

const (
<<<<<<< HEAD
	// Timeout to boot the Lux Node node
	BootLuxNodeTimeout = 5 * time.Minute

	// Timeout for the health API to check the Lux Node is ready
=======
	// Timeout to boot the Luxd node
	BootLuxNodeTimeout = 5 * time.Minute

	// Timeout for the health API to check the Luxd is ready
>>>>>>> b36c20f (Update executable to luxd)
	HealthCheckTimeout = 5 * time.Second

	DefaultLocalNodeURI = "http://127.0.0.1:9650"
)

var (
	NodeURIs = []string{DefaultLocalNodeURI, "http://127.0.0.1:9652", "http://127.0.0.1:9654", "http://127.0.0.1:9656", "http://127.0.0.1:9658"}
)
