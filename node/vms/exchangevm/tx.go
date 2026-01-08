// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package exchangevm

import (
	"context"
	"errors"
	"fmt"

	"github.com/luxfi/log"

	"github.com/luxfi/consensus/core/choices"
	"github.com/luxfi/consensus/engine/dag"
	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/vms/exchangevm/txs"
	"github.com/luxfi/sdk/node/vms/exchangevm/txs/executor"
)

var (
	_ dag.Tx = (*Tx)(nil)

	errTxNotProcessing  = errors.New("transaction is not processing")
	errUnexpectedReject = errors.New("attempting to reject transaction")
)

type Tx struct {
	vm *VM
	tx *txs.Tx
}

func (tx *Tx) ID() ids.ID {
	return tx.tx.ID()
}

// Height returns the height of this transaction (not used in XVM)
func (tx *Tx) Height() uint64 {
	return 0
}

// Parent returns the parent ID (not used in XVM DAG)
func (tx *Tx) Parent() ids.ID {
	return ids.Empty
}

// ParentIDs returns the IDs of the parent transactions (inputs)
func (tx *Tx) ParentIDs() []ids.ID {
	// Return the transaction IDs this transaction depends on
	parents := []ids.ID{}
	for _, in := range tx.tx.Unsigned.InputUTXOs() {
		if in.Symbolic() {
			continue
		}
		txID, _ := in.InputSource()
		parents = append(parents, txID)
	}
	return parents
}

func (tx *Tx) Accept(ctx context.Context) error {
	if s := tx.Status(); s != choices.Processing {
		return fmt.Errorf("%w: %s", errTxNotProcessing, s)
	}

	tx.vm.onAccept(tx.tx)

	executor := &executor.Executor{
		Codec:  tx.vm.txBackend.Codec,
		State:  tx.vm.state,
		Tx:     tx.tx,
		Inputs: set.NewSet[ids.ID](0), // Initialize empty set for imported inputs
	}
	err := tx.tx.Unsigned.Visit(executor)
	if err != nil {
		return fmt.Errorf("error staging accepted state changes: %w", err)
	}

	tx.vm.state.AddTx(tx.tx)

	commitBatch, err := tx.vm.state.CommitBatch()
	if err != nil {
		txID := tx.tx.ID()
		return fmt.Errorf("couldn't create commitBatch while processing tx %s: %w", txID, err)
	}

	defer tx.vm.state.Abort()
	// Convert the atomicRequests to interface{} type for SharedMemory
	requests := make(map[ids.ID]interface{}, len(executor.AtomicRequests))
	for chainID, reqs := range executor.AtomicRequests {
		requests[chainID] = reqs
	}
	err = tx.vm.SharedMemory.Apply(
		requests,
		commitBatch,
	)
	if err != nil {
		txID := tx.tx.ID()
		return fmt.Errorf("error committing accepted state changes while processing tx %s: %w", txID, err)
	}

	return tx.vm.metrics.MarkTxAccepted(tx.tx)
}

func (*Tx) Reject(ctx context.Context) error {
	return errUnexpectedReject
}

func (tx *Tx) Status() choices.Status {
	txID := tx.tx.ID()
	_, err := tx.vm.state.GetTx(txID)
	switch err {
	case nil:
		return choices.Accepted
	case database.ErrNotFound:
		return choices.Processing
	default:
		tx.vm.log.Error("failed looking up tx status",
			log.Stringer("txID", txID),
			log.String("error", err.Error()),
		)
		return choices.Processing
	}
}

func (tx *Tx) MissingDependencies() (set.Set[ids.ID], error) {
	txIDs := make(set.Set[ids.ID])
	for _, in := range tx.tx.Unsigned.InputUTXOs() {
		if in.Symbolic() {
			continue
		}
		txID, _ := in.InputSource()

		_, err := tx.vm.state.GetTx(txID)
		switch err {
		case nil:
			// Tx was already accepted
		case database.ErrNotFound:
			txIDs.Add(txID)
		default:
			return nil, err
		}
	}
	return txIDs, nil
}

func (tx *Tx) Bytes() []byte {
	return tx.tx.Bytes()
}

func (tx *Tx) Verify(ctx context.Context) error {
	if s := tx.Status(); s != choices.Processing {
		return fmt.Errorf("%w: %s", errTxNotProcessing, s)
	}
	return tx.tx.Unsigned.Visit(&executor.SemanticVerifier{
		Backend: tx.vm.txBackend,
		State:   tx.vm.state,
		Tx:      tx.tx,
	})
}
