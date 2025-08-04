// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package customrawdb

import (
	"encoding/binary"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/ethdb"
)

// ReadBloomBits retrieves the bloom bits for a specific section and bit index.
func ReadBloomBits(db ethdb.KeyValueReader, bit uint, section uint64, head common.Hash) ([]byte, error) {
	// Use the same key scheme as go-ethereum
	key := make([]byte, 10+1+8+32)
	copy(key, []byte("blt"))                   // Bloom trie key prefix
	key[3] = byte(bit)                         // Bloom bit index
	binary.BigEndian.PutUint64(key[4:12], section) // Section index
	copy(key[12:], head[:])                    // Block hash

	return db.Get(key)
}

// WriteBloomBits stores the bloom bits for a specific section and bit index.
func WriteBloomBits(db ethdb.KeyValueWriter, bit uint, section uint64, head common.Hash, bits []byte) error {
	key := make([]byte, 10+1+8+32)
	copy(key, []byte("blt"))
	key[3] = byte(bit)
	binary.BigEndian.PutUint64(key[4:12], section)
	copy(key[12:], head[:])

	return db.Put(key, bits)
}

// DeleteBloombits deletes all bloom bits for a given section range.
func DeleteBloombits(db ethdb.Database, start, end uint64) {
	// Iterate through all possible bit indices (0-2047)
	for bit := uint(0); bit < 2048; bit++ {
		for section := start; section <= end; section++ {
			// Read canonical hash for the section
			head := rawdb.ReadCanonicalHash(db, (section+1)*4096-1)
			if head == (common.Hash{}) {
				continue
			}
			
			key := make([]byte, 10+1+8+32)
			copy(key, []byte("blt"))
			key[3] = byte(bit)
			binary.BigEndian.PutUint64(key[4:12], section)
			copy(key[12:], head[:])
			
			db.Delete(key)
		}
	}
}