// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"context"

	consensusctx "github.com/luxfi/consensus/context"

	"errors"

	"github.com/luxfi/ids"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/node/vms/components/verify"
	"github.com/luxfi/sdk/node/vms/platformvm/fx"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

var (
	_ UnsignedTx = (*TransferChainOwnershipTx)(nil)

	ErrTransferPermissionlessChain = errors.New("cannot transfer ownership of a permissionless chain")
)

type TransferChainOwnershipTx struct {
	// Metadata, inputs and outputs
	BaseTx `serialize:"true"`
	// ID of the chain this tx is modifying
	Chain ids.ID `serialize:"true" json:"chainID"`
	// Proves that the issuer has the right to modify the chain.
	ChainAuth verify.Verifiable `serialize:"true" json:"chainAuthorization"`
	// Who is now authorized to manage this chain
	Owner fx.Owner `serialize:"true" json:"newOwner"`
}

// InitCtx sets the FxID fields in the inputs and outputs of this
// [TransferChainOwnershipTx]. Also sets the [ctx] to the given [vm.ctx] so
// that the addresses can be json marshalled into human readable format
func (tx *TransferChainOwnershipTx) InitCtx(ctx *consensusctx.Context) {
	tx.BaseTx.InitCtx(ctx)
	// Initialize context for Owner if it's *secp256k1fx.OutputOwners
	if owner, ok := tx.Owner.(*secp256k1fx.OutputOwners); ok {
		owner.InitCtx(ctx)
	}
}

func (tx *TransferChainOwnershipTx) SyntacticVerify(ctx *consensusctx.Context) error {
	switch {
	case tx == nil:
		return ErrNilTx
	case tx.SyntacticallyVerified:
		// already passed syntactic verification
		return nil
	case tx.Chain == constants.PrimaryNetworkID:
		return ErrTransferPermissionlessChain
	}

	if err := tx.BaseTx.SyntacticVerify(ctx); err != nil {
		return err
	}
	if err := verify.All(tx.ChainAuth, tx.Owner); err != nil {
		return err
	}

	tx.SyntacticallyVerified = true
	return nil
}

func (tx *TransferChainOwnershipTx) Visit(visitor Visitor) error {
	return visitor.TransferChainOwnershipTx(tx)
}

// InitializeWithContext initializes the transaction with consensus context
func (tx *TransferChainOwnershipTx) InitializeWithContext(ctx context.Context) error {
	// Initialize any context-dependent fields here
	return nil
}
