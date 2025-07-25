// (c) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"fmt"

	"github.com/luxfi/node/ids"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/rlp"
)

// GossipEthTx wraps a types.Transaction for gossip
type GossipEthTx struct {
	Tx *types.Transaction
}

// GossipID returns the ID of this gossipable message
func (g *GossipEthTx) GossipID() ids.ID {
	if g.Tx == nil {
		return ids.Empty
	}
	return ids.ID(g.Tx.Hash())
}

// GossipEthTxMarshaller handles marshalling/unmarshalling of eth txs
type GossipEthTxMarshaller struct{}

// MarshalGossip marshals the transaction for gossip
func (g GossipEthTxMarshaller) MarshalGossip(tx *GossipEthTx) ([]byte, error) {
	if tx == nil || tx.Tx == nil {
		return nil, fmt.Errorf("cannot marshal nil transaction")
	}
	return rlp.EncodeToBytes(tx.Tx)
}

// UnmarshalGossip unmarshals the transaction from gossip
func (g GossipEthTxMarshaller) UnmarshalGossip(bytes []byte) (*GossipEthTx, error) {
	tx := &types.Transaction{}
	if err := rlp.DecodeBytes(bytes, tx); err != nil {
		return nil, err
	}
	return &GossipEthTx{Tx: tx}, nil
}