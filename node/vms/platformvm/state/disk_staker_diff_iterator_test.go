// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thepudds/fzgen/fuzzer"

	"github.com/luxfi/database/memdb"
	"github.com/luxfi/ids"
)

func FuzzMarshalDiffKey(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		require := require.New(t)

		var (
			chainID ids.ID
			height  uint64
			nodeID  ids.NodeID
		)
		fz := fuzzer.NewFuzzer(data)
		fz.Fill(&chainID, &height, &nodeID)

		key := marshalDiffKey(chainID, height, nodeID)
		parsedChainID, parsedHeight, parsedNodeID, err := unmarshalDiffKey(key)
		require.NoError(err)
		require.Equal(chainID, parsedChainID)
		require.Equal(height, parsedHeight)
		require.Equal(nodeID, parsedNodeID)
	})
}

func FuzzUnmarshalDiffKey(f *testing.F) {
	f.Fuzz(func(t *testing.T, key []byte) {
		require := require.New(t)

		chainID, height, nodeID, err := unmarshalDiffKey(key)
		if err != nil {
			require.ErrorIs(err, errUnexpectedDiffKeyLength)
			return
		}

		formattedKey := marshalDiffKey(chainID, height, nodeID)
		require.Equal(key, formattedKey)
	})
}

func TestDiffIteration(t *testing.T) {
	require := require.New(t)

	db := memdb.New()

	chainID0 := ids.GenerateTestID()
	chainID1 := ids.GenerateTestID()

	nodeID0 := ids.BuildTestNodeID([]byte{0x00})
	nodeID1 := ids.BuildTestNodeID([]byte{0x01})

	chainID0Height0NodeID0 := marshalDiffKey(chainID0, 0, nodeID0)
	chainID0Height1NodeID0 := marshalDiffKey(chainID0, 1, nodeID0)
	chainID0Height1NodeID1 := marshalDiffKey(chainID0, 1, nodeID1)

	chainID1Height0NodeID0 := marshalDiffKey(chainID1, 0, nodeID0)
	chainID1Height1NodeID0 := marshalDiffKey(chainID1, 1, nodeID0)
	chainID1Height1NodeID1 := marshalDiffKey(chainID1, 1, nodeID1)

	require.NoError(db.Put(chainID0Height0NodeID0, nil))
	require.NoError(db.Put(chainID0Height1NodeID0, nil))
	require.NoError(db.Put(chainID0Height1NodeID1, nil))
	require.NoError(db.Put(chainID1Height0NodeID0, nil))
	require.NoError(db.Put(chainID1Height1NodeID0, nil))
	require.NoError(db.Put(chainID1Height1NodeID1, nil))

	{
		it := db.NewIteratorWithStartAndPrefix(marshalStartDiffKey(chainID0, 0), chainID0[:])
		defer it.Release()

		expectedKeys := [][]byte{
			chainID0Height0NodeID0,
		}
		for _, expectedKey := range expectedKeys {
			require.True(it.Next())
			require.Equal(expectedKey, it.Key())
		}
		require.False(it.Next())
		require.NoError(it.Error())
	}

	{
		it := db.NewIteratorWithStartAndPrefix(marshalStartDiffKey(chainID0, 1), chainID0[:])
		defer it.Release()

		expectedKeys := [][]byte{
			chainID0Height1NodeID0,
			chainID0Height1NodeID1,
			chainID0Height0NodeID0,
		}
		for _, expectedKey := range expectedKeys {
			require.True(it.Next())
			require.Equal(expectedKey, it.Key())
		}
		require.False(it.Next())
		require.NoError(it.Error())
	}
}
