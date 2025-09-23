// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package pathdb

import (
	"math/rand"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/trie/trienode"
)

// Test helper functions for pathdb tests

// randBytes generates random bytes for testing
func randBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

// randomAddress generates a random address for testing
func randomAddress() common.Address {
	return common.BytesToAddress(randBytes(20))
}

// randomNode generates a random node for testing
func randomNode() *trienode.Node {
	return trienode.New(common.BytesToHash(randBytes(32)), randBytes(100))
}
