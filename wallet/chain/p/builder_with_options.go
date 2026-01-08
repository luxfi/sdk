// Copyright (C) 2019-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package p

import (
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/platformvm/signer"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/wallet/primary/common"
)

var _ Builder = (*builderWithOptions)(nil)

type builderWithOptions struct {
	Builder
	options []common.Option
}

// NewBuilderWithOptions returns a new transaction builder that will use the
// given options by default.
//
//   - [builder] is the builder that will be called to perform the underlying
//     operations.
//   - [options] will be provided to the builder in addition to the options
//     provided in the method calls.
func NewBuilderWithOptions(builder Builder, options ...common.Option) Builder {
	return &builderWithOptions{
		Builder: builder,
		options: options,
	}
}

func (b *builderWithOptions) GetBalance(
	options ...common.Option,
) (map[ids.ID]uint64, error) {
	return b.Builder.GetBalance(
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) GetImportableBalance(
	chainID ids.ID,
	options ...common.Option,
) (map[ids.ID]uint64, error) {
	return b.Builder.GetImportableBalance(
		chainID,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewAddValidatorTx(
	vdr *txs.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	shares uint32,
	options ...common.Option,
) (*txs.AddValidatorTx, error) {
	return b.Builder.NewAddValidatorTx(
		vdr,
		rewardsOwner,
		shares,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewAddChainValidatorTx(
	vdr *txs.ChainValidator,
	options ...common.Option,
) (*txs.AddChainValidatorTx, error) {
	return b.Builder.NewAddChainValidatorTx(
		vdr,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) RemoveChainValidatorTx(
	nodeID ids.NodeID,
	netID ids.ID,
	options ...common.Option,
) (*txs.RemoveChainValidatorTx, error) {
	return b.Builder.NewRemoveChainValidatorTx(
		nodeID,
		netID,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewAddDelegatorTx(
	vdr *txs.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.AddDelegatorTx, error) {
	return b.Builder.NewAddDelegatorTx(
		vdr,
		rewardsOwner,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewCreateChainTx(
	netID ids.ID,
	genesis []byte,
	vmID ids.ID,
	fxIDs []ids.ID,
	chainName string,
	options ...common.Option,
) (*txs.CreateChainTx, error) {
	return b.Builder.NewCreateChainTx(
		netID,
		genesis,
		vmID,
		fxIDs,
		chainName,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewCreateSubnetTx(
	owner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.CreateSubnetTx, error) {
	return b.Builder.NewCreateSubnetTx(
		owner,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewImportTx(
	sourceChainID ids.ID,
	to *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.ImportTx, error) {
	return b.Builder.NewImportTx(
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
	return b.Builder.NewExportTx(
		chainID,
		outputs,
		common.UnionOptions(b.options, options)...,
	)
}

func (b *builderWithOptions) NewTransformChainTx(
	netID ids.ID,
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
	return b.Builder.NewTransformChainTx(
		netID,
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
	return b.Builder.NewAddPermissionlessValidatorTx(
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
	return b.Builder.NewAddPermissionlessDelegatorTx(
		vdr,
		assetID,
		rewardsOwner,
		common.UnionOptions(b.options, options)...,
	)
}
