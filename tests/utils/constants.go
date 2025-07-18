// Copyright (C) 2019-2022, Hanzo Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"fmt"
	"time"
)

const (
	// Timeout to boot the Lux node
	BootLuxNodeTimeout = 5 * time.Minute

	// Timeout for the health API to check the Lux is ready
	HealthCheckTimeout = 5 * time.Second

	DefaultLocalNodeURI = "http://127.0.0.1:9630"
)

// GetDefaultChainURI returns the default chain URI for the given blockchain ID
func GetDefaultChainURI(blockchainID string) string {
	return fmt.Sprintf("%s/ext/bc/%s/rpc", DefaultLocalNodeURI, blockchainID)
}
