// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package wallet provides P-Chain wallet implementation.
package wallet

import (
	"time"

	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
	"github.com/luxfi/proto/p/txs"
	"github.com/luxfi/sdk/wallet/chain/p/builder"
	"github.com/luxfi/sdk/wallet/primary/common"
	lux "github.com/luxfi/utxo"
	"github.com/luxfi/utxo/secp256k1fx"

	vmsigner "github.com/luxfi/proto/p/signer"
	walletsigner "github.com/luxfi/sdk/wallet/chain/p/signer"
)

var _ Wallet = (*wallet)(nil)

type Client interface {
	// IssueTx issues the signed tx.
	IssueTx(
		tx *txs.Tx,
		options ...common.Option,
	) error
}

type Wallet interface {
	Client

	// Builder returns the builder that will be used to create the transactions.
	Builder() builder.Builder

	// Signer returns the signer that will be used to sign the transactions.
	Signer() walletsigner.Signer

	// IssueBaseTx creates, signs, and issues a new simple value transfer.
	//
	// - [outputs] specifies all the recipients and amounts that should be sent
	//   from this transaction.
	IssueBaseTx(
		outputs []*lux.TransferableOutput,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueAddValidatorTx creates, signs, and issues a new validator of the
	// primary network the wallet is currently connected to.
	//
	// This is the universal "add a validator to a network" transaction.
	// Validators validate networks (not chains); chains live on networks.
	// The target network is determined by the wallet's connection context —
	// the wallet's RPC endpoint dictates which primary network's P-chain
	// receives the tx:
	//
	//   - the Lux primary network (networkID 1/2/3/1337), OR
	//   - a sovereign L1's own primary network (its own dedicated networkID).
	//
	// To target a different network, dial that network first then call this.
	//
	// - [vdr] specifies all the details of the validation period such as the
	//   startTime, endTime, stake weight, and nodeID.
	// - [rewardsOwner] specifies the owner of all the rewards this validator
	//   may accrue during its validation period.
	// - [shares] specifies the fraction (out of 1,000,000) that this validator
	//   will take from delegation rewards. If 1,000,000 is provided, 100% of
	//   the delegation reward will be sent to the validator's [rewardsOwner].
	IssueAddValidatorTx(
		vdr *txs.Validator,
		rewardsOwner *secp256k1fx.OutputOwners,
		shares uint32,
		options ...common.Option,
	) (*txs.Tx, error)

	// AddValidator is the canonical one-call convenience helper to add a
	// validator to the network the wallet is currently connected to.
	//
	// It is exactly equivalent to IssueAddValidatorTx — same parameters,
	// same return — and exists to make the decomplected model explicit at
	// the API surface: from a caller's perspective, there is ONE way to
	// add a validator (this method), and the target network is determined
	// by the wallet's connection context. Whether the caller is wiring up
	// the Lux primary network or a downstream sovereign L1's own primary
	// network, the call is the same — just dial the right network first.
	//
	// To target a different network, dial that network first then call this.
	AddValidator(
		vdr *txs.Validator,
		rewardsOwner *secp256k1fx.OutputOwners,
		delegationShares uint32,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueAddChainValidatorTx creates, signs, and issues a new validator of a
	// chain.
	//
	// - [vdr] specifies all the details of the validation period such as the
	//   startTime, endTime, sampling weight, nodeID, and chainID.
	//
	// Deprecated: validators validate networks, not chains. Chains live on
	// networks (created by CreateChainTx). Use IssueAddValidatorTx — that's
	// the universal "add a validator to a network" path, whether the target
	// is the Lux primary network or a sovereign L1's own primary network.
	// The wallet's connection context determines which network is targeted.
	// This method is retained for one release cycle of wire compatibility
	// only and will be removed.
	IssueAddChainValidatorTx(
		vdr *txs.ChainValidator,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueRemoveChainValidatorTx creates, signs, and issues a transaction
	// that removes a validator of a chain.
	//
	// - [nodeID] is the validator being removed from [chainID].
	//
	// Deprecated: retained for one release cycle of wire compatibility only.
	// New code should manage validator lifecycle through the network-level
	// validator txs (AddValidatorTx + the L1 validator txs for sovereign L1s).
	IssueRemoveChainValidatorTx(
		nodeID ids.NodeID,
		chainID ids.ID,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueAddDelegatorTx creates, signs, and issues a new delegator to a
	// validator on the primary network.
	//
	// - [vdr] specifies all the details of the delegation period such as the
	//   startTime, endTime, stake weight, and validator's nodeID.
	// - [rewardsOwner] specifies the owner of all the rewards this delegator
	//   may accrue at the end of its delegation period.
	IssueAddDelegatorTx(
		vdr *txs.Validator,
		rewardsOwner *secp256k1fx.OutputOwners,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueCreateChainTx creates, signs, and issues a new chain in the named
	// network.
	//
	// - [chainID] specifies the chain to launch the chain in.
	// - [genesis] specifies the initial state of the new chain.
	// - [vmID] specifies the vm that the new chain will run.
	// - [fxIDs] specifies all the feature extensions that the vm should be
	//   running with.
	// - [chainName] specifies a human readable name for the chain.
	IssueCreateChainTx(
		chainID ids.ID,
		genesis []byte,
		vmID ids.ID,
		fxIDs []ids.ID,
		chainName string,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueCreateNetworkTx creates, signs, and issues a new network with the
	// specified owner.
	//
	// - [owner] specifies who has the ability to create new chains and add new
	//   validators to the network.
	IssueCreateNetworkTx(
		owner *secp256k1fx.OutputOwners,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueTransferChainOwnershipTx creates, signs, and issues a transaction that
	// changes the owner of the named chain.
	//
	// - [chainID] specifies the chain to be modified
	// - [owner] specifies who has the ability to create new chains and add new
	//   validators to the chain.
	IssueTransferChainOwnershipTx(
		chainID ids.ID,
		owner *secp256k1fx.OutputOwners,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueConvertNetworkToL1Tx creates, signs, and issues a transaction that
	// converts the chain to a Permissionless L1.
	//
	// - [chainID] specifies the chain to be converted
	// - [managerChainID] specifies which chain the manager is deployed on
	// - [address] specifies the address of the manager
	// - [validators] specifies the initial L1 validators of the L1
	IssueConvertNetworkToL1Tx(
		chainID ids.ID,
		managerChainID ids.ID,
		address []byte,
		validators []*txs.ConvertNetworkToL1Validator,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueRegisterL1ValidatorTx creates, signs, and issues a transaction that
	// adds a validator to an L1.
	//
	// - [balance] that the validator should allocate to continuous fees
	// - [proofOfPossession] is the BLS PoP for the key included in the Warp
	//   message
	// - [message] is the Warp message that authorizes this validator to be
	//   added
	IssueRegisterL1ValidatorTx(
		balance uint64,
		proofOfPossession [bls.SignatureLen]byte,
		message []byte,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueSetL1ValidatorWeightTx creates, signs, and issues a transaction that
	// sets the weight of a validator on an L1.
	//
	// - [message] is the Warp message that authorizes this validator's weight
	//   to be changed
	IssueSetL1ValidatorWeightTx(
		message []byte,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueIncreaseL1ValidatorBalanceTx creates, signs, and issues a
	// transaction that increases the balance of a validator on an L1 for the
	// continuous fee.
	//
	// - [validationID] of the validator
	// - [balance] amount to increase the validator's balance by
	IssueIncreaseL1ValidatorBalanceTx(
		validationID ids.ID,
		balance uint64,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueDisableL1ValidatorTx creates, signs, and issues a transaction that
	// disables an L1 validator and returns the remaining funds allocated to the
	// continuous fee to the remaining balance owner.
	//
	// - [validationID] of the validator to disable
	IssueDisableL1ValidatorTx(
		validationID ids.ID,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueImportTx creates, signs, and issues an import transaction that
	// attempts to consume all the available UTXOs and import the funds to [to].
	//
	// - [chainID] specifies the chain to be importing funds from.
	// - [to] specifies where to send the imported funds to.
	IssueImportTx(
		chainID ids.ID,
		to *secp256k1fx.OutputOwners,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueExportTx creates, signs, and issues an export transaction that
	// attempts to send all the provided [outputs] to the requested [chainID].
	//
	// - [chainID] specifies the chain to be exporting the funds to.
	// - [outputs] specifies the outputs to send to the [chainID].
	IssueExportTx(
		chainID ids.ID,
		outputs []*lux.TransferableOutput,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueTransformChainTx creates a transform chain transaction that attempts
	// to convert the provided [chainID] from a permissioned chain to a
	// permissionless chain. This transaction will convert
	// [maxSupply] - [initialSupply] of [assetID] to staking rewards.
	//
	// - [chainID] specifies the chain to transform.
	// - [assetID] specifies the asset to use to reward stakers on the chain.
	// - [initialSupply] is the amount of [assetID] that will be in circulation
	//   after this transaction is accepted.
	// - [maxSupply] is the maximum total amount of [assetID] that should ever
	//   exist.
	// - [minConsumptionRate] is the rate that a staker will receive rewards
	//   if they stake with a duration of 0.
	// - [maxConsumptionRate] is the maximum rate that staking rewards should be
	//   consumed from the reward pool per year.
	// - [minValidatorStake] is the minimum amount of funds required to become a
	//   validator.
	// - [maxValidatorStake] is the maximum amount of funds a single validator
	//   can be allocated, including delegated funds.
	// - [minStakeDuration] is the minimum number of seconds a staker can stake
	//   for.
	// - [maxStakeDuration] is the maximum number of seconds a staker can stake
	//   for.
	// - [minValidatorStake] is the minimum amount of funds required to become a
	//   delegator.
	// - [maxValidatorWeightFactor] is the factor which calculates the maximum
	//   amount of delegation a validator can receive. A value of 1 effectively
	//   disables delegation.
	// - [uptimeRequirement] is the minimum percentage a validator must be
	//   online and responsive to receive a reward.
	IssueTransformChainTx(
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
	) (*txs.Tx, error)

	// IssueAddPermissionlessValidatorTx creates, signs, and issues a new
	// validator of the specified chain.
	//
	// - [vdr] specifies all the details of the validation period such as the
	//   chainID, startTime, endTime, stake weight, and nodeID.
	// - [signer] if the chainID is the primary network, this is the BLS key
	//   for this validator. Otherwise, this value should be the empty signer.
	// - [assetID] specifies the asset to stake.
	// - [validationRewardsOwner] specifies the owner of all the rewards this
	//   validator earns for its validation period.
	// - [delegationRewardsOwner] specifies the owner of all the rewards this
	//   validator earns for delegations during its validation period.
	// - [shares] specifies the fraction (out of 1,000,000) that this validator
	//   will take from delegation rewards. If 1,000,000 is provided, 100% of
	//   the delegation reward will be sent to the validator's [rewardsOwner].
	IssueAddPermissionlessValidatorTx(
		vdr *txs.ChainValidator,
		signer vmsigner.Signer,
		assetID ids.ID,
		validationRewardsOwner *secp256k1fx.OutputOwners,
		delegationRewardsOwner *secp256k1fx.OutputOwners,
		shares uint32,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueAddPermissionlessDelegatorTx creates, signs, and issues a new
	// delegator of the specified chain on the specified nodeID.
	//
	// - [vdr] specifies all the details of the delegation period such as the
	//   chainID, startTime, endTime, stake weight, and nodeID.
	// - [assetID] specifies the asset to stake.
	// - [rewardsOwner] specifies the owner of all the rewards this delegator
	//   earns during its delegation period.
	IssueAddPermissionlessDelegatorTx(
		vdr *txs.ChainValidator,
		assetID ids.ID,
		rewardsOwner *secp256k1fx.OutputOwners,
		options ...common.Option,
	) (*txs.Tx, error)

	// IssueUnsignedTx signs and issues the unsigned tx.
	IssueUnsignedTx(
		utx txs.UnsignedTx,
		options ...common.Option,
	) (*txs.Tx, error)
}

func New(
	client Client,
	builder builder.Builder,
	signer walletsigner.Signer,
) Wallet {
	return &wallet{
		Client:  client,
		builder: builder,
		signer:  signer,
	}
}

type wallet struct {
	Client
	builder builder.Builder
	signer  walletsigner.Signer
}

func (w *wallet) Builder() builder.Builder {
	return w.builder
}

func (w *wallet) Signer() walletsigner.Signer {
	return w.signer
}

func (w *wallet) IssueBaseTx(
	outputs []*lux.TransferableOutput,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewBaseTx(outputs, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueAddValidatorTx(
	vdr *txs.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	shares uint32,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewAddValidatorTx(vdr, rewardsOwner, shares, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

// AddValidator adds a validator to the network the wallet is currently
// connected to (Lux primary or a sovereign L1's own primary — same tx,
// just different target networkID determined by the wallet's RPC endpoint).
// To target a different network, dial that network first then call this.
//
// This is the recommended one-call helper for the common case where the
// caller already holds a wallet dialed to the right network — there is no
// need to thread chainIDs or auth proofs through. From a caller's
// perspective there is ONE add-validator entry point: this method.
func (w *wallet) AddValidator(
	vdr *txs.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	delegationShares uint32,
	options ...common.Option,
) (*txs.Tx, error) {
	return w.IssueAddValidatorTx(vdr, rewardsOwner, delegationShares, options...)
}

func (w *wallet) IssueAddChainValidatorTx(
	vdr *txs.ChainValidator,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewAddChainValidatorTx(vdr, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueRemoveChainValidatorTx(
	nodeID ids.NodeID,
	chainID ids.ID,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewRemoveChainValidatorTx(nodeID, chainID, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueAddDelegatorTx(
	vdr *txs.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewAddDelegatorTx(vdr, rewardsOwner, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueCreateChainTx(
	chainID ids.ID,
	genesis []byte,
	vmID ids.ID,
	fxIDs []ids.ID,
	chainName string,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewCreateChainTx(chainID, genesis, vmID, fxIDs, chainName, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueCreateNetworkTx(
	owner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewCreateNetworkTx(owner, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueTransferChainOwnershipTx(
	chainID ids.ID,
	owner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewTransferChainOwnershipTx(chainID, owner, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueConvertNetworkToL1Tx(
	chainID ids.ID,
	managerChainID ids.ID,
	address []byte,
	validators []*txs.ConvertNetworkToL1Validator,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewConvertNetworkToL1Tx(chainID, managerChainID, address, validators, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueRegisterL1ValidatorTx(
	balance uint64,
	proofOfPossession [bls.SignatureLen]byte,
	message []byte,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewRegisterL1ValidatorTx(balance, proofOfPossession, message, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueSetL1ValidatorWeightTx(
	message []byte,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewSetL1ValidatorWeightTx(message, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueIncreaseL1ValidatorBalanceTx(
	validationID ids.ID,
	balance uint64,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewIncreaseL1ValidatorBalanceTx(validationID, balance, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueDisableL1ValidatorTx(
	validationID ids.ID,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewDisableL1ValidatorTx(validationID, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueImportTx(
	sourceChainID ids.ID,
	to *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewImportTx(sourceChainID, to, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueExportTx(
	chainID ids.ID,
	outputs []*lux.TransferableOutput,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewExportTx(chainID, outputs, options...)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

// Removed in regenesis
func (w *wallet) IssueTransformChainTx(
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
) (*txs.Tx, error) {
	utx, err := w.builder.NewTransformChainTx(
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
		options...,
	)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueAddPermissionlessValidatorTx(
	vdr *txs.ChainValidator,
	signer vmsigner.Signer,
	assetID ids.ID,
	validationRewardsOwner *secp256k1fx.OutputOwners,
	delegationRewardsOwner *secp256k1fx.OutputOwners,
	shares uint32,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewAddPermissionlessValidatorTx(
		vdr,
		signer,
		assetID,
		validationRewardsOwner,
		delegationRewardsOwner,
		shares,
		options...,
	)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueAddPermissionlessDelegatorTx(
	vdr *txs.ChainValidator,
	assetID ids.ID,
	rewardsOwner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.Tx, error) {
	utx, err := w.builder.NewAddPermissionlessDelegatorTx(
		vdr,
		assetID,
		rewardsOwner,
		options...,
	)
	if err != nil {
		return nil, err
	}
	return w.IssueUnsignedTx(utx, options...)
}

func (w *wallet) IssueUnsignedTx(
	utx txs.UnsignedTx,
	options ...common.Option,
) (*txs.Tx, error) {
	ops := common.NewOptions(options)
	ctx := ops.Context()
	tx, err := walletsigner.SignUnsigned(ctx, w.signer, utx)
	if err != nil {
		return nil, err
	}

	return tx, w.IssueTx(tx, options...)
}
