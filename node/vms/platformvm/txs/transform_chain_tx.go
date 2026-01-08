// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"context"
	"errors"
	"fmt"

	consensusctx "github.com/luxfi/consensus/context"

	"github.com/luxfi/ids"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/node/vms/components/verify"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
)

var (
	_ UnsignedTx = (*TransformChainTx)(nil)

	errCantTransformPrimaryNetwork       = errors.New("cannot transform primary network")
	errEmptyAssetID                      = errors.New("empty asset ID is not valid")
	errAssetIDCantBeLUX                  = errors.New("asset ID can't be LUX")
	errInitialSupplyZero                 = errors.New("initial supply must be non-0")
	errInitialSupplyGreaterThanMaxSupply = errors.New("initial supply can't be greater than maximum supply")
	errMinConsumptionRateTooLarge        = errors.New("min consumption rate must be less than or equal to max consumption rate")
	errMaxConsumptionRateTooLarge        = fmt.Errorf("max consumption rate must be less than or equal to %d", reward.PercentDenominator)
	errMinValidatorStakeZero             = errors.New("min validator stake must be non-0")
	errMinValidatorStakeAboveSupply      = errors.New("min validator stake must be less than or equal to initial supply")
	errMinValidatorStakeAboveMax         = errors.New("min validator stake must be less than or equal to max validator stake")
	errMaxValidatorStakeTooLarge         = errors.New("max validator stake must be less than or equal to max supply")
	errMinStakeDurationZero              = errors.New("min stake duration must be non-0")
	errMinStakeDurationTooLarge          = errors.New("min stake duration must be less than or equal to max stake duration")
	errMinDelegationFeeTooLarge          = fmt.Errorf("min delegation fee must be less than or equal to %d", reward.PercentDenominator)
	errMinDelegatorStakeZero             = errors.New("min delegator stake must be non-0")
	errMaxValidatorWeightFactorZero      = errors.New("max validator weight factor must be non-0")
	errUptimeRequirementTooLarge         = fmt.Errorf("uptime requirement must be less than or equal to %d", reward.PercentDenominator)
)

// TransformChainTx is an unsigned transformChainTx
type TransformChainTx struct {
	// Metadata, inputs and outputs
	BaseTx `serialize:"true"`
	// ID of the Chain to transform
	// Restrictions:
	// - Must not be the Primary Network ID
	Chain ids.ID `serialize:"true" json:"chainID"`
	// Asset to use when staking on the Net
	// Restrictions:
	// - Must not be the Empty ID
	// - Must not be the LUX ID
	AssetID ids.ID `serialize:"true" json:"assetID"`
	// Amount to initially specify as the current supply
	// Restrictions:
	// - Must be > 0
	InitialSupply uint64 `serialize:"true" json:"initialSupply"`
	// Amount to specify as the maximum token supply
	// Restrictions:
	// - Must be >= [InitialSupply]
	MaximumSupply uint64 `serialize:"true" json:"maximumSupply"`
	// MinConsumptionRate is the rate to allocate funds if the validator's stake
	// duration is 0
	MinConsumptionRate uint64 `serialize:"true" json:"minConsumptionRate"`
	// MaxConsumptionRate is the rate to allocate funds if the validator's stake
	// duration is equal to the minting period
	// Restrictions:
	// - Must be >= [MinConsumptionRate]
	// - Must be <= [reward.PercentDenominator]
	MaxConsumptionRate uint64 `serialize:"true" json:"maxConsumptionRate"`
	// MinValidatorStake is the minimum amount of funds required to become a
	// validator.
	// Restrictions:
	// - Must be > 0
	// - Must be <= [InitialSupply]
	MinValidatorStake uint64 `serialize:"true" json:"minValidatorStake"`
	// MaxValidatorStake is the maximum amount of funds a single validator can
	// be allocated, including delegated funds.
	// Restrictions:
	// - Must be >= [MinValidatorStake]
	// - Must be <= [MaximumSupply]
	MaxValidatorStake uint64 `serialize:"true" json:"maxValidatorStake"`
	// MinStakeDuration is the minimum number of seconds a staker can stake for.
	// Restrictions:
	// - Must be > 0
	MinStakeDuration uint32 `serialize:"true" json:"minStakeDuration"`
	// MaxStakeDuration is the maximum number of seconds a staker can stake for.
	// Restrictions:
	// - Must be >= [MinStakeDuration]
	// - Must be <= [GlobalMaxStakeDuration]
	MaxStakeDuration uint32 `serialize:"true" json:"maxStakeDuration"`
	// MinDelegationFee is the minimum percentage a validator must charge a
	// delegator for delegating.
	// Restrictions:
	// - Must be <= [reward.PercentDenominator]
	MinDelegationFee uint32 `serialize:"true" json:"minDelegationFee"`
	// MinDelegatorStake is the minimum amount of funds required to become a
	// delegator.
	// Restrictions:
	// - Must be > 0
	MinDelegatorStake uint64 `serialize:"true" json:"minDelegatorStake"`
	// MaxValidatorWeightFactor is the factor which calculates the maximum
	// amount of delegation a validator can receive.
	// Note: a value of 1 effectively disables delegation.
	// Restrictions:
	// - Must be > 0
	MaxValidatorWeightFactor byte `serialize:"true" json:"maxValidatorWeightFactor"`
	// UptimeRequirement is the minimum percentage a validator must be online
	// and responsive to receive a reward.
	// Restrictions:
	// - Must be <= [reward.PercentDenominator]
	UptimeRequirement uint32 `serialize:"true" json:"uptimeRequirement"`
	// Authorizes this transformation
	ChainAuth verify.Verifiable `serialize:"true" json:"chainAuthorization"`
}

func (tx *TransformChainTx) SyntacticVerify(ctx *consensusctx.Context) error {
	switch {
	case tx == nil:
		return ErrNilTx
	case tx.SyntacticallyVerified: // already passed syntactic verification
		return nil
	case tx.Chain == constants.PrimaryNetworkID:
		return errCantTransformPrimaryNetwork
	case tx.AssetID == ids.Empty:
		return errEmptyAssetID
	case tx.AssetID == ctx.XAssetID:
		return errAssetIDCantBeLUX
	case tx.InitialSupply == 0:
		return errInitialSupplyZero
	case tx.InitialSupply > tx.MaximumSupply:
		return errInitialSupplyGreaterThanMaxSupply
	case tx.MinConsumptionRate > tx.MaxConsumptionRate:
		return errMinConsumptionRateTooLarge
	case tx.MaxConsumptionRate > reward.PercentDenominator:
		return errMaxConsumptionRateTooLarge
	case tx.MinValidatorStake == 0:
		return errMinValidatorStakeZero
	case tx.MinValidatorStake > tx.InitialSupply:
		return errMinValidatorStakeAboveSupply
	case tx.MinValidatorStake > tx.MaxValidatorStake:
		return errMinValidatorStakeAboveMax
	case tx.MaxValidatorStake > tx.MaximumSupply:
		return errMaxValidatorStakeTooLarge
	case tx.MinStakeDuration == 0:
		return errMinStakeDurationZero
	case tx.MinStakeDuration > tx.MaxStakeDuration:
		return errMinStakeDurationTooLarge
	case tx.MinDelegationFee > reward.PercentDenominator:
		return errMinDelegationFeeTooLarge
	case tx.MinDelegatorStake == 0:
		return errMinDelegatorStakeZero
	case tx.MaxValidatorWeightFactor == 0:
		return errMaxValidatorWeightFactorZero
	case tx.UptimeRequirement > reward.PercentDenominator:
		return errUptimeRequirementTooLarge
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

func (tx *TransformChainTx) Visit(visitor Visitor) error {
	return visitor.TransformChainTx(tx)
}

// InitializeWithContext initializes the transaction with consensus context
func (tx *TransformChainTx) InitializeWithContext(ctx context.Context) error {
	// Initialize any context-dependent fields here
	return nil
}
