// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package customrawdb

import (
	"fmt"
	"math/big"

	"github.com/luxfi/evm/plugin/evm/customtypes"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/rlp"
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

// WriteBlockGasCost writes the BlockGasCost for a header to a separate key.
func WriteBlockGasCost(db ethdb.KeyValueWriter, hash common.Hash, number uint64, cost *big.Int) {
	if cost == nil {
		return
	}
	key := blockGasCostKey(number, hash)
	data, err := rlp.EncodeToBytes(cost)
	if err != nil {
		return
	}
	_ = db.Put(key, data)
}

// ReadBlockGasCost reads the BlockGasCost for a header from its separate key.
func ReadBlockGasCost(db ethdb.Reader, hash common.Hash, number uint64) *big.Int {
	key := blockGasCostKey(number, hash)
	data, err := db.Get(key)
	if err != nil || len(data) == 0 {
		return nil
	}
	var cost big.Int
	if err := rlp.DecodeBytes(data, &cost); err != nil {
		return nil
	}
	return &cost
}

// WriteHeader writes a header with its HeaderExtra (BlockGasCost) to the database.
// This uses rawdb.WriteHeader for the header itself and stores BlockGasCost separately.
func WriteHeader(db ethdb.KeyValueWriter, header *types.Header) {
	// Write the header using geth's standard encoding
	rawdb.WriteHeader(db, header)

	// Get the extra data (BlockGasCost) if available and write it separately
	if extra := customtypes.GetHeaderExtra(header); extra != nil && extra.BlockGasCost != nil {
		WriteBlockGasCost(db, header.Hash(), header.Number.Uint64(), extra.BlockGasCost)
	}
}

// WriteBlock writes a block with its HeaderExtra (BlockGasCost) to the database.
// This is a replacement for rawdb.WriteBlock that preserves custom header fields.
func WriteBlock(db ethdb.KeyValueWriter, block *types.Block) {
	WriteHeader(db, block.Header())
	rawdb.WriteBody(db, block.Hash(), block.NumberU64(), block.Body())
}

// ReadHeader reads a header from the database and restores its HeaderExtra.
func ReadHeader(db ethdb.Reader, hash common.Hash, number uint64) *types.Header {
	header := rawdb.ReadHeader(db, hash, number)
	if header == nil {
		return nil
	}

	// Restore HeaderExtra from the separate BlockGasCost key
	RestoreHeaderExtra(db, header, hash, number)
	return header
}

// RestoreHeaderExtra reads the BlockGasCost from its separate key and
// stores it in the header's in-memory HeaderExtra.
func RestoreHeaderExtra(db ethdb.Reader, header *types.Header, hash common.Hash, number uint64) {
	// Check if we already have the extra in memory
	if extra := customtypes.GetHeaderExtra(header); extra != nil && extra.BlockGasCost != nil {
		return
	}

	// Read BlockGasCost from its separate key
	cost := ReadBlockGasCost(db, hash, number)
	if cost != nil {
		customtypes.SetHeaderExtra(header, &customtypes.HeaderExtra{BlockGasCost: cost})
	}
}
