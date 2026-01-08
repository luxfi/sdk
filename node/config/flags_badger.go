// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"flag"
	"path/filepath"
)

// BadgerDB flags for C-Chain
type BadgerDBConfig struct {
	Enable          bool
	DataDir         string
	EnableAncient   bool
	AncientDir      string
	ReadOnly        bool
	SharedAncient   bool
	FreezeThreshold uint64
}

// AddBadgerDBFlags adds BadgerDB-related flags to the flag set
func AddBadgerDBFlags(fs *flag.FlagSet) *BadgerDBConfig {
	config := &BadgerDBConfig{}

	fs.BoolVar(&config.Enable, "cchain-badger", false,
		"Enable BadgerDB for C-Chain instead of default database")

	fs.StringVar(&config.DataDir, "cchain-badger-dir", "",
		"BadgerDB data directory (default: <datadir>/cchain-badger)")

	fs.BoolVar(&config.EnableAncient, "cchain-ancient", false,
		"Enable ancient store for historical blockchain data")

	fs.StringVar(&config.AncientDir, "cchain-ancient-dir", "",
		"Ancient store directory (default: <badger-dir>/ancient)")

	fs.BoolVar(&config.ReadOnly, "cchain-ancient-readonly", false,
		"Open ancient store in read-only mode (allows sharing)")

	fs.BoolVar(&config.SharedAncient, "cchain-ancient-shared", false,
		"Enable shared access to ancient store (requires readonly)")

	fs.Uint64Var(&config.FreezeThreshold, "cchain-freeze-threshold", 90000,
		"Number of recent blocks to keep in main DB before freezing to ancient")

	return config
}

// Validate validates the BadgerDB configuration
func (c *BadgerDBConfig) Validate(dataDir string) error {
	// Set defaults
	if c.Enable && c.DataDir == "" {
		c.DataDir = filepath.Join(dataDir, "cchain-badger")
	}

	if c.EnableAncient && c.AncientDir == "" {
		c.AncientDir = filepath.Join(c.DataDir, "ancient")
	}

	// Shared ancient requires read-only
	if c.SharedAncient && !c.ReadOnly {
		c.ReadOnly = true
	}

	return nil
}
