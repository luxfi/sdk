// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	consensusctx "github.com/luxfi/consensus/context"

	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

var (
	_ UnsignedTx             = (*OperationTx)(nil)
	_ secp256k1fx.UnsignedTx = (*OperationTx)(nil)
)

// OperationTx is a transaction with no credentials.
type OperationTx struct {
	BaseTx `serialize:"true"`

	Ops []*Operation `serialize:"true" json:"operations"`
}

func (t *OperationTx) InitCtx(ctx *consensusctx.Context) {
	// FxOperation doesn't have InitCtx method
	// for _, op := range t.Ops {
	//     op.Op.InitCtx(ctx)
	// }
	t.BaseTx.InitCtx(ctx)
}

// InitializeContext initializes the context for this transaction
func (t *OperationTx) InitializeContext(ctx *consensusctx.Context) error {
	t.InitCtx(ctx)
	return nil
}

// Operations track which ops this transaction is performing. The returned array
// should not be modified.
func (t *OperationTx) Operations() []*Operation {
	return t.Ops
}

func (t *OperationTx) InputUTXOs() []*lux.UTXOID {
	utxos := t.BaseTx.InputUTXOs()
	for _, op := range t.Ops {
		utxos = append(utxos, op.UTXOIDs...)
	}
	return utxos
}

func (t *OperationTx) InputIDs() set.Set[ids.ID] {
	inputs := t.BaseTx.InputIDs()
	for _, op := range t.Ops {
		for _, utxo := range op.UTXOIDs {
			inputs.Add(utxo.InputID())
		}
	}
	return inputs
}

// NumCredentials returns the number of expected credentials
func (t *OperationTx) NumCredentials() int {
	return t.BaseTx.NumCredentials() + len(t.Ops)
}

func (t *OperationTx) Visit(v Visitor) error {
	return v.OperationTx(t)
}

// InitializeWithContext initializes the transaction with consensus context
func (tx *OperationTx) InitializeWithContext(ctx *consensusctx.Context) error {
	// Initialize any context-dependent fields here
	return nil
}
