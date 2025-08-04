// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package customrawdb

import (
	"fmt"

	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/ethdb"
)

// InspectDatabase traverses the entire database and checks the size
// of all different categories of data.
func InspectDatabase(db ethdb.Database, keyPrefix, keyStart []byte) error {
	// For now, just delegate to the standard rawdb.InspectDatabase
	// TODO: Add custom statistics tracking for evm-specific data
	return rawdb.InspectDatabase(db, keyPrefix, keyStart)
}

// ParseStateSchemeExt parses the state scheme from the provided string.
func ParseStateSchemeExt(provided string, disk ethdb.Database) (string, error) {
	// Check for custom scheme
	if provided == FirewoodScheme {
		if diskScheme := rawdb.ReadStateScheme(disk); diskScheme != "" {
			// Valid scheme on disk mismatched
			return "", fmt.Errorf("State scheme %s already set on disk, can't use Firewood", diskScheme)
		}
		// If no conflicting scheme is found, is valid.
		return FirewoodScheme, nil
	}

	// Check for valid eth scheme
	return rawdb.ParseStateScheme(provided, disk)
}
