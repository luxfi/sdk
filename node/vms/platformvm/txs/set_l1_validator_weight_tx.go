// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	consensusctx "github.com/luxfi/consensus/context"

	"github.com/luxfi/sdk/node/vms/types"
)

var _ UnsignedTx = (*SetL1ValidatorWeightTx)(nil)

type SetL1ValidatorWeightTx struct {
	// Metadata, inputs and outputs
	BaseTx `serialize:"true"`
	// Message is expected to be a signed Warp message containing an
	// AddressedCall payload with the SetL1ValidatorWeight message.
	Message types.JSONByteSlice `serialize:"true" json:"message"`
}

func (tx *SetL1ValidatorWeightTx) SyntacticVerify(ctx *consensusctx.Context) error {
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

	tx.SyntacticallyVerified = true
	return nil
}

func (tx *SetL1ValidatorWeightTx) Visit(visitor Visitor) error {
	return visitor.SetL1ValidatorWeightTx(tx)
}
