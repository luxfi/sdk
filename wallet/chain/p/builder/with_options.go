// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/node/utils/crypto/bls"
	"github.com/luxfi/node/vms/components/lux"
	"github.com/luxfi/node/vms/platformvm/signer"
	"github.com/luxfi/node/vms/platformvm/txs"
	"github.com/luxfi/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/wallet/primary/common"
)

var _ Builder = (*withOptions)(nil)

type withOptions struct {
	builder Builder
	options []common.Option
}

// WithOptions returns a new builder that will use the given options by default.
//
//   - [builder] is the builder that will be called to perform the underlying
//     operations.
//   - [options] will be provided to the builder in addition to the options
//     provided in the method calls.
func WithOptions(builder Builder, options ...common.Option) Builder {
	return &withOptions{
		builder: builder,
		options: options,
	}
}

func (w *withOptions) Context() *Context {
	return w.builder.Context()
}

func (w *withOptions) GetBalance(
	options ...common.Option,
) (map[ids.ID]uint64, error) {
	return w.builder.GetBalance(
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) GetImportableBalance(
	chainID ids.ID,
	options ...common.Option,
) (map[ids.ID]uint64, error) {
	return w.builder.GetImportableBalance(
		chainID,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) NewBaseTx(
	outputs []*lux.TransferableOutput,
	options ...common.Option,
) (*txs.BaseTx, error) {
	return w.builder.NewBaseTx(
		outputs,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) NewAddValidatorTx(
	vdr *txs.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	shares uint32,
	options ...common.Option,
) (*txs.AddValidatorTx, error) {
	return w.builder.NewAddValidatorTx(
		vdr,
		rewardsOwner,
		shares,
		common.UnionOptions(w.options, options)...,
	)
}

// Removed in regenesis
func (w *withOptions) NewAddNetValidatorTx(
	vdr *txs.NetValidator,
	options ...common.Option,
) (*txs.AddNetValidatorTx, error) {
	return w.builder.NewAddNetValidatorTx(
		vdr,
		common.UnionOptions(w.options, options)...,
	)
}

// Removed in regenesis
func (w *withOptions) NewRemoveNetValidatorTx(
	nodeID ids.NodeID,
	subnetID ids.ID,
	options ...common.Option,
) (*txs.RemoveNetValidatorTx, error) {
	return w.builder.NewRemoveNetValidatorTx(
		nodeID,
		subnetID,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) NewAddDelegatorTx(
	vdr *txs.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.AddDelegatorTx, error) {
	return w.builder.NewAddDelegatorTx(
		vdr,
		rewardsOwner,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) NewCreateChainTx(
	subnetID ids.ID,
	genesis []byte,
	vmID ids.ID,
	fxIDs []ids.ID,
	chainName string,
	options ...common.Option,
) (*txs.CreateChainTx, error) {
	return w.builder.NewCreateChainTx(
		subnetID,
		genesis,
		vmID,
		fxIDs,
		chainName,
		common.UnionOptions(w.options, options)...,
	)
}

// Removed in regenesis
func (w *withOptions) NewCreateNetTx(
	owner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.CreateNetTx, error) {
	return w.builder.NewCreateNetTx(
		owner,
		common.UnionOptions(w.options, options)...,
	)
}

// Removed in regenesis
func (w *withOptions) NewTransferNetOwnershipTx(
	subnetID ids.ID,
	owner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.TransferNetOwnershipTx, error) {
	return w.builder.NewTransferNetOwnershipTx(
		subnetID,
		owner,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) NewConvertNetToL1Tx(
	subnetID ids.ID,
	chainID ids.ID,
	address []byte,
	validators []*txs.ConvertNetToL1Validator,
	options ...common.Option,
) (*txs.ConvertNetToL1Tx, error) {
	return w.builder.NewConvertNetToL1Tx(
		subnetID,
		chainID,
		address,
		validators,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) NewRegisterL1ValidatorTx(
	balance uint64,
	proofOfPossession [bls.SignatureLen]byte,
	message []byte,
	options ...common.Option,
) (*txs.RegisterL1ValidatorTx, error) {
	return w.builder.NewRegisterL1ValidatorTx(
		balance,
		proofOfPossession,
		message,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) NewSetL1ValidatorWeightTx(
	message []byte,
	options ...common.Option,
) (*txs.SetL1ValidatorWeightTx, error) {
	return w.builder.NewSetL1ValidatorWeightTx(
		message,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) NewIncreaseL1ValidatorBalanceTx(
	validationID ids.ID,
	balance uint64,
	options ...common.Option,
) (*txs.IncreaseL1ValidatorBalanceTx, error) {
	return w.builder.NewIncreaseL1ValidatorBalanceTx(
		validationID,
		balance,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) NewDisableL1ValidatorTx(
	validationID ids.ID,
	options ...common.Option,
) (*txs.DisableL1ValidatorTx, error) {
	return w.builder.NewDisableL1ValidatorTx(
		validationID,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) NewImportTx(
	sourceChainID ids.ID,
	to *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.ImportTx, error) {
	return w.builder.NewImportTx(
		sourceChainID,
		to,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) NewExportTx(
	chainID ids.ID,
	outputs []*lux.TransferableOutput,
	options ...common.Option,
) (*txs.ExportTx, error) {
	return w.builder.NewExportTx(
		chainID,
		outputs,
		common.UnionOptions(w.options, options)...,
	)
}

// Removed in regenesis
func (w *withOptions) NewTransformNetTx(
	subnetID ids.ID,
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
) (*txs.TransformNetTx, error) {
	return w.builder.NewTransformNetTx(
		subnetID,
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
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) NewAddPermissionlessValidatorTx(
	vdr *txs.NetValidator,
	signer signer.Signer,
	assetID ids.ID,
	validationRewardsOwner *secp256k1fx.OutputOwners,
	delegationRewardsOwner *secp256k1fx.OutputOwners,
	shares uint32,
	options ...common.Option,
) (*txs.AddPermissionlessValidatorTx, error) {
	return w.builder.NewAddPermissionlessValidatorTx(
		vdr,
		signer,
		assetID,
		validationRewardsOwner,
		delegationRewardsOwner,
		shares,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) NewAddPermissionlessDelegatorTx(
	vdr *txs.NetValidator,
	assetID ids.ID,
	rewardsOwner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.AddPermissionlessDelegatorTx, error) {
	return w.builder.NewAddPermissionlessDelegatorTx(
		vdr,
		assetID,
		rewardsOwner,
		common.UnionOptions(w.options, options)...,
	)
}
