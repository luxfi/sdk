// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	consensusctx "github.com/luxfi/consensus/context"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/vms/components/verify"
)

var _ UnsignedTx = (*DisableL1ValidatorTx)(nil)

type DisableL1ValidatorTx struct {
	// Metadata, inputs and outputs
	BaseTx `serialize:"true"`
	// ID corresponding to the validator
	ValidationID ids.ID `serialize:"true" json:"validationID"`
	// Authorizes this validator to be disabled
	DisableAuth verify.Verifiable `serialize:"true" json:"disableAuthorization"`
}

func (tx *DisableL1ValidatorTx) SyntacticVerify(ctx *consensusctx.Context) error {
	switch {
	case tx == nil:
		return ErrNilTx
	case tx.SyntacticallyVerified:
		// already passed syntactic verification
		return nil
	}

	if err := tx.BaseTx.SyntacticVerify(ctx); err != nil {
		return err
	}
	if err := tx.DisableAuth.Verify(); err != nil {
		return err
	}

	tx.SyntacticallyVerified = true
	return nil
}

func (tx *DisableL1ValidatorTx) Visit(visitor Visitor) error {
	return visitor.DisableL1ValidatorTx(tx)
}
