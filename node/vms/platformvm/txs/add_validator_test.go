// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/ids"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/utils/timer/mockable"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
	"github.com/luxfi/sdk/node/vms/platformvm/stakeable"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

func TestAddValidatorTxSyntacticVerify(t *testing.T) {
	require := require.New(t)
	clk := mockable.Clock{}
	luxAssetID := ids.GenerateTestID()
	nodeID := ids.GenerateTestNodeID()
	testChainID := ids.GenerateTestID() // Use a test chain ID instead of empty
	ctx := &consensusctx.Context{
		NetworkID: constants.UnitTestID,
		
		
		ChainID:   ids.GenerateTestID(),
	}
	ctx = &consensusctx.Context{
		
		ChainID:    testChainID,
		XAssetID: luxAssetID,
		NodeID:     nodeID,
	}
	signers := [][]*secp256k1.PrivateKey{preFundedKeys}

	var (
		stx            *Tx
		addValidatorTx *AddValidatorTx
		err            error
	)

	// Case : signed tx is nil
	err = stx.SyntacticVerify(ctx)
	require.ErrorIs(err, ErrNilSignedTx)

	// Case : unsigned tx is nil
	err = addValidatorTx.SyntacticVerify(ctx)
	require.ErrorIs(err, ErrNilTx)

	validatorWeight := uint64(2022)
	rewardAddress := preFundedKeys[0].Address()
	inputs := []*lux.TransferableInput{{
		UTXOID: lux.UTXOID{
			TxID:        ids.ID{'t', 'x', 'I', 'D'},
			OutputIndex: 2,
		},
		Asset: lux.Asset{ID: luxAssetID},
		In: &secp256k1fx.TransferInput{
			Amt:   uint64(5678),
			Input: secp256k1fx.Input{SigIndices: []uint32{0}},
		},
	}}
	outputs := []*lux.TransferableOutput{{
		Asset: lux.Asset{ID: luxAssetID},
		Out: &secp256k1fx.TransferOutput{
			Amt: uint64(1234),
			OutputOwners: secp256k1fx.OutputOwners{
				Threshold: 1,
				Addrs:     []ids.ShortID{preFundedKeys[0].Address()},
			},
		},
	}}
	stakes := []*lux.TransferableOutput{{
		Asset: lux.Asset{ID: luxAssetID},
		Out: &stakeable.LockOut{
			Locktime: uint64(clk.Time().Add(time.Second).Unix()),
			TransferableOut: &secp256k1fx.TransferOutput{
				Amt: validatorWeight,
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     []ids.ShortID{preFundedKeys[0].Address()},
				},
			},
		},
	}}
	addValidatorTx = &AddValidatorTx{
		BaseTx: BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    ctx.NetworkID,
			BlockchainID: ctx.ChainID,
			Ins:          inputs,
			Outs:         outputs,
		}},
		Validator: Validator{
			NodeID: ctx.NodeID,
			Start:  uint64(clk.Time().Unix()),
			End:    uint64(clk.Time().Add(time.Hour).Unix()),
			Wght:   validatorWeight,
		},
		StakeOuts: stakes,
		RewardsOwner: &secp256k1fx.OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs:     []ids.ShortID{rewardAddress},
		},
		DelegationShares: reward.PercentDenominator,
	}

	// Case: valid tx
	stx, err = NewSigned(addValidatorTx, Codec, signers)
	require.NoError(err)
	require.NoError(stx.SyntacticVerify(ctx))

	// Case: Wrong network ID
	addValidatorTx.SyntacticallyVerified = false
	addValidatorTx.NetworkID++
	stx, err = NewSigned(addValidatorTx, Codec, signers)
	require.NoError(err)
	err = stx.SyntacticVerify(ctx)
	require.ErrorIs(err, lux.ErrWrongNetworkID)
	addValidatorTx.NetworkID--

	// Case: Stake owner has no addresses
	addValidatorTx.SyntacticallyVerified = false
	addValidatorTx.StakeOuts[0].
		Out.(*stakeable.LockOut).
		TransferableOut.(*secp256k1fx.TransferOutput).
		Addrs = nil
	stx, err = NewSigned(addValidatorTx, Codec, signers)
	require.NoError(err)
	err = stx.SyntacticVerify(ctx)
	require.ErrorIs(err, secp256k1fx.ErrOutputUnspendable)
	addValidatorTx.StakeOuts = stakes

	// Case: Rewards owner has no addresses
	addValidatorTx.SyntacticallyVerified = false
	addValidatorTx.RewardsOwner.(*secp256k1fx.OutputOwners).Addrs = nil
	stx, err = NewSigned(addValidatorTx, Codec, signers)
	require.NoError(err)
	err = stx.SyntacticVerify(ctx)
	require.ErrorIs(err, secp256k1fx.ErrOutputUnspendable)
	addValidatorTx.RewardsOwner.(*secp256k1fx.OutputOwners).Addrs = []ids.ShortID{rewardAddress}

	// Case: Too many shares
	addValidatorTx.SyntacticallyVerified = false
	addValidatorTx.DelegationShares++ // 1 more than max amount
	stx, err = NewSigned(addValidatorTx, Codec, signers)
	require.NoError(err)
	err = stx.SyntacticVerify(ctx)
	require.ErrorIs(err, errTooManyShares)
	addValidatorTx.DelegationShares--
}

func TestAddValidatorTxSyntacticVerifyNotLUX(t *testing.T) {
	require := require.New(t)
	clk := mockable.Clock{}
	luxAssetID := ids.GenerateTestID()
	nodeID := ids.GenerateTestNodeID()
	testChainID := ids.GenerateTestID() // Use a test chain ID instead of empty
	ctx := &consensusctx.Context{
		NetworkID: constants.UnitTestID,
		
		
		ChainID:   ids.GenerateTestID(),
	}
	ctx = &consensusctx.Context{
		
		ChainID:    testChainID,
		XAssetID: luxAssetID,
		NodeID:     nodeID,
	}
	signers := [][]*secp256k1.PrivateKey{preFundedKeys}

	var (
		stx            *Tx
		addValidatorTx *AddValidatorTx
		err            error
	)

	assetID := ids.GenerateTestID()
	validatorWeight := uint64(2022)
	rewardAddress := preFundedKeys[0].Address()
	inputs := []*lux.TransferableInput{{
		UTXOID: lux.UTXOID{
			TxID:        ids.ID{'t', 'x', 'I', 'D'},
			OutputIndex: 2,
		},
		Asset: lux.Asset{ID: assetID},
		In: &secp256k1fx.TransferInput{
			Amt:   uint64(5678),
			Input: secp256k1fx.Input{SigIndices: []uint32{0}},
		},
	}}
	outputs := []*lux.TransferableOutput{{
		Asset: lux.Asset{ID: assetID},
		Out: &secp256k1fx.TransferOutput{
			Amt: uint64(1234),
			OutputOwners: secp256k1fx.OutputOwners{
				Threshold: 1,
				Addrs:     []ids.ShortID{preFundedKeys[0].Address()},
			},
		},
	}}
	stakes := []*lux.TransferableOutput{{
		Asset: lux.Asset{ID: assetID},
		Out: &stakeable.LockOut{
			Locktime: uint64(clk.Time().Add(time.Second).Unix()),
			TransferableOut: &secp256k1fx.TransferOutput{
				Amt: validatorWeight,
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     []ids.ShortID{preFundedKeys[0].Address()},
				},
			},
		},
	}}
	addValidatorTx = &AddValidatorTx{
		BaseTx: BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    ctx.NetworkID,
			BlockchainID: ctx.ChainID,
			Ins:          inputs,
			Outs:         outputs,
		}},
		Validator: Validator{
			NodeID: ctx.NodeID,
			Start:  uint64(clk.Time().Unix()),
			End:    uint64(clk.Time().Add(time.Hour).Unix()),
			Wght:   validatorWeight,
		},
		StakeOuts: stakes,
		RewardsOwner: &secp256k1fx.OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs:     []ids.ShortID{rewardAddress},
		},
		DelegationShares: reward.PercentDenominator,
	}

	stx, err = NewSigned(addValidatorTx, Codec, signers)
	require.NoError(err)

	err = stx.SyntacticVerify(ctx)
	require.ErrorIs(err, errStakeMustBeLUX)
}

func TestAddValidatorTxNotDelegatorTx(t *testing.T) {
	txIntf := any((*AddValidatorTx)(nil))
	_, ok := txIntf.(DelegatorTx)
	require.False(t, ok)
}
