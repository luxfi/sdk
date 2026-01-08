// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package bloom

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	var (
		require = require.New(t)
		count   = 10000
		p       = 0.1
	)

	numHashes, numEntries := OptimalParameters(count, p)
	f, err := New(numHashes, numEntries)
	require.NoError(err)
	require.NotNil(f)

	salt := []byte("test salt")
	Add(f, []byte("hello"), salt)

	contains := Contains(f, []byte("hello"), salt)
	require.True(contains, "should have contained the key")

	contains = Contains(f, []byte("bye"), salt)
	require.False(contains, "shouldn't have contained the key")
}
