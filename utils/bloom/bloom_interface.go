// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package bloom

// BloomFilter is the interface for bloom filter implementations that support
// adding and checking raw byte slices
type BloomFilter interface {
	// Add adds to filter, assumed thread safe
	Add(...[]byte)

	// Check checks filter, assumed thread safe
	Check([]byte) bool
}
