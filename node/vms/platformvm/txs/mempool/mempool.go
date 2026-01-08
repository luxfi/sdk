// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mempool

import (
	"errors"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/metric"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	txmempool "github.com/luxfi/sdk/node/vms/txs/mempool"
)

var (
	ErrCantIssueAdvanceTimeTx     = errors.New("can not issue an advance time tx")
	ErrCantIssueRewardValidatorTx = errors.New("can not issue a reward validator tx")
	errMempoolFull                = errors.New("mempool is full")
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
	switch tx.Unsigned.(type) {
	case *txs.AdvanceTimeTx:
		return ErrCantIssueAdvanceTimeTx
	case *txs.RewardValidatorTx:
		return ErrCantIssueRewardValidatorTx
	default:
		return m.Mempool.Add(tx)
	}
}

func (m *Mempool) HasTxs() bool {
	return m.Len() > 0
}

func (m *Mempool) Has(txID ids.ID) bool {
	_, exists := m.Get(txID)
	return exists
}

func (m *Mempool) PeekTxs(n int) []*txs.Tx {
	var result []*txs.Tx
	count := 0
	m.Iterate(func(tx *txs.Tx) bool {
		if count >= n {
			return false
		}
		result = append(result, tx)
		count++
		return true
	})
	return result
}

func (m *Mempool) DropExpiredStakerTxs(minStartTime time.Time) []ids.ID {
	var droppedTxIDs []ids.ID
	var txsToRemove []*txs.Tx

	m.Iterate(func(tx *txs.Tx) bool {
		// Check if this is a staker transaction
		switch stakerTx := tx.Unsigned.(type) {
		case *txs.AddValidatorTx:
			if stakerTx.StartTime().Before(minStartTime) {
				droppedTxIDs = append(droppedTxIDs, tx.ID())
				txsToRemove = append(txsToRemove, tx)
			}
		case *txs.AddDelegatorTx:
			if stakerTx.StartTime().Before(minStartTime) {
				droppedTxIDs = append(droppedTxIDs, tx.ID())
				txsToRemove = append(txsToRemove, tx)
			}
		case *txs.AddPermissionlessValidatorTx:
			if stakerTx.StartTime().Before(minStartTime) {
				droppedTxIDs = append(droppedTxIDs, tx.ID())
				txsToRemove = append(txsToRemove, tx)
			}
		case *txs.AddPermissionlessDelegatorTx:
			if stakerTx.StartTime().Before(minStartTime) {
				droppedTxIDs = append(droppedTxIDs, tx.ID())
				txsToRemove = append(txsToRemove, tx)
			}
		}
		return true
	})

	if len(txsToRemove) > 0 {
		m.Remove(txsToRemove...)
	}

	return droppedTxIDs
}

func (m *Mempool) Remove(txs ...*txs.Tx) {
	m.Mempool.Remove(txs...)
}
