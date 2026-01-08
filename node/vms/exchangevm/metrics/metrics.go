// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"errors"

	"github.com/luxfi/metric"
	utilmetric "github.com/luxfi/sdk/utils/metric"

	"github.com/luxfi/sdk/utils/wrappers"
	"github.com/luxfi/sdk/node/vms/exchangevm/block"
	"github.com/luxfi/sdk/node/vms/exchangevm/txs"
)

var _ Metrics = (*metricsImpl)(nil)

type Metrics interface {
	utilmetric.APIInterceptor

	IncTxRefreshes()
	IncTxRefreshHits()
	IncTxRefreshMisses()

	// MarkBlockAccepted updates all metrics relating to the acceptance of a
	// block, including the underlying acceptance of the contained transactions.
	MarkBlockAccepted(b block.Block) error
	// MarkTxAccepted updates all metrics relating to the acceptance of a
	// transaction.
	//
	// Note: This is not intended to be called during the acceptance of a block,
	// as MarkBlockAccepted already handles updating transaction related
	// metric.
	MarkTxAccepted(tx *txs.Tx) error
}

type metricsImpl struct {
	txMetrics *txMetrics

	numTxRefreshes, numTxRefreshHits, numTxRefreshMisses metric.Counter

	utilmetric.APIInterceptor
}

func (m *metricsImpl) IncTxRefreshes() {
	m.numTxRefreshes.Inc()
}

func (m *metricsImpl) IncTxRefreshHits() {
	m.numTxRefreshHits.Inc()
}

func (m *metricsImpl) IncTxRefreshMisses() {
	m.numTxRefreshMisses.Inc()
}

func (m *metricsImpl) MarkBlockAccepted(b block.Block) error {
	for _, tx := range b.Txs() {
		if err := tx.Unsigned.Visit(m.txMetrics); err != nil {
			return err
		}
	}
	return nil
}

func (m *metricsImpl) MarkTxAccepted(tx *txs.Tx) error {
	return tx.Unsigned.Visit(m.txMetrics)
}

func New(registerer metric.Registerer) (Metrics, error) {
	registry, ok := registerer.(metric.Registry)
	if !ok {
		return nil, errors.New("registerer must implement metric.Registry")
	}
	txMetrics, err := newTxMetrics(registry)
	errs := wrappers.Errs{Err: err}

	m := &metricsImpl{txMetrics: txMetrics}

	m.numTxRefreshes = metric.NewCounter(metric.CounterOpts{
		Name: "tx_refreshes",
		Help: "Number of times unique txs have been refreshed",
	})
	m.numTxRefreshHits = metric.NewCounter(metric.CounterOpts{
		Name: "tx_refresh_hits",
		Help: "Number of times unique txs have not been unique, but were cached",
	})
	m.numTxRefreshMisses = metric.NewCounter(metric.CounterOpts{
		Name: "tx_refresh_misses",
		Help: "Number of times unique txs have not been unique and weren't cached",
	})

	apiRequestMetric, err := utilmetric.NewAPIInterceptor(registry)
	m.APIInterceptor = apiRequestMetric
	errs.Add(err)
	// Metrics are self-registering when created with NewCounter etc.
	return m, errs.Err
}
