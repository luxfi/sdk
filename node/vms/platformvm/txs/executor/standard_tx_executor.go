// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/luxfi/log"

	"github.com/luxfi/const"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/chains/atomic"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/math"
	"github.com/luxfi/sdk/node/vms/components/gas"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/platformvm/signer"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/platformvm/txs/fee"
	"github.com/luxfi/sdk/node/vms/platformvm/warp"
	"github.com/luxfi/sdk/node/vms/platformvm/warp/message"
	"github.com/luxfi/sdk/node/vms/platformvm/warp/payload"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

// TODO: Before Etna, ensure that the maximum number of expiries to track is
// limited to a reasonable number by this window.
const (
	second                            = 1
	minute                            = 60 * second
	hour                              = 60 * minute
	day                               = 24 * hour
	RegisterL1ValidatorTxExpiryWindow = day
)

var (
	_ txs.Visitor = (*standardTxExecutor)(nil)

	errEmptyNodeID                     = errors.New("validator nodeID cannot be empty")
	errMaxStakeDurationTooLarge        = errors.New("max stake duration must be less than or equal to the global max stake duration")
	errMissingStartTimePreDurango      = errors.New("staker transactions must have a StartTime pre-Durango")
	errEtnaUpgradeNotActive            = errors.New("attempting to use an Etna-upgrade feature prior to activation")
	errTransformChainTxPostEtna        = errors.New("TransformChainTx is not permitted post-Etna")
	errMaxNumActiveValidators          = errors.New("already at the max number of active validators")
	errCouldNotLoadChainToL1Conversion = errors.New("could not load chain conversion")
	errWrongWarpMessageSourceChainID   = errors.New("wrong warp message source chain ID")
	errWrongWarpMessageSourceAddress   = errors.New("wrong warp message source address")
	errWarpMessageExpired              = errors.New("warp message expired")
	errWarpMessageNotYetAllowed        = errors.New("warp message not yet allowed")
	errWarpMessageAlreadyIssued        = errors.New("warp message already issued")
	errCouldNotLoadL1Validator         = errors.New("could not load L1 validator")
	errWarpMessageContainsStaleNonce   = errors.New("warp message contains stale nonce")
	errRemovingLastValidator           = errors.New("attempting to remove the last L1 validator from a converted subnet")
	errStateCorruption                 = errors.New("state corruption")
)

// StandardTx executes the standard transaction [tx].
//
// [state] is modified to represent the state of the chain after the execution
// of [tx].
//
// Returns:
//   - The IDs of any import UTXOs consumed.
//   - The, potentially nil, atomic requests that should be performed against
//     shared memory when this transaction is accepted.
//   - A, potentially nil, function that should be called when this transaction
//     is accepted.
func StandardTx(
	backend *Backend,
	feeCalculator fee.Calculator,
	tx *txs.Tx,
	state state.Diff,
) (set.Set[ids.ID], map[ids.ID]*atomic.Requests, func(), error) {
	standardExecutor := standardTxExecutor{
		backend:       backend,
		feeCalculator: feeCalculator,
		tx:            tx,
		state:         state,
	}
	if err := tx.Unsigned.Visit(&standardExecutor); err != nil {
		txID := tx.ID()
		return nil, nil, nil, fmt.Errorf("standard tx %s failed execution: %w", txID, err)
	}
	return standardExecutor.inputs, standardExecutor.atomicRequests, standardExecutor.onAccept, nil
}

type standardTxExecutor struct {
	// inputs, to be filled before visitor methods are called
	backend       *Backend
	state         state.Diff // state is expected to be modified
	feeCalculator fee.Calculator
	tx            *txs.Tx

	// outputs of visitor execution
	onAccept       func() // may be nil
	inputs         set.Set[ids.ID]
	atomicRequests map[ids.ID]*atomic.Requests // may be nil
}

func (*standardTxExecutor) AdvanceTimeTx(*txs.AdvanceTimeTx) error {
	return ErrWrongTxType
}

func (*standardTxExecutor) RewardValidatorTx(*txs.RewardValidatorTx) error {
	return ErrWrongTxType
}

func (e *standardTxExecutor) AddValidatorTx(tx *txs.AddValidatorTx) error {
	if tx.Validator.NodeID == ids.EmptyNodeID {
		return errEmptyNodeID
	}

	if _, err := verifyAddValidatorTx(
		e.backend,
		e.feeCalculator,
		e.state,
		e.tx,
		tx,
	); err != nil {
		return err
	}

	if err := e.putStaker(tx); err != nil {
		return err
	}

	txID := e.tx.ID()
	lux.Consume(e.state, tx.Ins)
	lux.Produce(e.state, txID, tx.Outs)

	if e.backend.Config.PartialSyncPrimaryNetwork && tx.Validator.NodeID == e.backend.Ctx.NodeID {
		e.backend.Log.Warn("verified transaction that would cause this node to become unhealthy",
			log.String("reason", "primary network is not being fully synced"),
			log.Stringer("txID", txID),
			log.String("txType", "addValidator"),
			log.Stringer("nodeID", tx.Validator.NodeID),
		)
	}
	return nil
}

func (e *standardTxExecutor) AddChainValidatorTx(tx *txs.AddChainValidatorTx) error {
	if err := verifyAddChainValidatorTx(
		e.backend,
		e.feeCalculator,
		e.state,
		e.tx,
		tx,
	); err != nil {
		return err
	}

	if err := e.putStaker(tx); err != nil {
		return err
	}

	txID := e.tx.ID()
	lux.Consume(e.state, tx.Ins)
	lux.Produce(e.state, txID, tx.Outs)
	return nil
}

func (e *standardTxExecutor) AddDelegatorTx(tx *txs.AddDelegatorTx) error {
	if _, err := verifyAddDelegatorTx(
		e.backend,
		e.feeCalculator,
		e.state,
		e.tx,
		tx,
	); err != nil {
		return err
	}

	if err := e.putStaker(tx); err != nil {
		return err
	}

	txID := e.tx.ID()
	lux.Consume(e.state, tx.Ins)
	lux.Produce(e.state, txID, tx.Outs)
	return nil
}

func (e *standardTxExecutor) CreateChainTx(tx *txs.CreateChainTx) error {
	if err := e.tx.SyntacticVerify(e.backend.Ctx); err != nil {
		return err
	}

	var (
		currentTimestamp = e.state.GetTimestamp()
		isDurangoActive  = e.backend.Config.UpgradeConfig.IsDurangoActivated(currentTimestamp)
	)
	if err := lux.VerifyMemoFieldLength(tx.Memo, isDurangoActive); err != nil {
		return err
	}

	// Verify chain name uniqueness (case-insensitive)
	if tx.BlockchainName != "" && e.state.IsChainNameTaken(tx.BlockchainName) {
		return fmt.Errorf("chain name %q is already taken", tx.BlockchainName)
	}

	baseTxCreds, err := verifyPoASubnetAuthorization(e.backend.Fx, e.state, e.tx, tx.ChainID, tx.ChainAuth)
	if err != nil {
		return err
	}

	// Verify the flowcheck
	fee, err := e.feeCalculator.CalculateFee(tx)
	if err != nil {
		return err
	}
	if err := e.backend.FlowChecker.VerifySpend(
		tx,
		e.state,
		tx.Ins,
		tx.Outs,
		baseTxCreds,
		map[ids.ID]uint64{
			e.backend.Ctx.XAssetID: fee,
		},
	); err != nil {
		return err
	}

	txID := e.tx.ID()

	// Consume the UTXOS
	lux.Consume(e.state, tx.Ins)
	// Produce the UTXOS
	lux.Produce(e.state, txID, tx.Outs)
	// Add the new chain to the database
	e.state.AddChain(e.tx)

	// If this proposal is committed and this node is a member of the subnet
	// that validates the blockchain, create the blockchain
	e.onAccept = func() {
		e.backend.Config.CreateChain(txID, tx)
	}
	return nil
}

func (e *standardTxExecutor) CreateSubnetTx(tx *txs.CreateSubnetTx) error {
	// Make sure this transaction is well formed.
	if err := e.tx.SyntacticVerify(e.backend.Ctx); err != nil {
		return err
	}

	var (
		currentTimestamp = e.state.GetTimestamp()
		isDurangoActive  = e.backend.Config.UpgradeConfig.IsDurangoActivated(currentTimestamp)
	)
	if err := lux.VerifyMemoFieldLength(tx.Memo, isDurangoActive); err != nil {
		return err
	}

	// Verify the flowcheck
	fee, err := e.feeCalculator.CalculateFee(tx)
	if err != nil {
		return err
	}
	if err := e.backend.FlowChecker.VerifySpend(
		tx,
		e.state,
		tx.Ins,
		tx.Outs,
		e.tx.Creds,
		map[ids.ID]uint64{
			e.backend.Ctx.XAssetID: fee,
		},
	); err != nil {
		return err
	}

	txID := e.tx.ID()

	// Consume the UTXOS
	lux.Consume(e.state, tx.Ins)
	// Produce the UTXOS
	lux.Produce(e.state, txID, tx.Outs)
	// Add the new subnet to the database
	e.state.AddNet(txID)
	e.state.SetNetOwner(txID, tx.Owner)
	return nil
}

func (e *standardTxExecutor) ImportTx(tx *txs.ImportTx) error {
	if err := e.tx.SyntacticVerify(e.backend.Ctx); err != nil {
		return err
	}

	var (
		currentTimestamp = e.state.GetTimestamp()
		isDurangoActive  = e.backend.Config.UpgradeConfig.IsDurangoActivated(currentTimestamp)
	)
	if err := lux.VerifyMemoFieldLength(tx.Memo, isDurangoActive); err != nil {
		return err
	}

	e.inputs = set.NewSet[ids.ID](len(tx.ImportedInputs))
	utxoIDs := make([][]byte, len(tx.ImportedInputs))
	for i, in := range tx.ImportedInputs {
		utxoID := in.UTXOID.InputID()

		e.inputs.Add(utxoID)
		utxoIDs[i] = utxoID[:]
	}

	// Skip verification of the shared memory inputs if the other primary
	// network chains are not guaranteed to be up-to-date.
	var allUTXOBytes [][]byte
	if e.backend.Bootstrapped.Get() && !e.backend.Config.PartialSyncPrimaryNetwork {
		// TODO: Restore SameNet check once ChainContext types are aligned
		// if err := verify.SameNet(context.TODO(), e.backend.Ctx, tx.SourceChain); err != nil {
		// 	return err
		// }

		if e.backend.Ctx.SharedMemory != nil {
			if sm, ok := e.backend.Ctx.SharedMemory.(atomic.SharedMemory); ok {
				var err error
				allUTXOBytes, err = sm.Get(tx.SourceChain, utxoIDs)
				if err != nil {
					return fmt.Errorf("failed to get shared memory: %w", err)
				}
			}
		}

		utxos := make([]*lux.UTXO, len(tx.Ins)+len(tx.ImportedInputs))
		for index, input := range tx.Ins {
			utxo, err := e.state.GetUTXO(input.InputID())
			if err != nil {
				return fmt.Errorf("failed to get UTXO %s: %w", &input.UTXOID, err)
			}
			utxos[index] = utxo
		}
		for i, utxoBytes := range allUTXOBytes {
			utxo := &lux.UTXO{}
			if _, err := txs.Codec.Unmarshal(utxoBytes, utxo); err != nil {
				return fmt.Errorf("failed to unmarshal UTXO: %w", err)
			}
			utxos[i+len(tx.Ins)] = utxo
		}

		ins := make([]*lux.TransferableInput, len(tx.Ins)+len(tx.ImportedInputs))
		copy(ins, tx.Ins)
		copy(ins[len(tx.Ins):], tx.ImportedInputs)

		// Verify the flowcheck
		fee, err := e.feeCalculator.CalculateFee(tx)
		if err != nil {
			return err
		}
		if err := e.backend.FlowChecker.VerifySpendUTXOs(
			tx,
			utxos,
			ins,
			tx.Outs,
			e.tx.Creds,
			map[ids.ID]uint64{
				e.backend.Ctx.XAssetID: fee,
			},
		); err != nil {
			return err
		}
	}

	txID := e.tx.ID()

	// Consume the UTXOS
	lux.Consume(e.state, tx.Ins)
	// Produce the UTXOS
	lux.Produce(e.state, txID, tx.Outs)

	// Note: We apply atomic requests even if we are not verifying atomic
	// requests to ensure the shared state will be correct if we later start
	// verifying the requests.
	e.atomicRequests = map[ids.ID]*atomic.Requests{
		tx.SourceChain: {
			RemoveRequests: utxoIDs,
		},
	}
	return nil
}

func (e *standardTxExecutor) ExportTx(tx *txs.ExportTx) error {
	if err := e.tx.SyntacticVerify(e.backend.Ctx); err != nil {
		return err
	}

	var (
		currentTimestamp = e.state.GetTimestamp()
		isDurangoActive  = e.backend.Config.UpgradeConfig.IsDurangoActivated(currentTimestamp)
	)
	if err := lux.VerifyMemoFieldLength(tx.Memo, isDurangoActive); err != nil {
		return err
	}

	outs := make([]*lux.TransferableOutput, len(tx.Outs)+len(tx.ExportedOutputs))
	copy(outs, tx.Outs)
	copy(outs[len(tx.Outs):], tx.ExportedOutputs)

	if e.backend.Bootstrapped.Get() {
		// TODO: Fix ChainContext type mismatch
		// if err := verify.SameNet(context.TODO(), e.backend.Ctx, tx.DestinationChain); err != nil {
		// 	return err
		// }
	}

	// Verify the flowcheck
	fee, err := e.feeCalculator.CalculateFee(tx)
	if err != nil {
		return err
	}
	if err := e.backend.FlowChecker.VerifySpend(
		tx,
		e.state,
		tx.Ins,
		outs,
		e.tx.Creds,
		map[ids.ID]uint64{
			e.backend.Ctx.XAssetID: fee,
		},
	); err != nil {
		return fmt.Errorf("failed verifySpend: %w", err)
	}

	txID := e.tx.ID()

	// Consume the UTXOS
	lux.Consume(e.state, tx.Ins)
	// Produce the UTXOS
	lux.Produce(e.state, txID, tx.Outs)

	// Note: We apply atomic requests even if we are not verifying atomic
	// requests to ensure the shared state will be correct if we later start
	// verifying the requests.
	elems := make([]*atomic.Element, len(tx.ExportedOutputs))
	for i, out := range tx.ExportedOutputs {
		utxo := &lux.UTXO{
			UTXOID: lux.UTXOID{
				TxID:        txID,
				OutputIndex: uint32(len(tx.Outs) + i),
			},
			Asset: lux.Asset{ID: out.AssetID()},
			Out:   out.Out,
		}

		utxoBytes, err := txs.Codec.Marshal(txs.CodecVersion, utxo)
		if err != nil {
			return fmt.Errorf("failed to marshal UTXO: %w", err)
		}
		utxoID := utxo.InputID()
		elem := &atomic.Element{
			Key:   utxoID[:],
			Value: utxoBytes,
		}
		if out, ok := utxo.Out.(lux.Addressable); ok {
			elem.Traits = out.Addresses()
		}

		elems[i] = elem
	}
	e.atomicRequests = map[ids.ID]*atomic.Requests{
		tx.DestinationChain: {
			PutRequests: elems,
		},
	}
	return nil
}

// Verifies a [*txs.RemoveChainValidatorTx] and, if it passes, executes it on
// [e.State]. For verification rules, see [verifyRemoveChainValidatorTx]. This
// transaction will result in [tx.NodeID] being removed as a validator of
// [tx.ChainID].
// Note: [tx.NodeID] may be either a current or pending validator.
func (e *standardTxExecutor) RemoveChainValidatorTx(tx *txs.RemoveChainValidatorTx) error {
	staker, isCurrentValidator, err := verifyRemoveChainValidatorTx(
		e.backend,
		e.feeCalculator,
		e.state,
		e.tx,
		tx,
	)
	if err != nil {
		return err
	}

	if isCurrentValidator {
		e.state.DeleteCurrentValidator(staker)
	} else {
		e.state.DeletePendingValidator(staker)
	}

	// Invariant: There are no permissioned net delegators to remove.

	txID := e.tx.ID()
	lux.Consume(e.state, tx.Ins)
	lux.Produce(e.state, txID, tx.Outs)

	return nil
}

func (e *standardTxExecutor) TransformChainTx(tx *txs.TransformChainTx) error {
	currentTimestamp := e.state.GetTimestamp()
	if e.backend.Config.UpgradeConfig.IsEtnaActivated(currentTimestamp) {
		return errTransformChainTxPostEtna
	}

	if err := e.tx.SyntacticVerify(e.backend.Ctx); err != nil {
		return err
	}

	isDurangoActive := e.backend.Config.UpgradeConfig.IsDurangoActivated(currentTimestamp)
	if err := lux.VerifyMemoFieldLength(tx.Memo, isDurangoActive); err != nil {
		return err
	}

	// Note: math.MaxInt32 * time.Second < math.MaxInt64 - so this can never
	// overflow.
	if time.Duration(tx.MaxStakeDuration)*time.Second > e.backend.Config.MaxStakeDuration {
		return errMaxStakeDurationTooLarge
	}

	baseTxCreds, err := verifyPoASubnetAuthorization(e.backend.Fx, e.state, e.tx, tx.Chain, tx.ChainAuth)
	if err != nil {
		return err
	}

	// Verify the flowcheck
	fee, err := e.feeCalculator.CalculateFee(tx)
	if err != nil {
		return err
	}
	totalRewardAmount := tx.MaximumSupply - tx.InitialSupply
	if err := e.backend.FlowChecker.VerifySpend(
		tx,
		e.state,
		tx.Ins,
		tx.Outs,
		baseTxCreds,
		// Invariant: [tx.AssetID != e.XAssetID]. This prevents the first
		//            entry in this map literal from being overwritten by the
		//            second entry.
		map[ids.ID]uint64{
			e.backend.Ctx.XAssetID: fee,
			tx.AssetID:               totalRewardAmount,
		},
	); err != nil {
		return err
	}

	txID := e.tx.ID()

	// Consume the UTXOS
	lux.Consume(e.state, tx.Ins)
	// Produce the UTXOS
	lux.Produce(e.state, txID, tx.Outs)
	// Transform the new subnet in the database
	e.state.AddNetTransformation(e.tx)
	e.state.SetCurrentSupply(tx.Chain, tx.InitialSupply)
	return nil
}

func (e *standardTxExecutor) AddPermissionlessValidatorTx(tx *txs.AddPermissionlessValidatorTx) error {
	if err := verifyAddPermissionlessValidatorTx(
		e.backend,
		e.feeCalculator,
		e.state,
		e.tx,
		tx,
	); err != nil {
		return err
	}

	if err := e.putStaker(tx); err != nil {
		return err
	}

	txID := e.tx.ID()
	lux.Consume(e.state, tx.Ins)
	lux.Produce(e.state, txID, tx.Outs)

	if e.backend.Config.PartialSyncPrimaryNetwork &&
		tx.Chain == constants.PrimaryNetworkID &&
		tx.Validator.NodeID == e.backend.Ctx.NodeID {
		e.backend.Log.Warn("verified transaction that would cause this node to become unhealthy",
			log.String("reason", "primary network is not being fully synced"),
			log.Stringer("txID", txID),
			log.String("txType", "addPermissionlessValidator"),
			log.Stringer("nodeID", tx.Validator.NodeID),
		)
	}

	return nil
}

func (e *standardTxExecutor) AddPermissionlessDelegatorTx(tx *txs.AddPermissionlessDelegatorTx) error {
	if err := verifyAddPermissionlessDelegatorTx(
		e.backend,
		e.feeCalculator,
		e.state,
		e.tx,
		tx,
	); err != nil {
		return err
	}

	if err := e.putStaker(tx); err != nil {
		return err
	}

	txID := e.tx.ID()
	lux.Consume(e.state, tx.Ins)
	lux.Produce(e.state, txID, tx.Outs)
	return nil
}

// Verifies a [*txs.TransferChainOwnershipTx] and, if it passes, executes it on
// [e.State]. For verification rules, see [verifyTransferChainOwnershipTx].
// This transaction will result in the ownership of [tx.Chain] being transferred
// to [tx.Owner].
func (e *standardTxExecutor) TransferChainOwnershipTx(tx *txs.TransferChainOwnershipTx) error {
	err := verifyTransferChainOwnershipTx(
		e.backend,
		e.feeCalculator,
		e.state,
		e.tx,
		tx,
	)
	if err != nil {
		return err
	}

	e.state.SetNetOwner(tx.Chain, tx.Owner)

	txID := e.tx.ID()
	lux.Consume(e.state, tx.Ins)
	lux.Produce(e.state, txID, tx.Outs)
	return nil
}

func (e *standardTxExecutor) BaseTx(tx *txs.BaseTx) error {
	var (
		currentTimestamp = e.state.GetTimestamp()
		upgrades         = e.backend.Config.UpgradeConfig
	)
	if !upgrades.IsDurangoActivated(currentTimestamp) {
		return ErrDurangoUpgradeNotActive
	}

	// Verify the tx is well-formed
	if err := e.tx.SyntacticVerify(e.backend.Ctx); err != nil {
		return err
	}

	if err := lux.VerifyMemoFieldLength(tx.Memo, true /*=isDurangoActive*/); err != nil {
		return err
	}

	// Verify the flowcheck
	fee, err := e.feeCalculator.CalculateFee(tx)
	if err != nil {
		return err
	}
	if err := e.backend.FlowChecker.VerifySpend(
		tx,
		e.state,
		tx.Ins,
		tx.Outs,
		e.tx.Creds,
		map[ids.ID]uint64{
			e.backend.Ctx.XAssetID: fee,
		},
	); err != nil {
		return err
	}

	txID := e.tx.ID()
	// Consume the UTXOS
	lux.Consume(e.state, tx.Ins)
	// Produce the UTXOS
	lux.Produce(e.state, txID, tx.Outs)
	return nil
}

func (e *standardTxExecutor) ConvertChainToL1Tx(tx *txs.ConvertChainToL1Tx) error {
	var (
		currentTimestamp = e.state.GetTimestamp()
		upgrades         = e.backend.Config.UpgradeConfig
	)
	if !upgrades.IsEtnaActivated(currentTimestamp) {
		return errEtnaUpgradeNotActive
	}

	if err := e.tx.SyntacticVerify(e.backend.Ctx); err != nil {
		return err
	}

	if err := lux.VerifyMemoFieldLength(tx.Memo, true /*=isDurangoActive*/); err != nil {
		return err
	}

	baseTxCreds, err := verifyPoASubnetAuthorization(e.backend.Fx, e.state, e.tx, tx.Chain, tx.ChainAuth)
	if err != nil {
		return err
	}

	// Verify the flowcheck
	fee, err := e.feeCalculator.CalculateFee(tx)
	if err != nil {
		return err
	}

	var (
		startTime                = uint64(currentTimestamp.Unix())
		currentFees              = e.state.GetAccruedFees()
		subnetToL1ConversionData = message.ChainToL1ConversionData{
			ChainID:        tx.Chain,
			ManagerChainID: tx.ManagerChainID,
			ManagerAddress: tx.Address,
			Validators:     make([]message.ChainToL1ConversionValidatorData, len(tx.Validators)),
		}
	)
	for i, vdr := range tx.Validators {
		nodeID, err := ids.ToNodeID(vdr.NodeID)
		if err != nil {
			return err
		}

		remainingBalanceOwner, err := txs.Codec.Marshal(txs.CodecVersion, &vdr.RemainingBalanceOwner)
		if err != nil {
			return err
		}
		deactivationOwner, err := txs.Codec.Marshal(txs.CodecVersion, &vdr.DeactivationOwner)
		if err != nil {
			return err
		}

		l1Validator := state.L1Validator{
			ValidationID:          tx.Chain.Append(uint32(i)),
			ChainID:               tx.Chain,
			NodeID:                nodeID,
			PublicKey:             bls.PublicKeyToUncompressedBytes(vdr.Signer.Key()),
			RemainingBalanceOwner: remainingBalanceOwner,
			DeactivationOwner:     deactivationOwner,
			StartTime:             startTime,
			Weight:                vdr.Weight,
			MinNonce:              0,
			EndAccumulatedFee:     0, // If Balance is 0, this is 0
		}
		if vdr.Balance != 0 {
			// We are attempting to add an active validator
			if gas.Gas(e.state.NumActiveL1Validators()) >= e.backend.Config.ValidatorFeeConfig.Capacity {
				return errMaxNumActiveValidators
			}

			l1Validator.EndAccumulatedFee, err = math.Add(vdr.Balance, currentFees)
			if err != nil {
				return err
			}

			fee, err = math.Add(fee, vdr.Balance)
			if err != nil {
				return err
			}
		}

		if err := e.state.PutL1Validator(l1Validator); err != nil {
			return err
		}

		subnetToL1ConversionData.Validators[i] = message.ChainToL1ConversionValidatorData{
			NodeID:       vdr.NodeID,
			BLSPublicKey: vdr.Signer.PublicKey,
			Weight:       vdr.Weight,
		}
	}
	if err := e.backend.FlowChecker.VerifySpend(
		tx,
		e.state,
		tx.Ins,
		tx.Outs,
		baseTxCreds,
		map[ids.ID]uint64{
			e.backend.Ctx.XAssetID: fee,
		},
	); err != nil {
		return err
	}

	conversionID, err := message.ChainToL1ConversionID(subnetToL1ConversionData)
	if err != nil {
		return err
	}

	txID := e.tx.ID()

	// Consume the UTXOS
	lux.Consume(e.state, tx.Ins)
	// Produce the UTXOS
	lux.Produce(e.state, txID, tx.Outs)
	// Track the subnet conversion in the database
	e.state.SetNetToL1Conversion(
		tx.Chain,
		state.NetToL1Conversion{
			ConversionID: conversionID,
			ChainID:      tx.ManagerChainID,
			Addr:         tx.Address,
		},
	)
	return nil
}

func (e *standardTxExecutor) RegisterL1ValidatorTx(tx *txs.RegisterL1ValidatorTx) error {
	var (
		currentTimestamp = e.state.GetTimestamp()
		upgrades         = e.backend.Config.UpgradeConfig
	)
	if !upgrades.IsEtnaActivated(currentTimestamp) {
		return errEtnaUpgradeNotActive
	}

	if err := e.tx.SyntacticVerify(e.backend.Ctx); err != nil {
		return err
	}

	if err := lux.VerifyMemoFieldLength(tx.Memo, true /*=isDurangoActive*/); err != nil {
		return err
	}

	// Verify the flowcheck
	fee, err := e.feeCalculator.CalculateFee(tx)
	if err != nil {
		return err
	}
	fee, err = math.Add(fee, tx.Balance)
	if err != nil {
		return err
	}

	if err := e.backend.FlowChecker.VerifySpend(
		tx,
		e.state,
		tx.Ins,
		tx.Outs,
		e.tx.Creds,
		map[ids.ID]uint64{
			e.backend.Ctx.XAssetID: fee,
		},
	); err != nil {
		return err
	}

	// Parse the warp message.
	warpMessage, err := warp.ParseMessage(tx.Message)
	if err != nil {
		return err
	}
	addressedCall, err := payload.ParseAddressedCall(warpMessage.Payload)
	if err != nil {
		return err
	}
	msg, err := message.ParseRegisterL1Validator(addressedCall.Payload)
	if err != nil {
		return err
	}
	if err := msg.Verify(); err != nil {
		return err
	}

	// Verify that the warp message was sent from the expected chain and
	// address.
	if err := verifyL1Conversion(e.state, msg.ChainID, warpMessage.SourceChainID, addressedCall.SourceAddress); err != nil {
		return err
	}

	// Verify that the message contains a valid expiry time.
	currentTimestampUnix := uint64(currentTimestamp.Unix())
	if msg.Expiry <= currentTimestampUnix {
		return fmt.Errorf("%w at %d and it is currently %d", errWarpMessageExpired, msg.Expiry, currentTimestampUnix)
	}
	if secondsUntilExpiry := msg.Expiry - currentTimestampUnix; secondsUntilExpiry > RegisterL1ValidatorTxExpiryWindow {
		return fmt.Errorf("%w because time is %d seconds in the future but the limit is %d", errWarpMessageNotYetAllowed, secondsUntilExpiry, RegisterL1ValidatorTxExpiryWindow)
	}

	// Verify that this warp message isn't being replayed.
	validationID := msg.ValidationID()
	expiry := state.ExpiryEntry{
		Timestamp:    msg.Expiry,
		ValidationID: validationID,
	}
	isDuplicate, err := e.state.HasExpiry(expiry)
	if err != nil {
		return err
	}
	if isDuplicate {
		return fmt.Errorf("%w for validationID %s", errWarpMessageAlreadyIssued, validationID)
	}

	// Verify proof of possession provided by the transaction against the public
	// key provided by the warp message.
	pop := signer.ProofOfPossession{
		PublicKey:         msg.BLSPublicKey,
		ProofOfPossession: tx.ProofOfPossession,
	}
	if err := pop.Verify(); err != nil {
		return err
	}

	// Create the L1 validator.
	nodeID, err := ids.ToNodeID(msg.NodeID)
	if err != nil {
		return err
	}
	remainingBalanceOwner, err := txs.Codec.Marshal(txs.CodecVersion, &msg.RemainingBalanceOwner)
	if err != nil {
		return err
	}
	deactivationOwner, err := txs.Codec.Marshal(txs.CodecVersion, &msg.DisableOwner)
	if err != nil {
		return err
	}
	l1Validator := state.L1Validator{
		ValidationID:          validationID,
		ChainID:               msg.ChainID,
		NodeID:                nodeID,
		PublicKey:             bls.PublicKeyToUncompressedBytes(pop.Key()),
		RemainingBalanceOwner: remainingBalanceOwner,
		DeactivationOwner:     deactivationOwner,
		StartTime:             currentTimestampUnix,
		Weight:                msg.Weight,
		MinNonce:              0,
		EndAccumulatedFee:     0, // If Balance is 0, this is will remain 0
	}

	// If the balance is non-zero, this validator should be initially active.
	if tx.Balance != 0 {
		// Verify that there is space for an active validator.
		if gas.Gas(e.state.NumActiveL1Validators()) >= e.backend.Config.ValidatorFeeConfig.Capacity {
			return errMaxNumActiveValidators
		}

		// Mark the validator as active.
		currentFees := e.state.GetAccruedFees()
		l1Validator.EndAccumulatedFee, err = math.Add(tx.Balance, currentFees)
		if err != nil {
			return err
		}
	}

	if err := e.state.PutL1Validator(l1Validator); err != nil {
		return err
	}

	txID := e.tx.ID()

	// Consume the UTXOS
	lux.Consume(e.state, tx.Ins)
	// Produce the UTXOS
	lux.Produce(e.state, txID, tx.Outs)
	// Prevent this warp message from being replayed
	e.state.PutExpiry(expiry)
	return nil
}

func (e *standardTxExecutor) SetL1ValidatorWeightTx(tx *txs.SetL1ValidatorWeightTx) error {
	var (
		currentTimestamp = e.state.GetTimestamp()
		upgrades         = e.backend.Config.UpgradeConfig
	)
	if !upgrades.IsEtnaActivated(currentTimestamp) {
		return errEtnaUpgradeNotActive
	}

	if err := e.tx.SyntacticVerify(e.backend.Ctx); err != nil {
		return err
	}

	if err := lux.VerifyMemoFieldLength(tx.Memo, true /*=isDurangoActive*/); err != nil {
		return err
	}

	// Verify the flowcheck
	fee, err := e.feeCalculator.CalculateFee(tx)
	if err != nil {
		return err
	}

	if err := e.backend.FlowChecker.VerifySpend(
		tx,
		e.state,
		tx.Ins,
		tx.Outs,
		e.tx.Creds,
		map[ids.ID]uint64{
			e.backend.Ctx.XAssetID: fee,
		},
	); err != nil {
		return err
	}

	// Parse the warp message.
	warpMessage, err := warp.ParseMessage(tx.Message)
	if err != nil {
		return err
	}
	addressedCall, err := payload.ParseAddressedCall(warpMessage.Payload)
	if err != nil {
		return err
	}
	msg, err := message.ParseL1ValidatorWeight(addressedCall.Payload)
	if err != nil {
		return err
	}
	if err := msg.Verify(); err != nil {
		return err
	}

	// Verify that the message contains a valid nonce for a current validator.
	l1Validator, err := e.state.GetL1Validator(msg.ValidationID)
	if err != nil {
		return fmt.Errorf("%w: %w", errCouldNotLoadL1Validator, err)
	}
	if msg.Nonce < l1Validator.MinNonce {
		return fmt.Errorf("%w %d must be at least %d", errWarpMessageContainsStaleNonce, msg.Nonce, l1Validator.MinNonce)
	}

	// Verify that the warp message was sent from the expected chain and
	// address.
	if err := verifyL1Conversion(e.state, l1Validator.ChainID, warpMessage.SourceChainID, addressedCall.SourceAddress); err != nil {
		return err
	}

	txID := e.tx.ID()

	// Check if we are removing the validator.
	if msg.Weight == 0 {
		// Verify that we are not removing the last validator.
		weight, err := e.state.WeightOfL1Validators(l1Validator.ChainID)
		if err != nil {
			return fmt.Errorf("could not load L1 validator weights: %w", err)
		}
		if weight == l1Validator.Weight {
			return errRemovingLastValidator
		}

		// If the validator is currently active, we need to refund the remaining
		// balance.
		if l1Validator.EndAccumulatedFee != 0 {
			var remainingBalanceOwner message.PChainOwner
			if _, err := txs.Codec.Unmarshal(l1Validator.RemainingBalanceOwner, &remainingBalanceOwner); err != nil {
				return fmt.Errorf("%w: remaining balance owner is malformed", errStateCorruption)
			}

			accruedFees := e.state.GetAccruedFees()
			if l1Validator.EndAccumulatedFee <= accruedFees {
				// This check should be unreachable. However, it prevents LUX
				// from being minted due to state corruption. This also prevents
				// invalid UTXOs from being created (with 0 value).
				return fmt.Errorf("%w: validator should have already been disabled", errStateCorruption)
			}
			remainingBalance := l1Validator.EndAccumulatedFee - accruedFees

			utxo := &lux.UTXO{
				UTXOID: lux.UTXOID{
					TxID:        txID,
					OutputIndex: uint32(len(tx.Outs)),
				},
				Asset: lux.Asset{
					ID: e.backend.Ctx.XAssetID,
				},
				Out: &secp256k1fx.TransferOutput{
					Amt: remainingBalance,
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: remainingBalanceOwner.Threshold,
						Addrs:     remainingBalanceOwner.Addresses,
					},
				},
			}
			e.state.AddUTXO(utxo)
		}
	}

	// If the weight is being set to 0, it is possible for the nonce increment
	// to overflow. However, the validator is being removed and the nonce
	// doesn't matter. If weight is not 0, [msg.Nonce] is enforced by
	// [msg.Verify()] to be less than MaxUInt64 and can therefore be incremented
	// without overflow.
	l1Validator.MinNonce = msg.Nonce + 1
	l1Validator.Weight = msg.Weight
	if err := e.state.PutL1Validator(l1Validator); err != nil {
		return err
	}

	// Consume the UTXOS
	lux.Consume(e.state, tx.Ins)
	// Produce the UTXOS
	lux.Produce(e.state, txID, tx.Outs)
	return nil
}

func (e *standardTxExecutor) IncreaseL1ValidatorBalanceTx(tx *txs.IncreaseL1ValidatorBalanceTx) error {
	var (
		currentTimestamp = e.state.GetTimestamp()
		upgrades         = e.backend.Config.UpgradeConfig
	)
	if !upgrades.IsEtnaActivated(currentTimestamp) {
		return errEtnaUpgradeNotActive
	}

	if err := e.tx.SyntacticVerify(e.backend.Ctx); err != nil {
		return err
	}

	if err := lux.VerifyMemoFieldLength(tx.Memo, true /*=isDurangoActive*/); err != nil {
		return err
	}

	// Verify the flowcheck
	fee, err := e.feeCalculator.CalculateFee(tx)
	if err != nil {
		return err
	}

	fee, err = math.Add(fee, tx.Balance)
	if err != nil {
		return err
	}

	if err := e.backend.FlowChecker.VerifySpend(
		tx,
		e.state,
		tx.Ins,
		tx.Outs,
		e.tx.Creds,
		map[ids.ID]uint64{
			e.backend.Ctx.XAssetID: fee,
		},
	); err != nil {
		return err
	}

	l1Validator, err := e.state.GetL1Validator(tx.ValidationID)
	if err != nil {
		return err
	}

	// If the validator is currently inactive, we are activating it.
	if l1Validator.EndAccumulatedFee == 0 {
		if gas.Gas(e.state.NumActiveL1Validators()) >= e.backend.Config.ValidatorFeeConfig.Capacity {
			return errMaxNumActiveValidators
		}

		l1Validator.EndAccumulatedFee = e.state.GetAccruedFees()
	}
	l1Validator.EndAccumulatedFee, err = math.Add(l1Validator.EndAccumulatedFee, tx.Balance)
	if err != nil {
		return err
	}

	if err := e.state.PutL1Validator(l1Validator); err != nil {
		return err
	}

	txID := e.tx.ID()

	// Consume the UTXOS
	lux.Consume(e.state, tx.Ins)
	// Produce the UTXOS
	lux.Produce(e.state, txID, tx.Outs)
	return nil
}

func (e *standardTxExecutor) DisableL1ValidatorTx(tx *txs.DisableL1ValidatorTx) error {
	var (
		currentTimestamp = e.state.GetTimestamp()
		upgrades         = e.backend.Config.UpgradeConfig
	)
	if !upgrades.IsEtnaActivated(currentTimestamp) {
		return errEtnaUpgradeNotActive
	}

	if err := e.tx.SyntacticVerify(e.backend.Ctx); err != nil {
		return err
	}

	if err := lux.VerifyMemoFieldLength(tx.Memo, true /*=isDurangoActive*/); err != nil {
		return err
	}

	l1Validator, err := e.state.GetL1Validator(tx.ValidationID)
	if err != nil {
		return fmt.Errorf("%w: %w", errCouldNotLoadL1Validator, err)
	}

	var disableOwner message.PChainOwner
	if _, err := txs.Codec.Unmarshal(l1Validator.DeactivationOwner, &disableOwner); err != nil {
		return err
	}

	baseTxCreds, err := verifyAuthorization(
		e.backend.Fx,
		e.tx,
		&secp256k1fx.OutputOwners{
			Threshold: disableOwner.Threshold,
			Addrs:     disableOwner.Addresses,
		},
		tx.DisableAuth,
	)
	if err != nil {
		return err
	}

	// Verify the flowcheck
	fee, err := e.feeCalculator.CalculateFee(tx)
	if err != nil {
		return err
	}

	if err := e.backend.FlowChecker.VerifySpend(
		tx,
		e.state,
		tx.Ins,
		tx.Outs,
		baseTxCreds,
		map[ids.ID]uint64{
			e.backend.Ctx.XAssetID: fee,
		},
	); err != nil {
		return err
	}

	txID := e.tx.ID()

	// Consume the UTXOS
	lux.Consume(e.state, tx.Ins)
	// Produce the UTXOS
	lux.Produce(e.state, txID, tx.Outs)

	// If the validator is already disabled, there is nothing to do.
	if l1Validator.EndAccumulatedFee == 0 {
		return nil
	}

	var remainingBalanceOwner message.PChainOwner
	if _, err := txs.Codec.Unmarshal(l1Validator.RemainingBalanceOwner, &remainingBalanceOwner); err != nil {
		return err
	}

	accruedFees := e.state.GetAccruedFees()
	if l1Validator.EndAccumulatedFee <= accruedFees {
		// This check should be unreachable. However, including it ensures
		// that LUX can't get minted out of thin air due to state
		// corruption.
		return fmt.Errorf("%w: validator should have already been disabled", errStateCorruption)
	}
	remainingBalance := l1Validator.EndAccumulatedFee - accruedFees

	utxo := &lux.UTXO{
		UTXOID: lux.UTXOID{
			TxID:        txID,
			OutputIndex: uint32(len(tx.Outs)),
		},
		Asset: lux.Asset{
			ID: e.backend.Ctx.XAssetID,
		},
		Out: &secp256k1fx.TransferOutput{
			Amt: remainingBalance,
			OutputOwners: secp256k1fx.OutputOwners{
				Threshold: remainingBalanceOwner.Threshold,
				Addrs:     remainingBalanceOwner.Addresses,
			},
		},
	}
	e.state.AddUTXO(utxo)

	// Disable the validator
	l1Validator.EndAccumulatedFee = 0
	return e.state.PutL1Validator(l1Validator)
}

// Creates the staker as defined in [stakerTx] and adds it to [e.State].
func (e *standardTxExecutor) putStaker(stakerTx txs.Staker) error {
	var (
		chainTime = e.state.GetTimestamp()
		txID      = e.tx.ID()
		staker    *state.Staker
		err       error
	)

	if !e.backend.Config.UpgradeConfig.IsDurangoActivated(chainTime) {
		// Pre-Durango, stakers set a future [StartTime] and are added to the
		// pending staker set. They are promoted to the current staker set once
		// the chain time reaches [StartTime].
		scheduledStakerTx, ok := stakerTx.(txs.ScheduledStaker)
		if !ok {
			return fmt.Errorf("%w: %T", errMissingStartTimePreDurango, stakerTx)
		}
		staker, err = state.NewPendingStaker(txID, scheduledStakerTx)
	} else {
		// Only calculate the potentialReward for permissionless stakers.
		// Recall that we only need to check if this is a permissioned
		// validator as there are no permissioned delegators
		var potentialReward uint64
		if !stakerTx.CurrentPriority().IsPermissionedValidator() {
			subnetID := stakerTx.ChainID()
			currentSupply, err := e.state.GetCurrentSupply(subnetID)
			if err != nil {
				return err
			}

			rewards, err := GetRewardsCalculator(e.backend, e.state, subnetID)
			if err != nil {
				return err
			}

			// Post-Durango, stakers are immediately added to the current staker
			// set. Their [StartTime] is the current chain time.
			stakeDuration := stakerTx.EndTime().Sub(chainTime)
			potentialReward = rewards.Calculate(
				stakeDuration,
				stakerTx.Weight(),
				currentSupply,
			)

			e.state.SetCurrentSupply(subnetID, currentSupply+potentialReward)
		}

		staker, err = state.NewCurrentStaker(txID, stakerTx, chainTime, potentialReward)
	}
	if err != nil {
		return err
	}

	switch priority := staker.Priority; {
	case priority.IsCurrentValidator():
		if err := e.state.PutCurrentValidator(staker); err != nil {
			return err
		}
	case priority.IsCurrentDelegator():
		e.state.PutCurrentDelegator(staker)
	case priority.IsPendingValidator():
		if err := e.state.PutPendingValidator(staker); err != nil {
			return err
		}
	case priority.IsPendingDelegator():
		e.state.PutPendingDelegator(staker)
	default:
		return fmt.Errorf("staker %s, unexpected priority %d", staker.TxID, priority)
	}
	return nil
}

// verifyL1Conversion verifies that the L1 conversion of [subnetID] references
// the [expectedChainID] and [expectedAddress].
func verifyL1Conversion(
	state state.Chain,
	subnetID ids.ID,
	expectedChainID ids.ID,
	expectedAddress []byte,
) error {
	subnetToL1Conversion, err := state.GetNetToL1Conversion(subnetID)
	if err != nil {
		return fmt.Errorf("%w for %s with: %w", errCouldNotLoadChainToL1Conversion, subnetID, err)
	}
	if expectedChainID != subnetToL1Conversion.ChainID {
		return fmt.Errorf("%w expected %s but had %s", errWrongWarpMessageSourceChainID, subnetToL1Conversion.ChainID, expectedChainID)
	}
	if !bytes.Equal(expectedAddress, subnetToL1Conversion.Addr) {
		return fmt.Errorf("%w expected 0x%x but got 0x%x", errWrongWarpMessageSourceAddress, subnetToL1Conversion.Addr, expectedAddress)
	}
	return nil
}
