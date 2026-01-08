// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"context"
	"errors"
	"fmt"

	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/ids"
	safemath "github.com/luxfi/math"
	"github.com/luxfi/const"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/components/verify"
	"github.com/luxfi/sdk/node/vms/platformvm/fx"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
	"github.com/luxfi/sdk/node/vms/platformvm/signer"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

var (
	_ ValidatorTx     = (*AddPermissionlessValidatorTx)(nil)
	_ ScheduledStaker = (*AddPermissionlessDelegatorTx)(nil)

	errEmptyNodeID             = errors.New("validator nodeID cannot be empty")
	errNoStake                 = errors.New("no stake")
	errInvalidSigner           = errors.New("invalid signer")
	errMultipleStakedAssets    = errors.New("multiple staked assets")
	errValidatorWeightMismatch = errors.New("validator weight mismatch")
)

// AddPermissionlessValidatorTx is an unsigned addPermissionlessValidatorTx
type AddPermissionlessValidatorTx struct {
	// Metadata, inputs and outputs
	BaseTx `serialize:"true"`
	// Describes the validator
	Validator `serialize:"true" json:"validator"`
	// ID of the chain this validator is validating
	Chain ids.ID `serialize:"true" json:"chainID"`
	// If the [Chain] is the primary network, [Signer] is the BLS key for this
	// validator. If the [Chain] is not the primary network, this value is the
	// empty signer
	// Note: We do not enforce that the BLS key is unique across all validators.
	//       This means that validators can share a key if they so choose.
	//       However, a NodeID does uniquely map to a BLS key
	Signer signer.Signer `serialize:"true" json:"signer"`
	// Where to send staked tokens when done validating
	StakeOuts []*lux.TransferableOutput `serialize:"true" json:"stake"`
	// Where to send validation rewards when done validating
	ValidatorRewardsOwner fx.Owner `serialize:"true" json:"validationRewardsOwner"`
	// Where to send delegation rewards when done validating
	DelegatorRewardsOwner fx.Owner `serialize:"true" json:"delegationRewardsOwner"`
	// Fee this validator charges delegators as a percentage, times 10,000
	// For example, if this validator has DelegationShares=300,000 then they
	// take 30% of rewards from delegators
	DelegationShares uint32 `serialize:"true" json:"shares"`
}

// InitCtx sets the FxID fields in the inputs and outputs of this
// [AddPermissionlessValidatorTx]. Also sets the [ctx] to the given [vm.ctx] so
// that the addresses can be json marshalled into human readable format
func (tx *AddPermissionlessValidatorTx) InitCtx(ctx *consensusctx.Context) {
	tx.BaseTx.InitCtx(ctx)
	for _, out := range tx.StakeOuts {
		out.FxID = secp256k1fx.ID
		out.InitCtx(ctx)
	}
	// Owner doesn't have InitCtx method
	// tx.ValidatorRewardsOwner.InitCtx(ctx)
	// tx.DelegatorRewardsOwner.InitCtx(ctx)
}

func (tx *AddPermissionlessValidatorTx) ChainID() ids.ID {
	return tx.Chain
}

func (tx *AddPermissionlessValidatorTx) NodeID() ids.NodeID {
	return tx.Validator.NodeID
}

func (tx *AddPermissionlessValidatorTx) PublicKey() (*bls.PublicKey, bool, error) {
	if err := tx.Signer.Verify(); err != nil {
		return nil, false, err
	}
	key := tx.Signer.Key()
	return key, key != nil, nil
}

func (tx *AddPermissionlessValidatorTx) PendingPriority() Priority {
	if tx.Chain == constants.PrimaryNetworkID {
		return PrimaryNetworkValidatorPendingPriority
	}
	return NetPermissionlessValidatorPendingPriority
}

func (tx *AddPermissionlessValidatorTx) CurrentPriority() Priority {
	if tx.Chain == constants.PrimaryNetworkID {
		return PrimaryNetworkValidatorCurrentPriority
	}
	return NetPermissionlessValidatorCurrentPriority
}

func (tx *AddPermissionlessValidatorTx) Stake() []*lux.TransferableOutput {
	return tx.StakeOuts
}

func (tx *AddPermissionlessValidatorTx) ValidationRewardsOwner() fx.Owner {
	return tx.ValidatorRewardsOwner
}

func (tx *AddPermissionlessValidatorTx) DelegationRewardsOwner() fx.Owner {
	return tx.DelegatorRewardsOwner
}

func (tx *AddPermissionlessValidatorTx) Shares() uint32 {
	return tx.DelegationShares
}

// SyntacticVerify returns nil iff [tx] is valid
func (tx *AddPermissionlessValidatorTx) SyntacticVerify(ctx *consensusctx.Context) error {
	switch {
	case tx == nil:
		return ErrNilTx
	case tx.SyntacticallyVerified: // already passed syntactic verification
		return nil
	case tx.Validator.NodeID == ids.EmptyNodeID:
		return errEmptyNodeID
	case len(tx.StakeOuts) == 0: // Ensure there is provided stake
		return errNoStake
	case tx.DelegationShares > reward.PercentDenominator:
		return errTooManyShares
	}

	if err := tx.BaseTx.SyntacticVerify(ctx); err != nil {
		return fmt.Errorf("failed to verify BaseTx: %w", err)
	}
	if err := verify.All(&tx.Validator, tx.Signer, tx.ValidatorRewardsOwner, tx.DelegatorRewardsOwner); err != nil {
		return fmt.Errorf("failed to verify validator, signer, or rewards owners: %w", err)
	}

	hasKey := tx.Signer.Key() != nil
	isPrimaryNetwork := tx.Chain == constants.PrimaryNetworkID
	if hasKey != isPrimaryNetwork {
		return fmt.Errorf(
			"%w: hasKey=%v != isPrimaryNetwork=%v",
			errInvalidSigner,
			hasKey,
			isPrimaryNetwork,
		)
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
		return fmt.Errorf("%w: weight %d != stake %d", errValidatorWeightMismatch, tx.Wght, totalStakeWeight)
	}

	// cache that this is valid
	tx.SyntacticallyVerified = true
	return nil
}

func (tx *AddPermissionlessValidatorTx) Visit(visitor Visitor) error {
	return visitor.AddPermissionlessValidatorTx(tx)
}

// InitializeWithContext initializes the transaction with consensus context
func (tx *AddPermissionlessValidatorTx) InitializeWithContext(ctx context.Context) error {
	// Initialize any context-dependent fields here
	return nil
}
