// Copyright (C) 2019-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/wallet/primary/common"
	"github.com/luxfi/vm/components/lux"
	"github.com/luxfi/vm/vms/platformvm/signer"
	"github.com/luxfi/vm/vms/platformvm/txs"
	"github.com/luxfi/vm/vms/secp256k1fx"
)

var _ Builder = (*builderWithOptions)(nil)

type builderWithOptions struct {
	builder Builder
	options []common.Option
}

// NewWithOptions returns a new builder that will use the given options by
// default.
//
//   - [builder] is the builder that will be called to perform the underlying
//     operations.
//   - [options] will be provided to the builder in addition to the options
//     provided in the method calls.
func NewWithOptions(builder Builder, options ...common.Option) Builder {
	return &builderWithOptions{
		builder: builder,
		options: options,
	}
}

func (b *builderWithOptions) Context() *Context {
	return b.builder.Context()
}

func (b *builderWithOptions) GetBalance(
	options ...common.Option,
) (map[ids.ID]uint64, error) {
	return b.builder.GetBalance(
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) GetImportableBalance(
	chainID ids.ID,
	options ...common.Option,
) (map[ids.ID]uint64, error) {
	return b.builder.GetImportableBalance(
		chainID,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewBaseTx(
	outputs []*lux.TransferableOutput,
	options ...common.Option,
) (*txs.BaseTx, error) {
	return b.builder.NewBaseTx(
		outputs,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewAddValidatorTx(
	vdr *txs.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	shares uint32,
	options ...common.Option,
) (*txs.AddValidatorTx, error) {
	return b.builder.NewAddValidatorTx(
		vdr,
		rewardsOwner,
		shares,
		common.UnionOptions(b.options, options)...,
	)
}

// Removed in regenesis
func (b *builderWithOptions) NewAddChainValidatorTx(
	vdr *txs.ChainValidator,
	options ...common.Option,
) (*txs.AddChainValidatorTx, error) {
	return b.builder.NewAddChainValidatorTx(
		vdr,
		common.UnionOptions(b.options, options)...,
	)
}

// Removed in regenesis
func (b *builderWithOptions) NewRemoveChainValidatorTx(
	nodeID ids.NodeID,
	chainID ids.ID,
	options ...common.Option,
) (*txs.RemoveChainValidatorTx, error) {
	return b.builder.NewRemoveChainValidatorTx(
		nodeID,
		chainID,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewAddDelegatorTx(
	vdr *txs.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.AddDelegatorTx, error) {
	return b.builder.NewAddDelegatorTx(
		vdr,
		rewardsOwner,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewCreateChainTx(
	chainID ids.ID,
	genesis []byte,
	vmID ids.ID,
	fxIDs []ids.ID,
	blockchainName string,
	options ...common.Option,
) (*txs.CreateChainTx, error) {
	return b.builder.NewCreateChainTx(
		chainID,
		genesis,
		vmID,
		fxIDs,
		blockchainName,
		common.UnionOptions(b.options, options)...,
	)
}

// Removed in regenesis
func (b *builderWithOptions) NewCreateSubnetTx(
	owner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.CreateSubnetTx, error) {
	return b.builder.NewCreateSubnetTx(
		owner,
		common.UnionOptions(b.options, options)...,
	)
}

// Removed in regenesis
func (b *builderWithOptions) NewTransferChainOwnershipTx(
	chainID ids.ID,
	owner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.TransferChainOwnershipTx, error) {
	return b.builder.NewTransferChainOwnershipTx(
		chainID,
		owner,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewImportTx(
	sourceChainID ids.ID,
	to *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.ImportTx, error) {
	return b.builder.NewImportTx(
		sourceChainID,
		to,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewExportTx(
	chainID ids.ID,
	outputs []*lux.TransferableOutput,
	options ...common.Option,
) (*txs.ExportTx, error) {
	return b.builder.NewExportTx(
		chainID,
		outputs,
		common.UnionOptions(b.options, options)...,
	)
}

// Removed in regenesis
func (b *builderWithOptions) NewTransformChainTx(
	chainID ids.ID,
	assetID ids.ID,
	initialSupply uint64,
	maxSupply uint64,
	minConsumptionRate uint64,
	maxConsumptionRate uint64,
	minValidatorStake uint64,
	maxValidatorStake uint64,
	minStakeDuration time.Duration,
	maxStakeDuration time.Duration,
	minDelegationFee uint32,
	minDelegatorStake uint64,
	maxValidatorWeightFactor byte,
	uptimeRequirement uint32,
	options ...common.Option,
) (*txs.TransformChainTx, error) {
	return b.builder.NewTransformChainTx(
		chainID,
		assetID,
		initialSupply,
		maxSupply,
		minConsumptionRate,
		maxConsumptionRate,
		minValidatorStake,
		maxValidatorStake,
		minStakeDuration,
		maxStakeDuration,
		minDelegationFee,
		minDelegatorStake,
		maxValidatorWeightFactor,
		uptimeRequirement,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewAddPermissionlessValidatorTx(
	vdr *txs.ChainValidator,
	signer signer.Signer,
	assetID ids.ID,
	validationRewardsOwner *secp256k1fx.OutputOwners,
	delegationRewardsOwner *secp256k1fx.OutputOwners,
	shares uint32,
	options ...common.Option,
) (*txs.AddPermissionlessValidatorTx, error) {
	return b.builder.NewAddPermissionlessValidatorTx(
		vdr,
		signer,
		assetID,
		validationRewardsOwner,
		delegationRewardsOwner,
		shares,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewAddPermissionlessDelegatorTx(
	vdr *txs.ChainValidator,
	assetID ids.ID,
	rewardsOwner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.AddPermissionlessDelegatorTx, error) {
	return b.builder.NewAddPermissionlessDelegatorTx(
		vdr,
		assetID,
		rewardsOwner,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewConvertChainToL1Tx(
	chainID ids.ID,
	managerChainID ids.ID,
	address []byte,
	validators []*txs.ConvertChainToL1Validator,
	options ...common.Option,
) (*txs.ConvertChainToL1Tx, error) {
	return b.builder.NewConvertChainToL1Tx(
		chainID,
		managerChainID,
		address,
		validators,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewRegisterL1ValidatorTx(
	balance uint64,
	proofOfPossession [96]byte,
	message []byte,
	options ...common.Option,
) (*txs.RegisterL1ValidatorTx, error) {
	return b.builder.NewRegisterL1ValidatorTx(
		balance,
		proofOfPossession,
		message,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewSetL1ValidatorWeightTx(
	message []byte,
	options ...common.Option,
) (*txs.SetL1ValidatorWeightTx, error) {
	return b.builder.NewSetL1ValidatorWeightTx(
		message,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewIncreaseL1ValidatorBalanceTx(
	validationID ids.ID,
	balance uint64,
	options ...common.Option,
) (*txs.IncreaseL1ValidatorBalanceTx, error) {
	return b.builder.NewIncreaseL1ValidatorBalanceTx(
		validationID,
		balance,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewDisableL1ValidatorTx(
	validationID ids.ID,
	options ...common.Option,
) (*txs.DisableL1ValidatorTx, error) {
	return b.builder.NewDisableL1ValidatorTx(
		validationID,
		common.UnionOptions(b.options, options)...,
	)
}
