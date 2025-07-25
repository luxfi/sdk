// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/luxfi/evm/core/txpool"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/node/ids"
)

// GossipEthTxPool is a wrapper around TxPool for gossip
type GossipEthTxPool struct {
	txPool *txpool.TxPool
}

// NewGossipEthTxPool creates a new gossip transaction pool
func NewGossipEthTxPool(txPool *txpool.TxPool, registerer prometheus.Registerer) (*GossipEthTxPool, error) {
	if txPool == nil {
		return nil, fmt.Errorf("txPool cannot be nil")
	}
	return &GossipEthTxPool{
		txPool: txPool,
	}, nil
}

// Add implements the gossip.Mempool interface
func (g *GossipEthTxPool) Add(tx *GossipEthTx) error {
	if tx == nil || tx.Tx == nil {
		return fmt.Errorf("invalid transaction")
	}
	// Add transaction to the pool
	errs := g.txPool.Add([]*types.Transaction{tx.Tx}, true, false)
	if len(errs) > 0 && errs[0] != nil {
		return errs[0]
	}
	return nil
}

// Has implements the gossip.Mempool interface
func (g *GossipEthTxPool) Has(id ids.ID) bool {
	return g.txPool.Has(common.Hash(id))
}

// GetFilter implements the gossip.Mempool interface
func (g *GossipEthTxPool) GetFilter() ([]byte, []byte) {
	// TODO: Implement bloom filter for transaction gossip
	return nil, nil
}