// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"time"

	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
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
func (w *withOptions) NewAddChainValidatorTx(
	vdr *txs.ChainValidator,
	options ...common.Option,
) (*txs.AddChainValidatorTx, error) {
	return w.builder.NewAddChainValidatorTx(
		vdr,
		common.UnionOptions(w.options, options)...,
	)
}

// Removed in regenesis
func (w *withOptions) NewRemoveChainValidatorTx(
	nodeID ids.NodeID,
	chainID ids.ID,
	options ...common.Option,
) (*txs.RemoveChainValidatorTx, error) {
	return w.builder.NewRemoveChainValidatorTx(
		nodeID,
		chainID,
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
	chainID ids.ID,
	genesis []byte,
	vmID ids.ID,
	fxIDs []ids.ID,
	blockchainName string,
	options ...common.Option,
) (*txs.CreateChainTx, error) {
	return w.builder.NewCreateChainTx(
		chainID,
		genesis,
		vmID,
		fxIDs,
		blockchainName,
		common.UnionOptions(w.options, options)...,
	)
}

// Removed in regenesis
func (w *withOptions) NewCreateSubnetTx(
	owner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.CreateSubnetTx, error) {
	return w.builder.NewCreateSubnetTx(
		owner,
		common.UnionOptions(w.options, options)...,
	)
}

// Removed in regenesis
func (w *withOptions) NewTransferChainOwnershipTx(
	chainID ids.ID,
	owner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.TransferChainOwnershipTx, error) {
	return w.builder.NewTransferChainOwnershipTx(
		chainID,
		owner,
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) NewConvertChainToL1Tx(
	chainID ids.ID,
	managerChainID ids.ID,
	address []byte,
	validators []*txs.ConvertChainToL1Validator,
	options ...common.Option,
) (*txs.ConvertChainToL1Tx, error) {
	return w.builder.NewConvertChainToL1Tx(
		chainID,
		managerChainID,
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
func (w *withOptions) NewTransformChainTx(
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
	return w.builder.NewTransformChainTx(
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
		common.UnionOptions(w.options, options)...,
	)
}

func (w *withOptions) NewAddPermissionlessValidatorTx(
	vdr *txs.ChainValidator,
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
	vdr *txs.ChainValidator,
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
