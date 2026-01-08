// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"github.com/luxfi/metric"

	"github.com/luxfi/sdk/node/vms/platformvm/block"
)

const blkLabel = "blk"

var (
	_ block.Visitor = (*blockMetrics)(nil)

	blkLabels = []string{blkLabel}
)

type blockMetrics struct {
	txMetrics *txMetrics
	abortBlockCounter metric.Counter
	commitBlockCounter metric.Counter
	proposalBlockCounter metric.Counter
	standardBlockCounter metric.Counter
	atomicBlockCounter metric.Counter
}

func newBlockMetrics(registerer metric.Registerer) (*blockMetrics, error) {
	txMetrics, err := newTxMetrics(registerer)
	if err != nil {
		return nil, err
	}

	m := &blockMetrics{
		txMetrics: txMetrics,
		abortBlockCounter: metric.NewCounter("blks_accepted_abort", "number of abort blocks accepted"),
		commitBlockCounter: metric.NewCounter("blks_accepted_commit", "number of commit blocks accepted"),
		proposalBlockCounter: metric.NewCounter("blks_accepted_proposal", "number of proposal blocks accepted"),
		standardBlockCounter: metric.NewCounter("blks_accepted_standard", "number of standard blocks accepted"),
		atomicBlockCounter: metric.NewCounter("blks_accepted_atomic", "number of atomic blocks accepted"),
	}
	return m, nil
}

func (m *blockMetrics) BanffAbortBlock(*block.BanffAbortBlock) error {
	m.abortBlockCounter.Inc()
	return nil
}

func (m *blockMetrics) BanffCommitBlock(*block.BanffCommitBlock) error {
	m.commitBlockCounter.Inc()
	return nil
}

func (m *blockMetrics) BanffProposalBlock(b *block.BanffProposalBlock) error {
	m.proposalBlockCounter.Inc()
	for _, tx := range b.Transactions {
		if err := tx.Unsigned.Visit(m.txMetrics); err != nil {
			return err
		}
	}
	return b.Tx.Unsigned.Visit(m.txMetrics)
}

func (m *blockMetrics) BanffStandardBlock(b *block.BanffStandardBlock) error {
	m.standardBlockCounter.Inc()
	for _, tx := range b.Transactions {
		if err := tx.Unsigned.Visit(m.txMetrics); err != nil {
			return err
		}
	}
	return nil
}

func (m *blockMetrics) ApricotAbortBlock(*block.ApricotAbortBlock) error {
	m.abortBlockCounter.Inc()
	return nil
}

func (m *blockMetrics) ApricotCommitBlock(*block.ApricotCommitBlock) error {
	m.commitBlockCounter.Inc()
	return nil
}

func (m *blockMetrics) ApricotProposalBlock(b *block.ApricotProposalBlock) error {
	m.proposalBlockCounter.Inc()
	return b.Tx.Unsigned.Visit(m.txMetrics)
}

func (m *blockMetrics) ApricotStandardBlock(b *block.ApricotStandardBlock) error {
	m.standardBlockCounter.Inc()
	for _, tx := range b.Transactions {
		if err := tx.Unsigned.Visit(m.txMetrics); err != nil {
			return err
		}
	}
	return nil
}

func (m *blockMetrics) ApricotAtomicBlock(b *block.ApricotAtomicBlock) error {
	m.atomicBlockCounter.Inc()
	return b.Tx.Unsigned.Visit(m.txMetrics)
}
