// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package gasprice

import (
	"context"
	"math/big"
	"sync"
	"testing"

	"github.com/luxfi/evm/core"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/evm/plugin/evm/customtypes"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/stretchr/testify/require"
)

// makeTestHeader creates a header with properly set extras for testing
func makeTestHeader(number int64, parentHash common.Hash, gasUsed uint64) *types.Header {
	header := &types.Header{
		Number:  big.NewInt(number),
		GasUsed: gasUsed,
		BaseFee: big.NewInt(25_000_000_000), // Default test base fee
	}
	if parentHash != (common.Hash{}) {
		header.ParentHash = parentHash
	}
	// Set header extras with BlockGasCost for SubnetEVM compatibility
	customtypes.SetHeaderExtra(header, &customtypes.HeaderExtra{
		BlockGasCost: big.NewInt(0), // Use 0 for test blocks
	})
	return header
}

func TestFeeInfoProvider(t *testing.T) {
	backend := newTestBackend(t, params.TestChainConfig, 2, testGenBlock(t, 55, 370))
	f, err := newFeeInfoProvider(backend, 1, 2)
	require.NoError(t, err)

	// check that accepted event was subscribed
	require.NotNil(t, backend.acceptedEvent)

	// check fee infos were cached
	require.Equal(t, 2, f.cache.Len())

	// some block that extends the current chain
	var wg sync.WaitGroup
	wg.Add(1)
	f.newHeaderAdded = func() { wg.Done() }
	header := makeTestHeader(3, backend.LastAcceptedBlock().Hash(), 21000)
	block := types.NewBlockWithHeader(header)
	backend.acceptedEvent <- core.ChainEvent{Block: block}

	// wait for the event to process before validating the new header was added.
	wg.Wait()
	feeInfo, ok := f.get(3)
	require.True(t, ok)
	require.NotNil(t, feeInfo)
}

func TestFeeInfoProviderCacheSize(t *testing.T) {
	size := 5
	overflow := 3
	backend := newTestBackend(t, params.TestChainConfig, 0, testGenBlock(t, 55, 370))
	f, err := newFeeInfoProvider(backend, 1, size)
	require.NoError(t, err)

	// add [overflow] more elements than what will fit in the cache
	// to test eviction behavior.
	for i := 0; i < size+feeCacheExtraSlots+overflow; i++ {
		header := makeTestHeader(int64(i), common.Hash{}, 21000)
		_, err := f.addHeader(context.Background(), header)
		require.NoError(t, err)
	}

	// these numbers should be evicted
	for i := 0; i < overflow; i++ {
		feeInfo, ok := f.get(uint64(i))
		require.False(t, ok)
		require.Nil(t, feeInfo)
	}

	// these numbers should be present
	for i := overflow; i < size+feeCacheExtraSlots+overflow; i++ {
		feeInfo, ok := f.get(uint64(i))
		require.True(t, ok)
		require.NotNil(t, feeInfo)
	}
}
