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
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

// Note: Consider refactoring to use table tests for better test organization
func TestAddChainValidatorTxSyntacticVerify(t *testing.T) {
	require := require.New(t)
	clk := mockable.Clock{}
	nodeID := ids.GenerateTestNodeID()
	testChainID := ids.GenerateTestID() // Use a test chain ID instead of empty
	ctx := &consensusctx.Context{
		NetworkID: constants.UnitTestID,
		
		
		ChainID:   testChainID,
		NodeID:    nodeID,
	}
	signers := [][]*secp256k1.PrivateKey{preFundedKeys}

	var (
		stx               *Tx
		addNetValidatorTx *AddChainValidatorTx
		err               error
	)

	// Case : signed tx is nil
	err = stx.SyntacticVerify(ctx)
	require.ErrorIs(err, ErrNilSignedTx)

	// Case : unsigned tx is nil
	err = addNetValidatorTx.SyntacticVerify(ctx)
	require.ErrorIs(err, ErrNilTx)

	validatorWeight := uint64(2022)
	netID := ids.ID{'s', 'u', 'b', 'n', 'e', 't', 'I', 'D'}
	inputs := []*lux.TransferableInput{{
		UTXOID: lux.UTXOID{
			TxID:        ids.ID{'t', 'x', 'I', 'D'},
			OutputIndex: 2,
		},
		Asset: lux.Asset{ID: ids.ID{'a', 's', 's', 'e', 't'}},
		In: &secp256k1fx.TransferInput{
			Amt:   uint64(5678),
			Input: secp256k1fx.Input{SigIndices: []uint32{0}},
		},
	}}
	outputs := []*lux.TransferableOutput{{
		Asset: lux.Asset{ID: ids.ID{'a', 's', 's', 'e', 't'}},
		Out: &secp256k1fx.TransferOutput{
			Amt: uint64(1234),
			OutputOwners: secp256k1fx.OutputOwners{
				Threshold: 1,
				Addrs:     []ids.ShortID{preFundedKeys[0].Address()},
			},
		},
	}}
	subnetAuth := &secp256k1fx.Input{
		SigIndices: []uint32{0, 1},
	}
	addNetValidatorTx = &AddChainValidatorTx{
		BaseTx: BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    ctx.NetworkID,
			BlockchainID: ctx.ChainID,
			Ins:          inputs,
			Outs:         outputs,
			Memo:         []byte{1, 2, 3, 4, 5, 6, 7, 8},
		}},
		ChainValidator: ChainValidator{
			Validator: Validator{
				NodeID: nodeID,
				Start:  uint64(clk.Time().Unix()),
				End:    uint64(clk.Time().Add(time.Hour).Unix()),
				Wght:   validatorWeight,
			},
			Chain: netID,
		},
		ChainAuth: subnetAuth,
	}

	// Case: valid tx
	stx, err = NewSigned(addNetValidatorTx, Codec, signers)
	require.NoError(err)
	require.NoError(stx.SyntacticVerify(ctx))

	// Case: Wrong network ID
	addNetValidatorTx.SyntacticallyVerified = false
	addNetValidatorTx.NetworkID++
	stx, err = NewSigned(addNetValidatorTx, Codec, signers)
	require.NoError(err)
	err = stx.SyntacticVerify(ctx)
	require.ErrorIs(err, lux.ErrWrongNetworkID)
	addNetValidatorTx.NetworkID--

	// Case: Specifies primary network NetID
	addNetValidatorTx.SyntacticallyVerified = false
	addNetValidatorTx.Chain = ids.Empty
	stx, err = NewSigned(addNetValidatorTx, Codec, signers)
	require.NoError(err)
	err = stx.SyntacticVerify(ctx)
	require.ErrorIs(err, errAddPrimaryNetworkValidator)
	addNetValidatorTx.Chain = netID

	// Case: No weight
	addNetValidatorTx.SyntacticallyVerified = false
	addNetValidatorTx.Wght = 0
	stx, err = NewSigned(addNetValidatorTx, Codec, signers)
	require.NoError(err)
	err = stx.SyntacticVerify(ctx)
	require.ErrorIs(err, ErrWeightTooSmall)
	addNetValidatorTx.Wght = validatorWeight

	// Case: Net auth indices not unique
	addNetValidatorTx.SyntacticallyVerified = false
	input := addNetValidatorTx.ChainAuth.(*secp256k1fx.Input)
	oldInput := *input
	input.SigIndices[0] = input.SigIndices[1]
	stx, err = NewSigned(addNetValidatorTx, Codec, signers)
	require.NoError(err)
	err = stx.SyntacticVerify(ctx)
	require.ErrorIs(err, secp256k1fx.ErrInputIndicesNotSortedUnique)
	*input = oldInput

	// Case: adding to Primary Network
	addNetValidatorTx.SyntacticallyVerified = false
	addNetValidatorTx.Chain = constants.PrimaryNetworkID
	stx, err = NewSigned(addNetValidatorTx, Codec, signers)
	require.NoError(err)
	err = stx.SyntacticVerify(ctx)
	require.ErrorIs(err, errAddPrimaryNetworkValidator)
}

func TestAddNetValidatorMarshal(t *testing.T) {
	require := require.New(t)
	clk := mockable.Clock{}
	nodeID := ids.GenerateTestNodeID()
	testChainID := ids.GenerateTestID() // Use a test chain ID instead of empty
	ctx := &consensusctx.Context{
		NetworkID: constants.UnitTestID,
		
		
		ChainID:   testChainID,
		NodeID:    nodeID,
	}
	signers := [][]*secp256k1.PrivateKey{preFundedKeys}

	var (
		stx               *Tx
		addNetValidatorTx *AddChainValidatorTx
		err               error
	)

	// create a valid tx
	validatorWeight := uint64(2022)
	netID := ids.ID{'s', 'u', 'b', 'n', 'e', 't', 'I', 'D'}
	inputs := []*lux.TransferableInput{{
		UTXOID: lux.UTXOID{
			TxID:        ids.ID{'t', 'x', 'I', 'D'},
			OutputIndex: 2,
		},
		Asset: lux.Asset{ID: ids.ID{'a', 's', 's', 'e', 't'}},
		In: &secp256k1fx.TransferInput{
			Amt:   uint64(5678),
			Input: secp256k1fx.Input{SigIndices: []uint32{0}},
		},
	}}
	outputs := []*lux.TransferableOutput{{
		Asset: lux.Asset{ID: ids.ID{'a', 's', 's', 'e', 't'}},
		Out: &secp256k1fx.TransferOutput{
			Amt: uint64(1234),
			OutputOwners: secp256k1fx.OutputOwners{
				Threshold: 1,
				Addrs:     []ids.ShortID{preFundedKeys[0].Address()},
			},
		},
	}}
	subnetAuth := &secp256k1fx.Input{
		SigIndices: []uint32{0, 1},
	}
	addNetValidatorTx = &AddChainValidatorTx{
		BaseTx: BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    ctx.NetworkID,
			BlockchainID: ctx.ChainID,
			Ins:          inputs,
			Outs:         outputs,
			Memo:         []byte{1, 2, 3, 4, 5, 6, 7, 8},
		}},
		ChainValidator: ChainValidator{
			Validator: Validator{
				NodeID: nodeID,
				Start:  uint64(clk.Time().Unix()),
				End:    uint64(clk.Time().Add(time.Hour).Unix()),
				Wght:   validatorWeight,
			},
			Chain: netID,
		},
		ChainAuth: subnetAuth,
	}

	// Case: valid tx
	stx, err = NewSigned(addNetValidatorTx, Codec, signers)
	require.NoError(err)
	require.NoError(stx.SyntacticVerify(ctx))

	txBytes, err := Codec.Marshal(CodecVersion, stx)
	require.NoError(err)

	parsedTx, err := Parse(Codec, txBytes)
	require.NoError(err)

	require.NoError(parsedTx.SyntacticVerify(ctx))
	require.Equal(stx, parsedTx)
}

func TestAddChainValidatorTxNotValidatorTx(t *testing.T) {
	txIntf := any((*AddChainValidatorTx)(nil))
	_, ok := txIntf.(ValidatorTx)
	require.False(t, ok)
}

func TestAddChainValidatorTxNotDelegatorTx(t *testing.T) {
	txIntf := any((*AddChainValidatorTx)(nil))
	_, ok := txIntf.(DelegatorTx)
	require.False(t, ok)
}

func TestAddChainValidatorTxNotPermissionlessStaker(t *testing.T) {
	txIntf := any((*AddChainValidatorTx)(nil))
	_, ok := txIntf.(PermissionlessStaker)
	require.False(t, ok)
}
