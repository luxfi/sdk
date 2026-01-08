// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package mempool

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/metric"

	"github.com/luxfi/sdk/utils"

	"github.com/luxfi/sdk/node/vms/exchangevm/txs"

	"github.com/luxfi/sdk/node/vms/components/lux"
)

func newMempool() (*Mempool, error) {
	return New("mempool", metric.NewNoOpRegistry())
}

func TestMempoolBasics(t *testing.T) {
	require := require.New(t)

	mempool, err := newMempool()
	require.NoError(err)

	// Test that mempool starts empty
	require.False(mempool.HasTxs())

	// Add a transaction
	tx := newTx(0, 32)
	require.NoError(mempool.Add(tx))

	// Verify mempool now has transactions
	require.True(mempool.HasTxs())
}

func newTx(index uint32, size int) *txs.Tx {
	tx := &txs.Tx{Unsigned: &txs.BaseTx{BaseTx: lux.BaseTx{
		Ins: []*lux.TransferableInput{{
			UTXOID: lux.UTXOID{
				TxID:        ids.ID{'t', 'x', 'I', 'D'},
				OutputIndex: index,
			},
		}},
	}}}
	tx.SetBytes(utils.RandomBytes(size), utils.RandomBytes(size))
	return tx
}
