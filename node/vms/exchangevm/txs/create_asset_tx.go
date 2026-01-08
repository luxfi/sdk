// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	consensusctx "github.com/luxfi/consensus/context"

	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

var (
	_ UnsignedTx             = (*CreateAssetTx)(nil)
	_ secp256k1fx.UnsignedTx = (*CreateAssetTx)(nil)
)

// CreateAssetTx is a transaction that creates a new asset.
type CreateAssetTx struct {
	BaseTx       `serialize:"true"`
	Name         string          `serialize:"true" json:"name"`
	Symbol       string          `serialize:"true" json:"symbol"`
	Denomination byte            `serialize:"true" json:"denomination"`
	States       []*InitialState `serialize:"true" json:"initialStates"`
}

func (t *CreateAssetTx) InitCtx(ctx *consensusctx.Context) {
	for _, state := range t.States {
		state.InitCtx(ctx)
	}
	t.BaseTx.InitCtx(ctx)
}

// InitializeContext initializes the context for this transaction
func (t *CreateAssetTx) InitializeContext(ctx *consensusctx.Context) error {
	t.InitCtx(ctx)
	return nil
}

// InitialStates track which virtual machines, and the initial state of these
// machines, this asset uses. The returned array should not be modified.
func (t *CreateAssetTx) InitialStates() []*InitialState {
	return t.States
}

func (t *CreateAssetTx) Visit(v Visitor) error {
	return v.CreateAssetTx(t)
}

// InitializeWithContext initializes the transaction with consensus context
func (tx *CreateAssetTx) InitializeWithContext(ctx *consensusctx.Context) error {
	// Initialize any context-dependent fields here
	return nil
}
