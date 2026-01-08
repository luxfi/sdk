// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"fmt"
	"strings"
)

// ChainDatabaseConfig holds per-chain database configuration
type ChainDatabaseConfig struct {
	// Default database type for all chains
	DefaultType string

	// Per-chain overrides
	PChainDBType string
	XChainDBType string
	CChainDBType string

	// Additional net configurations can be added here
	// NetDBTypes map[ids.ID]string
}

// GetDatabaseType returns the database type for a specific chain
func (c *ChainDatabaseConfig) GetDatabaseType(chainAlias string) string {
	switch strings.ToUpper(chainAlias) {
	case "P":
		if c.PChainDBType != "" {
			return c.PChainDBType
		}
	case "X":
		if c.XChainDBType != "" {
			return c.XChainDBType
		}
	case "C":
		if c.CChainDBType != "" {
			return c.CChainDBType
		}
	}
	// Return default if no specific override
	return c.DefaultType
}

// Validate ensures all database types are valid
func (c *ChainDatabaseConfig) Validate() error {
	validTypes := map[string]bool{
		"pebbledb": true,
		"badgerdb": true,
		"memdb":    true,
	}

	// Check default type
	if !validTypes[c.DefaultType] {
		return fmt.Errorf("invalid default database type: %s", c.DefaultType)
	}

	// Check chain-specific types
	for chain, dbType := range map[string]string{
		"P-Chain": c.PChainDBType,
		"X-Chain": c.XChainDBType,
		"C-Chain": c.CChainDBType,
	} {
		if dbType != "" && !validTypes[dbType] {
			return fmt.Errorf("invalid database type for %s: %s", chain, dbType)
		}
	}

	return nil
}
