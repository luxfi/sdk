// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package interfaces

import (
	"time"

	"github.com/luxfi/node/utils/constants"
)

// Config represents network upgrade configuration
type Config struct {
	DurangoTime time.Time
	EtnaTime    time.Time
}

// GetConfig returns the network upgrade configuration for the given network ID
func GetConfig(networkID uint32) Config {
	// TODO: This should be properly synced with node upgrade times
	switch networkID {
	case constants.MainnetID:
		return Config{
			DurangoTime: time.Date(2024, time.March, 6, 16, 0, 0, 0, time.UTC),
			EtnaTime:    time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC), // Placeholder
		}
	case constants.TestnetID:
		return Config{
			DurangoTime: time.Date(2024, time.February, 13, 16, 0, 0, 0, time.UTC),
			EtnaTime:    time.Date(2024, time.December, 1, 0, 0, 0, 0, time.UTC), // Placeholder
		}
	default:
		// For test networks, use current time to enable all upgrades
		now := time.Now()
		return Config{
			DurangoTime: now,
			EtnaTime:    now,
		}
	}
}

// UnscheduledActivationTime is a placeholder for unscheduled network upgrades
var UnscheduledActivationTime = time.Unix(0, 0)

// InitiallyActiveTime represents features that are active from genesis
var InitiallyActiveTime = time.Unix(0, 0)
