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
	_ UnsignedTx             = (*ImportTx)(nil)
	_ secp256k1fx.UnsignedTx = (*ImportTx)(nil)
)

// ImportTx is a transaction that imports an asset from another blockchain.
type ImportTx struct {
	BaseTx `serialize:"true"`

	// Which chain to consume the funds from
	SourceChain ids.ID `serialize:"true" json:"sourceChain"`

	// The inputs to this transaction
	ImportedIns []*lux.TransferableInput `serialize:"true" json:"importedInputs"`
}

// InputUTXOs track which UTXOs this transaction is consuming.
func (t *ImportTx) InputUTXOs() []*lux.UTXOID {
	utxos := t.BaseTx.InputUTXOs()
	for _, in := range t.ImportedIns {
		in.Symbol = true
		utxos = append(utxos, &in.UTXOID)
	}
	return utxos
}

func (t *ImportTx) InputIDs() set.Set[ids.ID] {
	inputs := t.BaseTx.InputIDs()
	for _, in := range t.ImportedIns {
		inputs.Add(in.InputID())
	}
	return inputs
}

// NumCredentials returns the number of expected credentials
func (t *ImportTx) NumCredentials() int {
	return t.BaseTx.NumCredentials() + len(t.ImportedIns)
}

func (t *ImportTx) InitCtx(ctx *consensusctx.Context) {
	// TransferableInput doesn't have InitCtx
	// for _, in := range t.ImportedIns {
	//     in.InitCtx(ctx)
	// }
	t.BaseTx.InitCtx(ctx)
}

// InitializeContext initializes the context for this transaction
func (t *ImportTx) InitializeContext(ctx *consensusctx.Context) error {
	t.InitCtx(ctx)
	return nil
}

func (t *ImportTx) Visit(v Visitor) error {
	return v.ImportTx(t)
}

// InitializeWithContext initializes the transaction with consensus context
func (tx *ImportTx) InitializeWithContext(ctx *consensusctx.Context) error {
	// Initialize any context-dependent fields here
	return nil
}
