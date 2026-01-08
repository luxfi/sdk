// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package utxo

import (
	"context"
	"errors"
	"fmt"


	"github.com/luxfi/ids"
	"github.com/luxfi/crypto/secp256k1"
	hashing "github.com/luxfi/crypto/hash"
	safemath "github.com/luxfi/math"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/utils/timer/mockable"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/components/verify"
	"github.com/luxfi/sdk/node/vms/platformvm/fx"
	"github.com/luxfi/sdk/node/vms/platformvm/stakeable"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

var (
	_ Handler = (*handler)(nil)

	ErrInsufficientFunds            = errors.New("insufficient funds")
	ErrInsufficientUnlockedFunds    = errors.New("insufficient unlocked funds")
	ErrInsufficientLockedFunds      = errors.New("insufficient locked funds")
	errWrongNumberCredentials       = errors.New("wrong number of credentials")
	errWrongNumberUTXOs             = errors.New("wrong number of UTXOs")
	errAssetIDMismatch              = errors.New("input asset ID does not match UTXO asset ID")
	errLocktimeMismatch             = errors.New("input locktime does not match UTXO locktime")
	errCantSign                     = errors.New("can't sign")
	errLockedFundsNotMarkedAsLocked = errors.New("locked funds not marked as locked")
)

// min returns the minimum of two uint64 values
func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// TODO: Stake and Authorize should be replaced by similar methods in the
// P-chain wallet
type Spender interface {
	// Spend the provided amount while deducting the provided fee.
	// Arguments:
	// - [keys] are the owners of the funds
	// - [amount] is the amount of funds that are trying to be staked
	// - [fee] is the amount of LUX that should be burned
	// - [changeAddr] is the address that change, if there is any, is sent to
	// Returns:
	// - [inputs] the inputs that should be consumed to fund the outputs
	// - [returnedOutputs] the outputs that should be immediately returned to
	//                     the UTXO set
	// - [stakedOutputs] the outputs that should be locked for the duration of
	//                   the staking period
	// - [signers] the proof of ownership of the funds being moved
	Spend(
		utxoReader lux.UTXOReader,
		keys []*secp256k1.PrivateKey,
		amount uint64,
		fee uint64,
		changeAddr ids.ShortID,
	) (
		[]*lux.TransferableInput, // inputs
		[]*lux.TransferableOutput, // returnedOutputs
		[]*lux.TransferableOutput, // stakedOutputs
		[][]*secp256k1.PrivateKey, // signers
		error,
	)

	// Authorize an operation on behalf of the named net with the provided
	// keys.
	Authorize(
		state state.Chain,
		netID ids.ID,
		keys []*secp256k1.PrivateKey,
	) (
		verify.Verifiable, // Input that names owners
		[]*secp256k1.PrivateKey, // Keys that prove ownership
		error,
	)
}

type Verifier interface {
	// Verify that [tx] is semantically valid.
	// [ins] and [outs] are the inputs and outputs of [tx].
	// [creds] are the credentials of [tx], which allow [ins] to be spent.
	// [unlockedProduced] is the map of assets that were produced and their
	// amounts.
	// The [ins] must have at least [unlockedProduced] than the [outs].
	//
	// Precondition: [tx] has already been syntactically verified.
	//
	// Note: [unlockedProduced] is modified by this method.
	VerifySpend(
		tx txs.UnsignedTx,
		utxoDB lux.UTXOGetter,
		ins []*lux.TransferableInput,
		outs []*lux.TransferableOutput,
		creds []verify.Verifiable,
		unlockedProduced map[ids.ID]uint64,
	) error

	// Verify that [tx] is semantically valid.
	// [utxos[i]] is the UTXO being consumed by [ins[i]].
	// [ins] and [outs] are the inputs and outputs of [tx].
	// [creds] are the credentials of [tx], which allow [ins] to be spent.
	// [unlockedProduced] is the map of assets that were produced and their
	// amounts.
	// The [ins] must have at least [unlockedProduced] more than the [outs].
	//
	// Precondition: [tx] has already been syntactically verified.
	//
	// Note: [unlockedProduced] is modified by this method.
	VerifySpendUTXOs(
		tx txs.UnsignedTx,
		utxos []*lux.UTXO,
		ins []*lux.TransferableInput,
		outs []*lux.TransferableOutput,
		creds []verify.Verifiable,
		unlockedProduced map[ids.ID]uint64,
	) error
}

type Handler interface {
	Spender
	Verifier
}

func NewHandler(
	ctx context.Context,
	clk *mockable.Clock,
	fx fx.Fx,
) Handler {
	return &handler{
		ctx: ctx,
		clk: clk,
		fx:  fx,
	}
}

type handler struct {
	ctx context.Context
	clk *mockable.Clock
	fx  fx.Fx
}

func (h *handler) Spend(
	utxoReader lux.UTXOReader,
	keys []*secp256k1.PrivateKey,
	amount uint64,
	fee uint64,
	changeAddr ids.ShortID,
) (
	[]*lux.TransferableInput, // inputs
	[]*lux.TransferableOutput, // returnedOutputs
	[]*lux.TransferableOutput, // stakedOutputs
	[][]*secp256k1.PrivateKey, // signers
	error,
) {
	addrs := set.NewSet[ids.ShortID](len(keys)) // The addresses controlled by [keys]
	for _, key := range keys {
		pk := key.PublicKey()
		pkBytes := pk.Bytes()
		addressBytes := secp256k1.PubkeyBytesToAddress(pkBytes)
		addr, err := ids.ToShortID(addressBytes)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("couldn't convert key to address: %w", err)
		}
		addrs.Add(addr)
	}
	utxos, err := lux.GetAllUTXOs(utxoReader, addrs) // The UTXOs controlled by [keys]
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("couldn't get UTXOs: %w", err)
	}

	kc := secp256k1fx.NewKeychain(keys...) // Keychain consumes UTXOs and creates new ones

	// Minimum time this transaction will be issued at
	now := uint64(h.clk.Time().Unix())

	ins := []*lux.TransferableInput{}
	returnedOuts := []*lux.TransferableOutput{}
	stakedOuts := []*lux.TransferableOutput{}
	signers := [][]*secp256k1.PrivateKey{}

	// Amount of LUX that has been staked
	amountStaked := uint64(0)

	// Consume locked UTXOs
	for _, utxo := range utxos {
		// If we have consumed more LUX than we are trying to stake, then we
		// have no need to consume more locked LUX
		if amountStaked >= amount {
			break
		}

		if assetID := utxo.AssetID(); assetID != XAssetID {
			continue // We only care about staking LUX, so ignore other assets
		}

		out, ok := utxo.Out.(*stakeable.LockOut)
		if !ok {
			// This output isn't locked, so it will be handled during the next
			// iteration of the UTXO set
			continue
		}
		if out.Locktime <= now {
			// This output is no longer locked, so it will be handled during the
			// next iteration of the UTXO set
			continue
		}

		inner, ok := out.TransferableOut.(*secp256k1fx.TransferOutput)
		if !ok {
			// We only know how to clone secp256k1 outputs for now
			continue
		}

		inIntf, inSigners, err := kc.Spend(out.TransferableOut, now)
		if err != nil {
			// We couldn't spend the output, so move on to the next one
			continue
		}
		in, ok := inIntf.(lux.TransferableIn)
		if !ok { // should never happen
			// TODO: Add logging when logger is available
			// log.Warn("wrong input type",
			// 	log.String("expectedType", "lux.TransferableIn"),
			// 	log.String("actualType", fmt.Sprintf("%T", inIntf)),
			// )
			continue
		}

		// The remaining value is initially the full value of the input
		remainingValue := in.Amount()

		// Stake any value that should be staked
		amountToStake := min(
			amount-amountStaked, // Amount we still need to stake
			remainingValue,      // Amount available to stake
		)
		amountStaked += amountToStake
		remainingValue -= amountToStake

		// Add the input to the consumed inputs
		ins = append(ins, &lux.TransferableInput{
			UTXOID: utxo.UTXOID,
			Asset:  lux.Asset{ID: XAssetID},
			In: &stakeable.LockIn{
				Locktime:       out.Locktime,
				TransferableIn: in,
			},
		})

		// Add the output to the staked outputs
		stakedOuts = append(stakedOuts, &lux.TransferableOutput{
			Asset: lux.Asset{ID: XAssetID},
			Out: &stakeable.LockOut{
				Locktime: out.Locktime,
				TransferableOut: &secp256k1fx.TransferOutput{
					Amt:          amountToStake,
					OutputOwners: inner.OutputOwners,
				},
			},
		})

		if remainingValue > 0 {
			// This input provided more value than was needed to be locked.
			// Some of it must be returned
			returnedOuts = append(returnedOuts, &lux.TransferableOutput{
				Asset: lux.Asset{ID: XAssetID},
				Out: &stakeable.LockOut{
					Locktime: out.Locktime,
					TransferableOut: &secp256k1fx.TransferOutput{
						Amt:          remainingValue,
						OutputOwners: inner.OutputOwners,
					},
				},
			})
		}

		// Add the signers needed for this input to the set of signers
		signers = append(signers, inSigners)
	}

	// Amount of LUX that has been burned
	amountBurned := uint64(0)

	for _, utxo := range utxos {
		// If we have consumed more LUX than we are trying to stake,
		// and we have burned more LUX than we need to,
		// then we have no need to consume more LUX
		if amountBurned >= fee && amountStaked >= amount {
			break
		}

		if assetID := utxo.AssetID(); assetID != XAssetID {
			continue // We only care about burning LUX, so ignore other assets
		}

		out := utxo.Out
		inner, ok := out.(*stakeable.LockOut)
		if ok {
			if inner.Locktime > now {
				// This output is currently locked, so this output can't be
				// burned. Additionally, it may have already been consumed
				// above. Regardless, we skip to the next UTXO
				continue
			}
			out = inner.TransferableOut
		}

		inIntf, inSigners, err := kc.Spend(out, now)
		if err != nil {
			// We couldn't spend this UTXO, so we skip to the next one
			continue
		}
		in, ok := inIntf.(lux.TransferableIn)
		if !ok {
			// Because we only use the secp Fx right now, this should never
			// happen
			continue
		}

		// The remaining value is initially the full value of the input
		remainingValue := in.Amount()

		// Burn any value that should be burned
		amountToBurn := min(
			fee-amountBurned, // Amount we still need to burn
			remainingValue,   // Amount available to burn
		)
		amountBurned += amountToBurn
		remainingValue -= amountToBurn

		// Stake any value that should be staked
		amountToStake := min(
			amount-amountStaked, // Amount we still need to stake
			remainingValue,      // Amount available to stake
		)
		amountStaked += amountToStake
		remainingValue -= amountToStake

		// Add the input to the consumed inputs
		ins = append(ins, &lux.TransferableInput{
			UTXOID: utxo.UTXOID,
			Asset:  lux.Asset{ID: XAssetID},
			In:     in,
		})

		if amountToStake > 0 {
			// Some of this input was put for staking
			stakedOuts = append(stakedOuts, &lux.TransferableOutput{
				Asset: lux.Asset{ID: XAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: amountToStake,
					OutputOwners: secp256k1fx.OutputOwners{
						Locktime:  0,
						Threshold: 1,
						Addrs:     []ids.ShortID{changeAddr},
					},
				},
			})
		}

		if remainingValue > 0 {
			// This input had extra value, so some of it must be returned
			returnedOuts = append(returnedOuts, &lux.TransferableOutput{
				Asset: lux.Asset{ID: XAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: remainingValue,
					OutputOwners: secp256k1fx.OutputOwners{
						Locktime:  0,
						Threshold: 1,
						Addrs:     []ids.ShortID{changeAddr},
					},
				},
			})
		}

		// Add the signers needed for this input to the set of signers
		signers = append(signers, inSigners)
	}

	if amountBurned < fee || amountStaked < amount {
		return nil, nil, nil, nil, fmt.Errorf(
			"%w (unlocked, locked) (%d, %d) but need (%d, %d)",
			ErrInsufficientFunds, amountBurned, amountStaked, fee, amount,
		)
	}

	lux.SortTransferableInputsWithSigners(ins, signers)  // sort inputs and keys
	lux.SortTransferableOutputs(returnedOuts, txs.Codec) // sort outputs
	lux.SortTransferableOutputs(stakedOuts, txs.Codec)   // sort outputs

	return ins, returnedOuts, stakedOuts, signers, nil
}

func (h *handler) Authorize(
	state state.Chain,
	netID ids.ID,
	keys []*secp256k1.PrivateKey,
) (
	verify.Verifiable, // Input that names owners
	[]*secp256k1.PrivateKey, // Keys that prove ownership
	error,
) {
	subnetOwner, err := state.GetNetOwner(netID)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to fetch net owner for %s: %w",
			netID,
			err,
		)
	}

	// Make sure the owners of the net match the provided keys
	owner, ok := subnetOwner.(*secp256k1fx.OutputOwners)
	if !ok {
		return nil, nil, fmt.Errorf("expected *secp256k1fx.OutputOwners but got %T", subnetOwner)
	}

	// Add the keys to a keychain
	kc := secp256k1fx.NewKeychain(keys...)

	// Make sure that the operation is valid after a minimum time
	now := uint64(h.clk.Time().Unix())

	// Attempt to prove ownership of the subnet
	indices, signers, matches := kc.Match(owner, now)
	if !matches {
		return nil, nil, errCantSign
	}

	return &secp256k1fx.Input{SigIndices: indices}, signers, nil
}

func (h *handler) VerifySpend(
	tx txs.UnsignedTx,
	utxoDB lux.UTXOGetter,
	ins []*lux.TransferableInput,
	outs []*lux.TransferableOutput,
	creds []verify.Verifiable,
	unlockedProduced map[ids.ID]uint64,
) error {
	utxos := make([]*lux.UTXO, len(ins))
	for index, input := range ins {
		utxo, err := utxoDB.GetUTXO(input.InputID())
		if err != nil {
			return fmt.Errorf(
				"failed to read consumed UTXO %s due to: %w",
				&input.UTXOID,
				err,
			)
		}
		utxos[index] = utxo
	}

	return h.VerifySpendUTXOs(tx, utxos, ins, outs, creds, unlockedProduced)
}

func (h *handler) VerifySpendUTXOs(
	tx txs.UnsignedTx,
	utxos []*lux.UTXO,
	ins []*lux.TransferableInput,
	outs []*lux.TransferableOutput,
	creds []verify.Verifiable,
	unlockedProduced map[ids.ID]uint64,
) error {
	if len(ins) != len(creds) {
		return fmt.Errorf(
			"%w: %d inputs != %d credentials",
			errWrongNumberCredentials,
			len(ins),
			len(creds),
		)
	}
	if len(ins) != len(utxos) {
		return fmt.Errorf(
			"%w: %d inputs != %d utxos",
			errWrongNumberUTXOs,
			len(ins),
			len(utxos),
		)
	}
	for _, cred := range creds { // Verify credentials are well-formed.
		if err := cred.Verify(); err != nil {
			return err
		}
	}

	// Time this transaction is being verified
	now := uint64(h.clk.Time().Unix())

	// Track the amount of unlocked transfers
	// assetID -> amount
	unlockedConsumed := make(map[ids.ID]uint64)

	// Track the amount of locked transfers and their owners
	// assetID -> locktime -> ownerID -> amount
	lockedProduced := make(map[ids.ID]map[uint64]map[ids.ID]uint64)
	lockedConsumed := make(map[ids.ID]map[uint64]map[ids.ID]uint64)

	for index, input := range ins {
		utxo := utxos[index] // The UTXO consumed by [input]

		realAssetID := utxo.AssetID()
		claimedAssetID := input.AssetID()
		if realAssetID != claimedAssetID {
			return fmt.Errorf(
				"%w: %s != %s",
				errAssetIDMismatch,
				claimedAssetID,
				realAssetID,
			)
		}

		out := utxo.Out
		locktime := uint64(0)
		// Set [locktime] to this UTXO's locktime, if applicable
		if inner, ok := out.(*stakeable.LockOut); ok {
			out = inner.TransferableOut
			locktime = inner.Locktime
		}

		in := input.In
		// The UTXO says it's locked until [locktime], but this input, which
		// consumes it, is not locked even though [locktime] hasn't passed. This
		// is invalid.
		if inner, ok := in.(*stakeable.LockIn); now < locktime && !ok {
			return errLockedFundsNotMarkedAsLocked
		} else if ok {
			if inner.Locktime != locktime {
				// This input is locked, but its locktime is wrong
				return fmt.Errorf(
					"%w: %d != %d",
					errLocktimeMismatch,
					inner.Locktime,
					locktime,
				)
			}
			in = inner.TransferableIn
		}

		// Verify that this tx's credentials allow [in] to be spent
		if err := h.fx.VerifyTransfer(tx, in, creds[index], out); err != nil {
			return fmt.Errorf("failed to verify transfer: %w", err)
		}

		amount := in.Amount()

		if now >= locktime {
			newUnlockedConsumed, err := safemath.Add64(unlockedConsumed[realAssetID], amount)
			if err != nil {
				return err
			}
			unlockedConsumed[realAssetID] = newUnlockedConsumed
			continue
		}

		owned, ok := out.(fx.Owned)
		if !ok {
			return fmt.Errorf("expected fx.Owned but got %T", out)
		}
		owner := owned.Owners()
		ownerBytes, err := txs.Codec.Marshal(txs.CodecVersion, owner)
		if err != nil {
			return fmt.Errorf("couldn't marshal owner: %w", err)
		}
		lockedConsumedAsset, ok := lockedConsumed[realAssetID]
		if !ok {
			lockedConsumedAsset = make(map[uint64]map[ids.ID]uint64)
			lockedConsumed[realAssetID] = lockedConsumedAsset
		}
		ownerID := hashing.ComputeHash256Array(ownerBytes)
		owners, ok := lockedConsumedAsset[locktime]
		if !ok {
			owners = make(map[ids.ID]uint64)
			lockedConsumedAsset[locktime] = owners
		}
		newAmount, err := safemath.Add64(owners[ownerID], amount)
		if err != nil {
			return err
		}
		owners[ownerID] = newAmount
	}

	for _, out := range outs {
		assetID := out.AssetID()

		output := out.Output()
		locktime := uint64(0)
		// Set [locktime] to this output's locktime, if applicable
		if inner, ok := output.(*stakeable.LockOut); ok {
			output = inner.TransferableOut
			locktime = inner.Locktime
		}

		amount := output.Amount()

		if locktime == 0 {
			newUnlockedProduced, err := safemath.Add64(unlockedProduced[assetID], amount)
			if err != nil {
				return err
			}
			unlockedProduced[assetID] = newUnlockedProduced
			continue
		}

		owned, ok := output.(fx.Owned)
		if !ok {
			return fmt.Errorf("expected fx.Owned but got %T", out)
		}
		owner := owned.Owners()
		ownerBytes, err := txs.Codec.Marshal(txs.CodecVersion, owner)
		if err != nil {
			return fmt.Errorf("couldn't marshal owner: %w", err)
		}
		lockedProducedAsset, ok := lockedProduced[assetID]
		if !ok {
			lockedProducedAsset = make(map[uint64]map[ids.ID]uint64)
			lockedProduced[assetID] = lockedProducedAsset
		}
		ownerID := hashing.ComputeHash256Array(ownerBytes)
		owners, ok := lockedProducedAsset[locktime]
		if !ok {
			owners = make(map[ids.ID]uint64)
			lockedProducedAsset[locktime] = owners
		}
		newAmount, err := safemath.Add64(owners[ownerID], amount)
		if err != nil {
			return err
		}
		owners[ownerID] = newAmount
	}

	// Make sure that for each assetID and locktime, tokens produced <= tokens consumed
	for assetID, producedAssetAmounts := range lockedProduced {
		lockedConsumedAsset := lockedConsumed[assetID]
		for locktime, producedAmounts := range producedAssetAmounts {
			consumedAmounts := lockedConsumedAsset[locktime]
			for ownerID, producedAmount := range producedAmounts {
				consumedAmount := consumedAmounts[ownerID]

				if producedAmount > consumedAmount {
					increase := producedAmount - consumedAmount
					unlockedConsumedAsset := unlockedConsumed[assetID]
					if increase > unlockedConsumedAsset {
						return fmt.Errorf(
							"%w: %s needs %d more %s for locktime %d",
							ErrInsufficientLockedFunds,
							ownerID,
							increase-unlockedConsumedAsset,
							assetID,
							locktime,
						)
					}
					unlockedConsumed[assetID] = unlockedConsumedAsset - increase
				}
			}
		}
	}

	for assetID, unlockedProducedAsset := range unlockedProduced {
		unlockedConsumedAsset := unlockedConsumed[assetID]
		// More unlocked tokens produced than consumed. Invalid.
		if unlockedProducedAsset > unlockedConsumedAsset {
			return fmt.Errorf(
				"%w: needs %d more %s",
				ErrInsufficientUnlockedFunds,
				unlockedProducedAsset-unlockedConsumedAsset,
				assetID,
			)
		}
	}
	return nil
}
