// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package mempool

import (
	"github.com/luxfi/metric"
	"github.com/luxfi/sdk/node/vms/exchangevm/txs"
	txmempool "github.com/luxfi/sdk/node/vms/txs/mempool"
)

type Mempool struct {
	txmempool.Mempool[*txs.Tx]
}

func New(namespace string, registerer metric.Registerer) (*Mempool, error) {
	metrics, err := txmempool.NewMetrics(namespace, registerer)
	if err != nil {
		return nil, err
	}
	pool := txmempool.New[*txs.Tx](
		metrics,
	)
	return &Mempool{Mempool: pool}, nil
}

func (m *Mempool) Add(tx *txs.Tx) error {
	return m.Mempool.Add(tx)
}

func (m *Mempool) HasTxs() bool {
	return m.Len() > 0
}
