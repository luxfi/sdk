// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package network

import (
	"errors"
	"testing"

	"github.com/luxfi/log"
	"github.com/luxfi/metric"
	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"

	"github.com/luxfi/sdk/node/vms/txs/mempool"

	pmempool "github.com/luxfi/sdk/node/vms/platformvm/txs/mempool"
)

var errFoo = errors.New("foo")

// Add should error if verification errors
func TestGossipMempoolAddVerificationError(t *testing.T) {
	require := require.New(t)

	txID := ids.GenerateTestID()
	tx := &txs.Tx{
		TxID: txID,
	}

	mempool, err := pmempool.New("", metric.NewRegistry())
	require.NoError(err)
	txVerifier := testTxVerifier{err: errFoo}

	gossipMempool, err := newGossipMempool(
		mempool,
		metric.NewRegistry(),
		log.NewNoOpLogger(),
		txVerifier,
		testConfig.ExpectedBloomFilterElements,
		testConfig.ExpectedBloomFilterFalsePositiveProbability,
		testConfig.MaxBloomFilterFalsePositiveProbability,
	)
	require.NoError(err)

	err = gossipMempool.Add(tx)
	require.ErrorIs(err, errFoo)
	require.False(gossipMempool.bloom.Has(tx))
}

// Adding a duplicate to the mempool should return an error
func TestMempoolDuplicate(t *testing.T) {
	require := require.New(t)

	testMempool, err := pmempool.New("", metric.NewRegistry())
	require.NoError(err)
	txVerifier := testTxVerifier{}

	txID := ids.GenerateTestID()
	tx := &txs.Tx{
		Unsigned: &txs.BaseTx{},
		TxID:     txID,
	}

	require.NoError(testMempool.Add(tx))
	gossipMempool, err := newGossipMempool(
		testMempool,
		metric.NewRegistry(),
		nil,
		txVerifier,
		testConfig.ExpectedBloomFilterElements,
		testConfig.ExpectedBloomFilterFalsePositiveProbability,
		testConfig.MaxBloomFilterFalsePositiveProbability,
	)
	require.NoError(err)

	err = gossipMempool.Add(tx)
	require.ErrorIs(err, mempool.ErrDuplicateTx)
	require.False(gossipMempool.bloom.Has(tx))
}

// Adding a tx to the mempool should add it to the bloom filter
func TestGossipAddBloomFilter(t *testing.T) {
	require := require.New(t)

	txID := ids.GenerateTestID()
	tx := &txs.Tx{
		Unsigned: &txs.BaseTx{},
		TxID:     txID,
	}

	txVerifier := testTxVerifier{}
	mempool, err := pmempool.New("", metric.NewRegistry())
	require.NoError(err)

	gossipMempool, err := newGossipMempool(
		mempool,
		metric.NewRegistry(),
		log.NewNoOpLogger(),
		txVerifier,
		testConfig.ExpectedBloomFilterElements,
		testConfig.ExpectedBloomFilterFalsePositiveProbability,
		testConfig.MaxBloomFilterFalsePositiveProbability,
	)
	require.NoError(err)

	require.NoError(gossipMempool.Add(tx))
	require.True(gossipMempool.bloom.Has(tx))
}
