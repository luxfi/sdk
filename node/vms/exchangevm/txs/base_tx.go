// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package txs

import (
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"

	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/math/set"
)

var (
	_ UnsignedTx             = (*BaseTx)(nil)
	_ secp256k1fx.UnsignedTx = (*BaseTx)(nil)
)

// BaseTx is the basis of all transactions.
type BaseTx struct {
	lux.BaseTx `serialize:"true"`

	bytes []byte
}

func (t *BaseTx) InitCtx(ctx *consensusctx.Context) {
	for _, out := range t.Outs {
		out.InitCtx(ctx)
	}
}

// InitializeContext initializes the context for this transaction
func (t *BaseTx) InitializeContext(ctx *consensusctx.Context) error {
	t.InitCtx(ctx)
	return nil
}

func (t *BaseTx) SetBytes(bytes []byte) {
	t.bytes = bytes
}

func (t *BaseTx) Bytes() []byte {
	return t.bytes
}

func (t *BaseTx) InputIDs() set.Set[ids.ID] {
	inputIDs := make(set.Set[ids.ID], len(t.Ins))
	for _, in := range t.Ins {
		inputIDs.Add(in.InputID())
	}
	return inputIDs
}

// InputUTXOs returns the UTXOIDs this transaction is consuming
func (t *BaseTx) InputUTXOs() []*lux.UTXOID {
	utxos := make([]*lux.UTXOID, len(t.Ins))
	for i, in := range t.Ins {
		utxos[i] = &in.UTXOID
	}
	return utxos
}

func (t *BaseTx) Visit(v Visitor) error {
	return v.BaseTx(t)
}

// NumCredentials returns the number of expected credentials
func (t *BaseTx) NumCredentials() int {
	return len(t.Ins)
}

// InitializeWithContext initializes the transaction with consensus context
func (tx *BaseTx) InitializeWithContext(ctx *consensusctx.Context) error {
	// Initialize any context-dependent fields here
	tx.InitCtx(ctx)
	return nil
}
