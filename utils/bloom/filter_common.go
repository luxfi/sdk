// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package bloom

import (
	"encoding/binary"

	"github.com/spaolacci/murmur3"
)

// marshalCommon serializes [hashSeeds] and [entries] into a byte slice.
func marshalCommon(hashSeeds []uint64, entries []byte) []byte {
	bytes := make([]byte, 1+len(hashSeeds)*bytesPerUint64+len(entries))
	bytes[0] = byte(len(hashSeeds))
	offset := 1
	for _, seed := range hashSeeds {
		binary.BigEndian.PutUint64(bytes[offset:], seed)
		offset += bytesPerUint64
	}
	copy(bytes[offset:], entries)
	return bytes
}

// containsCommon returns true if [hash] is in the filter defined by [hashSeeds] and [entries].
func containsCommon(hashSeeds []uint64, entries []byte, hash uint64) bool {
	for _, seed := range hashSeeds {
		if !containsWithSeed(entries, hash, seed) {
			return false
		}
	}
	return true
}

func containsWithSeed(entries []byte, hash, seed uint64) bool {
	index := getIndex(entries, hash, seed)
	byteIndex := index / bitsPerByte
	bitIndex := index % bitsPerByte
	return entries[byteIndex]&(1<<bitIndex) != 0
}

func getIndex(entries []byte, hash, seed uint64) uint64 {
	// If the filter has L entries, we only want to use hashes in the range
	// [0, L). We achieve this by incrementally shifting the hash to use
	// bits that we haven't used before. If we run out of bits, we
	// perform a "hash extension" by rehashing the original hash and the
	// seed.
	//
	// If the size of the bloom filter approaches MaxUint64, the hash may
	// overflow back to the beginning of the range. This means that the
	// same index can be used more than once for a single value.
	// This is OK, it just means the filter isn't as good as it could be
	// if we had more bits in the hash.
	numEntries := uint64(len(entries))
	entriesMask := numEntries*bitsPerByte - 1
	hash += seed
	// Note: It may be significantly more performant to use:
	// https://lemire.me/blog/2016/06/27/a-fast-alternative-to-the-modulo-reduction/
	// over the modulo operation.
	index := hash & entriesMask
	for index >= numEntries*bitsPerByte {
		hash = hash >> hashRotation
		// If the hash is zero, we have run out of bits and need to
		// rehash.
		if hash == 0 {
			hash = uint64(murmur3.Sum64(binary.BigEndian.AppendUint64(binary.BigEndian.AppendUint64(nil, hash), seed)))
		}
		index = hash & entriesMask
	}
	return index
}
