// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"context"
	"errors"

	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/const"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/vms/components/verify"
)

var (
	_ UnsignedTx = (*RemoveChainValidatorTx)(nil)

	ErrRemovePrimaryNetworkValidator = errors.New("can't remove primary network validator with RemoveChainValidatorTx")
)

// RemoveChainValidatorTx removes a validator from a chain.
type RemoveChainValidatorTx struct {
	BaseTx `serialize:"true"`
	// The node to remove from the chain.
	NodeID ids.NodeID `serialize:"true" json:"nodeID"`
	// The chain to remove the node from.
	Chain ids.ID `serialize:"true" json:"chainID"`
	// Proves that the issuer has the right to remove the node from the chain.
	ChainAuth verify.Verifiable `serialize:"true" json:"chainAuthorization"`
}

func (tx *RemoveChainValidatorTx) SyntacticVerify(ctx *consensusctx.Context) error {
	switch {
	case tx == nil:
		return ErrNilTx
	case tx.SyntacticallyVerified:
		// already passed syntactic verification
		return nil
	case tx.Chain == constants.PrimaryNetworkID:
		return ErrRemovePrimaryNetworkValidator
	}

	if err := tx.BaseTx.SyntacticVerify(ctx); err != nil {
		return err
	}
	if err := tx.ChainAuth.Verify(); err != nil {
		return err
	}

	tx.SyntacticallyVerified = true
	return nil
}

func (tx *RemoveChainValidatorTx) Visit(visitor Visitor) error {
	return visitor.RemoveChainValidatorTx(tx)
}

// InitializeWithContext initializes the transaction with consensus context
func (tx *RemoveChainValidatorTx) InitializeWithContext(ctx context.Context) error {
	// Initialize any context-dependent fields here
	return nil
}
