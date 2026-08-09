// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package builder provides P-Chain transaction builder implementation.
package builder

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luxfi/constants"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
	"github.com/luxfi/math"
	"github.com/luxfi/math/set"
	"github.com/luxfi/proto/p/fx"
	"github.com/luxfi/proto/p/signer"
	"github.com/luxfi/proto/p/stakeable"
	"github.com/luxfi/proto/p/security"
	"github.com/luxfi/proto/p/txs"
	"github.com/luxfi/proto/p/txs/fee"
	"github.com/luxfi/sdk/wallet/primary/common"
	"github.com/luxfi/utils"
	lux "github.com/luxfi/utxo"
	"github.com/luxfi/utxo/secp256k1fx"
	"github.com/luxfi/vm/components/gas"
	"github.com/luxfi/vm/components/verify"
)

var (
	ErrNoChangeAddress           = errors.New("no possible change address")
	ErrUnknownOutputType         = errors.New("unknown output type")
	ErrUnknownOwnerType          = errors.New("unknown owner type")
	ErrInsufficientAuthorization = errors.New("insufficient authorization")
	ErrInsufficientFunds         = errors.New("insufficient funds")

	_ Builder = (*builder)(nil)
)

// Builder provides a convenient interface for building unsigned P-chain
// transactions.
type Builder interface {
	// Context returns the configuration of the chain that this builder uses to
	// create transactions.
	Context() *Context

	// GetBalance calculates the amount of each asset that this builder has
	// control over.
	GetBalance(
		options ...common.Option,
	) (map[ids.ID]uint64, error)

	// GetImportableBalance calculates the amount of each asset that this
	// builder could import from the provided chain.
	//
	// - [chainID] specifies the chain the funds are from.
	GetImportableBalance(
		chainID ids.ID,
		options ...common.Option,
	) (map[ids.ID]uint64, error)

	// NewBaseTx creates a new simple value transfer.
	//
	// - [outputs] specifies all the recipients and amounts that should be sent
	//   from this transaction.
	NewBaseTx(
		outputs []*lux.TransferableOutput,
		options ...common.Option,
	) (*txs.BaseTx, error)

	// NewAddValidatorTx creates a new validator of the primary network the
	// builder is configured against.
	//
	// This is the universal "add a validator to a network" transaction —
	// validators validate networks (not chains); chains live on networks.
	// The target network is whichever primary network the builder's context
	// is bound to (the Lux primary network, networkID 1/2/3/1337; or a
	// sovereign L1's own primary network with its own dedicated networkID).
	//
	// Stake outputs are synthesized internally from the validator weight;
	// the caller does not pass StakeOuts.
	//
	// - [vdr] specifies all the details of the validation period such as the
	//   startTime, endTime, stake weight, and nodeID.
	// - [rewardsOwner] specifies the owner of all the rewards this validator
	//   may accrue during its validation period.
	// - [shares] specifies the fraction (out of 1,000,000) that this validator
	//   will take from delegation rewards. If 1,000,000 is provided, 100% of
	//   the delegation reward will be sent to the validator's [rewardsOwner].
	NewAddValidatorTx(
		vdr *txs.Validator,
		rewardsOwner *secp256k1fx.OutputOwners,
		shares uint32,
		options ...common.Option,
	) (*txs.AddValidatorTx, error)

	// NewAddChainValidatorTx creates a new validator of a chain.
	//
	// - [vdr] specifies all the details of the validation period such as the
	//   startTime, endTime, sampling weight, nodeID, and chainID.
	//
	// Deprecated: validators validate networks, not chains. Chains live on
	// networks (created by CreateChainTx). Use NewAddValidatorTx — that's
	// the universal "add a validator to a network" path, whether the target
	// is the Lux primary network or a sovereign L1's own primary network.
	// The wallet/builder's connection context determines which network is
	// targeted. Retained for one release cycle of wire compatibility only.
	NewAddChainValidatorTx(
		vdr *txs.ChainValidator,
		options ...common.Option,
	) (*txs.AddChainValidatorTx, error)

	// NewRemoveChainValidatorTx removes [nodeID] from the validator
	// set [chainID].
	NewRemoveChainValidatorTx(
		nodeID ids.NodeID,
		chainID ids.ID,
		options ...common.Option,
	) (*txs.RemoveChainValidatorTx, error)

	// NewAddDelegatorTx creates a new delegator to a validator on the primary
	// network.
	//
	// - [vdr] specifies all the details of the delegation period such as the
	//   startTime, endTime, stake weight, and validator's nodeID.
	// - [rewardsOwner] specifies the owner of all the rewards this delegator
	//   may accrue at the end of its delegation period.
	NewAddDelegatorTx(
		vdr *txs.Validator,
		rewardsOwner *secp256k1fx.OutputOwners,
		options ...common.Option,
	) (*txs.AddDelegatorTx, error)

	// NewCreateChainTx creates a new blockchain in the named chain.
	//
	// - [chainID] specifies the chain to launch the blockchain in.
	// - [genesis] specifies the initial state of the new blockchain.
	// - [vmID] specifies the vm that the new blockchain will run.
	// - [fxIDs] specifies all the feature extensions that the vm should be
	//   running with.
	// - [blockchainName] specifies a human readable name for the blockchain.
	NewCreateChainTx(
		chainID ids.ID,
		genesis []byte,
		vmID ids.ID,
		fxIDs []ids.ID,
		blockchainName string,
		options ...common.Option,
	) (*txs.CreateChainTx, error)

	// NewCreateNetworkTx creates a new network with the specified owner.
	//
	// - [owner] specifies who has the ability to create new chains and add new
	//   validators to the network.
	NewCreateNetworkTx(
		owner *secp256k1fx.OutputOwners,
		options ...common.Option,
	) (*txs.CreateNetworkTx, error)

	// NewTransferChainOwnershipTx changes the owner of the named chain.
	//
	// - [chainID] specifies the chain to be modified
	// - [owner] specifies who has the ability to create new blockchains and add new
	//   validators to the chain.
	NewTransferChainOwnershipTx(
		chainID ids.ID,
		owner *secp256k1fx.OutputOwners,
		options ...common.Option,
	) (*txs.TransferChainOwnershipTx, error)

	// NewConvertNetworkToL1Tx converts the chain to a Permissionless L1.
	//
	// - [chainID] specifies the chain to be converted
	// - [managerChainID] specifies which chain the manager is deployed on
	// - [address] specifies the address of the manager
	// - [validators] specifies the initial L1 validators of the L1
	NewConvertNetworkToL1Tx(
		chainID ids.ID,
		managerChainID ids.ID,
		address []byte,
		validators []*txs.ConvertNetworkToL1Validator,
		options ...common.Option,
	) (*txs.ConvertNetworkToL1Tx, error)

	// NewRegisterL1ValidatorTx adds a validator to an L1.
	//
	// - [balance] that the validator should allocate to continuous fees
	// - [proofOfPossession] is the BLS PoP for the key included in the Warp
	//   message
	// - [message] is the Warp message that authorizes this validator to be
	//   added
	NewRegisterL1ValidatorTx(
		balance uint64,
		proofOfPossession [bls.SignatureLen]byte,
		message []byte,
		options ...common.Option,
	) (*txs.RegisterL1ValidatorTx, error)

	// NewSetL1ValidatorWeightTx sets the weight of a validator on an L1.
	//
	// - [message] is the Warp message that authorizes this validator's weight
	//   to be changed
	NewSetL1ValidatorWeightTx(
		message []byte,
		options ...common.Option,
	) (*txs.SetL1ValidatorWeightTx, error)

	// NewIncreaseL1ValidatorBalanceTx increases the balance of a validator on
	// an L1 for the continuous fee.
	// the continuous fee.
	//
	// - [validationID] of the validator
	// - [balance] amount to increase the validator's balance by
	NewIncreaseL1ValidatorBalanceTx(
		validationID ids.ID,
		balance uint64,
		options ...common.Option,
	) (*txs.IncreaseL1ValidatorBalanceTx, error)

	// NewDisableL1ValidatorTx disables an L1 validator and returns the
	// remaining funds allocated to the continuous fee to the remaining balance
	// owner.
	//
	// - [validationID] of the validator to disable
	NewDisableL1ValidatorTx(
		validationID ids.ID,
		options ...common.Option,
	) (*txs.DisableL1ValidatorTx, error)

	// NewImportTx creates an import transaction that attempts to consume all
	// the available UTXOs and import the funds to [to].
	//
	// - [chainID] specifies the chain to be importing funds from.
	// - [to] specifies where to send the imported funds to.
	NewImportTx(
		chainID ids.ID,
		to *secp256k1fx.OutputOwners,
		options ...common.Option,
	) (*txs.ImportTx, error)

	// NewExportTx creates an export transaction that attempts to send all the
	// provided [outputs] to the requested [chainID].
	//
	// - [chainID] specifies the chain to be exporting the funds to.
	// - [outputs] specifies the outputs to send to the [chainID].
	NewExportTx(
		chainID ids.ID,
		outputs []*lux.TransferableOutput,
		options ...common.Option,
	) (*txs.ExportTx, error)

	// Removed in regenesis
	// // NewTransformChainTx creates a transform chain transaction that attempts
	// // to convert the provided [chainID] from a permissioned chain to a
	// // permissionless chain. This transaction will convert
	// // [maxSupply] - [initialSupply] of [assetID] to staking rewards.
	// //
	// // - [chainID] specifies the chain to transform.
	// // - [assetID] specifies the asset to use to reward stakers on the chain.
	// // - [initialSupply] is the amount of [assetID] that will be in circulation
	// //   after this transaction is accepted.
	// // - [maxSupply] is the maximum total amount of [assetID] that should ever
	// //   exist.
	// // - [minConsumptionRate] is the rate that a staker will receive rewards
	// //   if they stake with a duration of 0.
	// // - [maxConsumptionRate] is the maximum rate that staking rewards should be
	// //   consumed from the reward pool per year.
	// // - [minValidatorStake] is the minimum amount of funds required to become a
	// //   validator.
	// // - [maxValidatorStake] is the maximum amount of funds a single validator
	// //   can be allocated, including delegated funds.
	// // - [minStakeDuration] is the minimum number of seconds a staker can stake
	// //   for.
	// // - [maxStakeDuration] is the maximum number of seconds a staker can stake
	// //   for.
	// // - [minValidatorStake] is the minimum amount of funds required to become a
	// //   delegator.
	// // - [maxValidatorWeightFactor] is the factor which calculates the maximum
	// //   amount of delegation a validator can receive. A value of 1 effectively
	// //   disables delegation.
	// // - [uptimeRequirement] is the minimum percentage a validator must be
	// //   online and responsive to receive a reward.
	NewTransformChainTx(
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
	) (*txs.TransformChainTx, error)

	// NewAddPermissionlessValidatorTx creates a new validator of the specified
	// chain.
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
	NewAddPermissionlessValidatorTx(
		vdr *txs.ChainValidator,
		signer signer.Signer,
		assetID ids.ID,
		validationRewardsOwner *secp256k1fx.OutputOwners,
		delegationRewardsOwner *secp256k1fx.OutputOwners,
		shares uint32,
		options ...common.Option,
	) (*txs.AddPermissionlessValidatorTx, error)

	// NewAddPermissionlessDelegatorTx creates a new delegator of the specified
	// chain on the specified nodeID.
	//
	// - [vdr] specifies all the details of the delegation period such as the
	//   chainID, startTime, endTime, stake weight, and nodeID.
	// - [assetID] specifies the asset to stake.
	// - [rewardsOwner] specifies the owner of all the rewards this delegator
	//   earns during its delegation period.
	NewAddPermissionlessDelegatorTx(
		vdr *txs.ChainValidator,
		assetID ids.ID,
		rewardsOwner *secp256k1fx.OutputOwners,
		options ...common.Option,
	) (*txs.AddPermissionlessDelegatorTx, error)

	// Note: NewRegisterL1ValidatorTx, NewSetL1ValidatorWeightTx,
	// NewIncreaseL1ValidatorBalanceTx, and NewDisableL1ValidatorTx
	// are already defined above in this interface
}

type Backend interface {
	UTXOs(ctx context.Context, sourceChainID ids.ID) ([]*lux.UTXO, error)
	GetOwner(ctx context.Context, ownerID ids.ID) (fx.Owner, error)
}

type builder struct {
	addrs   set.Set[ids.ShortID]
	context *Context
	backend Backend
}

// New returns a new transaction builder.
//
//   - [addrs] is the set of addresses that the builder assumes can be used when
//     signing the transactions in the future.
//   - [context] provides the chain's configuration.
//   - [backend] provides the chain's state.
func New(
	addrs set.Set[ids.ShortID],
	context *Context,
	backend Backend,
) Builder {
	return &builder{
		addrs:   addrs,
		context: context,
		backend: backend,
	}
}

// getBlockchainID returns the blockchain ID to use for transactions
// Returns context.ChainID if set (for tests), otherwise PlatformChainID
func (b *builder) getBlockchainID() ids.ID {
	if b.context.ChainID != ids.Empty {
		return b.context.ChainID
	}
	return constants.PlatformChainID
}

func (b *builder) Context() *Context {
	return b.context
}

func (b *builder) GetBalance(
	options ...common.Option,
) (map[ids.ID]uint64, error) {
	ops := common.NewOptions(options)
	return b.getBalance(constants.PlatformChainID, ops)
}

func (b *builder) GetImportableBalance(
	chainID ids.ID,
	options ...common.Option,
) (map[ids.ID]uint64, error) {
	ops := common.NewOptions(options)
	return b.getBalance(chainID, ops)
}

func (b *builder) NewBaseTx(
	outputs []*lux.TransferableOutput,
	options ...common.Option,
) (*txs.BaseTx, error) {
	toBurn := map[ids.ID]uint64{}
	for _, out := range outputs {
		assetID := out.AssetID()
		amountToBurn, err := math.Add(toBurn[assetID], out.Out.Amount())
		if err != nil {
			return nil, err
		}
		toBurn[assetID] = amountToBurn
	}
	toStake := map[ids.ID]uint64{}

	ops := common.NewOptions(options)
	memo := ops.Memo()
	memoComplexity := gas.Dimensions{
		gas.Bandwidth: uint64(len(memo)),
	}
	outputComplexity, err := fee.OutputComplexity(outputs...)
	if err != nil {
		return nil, err
	}
	complexity, err := fee.IntrinsicBaseTxComplexities.Add(
		&memoComplexity,
		&outputComplexity,
	)
	if err != nil {
		return nil, err
	}

	inputs, changeOutputs, _, err := b.spend(
		toBurn,
		toStake,
		0,
		complexity,
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}
	outputs = append(outputs, changeOutputs...)
	lux.SortTransferableOutputs(outputs) // sort the outputs

	tx := &txs.BaseTx{BaseTx: lux.BaseTx{
		NetworkID:    b.context.NetworkID,
		BlockchainID: b.getBlockchainID(),
		Ins:          inputs,
		Outs:         outputs,
		Memo:         memo,
	}}
	return tx, b.initCtx(tx)
}

func (b *builder) NewAddValidatorTx(
	vdr *txs.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	shares uint32,
	options ...common.Option,
) (*txs.AddValidatorTx, error) {
	utxoAssetID := b.context.UTXOAssetID
	toBurn := map[ids.ID]uint64{}
	toStake := map[ids.ID]uint64{
		utxoAssetID: vdr.Wght,
	}
	ops := common.NewOptions(options)
	inputs, baseOutputs, stakeOutputs, err := b.spend(
		toBurn,
		toStake,
		0,
		gas.Dimensions{},
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}

	utils.Sort(rewardsOwner.Addrs)

	tx := &txs.AddValidatorTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: b.getBlockchainID(),
			Ins:          inputs,
			Outs:         baseOutputs,
			Memo:         ops.Memo(),
		}},
		Validator:        *vdr,
		StakeOuts:        stakeOutputs,
		RewardsOwner:     rewardsOwner,
		DelegationShares: shares,
	}
	return tx, b.initCtx(tx)
}

// Removed in regenesis
func (b *builder) NewAddChainValidatorTx(
	vdr *txs.ChainValidator,
	options ...common.Option,
) (*txs.AddChainValidatorTx, error) {
	toBurn := map[ids.ID]uint64{}
	toStake := map[ids.ID]uint64{}
	//
	ops := common.NewOptions(options)
	chainAuth, err := b.authorize(vdr.Chain, ops)
	if err != nil {
		return nil, err
	}

	memo := ops.Memo()
	memoComplexity := gas.Dimensions{
		gas.Bandwidth: uint64(len(memo)),
	}
	authComplexity, err := fee.AuthComplexity(chainAuth)
	if err != nil {
		return nil, err
	}
	complexity, err := fee.IntrinsicAddChainValidatorTxComplexities.Add(
		&memoComplexity,
		&authComplexity,
	)
	if err != nil {
		return nil, err
	}

	inputs, outputs, _, err := b.spend(
		toBurn,
		toStake,
		0,
		complexity,
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}

	tx := &txs.AddChainValidatorTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: b.getBlockchainID(),
			Ins:          inputs,
			Outs:         outputs,
			Memo:         memo,
		}},
		ChainValidator: *vdr,
		ChainAuth:      chainAuth,
	}
	return tx, b.initCtx(tx)
}

// Removed in regenesis
func (b *builder) NewRemoveChainValidatorTx(
	nodeID ids.NodeID,
	chainID ids.ID,
	options ...common.Option,
) (*txs.RemoveChainValidatorTx, error) {
	toBurn := map[ids.ID]uint64{}
	toStake := map[ids.ID]uint64{}
	//
	ops := common.NewOptions(options)
	chainAuth, err := b.authorize(chainID, ops)
	if err != nil {
		return nil, err
	}

	memo := ops.Memo()
	memoComplexity := gas.Dimensions{
		gas.Bandwidth: uint64(len(memo)),
	}
	authComplexity, err := fee.AuthComplexity(chainAuth)
	if err != nil {
		return nil, err
	}
	complexity, err := fee.IntrinsicRemoveChainValidatorTxComplexities.Add(
		&memoComplexity,
		&authComplexity,
	)
	if err != nil {
		return nil, err
	}

	inputs, outputs, _, err := b.spend(
		toBurn,
		toStake,
		0,
		complexity,
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}

	tx := &txs.RemoveChainValidatorTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: b.getBlockchainID(),
			Ins:          inputs,
			Outs:         outputs,
			Memo:         ops.Memo(),
		}},
		Chain:     chainID,
		NodeID:    nodeID,
		ChainAuth: chainAuth,
	}
	return tx, b.initCtx(tx)
}

func (b *builder) NewAddDelegatorTx(
	vdr *txs.Validator,
	rewardsOwner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.AddDelegatorTx, error) {
	utxoAssetID := b.context.UTXOAssetID
	toBurn := map[ids.ID]uint64{}
	toStake := map[ids.ID]uint64{
		utxoAssetID: vdr.Wght,
	}
	ops := common.NewOptions(options)
	inputs, baseOutputs, stakeOutputs, err := b.spend(
		toBurn,
		toStake,
		0,
		gas.Dimensions{},
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}

	utils.Sort(rewardsOwner.Addrs)
	tx := &txs.AddDelegatorTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: b.getBlockchainID(),
			Ins:          inputs,
			Outs:         baseOutputs,
			Memo:         ops.Memo(),
		}},
		Validator:              *vdr,
		StakeOuts:              stakeOutputs,
		DelegationRewardsOwner: rewardsOwner,
	}
	return tx, b.initCtx(tx)
}

func (b *builder) NewCreateChainTx(
	chainID ids.ID,
	genesis []byte,
	vmID ids.ID,
	fxIDs []ids.ID,
	blockchainName string,
	options ...common.Option,
) (*txs.CreateChainTx, error) {
	toBurn := map[ids.ID]uint64{}
	toStake := map[ids.ID]uint64{}

	ops := common.NewOptions(options)
	chainAuth, err := b.authorize(chainID, ops)
	if err != nil {
		return nil, err
	}

	memo := ops.Memo()
	bandwidth, err := math.Mul(uint64(len(fxIDs)), ids.IDLen)
	if err != nil {
		return nil, err
	}
	bandwidth, err = math.Add(bandwidth, uint64(len(blockchainName)))
	if err != nil {
		return nil, err
	}
	bandwidth, err = math.Add(bandwidth, uint64(len(genesis)))
	if err != nil {
		return nil, err
	}
	bandwidth, err = math.Add(bandwidth, uint64(len(memo)))
	if err != nil {
		return nil, err
	}
	dynamicComplexity := gas.Dimensions{
		gas.Bandwidth: bandwidth,
	}
	authComplexity, err := fee.AuthComplexity(chainAuth)
	if err != nil {
		return nil, err
	}
	complexity, err := fee.IntrinsicCreateChainTxComplexities.Add(
		&dynamicComplexity,
		&authComplexity,
	)
	if err != nil {
		return nil, err
	}

	inputs, outputs, _, err := b.spend(
		toBurn,
		toStake,
		0,
		complexity,
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}

	utils.Sort(fxIDs)
	tx := &txs.CreateChainTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: b.getBlockchainID(),
			Ins:          inputs,
			Outs:         outputs,
			Memo:         memo,
		}},
		ValidateNetworkID: chainID,
		BlockchainName:    blockchainName,
		VMID:              vmID,
		FxIDs:             fxIDs,
		GenesisData:       genesis,
		ChainAuth:         chainAuth,
	}
	return tx, b.initCtx(tx)
}

func (b *builder) NewCreateNetworkTx(
	owner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.CreateNetworkTx, error) {
	toBurn := map[ids.ID]uint64{}
	toStake := map[ids.ID]uint64{}
	//
	ops := common.NewOptions(options)
	memo := ops.Memo()
	memoComplexity := gas.Dimensions{
		gas.Bandwidth: uint64(len(memo)),
	}
	ownerComplexity, err := fee.OwnerComplexity(owner)
	if err != nil {
		return nil, err
	}
	complexity, err := fee.IntrinsicCreateNetworkTxComplexities.Add(
		&memoComplexity,
		&ownerComplexity,
	)
	if err != nil {
		return nil, err
	}
	//
	inputs, outputs, _, err := b.spend(
		toBurn,
		toStake,
		0,
		complexity,
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}
	//
	utils.Sort(owner.Addrs)
	tx := &txs.CreateNetworkTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: b.getBlockchainID(),
			Ins:          inputs,
			Outs:         outputs,
			Memo:         memo,
		}},
		Owner: owner,
		// Restake the parent, no own validator set: the network's security is
		// entirely inherited and its owner is the authority.
		Security: security.Mode{RestakeParent: true},
	}
	return tx, b.initCtx(tx)
}

// Removed in regenesis
func (b *builder) NewTransferChainOwnershipTx(
	chainID ids.ID,
	owner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.TransferChainOwnershipTx, error) {
	toBurn := map[ids.ID]uint64{}
	toStake := map[ids.ID]uint64{}

	ops := common.NewOptions(options)
	chainAuth, err := b.authorize(chainID, ops)
	if err != nil {
		return nil, err
	}

	memo := ops.Memo()
	memoComplexity := gas.Dimensions{
		gas.Bandwidth: uint64(len(memo)),
	}
	authComplexity, err := fee.AuthComplexity(chainAuth)
	if err != nil {
		return nil, err
	}
	ownerComplexity, err := fee.OwnerComplexity(owner)
	if err != nil {
		return nil, err
	}
	complexity, err := fee.IntrinsicTransferChainOwnershipTxComplexities.Add(
		&memoComplexity,
		&authComplexity,
		&ownerComplexity,
	)
	if err != nil {
		return nil, err
	}

	inputs, outputs, _, err := b.spend(
		toBurn,
		toStake,
		0,
		complexity,
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}

	utils.Sort(owner.Addrs)
	tx := &txs.TransferChainOwnershipTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: b.getBlockchainID(),
			Ins:          inputs,
			Outs:         outputs,
			Memo:         memo,
		}},
		Chain:     chainID,
		Owner:     owner,
		ChainAuth: chainAuth,
	}
	return tx, b.initCtx(tx)
}

func (b *builder) NewConvertNetworkToL1Tx(
	chainID ids.ID,
	managerChainID ids.ID,
	address []byte,
	validators []*txs.ConvertNetworkToL1Validator,
	options ...common.Option,
) (*txs.ConvertNetworkToL1Tx, error) {
	var luxToBurn uint64
	for _, vdr := range validators {
		var err error
		luxToBurn, err = math.Add(luxToBurn, vdr.Balance)
		if err != nil {
			return nil, err
		}
	}

	var (
		toBurn = map[ids.ID]uint64{
			b.context.UTXOAssetID: luxToBurn,
		}
		toStake = map[ids.ID]uint64{}
		ops     = common.NewOptions(options)
	)
	chainAuth, err := b.authorize(chainID, ops)
	if err != nil {
		return nil, err
	}

	memo := ops.Memo()
	additionalBytes, err := math.Add(uint64(len(memo)), uint64(len(address)))
	if err != nil {
		return nil, err
	}
	bytesComplexity := gas.Dimensions{
		gas.Bandwidth: additionalBytes,
	}
	validatorComplexity, err := fee.ConvertNetworkToL1ValidatorComplexity(validators...)
	if err != nil {
		return nil, err
	}
	authComplexity, err := fee.AuthComplexity(chainAuth)
	if err != nil {
		return nil, err
	}
	complexity, err := fee.IntrinsicConvertNetworkToL1TxComplexities.Add(
		&bytesComplexity,
		&validatorComplexity,
		&authComplexity,
	)
	if err != nil {
		return nil, err
	}

	inputs, outputs, _, err := b.spend(
		toBurn,
		toStake,
		0,
		complexity,
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}

	utils.Sort(validators)
	tx := &txs.ConvertNetworkToL1Tx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: constants.PlatformChainID,
			Ins:          inputs,
			Outs:         outputs,
			Memo:         memo,
		}},
		Chain:          chainID,
		ManagerChainID: managerChainID,
		Address:        address,
		Validators:     validators,
		ChainAuth:      chainAuth,
	}
	return tx, b.initCtx(tx)
}

func (b *builder) NewRegisterL1ValidatorTx(
	balance uint64,
	proofOfPossession [bls.SignatureLen]byte,
	message []byte,
	options ...common.Option,
) (*txs.RegisterL1ValidatorTx, error) {
	var (
		toBurn = map[ids.ID]uint64{
			b.context.UTXOAssetID: balance,
		}
		toStake = map[ids.ID]uint64{}

		ops            = common.NewOptions(options)
		memo           = ops.Memo()
		memoComplexity = gas.Dimensions{
			gas.Bandwidth: uint64(len(memo)),
		}
	)
	warpComplexity, err := fee.WarpComplexity(message)
	if err != nil {
		return nil, err
	}
	complexity, err := fee.IntrinsicRegisterL1ValidatorTxComplexities.Add(
		&memoComplexity,
		&warpComplexity,
	)
	if err != nil {
		return nil, err
	}

	inputs, outputs, _, err := b.spend(
		toBurn,
		toStake,
		0,
		complexity,
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}

	tx := &txs.RegisterL1ValidatorTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: constants.PlatformChainID,
			Ins:          inputs,
			Outs:         outputs,
			Memo:         memo,
		}},
		Balance:           balance,
		ProofOfPossession: proofOfPossession,
		Message:           message,
	}
	return tx, b.initCtx(tx)
}

func (b *builder) NewSetL1ValidatorWeightTx(
	message []byte,
	options ...common.Option,
) (*txs.SetL1ValidatorWeightTx, error) {
	var (
		toBurn         = map[ids.ID]uint64{}
		toStake        = map[ids.ID]uint64{}
		ops            = common.NewOptions(options)
		memo           = ops.Memo()
		memoComplexity = gas.Dimensions{
			gas.Bandwidth: uint64(len(memo)),
		}
	)
	warpComplexity, err := fee.WarpComplexity(message)
	if err != nil {
		return nil, err
	}
	complexity, err := fee.IntrinsicSetL1ValidatorWeightTxComplexities.Add(
		&memoComplexity,
		&warpComplexity,
	)
	if err != nil {
		return nil, err
	}

	inputs, outputs, _, err := b.spend(
		toBurn,
		toStake,
		0,
		complexity,
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}

	tx := &txs.SetL1ValidatorWeightTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: constants.PlatformChainID,
			Ins:          inputs,
			Outs:         outputs,
			Memo:         memo,
		}},
		Message: message,
	}
	return tx, b.initCtx(tx)
}

func (b *builder) NewIncreaseL1ValidatorBalanceTx(
	validationID ids.ID,
	balance uint64,
	options ...common.Option,
) (*txs.IncreaseL1ValidatorBalanceTx, error) {
	var (
		toBurn = map[ids.ID]uint64{
			b.context.UTXOAssetID: balance,
		}
		toStake        = map[ids.ID]uint64{}
		ops            = common.NewOptions(options)
		memo           = ops.Memo()
		memoComplexity = gas.Dimensions{
			gas.Bandwidth: uint64(len(memo)),
		}
	)
	complexity, err := fee.IntrinsicIncreaseL1ValidatorBalanceTxComplexities.Add(
		&memoComplexity,
	)
	if err != nil {
		return nil, err
	}

	inputs, outputs, _, err := b.spend(
		toBurn,
		toStake,
		0,
		complexity,
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}

	tx := &txs.IncreaseL1ValidatorBalanceTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: constants.PlatformChainID,
			Ins:          inputs,
			Outs:         outputs,
			Memo:         memo,
		}},
		ValidationID: validationID,
		Balance:      balance,
	}
	return tx, b.initCtx(tx)
}

func (b *builder) NewDisableL1ValidatorTx(
	validationID ids.ID,
	options ...common.Option,
) (*txs.DisableL1ValidatorTx, error) {
	var (
		toBurn  = map[ids.ID]uint64{}
		toStake = map[ids.ID]uint64{}
		ops     = common.NewOptions(options)
	)
	disableAuth, err := b.authorize(validationID, ops)
	if err != nil {
		return nil, err
	}

	memo := ops.Memo()
	memoComplexity := gas.Dimensions{
		gas.Bandwidth: uint64(len(memo)),
	}
	authComplexity, err := fee.AuthComplexity(disableAuth)
	if err != nil {
		return nil, err
	}

	complexity, err := fee.IntrinsicDisableL1ValidatorTxComplexities.Add(
		&memoComplexity,
		&authComplexity,
	)
	if err != nil {
		return nil, err
	}

	inputs, outputs, _, err := b.spend(
		toBurn,
		toStake,
		0,
		complexity,
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}

	tx := &txs.DisableL1ValidatorTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: constants.PlatformChainID,
			Ins:          inputs,
			Outs:         outputs,
			Memo:         memo,
		}},
		ValidationID: validationID,
		DisableAuth:  disableAuth,
	}
	return tx, b.initCtx(tx)
}

func (b *builder) NewImportTx(
	sourceChainID ids.ID,
	to *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.ImportTx, error) {
	ops := common.NewOptions(options)
	utxos, err := b.backend.UTXOs(ops.Context(), sourceChainID)
	if err != nil {
		return nil, err
	}

	var (
		addrs           = ops.Addresses(b.addrs)
		minIssuanceTime = ops.MinIssuanceTime()
		utxoAssetID     = b.context.UTXOAssetID

		importedInputs  = make([]*lux.TransferableInput, 0, len(utxos))
		importedAmounts = make(map[ids.ID]uint64)
	)
	// Iterate over the unlocked UTXOs
	for _, utxo := range utxos {
		out, ok := utxo.Out.(*secp256k1fx.TransferOutput)
		if !ok {
			continue
		}

		inputSigIndices, ok := common.MatchOwners(&out.OutputOwners, addrs, minIssuanceTime)
		if !ok {
			// We couldn't spend this UTXO, so we skip to the next one
			continue
		}

		importedInputs = append(importedInputs, &lux.TransferableInput{
			UTXOID: utxo.UTXOID,
			Asset:  utxo.Asset,
			In: &secp256k1fx.TransferInput{
				Amt: out.Amt,
				Input: secp256k1fx.Input{
					SigIndices: inputSigIndices,
				},
			},
		})

		assetID := utxo.AssetID()
		newImportedAmount, err := math.Add(importedAmounts[assetID], out.Amt)
		if err != nil {
			return nil, err
		}
		importedAmounts[assetID] = newImportedAmount
	}
	utils.Sort(importedInputs) // sort imported inputs

	if len(importedInputs) == 0 {
		return nil, fmt.Errorf(
			"%w: no UTXOs available to import",
			ErrInsufficientFunds,
		)
	}

	outputs := make([]*lux.TransferableOutput, 0, len(importedAmounts))
	for assetID, amount := range importedAmounts {
		if assetID == utxoAssetID {
			continue
		}

		outputs = append(outputs, &lux.TransferableOutput{
			Asset: lux.Asset{ID: assetID},
			Out: &secp256k1fx.TransferOutput{
				Amt:          amount,
				OutputOwners: *to,
			},
		})
	}

	memo := ops.Memo()
	memoComplexity := gas.Dimensions{
		gas.Bandwidth: uint64(len(memo)),
	}
	inputComplexity, err := fee.InputComplexity(importedInputs...)
	if err != nil {
		return nil, err
	}
	outputComplexity, err := fee.OutputComplexity(outputs...)
	if err != nil {
		return nil, err
	}
	complexity, err := fee.IntrinsicImportTxComplexities.Add(
		&memoComplexity,
		&inputComplexity,
		&outputComplexity,
	)
	if err != nil {
		return nil, err
	}

	var (
		toBurn  = map[ids.ID]uint64{}
		toStake = map[ids.ID]uint64{}
	)
	excessLUX := importedAmounts[utxoAssetID]

	inputs, changeOutputs, _, err := b.spend(
		toBurn,
		toStake,
		excessLUX,
		complexity,
		to,
		ops,
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate tx inputs/outputs: %w", err)
	}
	outputs = append(outputs, changeOutputs...)

	lux.SortTransferableOutputs(outputs) // sort imported outputs
	tx := &txs.ImportTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: b.getBlockchainID(),
			Ins:          inputs,
			Outs:         outputs,
			Memo:         memo,
		}},
		SourceChain:    sourceChainID,
		ImportedInputs: importedInputs,
	}
	return tx, b.initCtx(tx)
}

func (b *builder) NewExportTx(
	chainID ids.ID,
	outputs []*lux.TransferableOutput,
	options ...common.Option,
) (*txs.ExportTx, error) {
	toBurn := map[ids.ID]uint64{}
	for _, out := range outputs {
		assetID := out.AssetID()
		amountToBurn, err := math.Add(toBurn[assetID], out.Out.Amount())
		if err != nil {
			return nil, err
		}
		toBurn[assetID] = amountToBurn
	}

	toStake := map[ids.ID]uint64{}
	ops := common.NewOptions(options)
	memo := ops.Memo()
	memoComplexity := gas.Dimensions{
		gas.Bandwidth: uint64(len(memo)),
	}
	outputComplexity, err := fee.OutputComplexity(outputs...)
	if err != nil {
		return nil, err
	}
	complexity, err := fee.IntrinsicExportTxComplexities.Add(
		&memoComplexity,
		&outputComplexity,
	)
	if err != nil {
		return nil, err
	}

	inputs, changeOutputs, _, err := b.spend(
		toBurn,
		toStake,
		0,
		complexity,
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}

	lux.SortTransferableOutputs(outputs) // sort exported outputs
	tx := &txs.ExportTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: b.getBlockchainID(),
			Ins:          inputs,
			Outs:         changeOutputs,
			Memo:         memo,
		}},
		DestinationChain: chainID,
		ExportedOutputs:  outputs,
	}
	return tx, b.initCtx(tx)
}

// Removed in regenesis
func (b *builder) NewTransformChainTx(
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
	toBurn := map[ids.ID]uint64{
		assetID: maxSupply - initialSupply,
	}
	toStake := map[ids.ID]uint64{}

	ops := common.NewOptions(options)
	chainAuth, err := b.authorize(chainID, ops)
	if err != nil {
		return nil, err
	}

	inputs, outputs, _, err := b.spend(
		toBurn,
		toStake,
		0,
		gas.Dimensions{},
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}

	tx := &txs.TransformChainTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: b.getBlockchainID(),
			Ins:          inputs,
			Outs:         outputs,
			Memo:         ops.Memo(),
		}},
		Chain:                    chainID,
		AssetID:                  assetID,
		InitialSupply:            initialSupply,
		MaximumSupply:            maxSupply,
		MinConsumptionRate:       minConsumptionRate,
		MaxConsumptionRate:       maxConsumptionRate,
		MinValidatorStake:        minValidatorStake,
		MaxValidatorStake:        maxValidatorStake,
		MinStakeDuration:         uint32(minStakeDuration / time.Second),
		MaxStakeDuration:         uint32(maxStakeDuration / time.Second),
		MinDelegationFee:         minDelegationFee,
		MinDelegatorStake:        minDelegatorStake,
		MaxValidatorWeightFactor: maxValidatorWeightFactor,
		UptimeRequirement:        uptimeRequirement,
		ChainAuth:                chainAuth,
	}
	return tx, b.initCtx(tx)
}

func (b *builder) NewAddPermissionlessValidatorTx(
	vdr *txs.ChainValidator,
	signer signer.Signer,
	assetID ids.ID,
	validationRewardsOwner *secp256k1fx.OutputOwners,
	delegationRewardsOwner *secp256k1fx.OutputOwners,
	shares uint32,
	options ...common.Option,
) (*txs.AddPermissionlessValidatorTx, error) {
	toBurn := map[ids.ID]uint64{}
	toStake := map[ids.ID]uint64{
		assetID: vdr.Wght,
	}

	ops := common.NewOptions(options)
	memo := ops.Memo()
	memoComplexity := gas.Dimensions{
		gas.Bandwidth: uint64(len(memo)),
	}
	signerComplexity, err := fee.SignerComplexity(signer)
	if err != nil {
		return nil, err
	}
	validatorOwnerComplexity, err := fee.OwnerComplexity(validationRewardsOwner)
	if err != nil {
		return nil, err
	}
	delegatorOwnerComplexity, err := fee.OwnerComplexity(delegationRewardsOwner)
	if err != nil {
		return nil, err
	}
	complexity, err := fee.IntrinsicAddPermissionlessValidatorTxComplexities.Add(
		&memoComplexity,
		&signerComplexity,
		&validatorOwnerComplexity,
		&delegatorOwnerComplexity,
	)
	if err != nil {
		return nil, err
	}

	inputs, baseOutputs, stakeOutputs, err := b.spend(
		toBurn,
		toStake,
		0,
		complexity,
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}

	utils.Sort(validationRewardsOwner.Addrs)
	utils.Sort(delegationRewardsOwner.Addrs)
	tx := &txs.AddPermissionlessValidatorTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: b.getBlockchainID(),
			Ins:          inputs,
			Outs:         baseOutputs,
			Memo:         memo,
		}},
		Validator:             vdr.Validator,
		Chain:                 vdr.Chain,
		Signer:                signer,
		StakeOuts:             stakeOutputs,
		ValidatorRewardsOwner: validationRewardsOwner,
		DelegatorRewardsOwner: delegationRewardsOwner,
		DelegationShares:      shares,
	}
	return tx, b.initCtx(tx)
}

func (b *builder) NewAddPermissionlessDelegatorTx(
	vdr *txs.ChainValidator,
	assetID ids.ID,
	rewardsOwner *secp256k1fx.OutputOwners,
	options ...common.Option,
) (*txs.AddPermissionlessDelegatorTx, error) {
	toBurn := map[ids.ID]uint64{}
	toStake := map[ids.ID]uint64{
		assetID: vdr.Wght,
	}

	ops := common.NewOptions(options)
	memo := ops.Memo()
	memoComplexity := gas.Dimensions{
		gas.Bandwidth: uint64(len(memo)),
	}
	ownerComplexity, err := fee.OwnerComplexity(rewardsOwner)
	if err != nil {
		return nil, err
	}
	complexity, err := fee.IntrinsicAddPermissionlessDelegatorTxComplexities.Add(
		&memoComplexity,
		&ownerComplexity,
	)
	if err != nil {
		return nil, err
	}

	inputs, baseOutputs, stakeOutputs, err := b.spend(
		toBurn,
		toStake,
		0,
		complexity,
		nil,
		ops,
	)
	if err != nil {
		return nil, err
	}

	utils.Sort(rewardsOwner.Addrs)
	tx := &txs.AddPermissionlessDelegatorTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: b.getBlockchainID(),
			Ins:          inputs,
			Outs:         baseOutputs,
			Memo:         memo,
		}},
		Validator:              vdr.Validator,
		Chain:                  vdr.Chain,
		StakeOuts:              stakeOutputs,
		DelegationRewardsOwner: rewardsOwner,
	}
	return tx, b.initCtx(tx)
}

func (b *builder) getBalance(
	chainID ids.ID,
	options *common.Options,
) (
	balance map[ids.ID]uint64,
	err error,
) {
	utxos, err := b.backend.UTXOs(options.Context(), chainID)
	if err != nil {
		return nil, err
	}

	addrs := options.Addresses(b.addrs)
	minIssuanceTime := options.MinIssuanceTime()
	balance = make(map[ids.ID]uint64)

	// Iterate over the UTXOs
	for _, utxo := range utxos {
		outIntf := utxo.Out
		if lockedOut, ok := outIntf.(*stakeable.LockOut); ok {
			if !options.AllowStakeableLocked() && lockedOut.Locktime > minIssuanceTime {
				// This output is currently locked, so this output can't be
				// burned.
				continue
			}
			outIntf = lockedOut.TransferableOut
		}

		out, ok := outIntf.(*secp256k1fx.TransferOutput)
		if !ok {
			return nil, ErrUnknownOutputType
		}

		_, ok = common.MatchOwners(&out.OutputOwners, addrs, minIssuanceTime)
		if !ok {
			// We couldn't spend this UTXO, so we skip to the next one
			continue
		}

		assetID := utxo.AssetID()
		balance[assetID], err = math.Add(balance[assetID], out.Amt)
		if err != nil {
			return nil, err
		}
	}
	return balance, nil
}

// spend takes in the requested burn amounts and the requested stake amounts.
//
//   - [toBurn] maps assetID to the amount of the asset to spend without
//     producing an output. This is typically used for fees. However, it can
//     also be used to consume some of an asset that will be produced in
//     separate outputs, such as ExportedOutputs. Only unlocked UTXOs are able
//     to be burned here.
//   - [toStake] maps assetID to the amount of the asset to spend and place into
//     the staked outputs. First locked UTXOs are attempted to be used for these
//     funds, and then unlocked UTXOs will be attempted to be used. There is no
//     preferential ordering on the unlock times.
//   - [excessLUX] contains the amount of extra LUX that spend can produce in
//     the change outputs in addition to the consumed and not burned LUX.
//   - [complexity] contains the currently accrued transaction complexity that
//     will be used to calculate the required fees to be burned.
//   - [ownerOverride] optionally specifies the output owners to use for the
//     unlocked LUX change output if no additional LUX was needed to be
//     burned. If this value is nil, the default change owner is used.
func (b *builder) spend(
	toBurn map[ids.ID]uint64,
	toStake map[ids.ID]uint64,
	excessLUX uint64,
	complexity gas.Dimensions,
	ownerOverride *secp256k1fx.OutputOwners,
	options *common.Options,
) (
	inputs []*lux.TransferableInput,
	changeOutputs []*lux.TransferableOutput,
	stakeOutputs []*lux.TransferableOutput,
	err error,
) {
	utxos, err := b.backend.UTXOs(options.Context(), constants.PlatformChainID)
	if err != nil {
		return nil, nil, nil, err
	}

	addrs := options.Addresses(b.addrs)
	minIssuanceTime := options.MinIssuanceTime()

	addr, ok := addrs.Peek()
	if !ok {
		return nil, nil, nil, ErrNoChangeAddress
	}
	changeOwner := options.ChangeOwner(&secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs:     []ids.ShortID{addr},
	})
	if ownerOverride == nil {
		ownerOverride = changeOwner
	}

	s := spendHelper{
		weights:  b.context.ComplexityWeights,
		gasPrice: b.context.GasPrice,

		toBurn:     toBurn,
		toStake:    toStake,
		complexity: complexity,

		// Initialize the return values with empty slices to preserve backward
		// compatibility of the json representation of transactions with no
		// inputs or outputs.
		inputs:        make([]*lux.TransferableInput, 0),
		changeOutputs: make([]*lux.TransferableOutput, 0),
		stakeOutputs:  make([]*lux.TransferableOutput, 0),
	}

	utxosByLocktime := splitByLocktime(utxos, minIssuanceTime)
	for _, utxo := range utxosByLocktime.locked {
		assetID := utxo.AssetID()
		if !s.shouldConsumeLockedAsset(assetID) {
			continue
		}

		out, locktime, err := unwrapOutput(utxo.Out)
		if err != nil {
			return nil, nil, nil, err
		}

		inputSigIndices, ok := common.MatchOwners(&out.OutputOwners, addrs, minIssuanceTime)
		if !ok {
			// We couldn't spend this UTXO, so we skip to the next one
			continue
		}

		err = s.addInput(&lux.TransferableInput{
			UTXOID: utxo.UTXOID,
			Asset:  utxo.Asset,
			In: &stakeable.LockIn{
				Locktime: locktime,
				TransferableIn: &secp256k1fx.TransferInput{
					Amt: out.Amt,
					Input: secp256k1fx.Input{
						SigIndices: inputSigIndices,
					},
				},
			},
		})
		if err != nil {
			return nil, nil, nil, err
		}

		excess := s.consumeLockedAsset(assetID, out.Amt)
		err = s.addStakedOutput(&lux.TransferableOutput{
			Asset: utxo.Asset,
			Out: &stakeable.LockOut{
				Locktime: locktime,
				TransferableOut: &secp256k1fx.TransferOutput{
					Amt:          out.Amt - excess,
					OutputOwners: out.OutputOwners,
				},
			},
		})
		if err != nil {
			return nil, nil, nil, err
		}

		if excess == 0 {
			continue
		}

		// This input had extra value, so some of it must be returned
		err = s.addChangeOutput(&lux.TransferableOutput{
			Asset: utxo.Asset,
			Out: &stakeable.LockOut{
				Locktime: locktime,
				TransferableOut: &secp256k1fx.TransferOutput{
					Amt:          excess,
					OutputOwners: out.OutputOwners,
				},
			},
		})
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// Add all the remaining stake amounts assuming unlocked UTXOs.
	for assetID, amount := range s.toStake {
		if amount == 0 {
			continue
		}

		err = s.addStakedOutput(&lux.TransferableOutput{
			Asset: lux.Asset{
				ID: assetID,
			},
			Out: &secp256k1fx.TransferOutput{
				Amt:          amount,
				OutputOwners: *changeOwner,
			},
		})
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// LUX is handled last to account for fees.
	utxosByUTXOAssetID := splitByAssetID(utxosByLocktime.unlocked, b.context.UTXOAssetID)
	for _, utxo := range utxosByUTXOAssetID.other {
		assetID := utxo.AssetID()
		if !s.shouldConsumeAsset(assetID) {
			continue
		}

		out, _, err := unwrapOutput(utxo.Out)
		if err != nil {
			return nil, nil, nil, err
		}

		inputSigIndices, ok := common.MatchOwners(&out.OutputOwners, addrs, minIssuanceTime)
		if !ok {
			// We couldn't spend this UTXO, so we skip to the next one
			continue
		}

		err = s.addInput(&lux.TransferableInput{
			UTXOID: utxo.UTXOID,
			Asset:  utxo.Asset,
			In: &secp256k1fx.TransferInput{
				Amt: out.Amt,
				Input: secp256k1fx.Input{
					SigIndices: inputSigIndices,
				},
			},
		})
		if err != nil {
			return nil, nil, nil, err
		}

		excess := s.consumeAsset(assetID, out.Amt)
		if excess == 0 {
			continue
		}

		// This input had extra value, so some of it must be returned
		err = s.addChangeOutput(&lux.TransferableOutput{
			Asset: utxo.Asset,
			Out: &secp256k1fx.TransferOutput{
				Amt:          excess,
				OutputOwners: *changeOwner,
			},
		})
		if err != nil {
			return nil, nil, nil, err
		}
	}

	for _, utxo := range utxosByUTXOAssetID.requested {
		// If we don't need to burn or stake additional LUX and we have
		// already consumed enough LUX to pay the required fee, stop
		// consuming UTXOs before adding this one.
		requiredFee, err := s.calculateFee()
		if err != nil {
			return nil, nil, nil, err
		}
		if !s.shouldConsumeAsset(b.context.UTXOAssetID) && excessLUX >= requiredFee {
			break
		}

		out, _, err := unwrapOutput(utxo.Out)
		if err != nil {
			return nil, nil, nil, err
		}

		inputSigIndices, ok := common.MatchOwners(&out.OutputOwners, addrs, minIssuanceTime)
		if !ok {
			// We couldn't spend this UTXO, so we skip to the next one
			continue
		}

		err = s.addInput(&lux.TransferableInput{
			UTXOID: utxo.UTXOID,
			Asset:  utxo.Asset,
			In: &secp256k1fx.TransferInput{
				Amt: out.Amt,
				Input: secp256k1fx.Input{
					SigIndices: inputSigIndices,
				},
			},
		})
		if err != nil {
			return nil, nil, nil, err
		}

		excess := s.consumeAsset(b.context.UTXOAssetID, out.Amt)
		excessLUX, err = math.Add(excessLUX, excess)
		if err != nil {
			return nil, nil, nil, err
		}

		// If we need to consume additional LUX, we should be returning the
		// change to the change address.
		ownerOverride = changeOwner
	}

	if err := s.verifyAssetsConsumed(); err != nil {
		return nil, nil, nil, err
	}

	requiredFee, err := s.calculateFee()
	if err != nil {
		return nil, nil, nil, err
	}
	if excessLUX < requiredFee {
		return nil, nil, nil, fmt.Errorf(
			"%w: provided UTXOs needed %d more nLUX (%q)",
			ErrInsufficientFunds,
			requiredFee-excessLUX,
			b.context.UTXOAssetID,
		)
	}

	// Try to add a change output if we have excess LUX
	secpExcessLUXOutput := &secp256k1fx.TransferOutput{
		Amt:          0, // Populated later if used
		OutputOwners: *ownerOverride,
	}
	excessLUXOutput := &lux.TransferableOutput{
		Asset: lux.Asset{
			ID: b.context.UTXOAssetID,
		},
		Out: secpExcessLUXOutput,
	}

	// Calculate the complexity of the potential change output
	changeOutputComplexity, err := fee.OutputComplexity(excessLUXOutput)
	if err != nil {
		return nil, nil, nil, err
	}

	// Temporarily add the change output complexity to calculate the fee
	tempComplexity, err := s.complexity.Add(&changeOutputComplexity)
	if err != nil {
		return nil, nil, nil, err
	}

	requiredFeeWithChange, err := tempComplexity.ToGas(s.weights)
	if err != nil {
		return nil, nil, nil, err
	}
	feeWithChange, err := requiredFeeWithChange.Cost(s.gasPrice)
	if err != nil {
		return nil, nil, nil, err
	}

	if excessLUX > feeWithChange {
		// It is worth adding the change output
		// Permanently add the complexity
		s.complexity = tempComplexity
		secpExcessLUXOutput.Amt = excessLUX - feeWithChange
		s.changeOutputs = append(s.changeOutputs, excessLUXOutput)
	}
	// If excessLUX <= feeWithChange, we don't add the change output
	// and we don't modify s.complexity (it stays without the change output)

	utils.Sort(s.inputs)                         // sort inputs
	lux.SortTransferableOutputs(s.changeOutputs) // sort the change outputs
	lux.SortTransferableOutputs(s.stakeOutputs)  // sort stake outputs
	return s.inputs, s.changeOutputs, s.stakeOutputs, nil
}

func (b *builder) authorize(ownerID ids.ID, options *common.Options) (*secp256k1fx.Input, error) {
	ownerIntf, err := b.backend.GetOwner(options.Context(), ownerID)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch owner for %q: %w",
			ownerID,
			err,
		)
	}
	owner, ok := ownerIntf.(*secp256k1fx.OutputOwners)
	if !ok {
		return nil, ErrUnknownOwnerType
	}

	addrs := options.Addresses(b.addrs)
	minIssuanceTime := options.MinIssuanceTime()
	inputSigIndices, ok := common.MatchOwners(owner, addrs, minIssuanceTime)
	if !ok {
		// We can't authorize the chain
		return nil, ErrInsufficientAuthorization
	}
	return &secp256k1fx.Input{
		SigIndices: inputSigIndices,
	}, nil
}

func (b *builder) initCtx(tx txs.UnsignedTx) error {
	ctx, err := NewConsensusRuntime(b.context.NetworkID, b.context.UTXOAssetID)
	if err != nil {
		return err
	}

	tx.InitRuntime(ctx)
	return nil
}

type spendHelper struct {
	weights  gas.Dimensions
	gasPrice gas.Price

	toBurn     map[ids.ID]uint64
	toStake    map[ids.ID]uint64
	complexity gas.Dimensions

	inputs        []*lux.TransferableInput
	changeOutputs []*lux.TransferableOutput
	stakeOutputs  []*lux.TransferableOutput
}

func (s *spendHelper) addInput(input *lux.TransferableInput) error {
	newInputComplexity, err := fee.InputComplexity(input)
	if err != nil {
		return err
	}
	s.complexity, err = s.complexity.Add(&newInputComplexity)
	if err != nil {
		return err
	}

	s.inputs = append(s.inputs, input)
	return nil
}

func (s *spendHelper) addChangeOutput(output *lux.TransferableOutput) error {
	s.changeOutputs = append(s.changeOutputs, output)
	return s.addOutputComplexity(output)
}

func (s *spendHelper) addStakedOutput(output *lux.TransferableOutput) error {
	s.stakeOutputs = append(s.stakeOutputs, output)
	return s.addOutputComplexity(output)
}

func (s *spendHelper) addOutputComplexity(output *lux.TransferableOutput) error {
	newOutputComplexity, err := fee.OutputComplexity(output)
	if err != nil {
		return err
	}
	s.complexity, err = s.complexity.Add(&newOutputComplexity)
	return err
}

func (s *spendHelper) shouldConsumeLockedAsset(assetID ids.ID) bool {
	return s.toStake[assetID] != 0
}

func (s *spendHelper) shouldConsumeAsset(assetID ids.ID) bool {
	return s.toBurn[assetID] != 0 || s.shouldConsumeLockedAsset(assetID)
}

func (s *spendHelper) consumeLockedAsset(assetID ids.ID, amount uint64) uint64 {
	// Stake any value that should be staked
	toStake := min(
		s.toStake[assetID], // Amount we still need to stake
		amount,             // Amount available to stake
	)
	s.toStake[assetID] -= toStake
	return amount - toStake
}

func (s *spendHelper) consumeAsset(assetID ids.ID, amount uint64) uint64 {
	// Burn any value that should be burned
	toBurn := min(
		s.toBurn[assetID], // Amount we still need to burn
		amount,            // Amount available to burn
	)
	s.toBurn[assetID] -= toBurn

	// Stake any remaining value that should be staked
	return s.consumeLockedAsset(assetID, amount-toBurn)
}

func (s *spendHelper) calculateFee() (uint64, error) {
	gas, err := s.complexity.ToGas(s.weights)
	if err != nil {
		return 0, err
	}
	return gas.Cost(s.gasPrice)
}

func (s *spendHelper) verifyAssetsConsumed() error {
	for assetID, amount := range s.toStake {
		if amount == 0 {
			continue
		}

		return fmt.Errorf(
			"%w: provided UTXOs need %d more units of asset %q to stake",
			ErrInsufficientFunds,
			amount,
			assetID,
		)
	}
	for assetID, amount := range s.toBurn {
		if amount == 0 {
			continue
		}

		return fmt.Errorf(
			"%w: provided UTXOs need %d more units of asset %q",
			ErrInsufficientFunds,
			amount,
			assetID,
		)
	}
	return nil
}

type utxosByLocktime struct {
	unlocked []*lux.UTXO
	locked   []*lux.UTXO
}

// splitByLocktime separates the provided UTXOs into two slices:
// 1. UTXOs that are unlocked with the provided issuance time
// 2. UTXOs that are locked with the provided issuance time
func splitByLocktime(utxos []*lux.UTXO, minIssuanceTime uint64) utxosByLocktime {
	split := utxosByLocktime{
		unlocked: make([]*lux.UTXO, 0, len(utxos)),
		locked:   make([]*lux.UTXO, 0, len(utxos)),
	}
	for _, utxo := range utxos {
		if lockedOut, ok := utxo.Out.(*stakeable.LockOut); ok && minIssuanceTime < lockedOut.Locktime {
			split.locked = append(split.locked, utxo)
		} else {
			split.unlocked = append(split.unlocked, utxo)
		}
	}
	return split
}

type utxosByAssetID struct {
	requested []*lux.UTXO
	other     []*lux.UTXO
}

// splitByAssetID separates the provided UTXOs into two slices:
// 1. UTXOs with the provided assetID
// 2. UTXOs with a different assetID
func splitByAssetID(utxos []*lux.UTXO, assetID ids.ID) utxosByAssetID {
	split := utxosByAssetID{
		requested: make([]*lux.UTXO, 0, len(utxos)),
		other:     make([]*lux.UTXO, 0, len(utxos)),
	}
	for _, utxo := range utxos {
		if utxo.AssetID() == assetID {
			split.requested = append(split.requested, utxo)
		} else {
			split.other = append(split.other, utxo)
		}
	}
	return split
}

// unwrapOutput returns the *secp256k1fx.TransferOutput that was, potentially,
// wrapped by a *stakeable.LockOut.
//
// If the output was stakeable and locked, the locktime is returned. Otherwise,
// the locktime returned will be 0.
//
// If the output is not a, potentially wrapped, *secp256k1fx.TransferOutput, an
// error is returned.
func unwrapOutput(output verify.State) (*secp256k1fx.TransferOutput, uint64, error) {
	var locktime uint64
	if lockedOut, ok := output.(*stakeable.LockOut); ok {
		output = lockedOut.TransferableOut
		locktime = lockedOut.Locktime
	}

	unwrappedOutput, ok := output.(*secp256k1fx.TransferOutput)
	if !ok {
		return nil, 0, ErrUnknownOutputType
	}
	return unwrappedOutput, locktime, nil
}
