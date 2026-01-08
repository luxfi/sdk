// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package network

import (
	"testing"

	"github.com/luxfi/metric"
	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/vms/exchangevm/fxs"
	"github.com/luxfi/sdk/node/vms/exchangevm/txs"
	"github.com/luxfi/sdk/node/vms/exchangevm/txs/mempool"
	"github.com/luxfi/sdk/node/vms/components/lux"

	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

var _ TxVerifier = (*testVerifier)(nil)

type testVerifier struct {
	err error
}

func (v testVerifier) VerifyTx(*txs.Tx) error {
	return v.err
}

func TestMarshaller(t *testing.T) {
	require := require.New(t)

	parser, err := txs.NewParser(
		[]fxs.Fx{
			&secp256k1fx.Fx{},
		},
	)
	require.NoError(err)

	marhsaller := txParser{
		parser: parser,
	}

	want := &txs.Tx{Unsigned: &txs.BaseTx{}}
	require.NoError(want.Initialize(parser.Codec()))

	bytes, err := marhsaller.MarshalGossip(want)
	require.NoError(err)

	got, err := marhsaller.UnmarshalGossip(bytes)
	require.NoError(err)
	require.Equal(want.GossipID(), got.GossipID())
}

func TestGossipMempoolAdd(t *testing.T) {
	require := require.New(t)

	metrics := metric.NewRegistry()

	baseMempool, err := mempool.New("", metrics)
	require.NoError(err)

	mempool, err := newGossipMempool(
		baseMempool,
		metrics,
		nil,
		testVerifier{},
		DefaultConfig.ExpectedBloomFilterElements,
		DefaultConfig.ExpectedBloomFilterFalsePositiveProbability,
		DefaultConfig.MaxBloomFilterFalsePositiveProbability,
	)
	require.NoError(err)

	tx := &txs.Tx{
		Unsigned: &txs.BaseTx{
			BaseTx: lux.BaseTx{
				Ins: []*lux.TransferableInput{},
			},
		},
		TxID: ids.GenerateTestID(),
	}

	require.NoError(mempool.Add(tx))
	require.True(mempool.bloom.Has(tx))
}

func TestGossipMempoolAddVerified(t *testing.T) {
	require := require.New(t)

	metrics := metric.NewRegistry()

	baseMempool, err := mempool.New("", metrics)
	require.NoError(err)

	mempool, err := newGossipMempool(
		baseMempool,
		metrics,
		nil,
		testVerifier{
			err: errTest, // We shouldn't be attempting to verify the tx in this flow
		},
		DefaultConfig.ExpectedBloomFilterElements,
		DefaultConfig.ExpectedBloomFilterFalsePositiveProbability,
		DefaultConfig.MaxBloomFilterFalsePositiveProbability,
	)
	require.NoError(err)

	tx := &txs.Tx{
		Unsigned: &txs.BaseTx{
			BaseTx: lux.BaseTx{
				Ins: []*lux.TransferableInput{},
			},
		},
		TxID: ids.GenerateTestID(),
	}

	require.NoError(mempool.AddWithoutVerification(tx))
	require.True(mempool.bloom.Has(tx))
}
