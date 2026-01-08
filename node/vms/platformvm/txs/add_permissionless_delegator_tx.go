// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"context"
	"fmt"

	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/ids"
	safemath "github.com/luxfi/math"
	"github.com/luxfi/const"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/components/verify"
	"github.com/luxfi/sdk/node/vms/platformvm/fx"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

var (
	_ DelegatorTx     = (*AddPermissionlessDelegatorTx)(nil)
	_ ScheduledStaker = (*AddPermissionlessDelegatorTx)(nil)
)

// AddPermissionlessDelegatorTx is an unsigned addPermissionlessDelegatorTx
type AddPermissionlessDelegatorTx struct {
	// Metadata, inputs and outputs
	BaseTx `serialize:"true"`
	// Describes the validator
	Validator `serialize:"true" json:"validator"`
	// ID of the chain this validator is validating
	Chain ids.ID `serialize:"true" json:"chainID"`
	// Where to send staked tokens when done validating
	StakeOuts []*lux.TransferableOutput `serialize:"true" json:"stake"`
	// Where to send staking rewards when done validating
	DelegationRewardsOwner fx.Owner `serialize:"true" json:"rewardsOwner"`
}

// InitCtx sets the FxID fields in the inputs and outputs of this
// [AddPermissionlessDelegatorTx]. Also sets the [ctx] to the given [vm.ctx] so
// that the addresses can be json marshalled into human readable format
func (tx *AddPermissionlessDelegatorTx) InitCtx(ctx *consensusctx.Context) {
	tx.BaseTx.InitCtx(ctx)
	for _, out := range tx.StakeOuts {
		out.FxID = secp256k1fx.ID
		out.InitCtx(ctx)
	}
	// Owner doesn't have InitCtx method
}

func (tx *AddPermissionlessDelegatorTx) ChainID() ids.ID {
	return tx.Chain
}

func (tx *AddPermissionlessDelegatorTx) NodeID() ids.NodeID {
	return tx.Validator.NodeID
}

func (*AddPermissionlessDelegatorTx) PublicKey() (*bls.PublicKey, bool, error) {
	return nil, false, nil
}

func (tx *AddPermissionlessDelegatorTx) PendingPriority() Priority {
	if tx.Chain == constants.PrimaryNetworkID {
		return PrimaryNetworkDelegatorBanffPendingPriority
	}
	return NetPermissionlessDelegatorPendingPriority
}

func (tx *AddPermissionlessDelegatorTx) CurrentPriority() Priority {
	if tx.Chain == constants.PrimaryNetworkID {
		return PrimaryNetworkDelegatorCurrentPriority
	}
	return NetPermissionlessDelegatorCurrentPriority
}

func (tx *AddPermissionlessDelegatorTx) Stake() []*lux.TransferableOutput {
	return tx.StakeOuts
}

func (tx *AddPermissionlessDelegatorTx) RewardsOwner() fx.Owner {
	return tx.DelegationRewardsOwner
}

// SyntacticVerify returns nil iff [tx] is valid
func (tx *AddPermissionlessDelegatorTx) SyntacticVerify(ctx *consensusctx.Context) error {
	switch {
	case tx == nil:
		return ErrNilTx
	case tx.SyntacticallyVerified: // already passed syntactic verification
		return nil
	case len(tx.StakeOuts) == 0: // Ensure there is provided stake
		return errNoStake
	}

	if err := tx.BaseTx.SyntacticVerify(ctx); err != nil {
		return fmt.Errorf("failed to verify BaseTx: %w", err)
	}
	if err := verify.All(&tx.Validator, tx.DelegationRewardsOwner); err != nil {
		return fmt.Errorf("failed to verify validator or rewards owner: %w", err)
	}

	for _, out := range tx.StakeOuts {
		if err := out.Verify(); err != nil {
			return fmt.Errorf("failed to verify output: %w", err)
		}
	}

	firstStakeOutput := tx.StakeOuts[0]
	stakedAssetID := firstStakeOutput.AssetID()
	totalStakeWeight := firstStakeOutput.Output().Amount()
	for _, out := range tx.StakeOuts[1:] {
		newWeight, err := safemath.Add64(totalStakeWeight, out.Output().Amount())
		if err != nil {
			return err
		}
		totalStakeWeight = newWeight

		assetID := out.AssetID()
		if assetID != stakedAssetID {
			return fmt.Errorf("%w: %q and %q", errMultipleStakedAssets, stakedAssetID, assetID)
		}
	}

	switch {
	case !lux.IsSortedTransferableOutputs(tx.StakeOuts, Codec):
		return errOutputsNotSorted
	case totalStakeWeight != tx.Wght:
		return fmt.Errorf("%w, delegator weight %d total stake weight %d",
			errDelegatorWeightMismatch,
			tx.Wght,
			totalStakeWeight,
		)
	}

	// cache that this is valid
	tx.SyntacticallyVerified = true
	return nil
}

func (tx *AddPermissionlessDelegatorTx) Visit(visitor Visitor) error {
	return visitor.AddPermissionlessDelegatorTx(tx)
}

// InitializeWithContext initializes the transaction with consensus context
func (tx *AddPermissionlessDelegatorTx) InitializeWithContext(ctx context.Context) error {
	// Initialize any context-dependent fields here
	return nil
}
