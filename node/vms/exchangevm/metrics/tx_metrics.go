// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"github.com/luxfi/metric"

	"github.com/luxfi/sdk/node/vms/exchangevm/txs"
)

const txLabel = "tx"

var (
	_ txs.Visitor = (*txMetrics)(nil)

	txLabels = []string{txLabel}
)

type txMetrics struct {
	numTxs metric.CounterVec
}

func newTxMetrics(registerer metric.Registerer) (*txMetrics, error) {
	m := &txMetrics{
		numTxs: metric.NewCounterVec(
			metric.CounterOpts{
				Name: "txs_accepted",
				Help: "number of transactions accepted",
			},
			txLabels,
		),
	}
	return m, nil
}

func (m *txMetrics) BaseTx(*txs.BaseTx) error {
	m.numTxs.With(metric.Labels{
		txLabel: "base",
	}).Inc()
	return nil
}

func (m *txMetrics) CreateAssetTx(*txs.CreateAssetTx) error {
	m.numTxs.With(metric.Labels{
		txLabel: "create_asset",
	}).Inc()
	return nil
}

func (m *txMetrics) OperationTx(*txs.OperationTx) error {
	m.numTxs.With(metric.Labels{
		txLabel: "operation",
	}).Inc()
	return nil
}

func (m *txMetrics) ImportTx(*txs.ImportTx) error {
	m.numTxs.With(metric.Labels{
		txLabel: "import",
	}).Inc()
	return nil
}

func (m *txMetrics) ExportTx(*txs.ExportTx) error {
	m.numTxs.With(metric.Labels{
		txLabel: "export",
	}).Inc()
	return nil
}
