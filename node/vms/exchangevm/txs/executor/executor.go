// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"fmt"

	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/chains/atomic"
	"github.com/luxfi/sdk/node/codec"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/exchangevm/state"
	"github.com/luxfi/sdk/node/vms/exchangevm/txs"
)

var _ txs.Visitor = (*Executor)(nil)

type Executor struct {
	Codec          codec.Manager
	State          state.Chain // state will be modified
	Tx             *txs.Tx
	Inputs         set.Set[ids.ID]             // imported inputs
	AtomicRequests map[ids.ID]*atomic.Requests // may be nil
}

func (e *Executor) BaseTx(tx *txs.BaseTx) error {
	txID := e.Tx.ID()
	lux.Consume(e.State, tx.Ins)
	lux.Produce(e.State, txID, tx.Outs)
	return nil
}

func (e *Executor) CreateAssetTx(tx *txs.CreateAssetTx) error {
	if err := e.BaseTx(&tx.BaseTx); err != nil {
		return err
	}

	txID := e.Tx.ID()
	index := len(tx.Outs)
	for _, state := range tx.States {
		for _, out := range state.Outs {
			e.State.AddUTXO(&lux.UTXO{
				UTXOID: lux.UTXOID{
					TxID:        txID,
					OutputIndex: uint32(index),
				},
				Asset: lux.Asset{
					ID: txID,
				},
				Out: out,
			})
			index++
		}
	}
	return nil
}

func (e *Executor) OperationTx(tx *txs.OperationTx) error {
	if err := e.BaseTx(&tx.BaseTx); err != nil {
		return err
	}

	txID := e.Tx.ID()
	index := len(tx.Outs)
	for _, op := range tx.Ops {
		for _, utxoID := range op.UTXOIDs {
			e.State.DeleteUTXO(utxoID.InputID())
		}
		asset := op.AssetID()
		for _, out := range op.Op.Outs() {
			e.State.AddUTXO(&lux.UTXO{
				UTXOID: lux.UTXOID{
					TxID:        txID,
					OutputIndex: uint32(index),
				},
				Asset: lux.Asset{ID: asset},
				Out:   out,
			})
			index++
		}
	}
	return nil
}

func (e *Executor) ImportTx(tx *txs.ImportTx) error {
	if err := e.BaseTx(&tx.BaseTx); err != nil {
		return err
	}

	utxoIDs := make([][]byte, len(tx.ImportedIns))
	for i, in := range tx.ImportedIns {
		utxoID := in.UTXOID.InputID()

		e.Inputs.Add(utxoID)
		utxoIDs[i] = utxoID[:]
	}
	e.AtomicRequests = map[ids.ID]*atomic.Requests{
		tx.SourceChain: {
			RemoveRequests: utxoIDs,
		},
	}
	return nil
}

func (e *Executor) ExportTx(tx *txs.ExportTx) error {
	if err := e.BaseTx(&tx.BaseTx); err != nil {
		return err
	}

	txID := e.Tx.ID()
	index := len(tx.Outs)
	elems := make([]*atomic.Element, len(tx.ExportedOuts))
	for i, out := range tx.ExportedOuts {
		utxo := &lux.UTXO{
			UTXOID: lux.UTXOID{
				TxID:        txID,
				OutputIndex: uint32(index),
			},
			Asset: lux.Asset{ID: out.AssetID()},
			Out:   out.Out,
		}
		index++

		utxoBytes, err := e.Codec.Marshal(txs.CodecVersion, utxo)
		if err != nil {
			return fmt.Errorf("failed to marshal UTXO: %w", err)
		}
		utxoID := utxo.InputID()
		elem := &atomic.Element{
			Key:   utxoID[:],
			Value: utxoBytes,
		}
		if out, ok := utxo.Out.(lux.Addressable); ok {
			elem.Traits = out.Addresses()
		}

		elems[i] = elem
	}
	e.AtomicRequests = map[ids.ID]*atomic.Requests{
		tx.DestinationChain: {
			PutRequests: elems,
		},
	}
	return nil
}
