// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"context"
	"errors"

	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/const"
	"github.com/luxfi/ids"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/sdk/node/vms/components/verify"
)

var (
	_ StakerTx        = (*AddChainValidatorTx)(nil)
	_ ScheduledStaker = (*AddChainValidatorTx)(nil)

	errAddPrimaryNetworkValidator = errors.New("can't add primary network validator with AddChainValidatorTx")
)

// AddChainValidatorTx is an unsigned addChainValidatorTx
type AddChainValidatorTx struct {
	// Metadata, inputs and outputs
	BaseTx `serialize:"true"`
	// The validator
	ChainValidator `serialize:"true" json:"validator"`
	// Auth that will be allowing this validator into the network
	ChainAuth verify.Verifiable `serialize:"true" json:"chainAuthorization"`
}

func (tx *AddChainValidatorTx) NodeID() ids.NodeID {
	return tx.ChainValidator.NodeID
}

func (*AddChainValidatorTx) PublicKey() (*bls.PublicKey, bool, error) {
	return nil, false, nil
}

func (*AddChainValidatorTx) PendingPriority() Priority {
	return ChainPermissionedValidatorPendingPriority
}

func (*AddChainValidatorTx) CurrentPriority() Priority {
	return ChainPermissionedValidatorCurrentPriority
}

// SyntacticVerify returns nil iff [tx] is valid
func (tx *AddChainValidatorTx) SyntacticVerify(ctx *consensusctx.Context) error {
	switch {
	case tx == nil:
		return ErrNilTx
	case tx.SyntacticallyVerified: // already passed syntactic verification
		return nil
	case tx.Chain == constants.PrimaryNetworkID:
		return errAddPrimaryNetworkValidator
	}

	if err := tx.BaseTx.SyntacticVerify(ctx); err != nil {
		return err
	}
	if err := verify.All(&tx.Validator, tx.ChainAuth); err != nil {
		return err
	}

	// cache that this is valid
	tx.SyntacticallyVerified = true
	return nil
}

func (tx *AddChainValidatorTx) Visit(visitor Visitor) error {
	return visitor.AddChainValidatorTx(tx)
}

// InitializeWithContext initializes the transaction with consensus context
func (tx *AddChainValidatorTx) InitializeWithContext(ctx context.Context) error {
	// Initialize any context-dependent fields here
	return nil
}
