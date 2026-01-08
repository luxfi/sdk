// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"context"
	"reflect"
	"testing"

	"github.com/luxfi/mock/gomock"
	"github.com/stretchr/testify/require"

	consContext "github.com/luxfi/consensus/context"
	consensustest "github.com/luxfi/consensus/test/helpers"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/database"
	"github.com/luxfi/database/memdb"
	"github.com/luxfi/database/prefixdb"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/chains/atomic"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/utils/timer/mockable"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/components/verify"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/node/vms/exchangevm/fxs"
	"github.com/luxfi/sdk/node/vms/exchangevm/state"
	"github.com/luxfi/sdk/node/vms/exchangevm/state/statemock"
	"github.com/luxfi/sdk/node/vms/exchangevm/txs"
)

// testSharedMemory adapts atomic.SharedMemory to executor.SharedMemory
type testSharedMemory struct {
	sm atomic.SharedMemory
}

func (t *testSharedMemory) Get(peerChainID ids.ID, keys [][]byte) ([][]byte, error) {
	return t.sm.Get(peerChainID, keys)
}

func (t *testSharedMemory) Apply(requests map[ids.ID]interface{}, batch ...interface{}) error {
	// We don't use Apply in these tests, just Get
	return nil
}

func TestSemanticVerifierBaseTx(t *testing.T) {
	ctx := context.Background()
	cChainID := ids.GenerateTestID()

	typeToFxIndex := make(map[reflect.Type]int)
	secpFx := &secp256k1fx.Fx{}
	parser, err := txs.NewCustomParser(
		typeToFxIndex,
		new(mockable.Clock),
		nil,
		[]fxs.Fx{
			secpFx,
		},
	)
	require.NoError(t, err)

	codec := parser.Codec()
	txID := ids.GenerateTestID()
	utxoID := lux.UTXOID{
		TxID:        txID,
		OutputIndex: 2,
	}
	asset := lux.Asset{
		ID: ids.GenerateTestID(),
	}
	inputSigner := secp256k1fx.Input{
		SigIndices: []uint32{
			0,
		},
	}
	fxInput := secp256k1fx.TransferInput{
		Amt:   12345,
		Input: inputSigner,
	}
	input := lux.TransferableInput{
		UTXOID: utxoID,
		Asset:  asset,
		In:     &fxInput,
	}
	baseTx := txs.BaseTx{
		BaseTx: lux.BaseTx{
			Ins: []*lux.TransferableInput{
				&input,
			},
		},
	}

	testNetID := ids.GenerateTestID()
	backendObj := &Backend{
		Ctx: ctx,
		LuxCtx: &consContext.Context{
			ChainID:        ids.GenerateTestID(),
			ValidatorState: &testValidatorState{chainID: testNetID},
		},
		CChainID: cChainID,
		Config:   &feeConfig,
		Fxs: []*fxs.ParsedFx{
			{
				ID: secp256k1fx.ID,
				Fx: secpFx,
			},
		},
		TypeToFxIndex: typeToFxIndex,
		Codec:         codec,
		FeeAssetID:    ids.GenerateTestID(),
		Bootstrapped:  true,
	}
	require.NoError(t, secpFx.Bootstrapped())

	outputOwners := secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs: []ids.ShortID{
			keys[0].Address(),
		},
	}
	output := secp256k1fx.TransferOutput{
		Amt:          12345,
		OutputOwners: outputOwners,
	}
	utxo := lux.UTXO{
		UTXOID: utxoID,
		Asset:  asset,
		Out:    &output,
	}
	unsignedCreateAssetTx := txs.CreateAssetTx{
		States: []*txs.InitialState{{
			FxIndex: 0,
		}},
	}
	createAssetTx := txs.Tx{
		Unsigned: &unsignedCreateAssetTx,
	}

	tests := []struct {
		name      string
		stateFunc func(*gomock.Controller) state.Chain
		txFunc    func(*require.Assertions) *txs.Tx
		err       error
	}{
		{
			name: "valid",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil)
				state.EXPECT().GetTx(asset.ID).Return(&createAssetTx, nil)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &baseTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[0]},
					},
				))
				return tx
			},
			err: nil,
		},
		{
			name: "assetID mismatch",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				utxo := utxo
				utxo.Asset.ID = ids.GenerateTestID()

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &baseTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[0]},
					},
				))
				return tx
			},
			err: errAssetIDMismatch,
		},
		{
			name: "not allowed input feature extension",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				unsignedCreateAssetTx := unsignedCreateAssetTx
				unsignedCreateAssetTx.States = nil

				createAssetTx := txs.Tx{
					Unsigned: &unsignedCreateAssetTx,
				}

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil)
				state.EXPECT().GetTx(asset.ID).Return(&createAssetTx, nil)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &baseTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[0]},
					},
				))
				return tx
			},
			err: errIncompatibleFx,
		},
		{
			name: "invalid signature",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil)
				state.EXPECT().GetTx(asset.ID).Return(&createAssetTx, nil)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &baseTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[1]},
					},
				))
				return tx
			},
			err: secp256k1fx.ErrWrongSig,
		},
		{
			name: "missing UTXO",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(nil, database.ErrNotFound)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &baseTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[0]},
					},
				))
				return tx
			},
			err: database.ErrNotFound,
		},
		{
			name: "invalid UTXO amount",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				output := output
				output.Amt--

				utxo := utxo
				utxo.Out = &output

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil)
				state.EXPECT().GetTx(asset.ID).Return(&createAssetTx, nil)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &baseTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[0]},
					},
				))
				return tx
			},
			err: secp256k1fx.ErrMismatchedAmounts,
		},
		{
			name: "not allowed output feature extension",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				unsignedCreateAssetTx := unsignedCreateAssetTx
				unsignedCreateAssetTx.States = nil

				createAssetTx := txs.Tx{
					Unsigned: &unsignedCreateAssetTx,
				}

				state.EXPECT().GetTx(asset.ID).Return(&createAssetTx, nil)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				baseTx := baseTx
				baseTx.Ins = nil
				baseTx.Outs = []*lux.TransferableOutput{
					{
						Asset: asset,
						Out:   &output,
					},
				}
				tx := &txs.Tx{
					Unsigned: &baseTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{},
				))
				return tx
			},
			err: errIncompatibleFx,
		},
		{
			name: "unknown asset",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil)
				state.EXPECT().GetTx(asset.ID).Return(nil, database.ErrNotFound)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &baseTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[0]},
					},
				))
				return tx
			},
			err: database.ErrNotFound,
		},
		{
			name: "not an asset",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				tx := txs.Tx{
					Unsigned: &baseTx,
				}

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil)
				state.EXPECT().GetTx(asset.ID).Return(&tx, nil)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &baseTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[0]},
					},
				))
				return tx
			},
			err: errNotAnAsset,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)

			state := test.stateFunc(ctrl)
			tx := test.txFunc(require)

			err := tx.Unsigned.Visit(&SemanticVerifier{
				Backend: backendObj,
				State:   state,
				Tx:      tx,
			})
			require.ErrorIs(err, test.err)
		})
	}
}

func TestSemanticVerifierExportTx(t *testing.T) {
	ctx := context.Background()
	cChainID := ids.GenerateTestID()
	chainID := ids.GenerateTestID()

	typeToFxIndex := make(map[reflect.Type]int)
	secpFx := &secp256k1fx.Fx{}
	parser, err := txs.NewCustomParser(
		typeToFxIndex,
		new(mockable.Clock),
		nil,
		[]fxs.Fx{
			secpFx,
		},
	)
	require.NoError(t, err)

	codec := parser.Codec()
	txID := ids.GenerateTestID()
	utxoID := lux.UTXOID{
		TxID:        txID,
		OutputIndex: 2,
	}
	asset := lux.Asset{
		ID: ids.GenerateTestID(),
	}
	inputSigner := secp256k1fx.Input{
		SigIndices: []uint32{
			0,
		},
	}
	fxInput := secp256k1fx.TransferInput{
		Amt:   12345,
		Input: inputSigner,
	}
	input := lux.TransferableInput{
		UTXOID: utxoID,
		Asset:  asset,
		In:     &fxInput,
	}
	baseTx := txs.BaseTx{
		BaseTx: lux.BaseTx{
			NetworkID:    constants.UnitTestID,
			BlockchainID: chainID,
			Ins: []*lux.TransferableInput{
				&input,
			},
		},
	}
	exportTx := txs.ExportTx{
		BaseTx:           baseTx,
		DestinationChain: cChainID,
	}

	backendObj := &Backend{
		Ctx: ctx,
		LuxCtx: &consContext.Context{
			ChainID:        chainID, // Use same chainID as baseTx
			ValidatorState: &testValidatorState{chainID: chainID},
		},
		CChainID: cChainID,
		Config:   &feeConfig,
		Fxs: []*fxs.ParsedFx{
			{
				ID: secp256k1fx.ID,
				Fx: secpFx,
			},
		},
		TypeToFxIndex: typeToFxIndex,
		Codec:         codec,
		FeeAssetID:    ids.GenerateTestID(),
		Bootstrapped:  true,
	}
	require.NoError(t, secpFx.Bootstrapped())

	outputOwners := secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs: []ids.ShortID{
			keys[0].Address(),
		},
	}
	output := secp256k1fx.TransferOutput{
		Amt:          12345,
		OutputOwners: outputOwners,
	}
	utxo := lux.UTXO{
		UTXOID: utxoID,
		Asset:  asset,
		Out:    &output,
	}
	unsignedCreateAssetTx := txs.CreateAssetTx{
		States: []*txs.InitialState{{
			FxIndex: 0,
		}},
	}
	createAssetTx := txs.Tx{
		Unsigned: &unsignedCreateAssetTx,
	}

	tests := []struct {
		name      string
		stateFunc func(*gomock.Controller) state.Chain
		txFunc    func(*require.Assertions) *txs.Tx
		err       error
	}{
		{
			name: "valid",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil)
				state.EXPECT().GetTx(asset.ID).Return(&createAssetTx, nil)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &exportTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[0]},
					},
				))
				return tx
			},
			err: nil,
		},
		{
			name: "assetID mismatch",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				utxo := utxo
				utxo.Asset.ID = ids.GenerateTestID()

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &exportTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[0]},
					},
				))
				return tx
			},
			err: errAssetIDMismatch,
		},
		{
			name: "not allowed input feature extension",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				unsignedCreateAssetTx := unsignedCreateAssetTx
				unsignedCreateAssetTx.States = nil

				createAssetTx := txs.Tx{
					Unsigned: &unsignedCreateAssetTx,
				}

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil)
				state.EXPECT().GetTx(asset.ID).Return(&createAssetTx, nil)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &exportTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[0]},
					},
				))
				return tx
			},
			err: errIncompatibleFx,
		},
		{
			name: "invalid signature",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil)
				state.EXPECT().GetTx(asset.ID).Return(&createAssetTx, nil)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &exportTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[1]},
					},
				))
				return tx
			},
			err: secp256k1fx.ErrWrongSig,
		},
		{
			name: "missing UTXO",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(nil, database.ErrNotFound)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &exportTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[0]},
					},
				))
				return tx
			},
			err: database.ErrNotFound,
		},
		{
			name: "invalid UTXO amount",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				output := output
				output.Amt--

				utxo := utxo
				utxo.Out = &output

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil)
				state.EXPECT().GetTx(asset.ID).Return(&createAssetTx, nil)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &exportTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[0]},
					},
				))
				return tx
			},
			err: secp256k1fx.ErrMismatchedAmounts,
		},
		{
			name: "not allowed output feature extension",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				unsignedCreateAssetTx := unsignedCreateAssetTx
				unsignedCreateAssetTx.States = nil

				createAssetTx := txs.Tx{
					Unsigned: &unsignedCreateAssetTx,
				}

				state.EXPECT().GetTx(asset.ID).Return(&createAssetTx, nil)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				exportTx := exportTx
				exportTx.Ins = nil
				exportTx.ExportedOuts = []*lux.TransferableOutput{
					{
						Asset: asset,
						Out:   &output,
					},
				}
				tx := &txs.Tx{
					Unsigned: &exportTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{},
				))
				return tx
			},
			err: errIncompatibleFx,
		},
		{
			name: "unknown asset",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil)
				state.EXPECT().GetTx(asset.ID).Return(nil, database.ErrNotFound)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &exportTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[0]},
					},
				))
				return tx
			},
			err: database.ErrNotFound,
		},
		{
			name: "not an asset",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)

				tx := txs.Tx{
					Unsigned: &baseTx,
				}

				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil)
				state.EXPECT().GetTx(asset.ID).Return(&tx, nil)

				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &exportTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[0]},
					},
				))
				return tx
			},
			err: errNotAnAsset,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)

			state := test.stateFunc(ctrl)
			tx := test.txFunc(require)

			err := tx.Unsigned.Visit(&SemanticVerifier{
				Backend: backendObj,
				State:   state,
				Tx:      tx,
			})
			require.ErrorIs(err, test.err)
		})
	}
}

// testValidatorState is a simple stub for consensusctx.ValidatorState used in tests
type testValidatorState struct {
	chainID ids.ID // The chain/subnet ID this validator state returns
}

func (t *testValidatorState) GetChainID(_ ids.ID) (ids.ID, error) {
	// Returns the subnet/chain ID for the given chain
	return t.chainID, nil
}

func (t *testValidatorState) GetNetworkID(_ ids.ID) (ids.ID, error) {
	return t.chainID, nil
}

func (t *testValidatorState) GetSubnetID(_ ids.ID) (ids.ID, error) {
	return ids.Empty, nil
}

func (t *testValidatorState) GetValidatorSet(height uint64, netID ids.ID) (map[ids.NodeID]uint64, error) {
	return make(map[ids.NodeID]uint64), nil
}

func (t *testValidatorState) GetCurrentHeight(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (t *testValidatorState) GetMinimumHeight(ctx context.Context) (uint64, error) {
	return 0, nil
}

func TestSemanticVerifierExportTxDifferentNet(t *testing.T) {
	ctrl := gomock.NewController(t)
	cChainID := ids.GenerateTestID()

	ctx := consensustest.Context(t, consensustest.XChainID)

	// Set up a validator state that returns a different chainID to trigger the error
	ctx.ValidatorState = &testValidatorState{
		chainID: ids.GenerateTestID(), // Different from ctx.NetID
	}

	typeToFxIndex := make(map[reflect.Type]int)
	secpFx := &secp256k1fx.Fx{}
	parser, err := txs.NewCustomParser(
		typeToFxIndex,
		new(mockable.Clock),
		nil,
		[]fxs.Fx{
			secpFx,
		},
	)
	require.NoError(t, err)

	codec := parser.Codec()
	txID := ids.GenerateTestID()
	utxoID := lux.UTXOID{
		TxID:        txID,
		OutputIndex: 2,
	}
	asset := lux.Asset{
		ID: ids.GenerateTestID(),
	}
	inputSigner := secp256k1fx.Input{
		SigIndices: []uint32{
			0,
		},
	}
	fxInput := secp256k1fx.TransferInput{
		Amt:   12345,
		Input: inputSigner,
	}
	input := lux.TransferableInput{
		UTXOID: utxoID,
		Asset:  asset,
		In:     &fxInput,
	}
	baseTx := txs.BaseTx{
		BaseTx: lux.BaseTx{
			Ins: []*lux.TransferableInput{
				&input,
			},
		},
	}
	exportTx := txs.ExportTx{
		BaseTx:           baseTx,
		DestinationChain: cChainID,
	}

	backendObj := &Backend{
		Ctx:      context.Background(),
		LuxCtx:   ctx,
		CChainID: cChainID,
		Config:   &feeConfig,
		Fxs: []*fxs.ParsedFx{
			{
				ID: secp256k1fx.ID,
				Fx: secpFx,
			},
		},
		TypeToFxIndex: typeToFxIndex,
		Codec:         codec,
		FeeAssetID:    ids.GenerateTestID(),
		Bootstrapped:  true,
	}
	require.NoError(t, secpFx.Bootstrapped())

	outputOwners := secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs: []ids.ShortID{
			keys[0].Address(),
		},
	}
	output := secp256k1fx.TransferOutput{
		Amt:          12345,
		OutputOwners: outputOwners,
	}
	utxo := lux.UTXO{
		UTXOID: utxoID,
		Asset:  asset,
		Out:    &output,
	}
	unsignedCreateAssetTx := txs.CreateAssetTx{
		States: []*txs.InitialState{{
			FxIndex: 0,
		}},
	}
	createAssetTx := txs.Tx{
		Unsigned: &unsignedCreateAssetTx,
	}

	state := statemock.NewChain(ctrl)

	state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil)
	state.EXPECT().GetTx(asset.ID).Return(&createAssetTx, nil)

	tx := &txs.Tx{
		Unsigned: &exportTx,
	}
	require.NoError(t, tx.SignSECP256K1Fx(
		codec,
		[][]*secp256k1.PrivateKey{
			{keys[0]},
		},
	))

	err = tx.Unsigned.Visit(&SemanticVerifier{
		Backend: backendObj,
		State:   state,
		Tx:      tx,
	})
	require.ErrorIs(t, err, verify.ErrMismatchedNetIDs)
}

func TestSemanticVerifierImportTx(t *testing.T) {
	// Create consensus context for chain operations
	cChainID := ids.GenerateTestID()
	chainID := ids.GenerateTestID()
	_ = consensustest.Context(t, chainID)
	ctx := context.Background() // Use standard context for Backend
	m := atomic.NewMemory(prefixdb.New([]byte{0}, memdb.New()))

	typeToFxIndex := make(map[reflect.Type]int)
	fx := &secp256k1fx.Fx{}
	parser, err := txs.NewCustomParser(
		typeToFxIndex,
		new(mockable.Clock),
		nil,
		[]fxs.Fx{
			fx,
		},
	)
	require.NoError(t, err)

	codec := parser.Codec()
	utxoID := lux.UTXOID{
		TxID:        ids.GenerateTestID(),
		OutputIndex: 2,
	}

	asset := lux.Asset{
		ID: ids.GenerateTestID(),
	}
	outputOwners := secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs: []ids.ShortID{
			keys[0].Address(),
		},
	}
	baseTx := txs.BaseTx{
		BaseTx: lux.BaseTx{
			NetworkID:    constants.UnitTestID,
			BlockchainID: chainID,
			Outs: []*lux.TransferableOutput{{
				Asset: asset,
				Out: &secp256k1fx.TransferOutput{
					Amt:          1000,
					OutputOwners: outputOwners,
				},
			}},
		},
	}
	input := lux.TransferableInput{
		UTXOID: utxoID,
		Asset:  asset,
		In: &secp256k1fx.TransferInput{
			Amt: 12345,
			Input: secp256k1fx.Input{
				SigIndices: []uint32{0},
			},
		},
	}
	unsignedImportTx := txs.ImportTx{
		BaseTx:      baseTx,
		SourceChain: cChainID,
		ImportedIns: []*lux.TransferableInput{
			&input,
		},
	}
	importTx := &txs.Tx{
		Unsigned: &unsignedImportTx,
	}
	require.NoError(t, importTx.SignSECP256K1Fx(
		codec,
		[][]*secp256k1.PrivateKey{
			{keys[0]},
		},
	))

	backendObj := &Backend{
		Ctx: ctx,
		LuxCtx: &consContext.Context{
			ChainID:        chainID, // Use same chainID as baseTx
			ValidatorState: &testValidatorState{chainID: chainID},
		},
		CChainID: cChainID,
		Config:   &feeConfig,
		Fxs: []*fxs.ParsedFx{
			{
				ID: secp256k1fx.ID,
				Fx: fx,
			},
		},
		TypeToFxIndex: typeToFxIndex,
		Codec:         codec,
		FeeAssetID:    ids.GenerateTestID(),
		Bootstrapped:  true,
		SharedMemory:  &testSharedMemory{sm: m.NewSharedMemory(chainID)},
	}
	require.NoError(t, fx.Bootstrapped())

	output := secp256k1fx.TransferOutput{
		Amt:          12345,
		OutputOwners: outputOwners,
	}
	utxo := lux.UTXO{
		UTXOID: utxoID,
		Asset:  asset,
		Out:    &output,
	}
	utxoBytes, err := codec.Marshal(txs.CodecVersion, utxo)
	require.NoError(t, err)

	peerSharedMemory := m.NewSharedMemory(cChainID)
	inputID := utxo.InputID()
	require.NoError(t, peerSharedMemory.Apply(map[ids.ID]*atomic.Requests{chainID: {PutRequests: []*atomic.Element{{
		Key:   inputID[:],
		Value: utxoBytes,
		Traits: [][]byte{
			keys[0].PublicKey().Address().Bytes(),
		},
	}}}}))

	unsignedCreateAssetTx := txs.CreateAssetTx{
		States: []*txs.InitialState{{
			FxIndex: 0,
		}},
	}
	createAssetTx := txs.Tx{
		Unsigned: &unsignedCreateAssetTx,
	}
	tests := []struct {
		name        string
		stateFunc   func(*gomock.Controller) state.Chain
		txFunc      func(*require.Assertions) *txs.Tx
		expectedErr error
	}{
		{
			name: "valid",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)
				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil).AnyTimes()
				state.EXPECT().GetTx(asset.ID).Return(&createAssetTx, nil).AnyTimes()
				return state
			},
			txFunc: func(*require.Assertions) *txs.Tx {
				return importTx
			},
			expectedErr: nil,
		},
		{
			name: "not allowed input feature extension",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)
				unsignedCreateAssetTx := unsignedCreateAssetTx
				unsignedCreateAssetTx.States = nil
				createAssetTx := txs.Tx{
					Unsigned: &unsignedCreateAssetTx,
				}
				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil).AnyTimes()
				state.EXPECT().GetTx(asset.ID).Return(&createAssetTx, nil).AnyTimes()
				return state
			},
			txFunc: func(*require.Assertions) *txs.Tx {
				return importTx
			},
			expectedErr: errIncompatibleFx,
		},
		{
			name: "invalid signature",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)
				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil).AnyTimes()
				state.EXPECT().GetTx(asset.ID).Return(&createAssetTx, nil).AnyTimes()
				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &unsignedImportTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					[][]*secp256k1.PrivateKey{
						{keys[1]},
					},
				))
				return tx
			},
			expectedErr: secp256k1fx.ErrWrongSig,
		},
		{
			name: "not allowed output feature extension",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)
				unsignedCreateAssetTx := unsignedCreateAssetTx
				unsignedCreateAssetTx.States = nil
				createAssetTx := txs.Tx{
					Unsigned: &unsignedCreateAssetTx,
				}
				state.EXPECT().GetTx(asset.ID).Return(&createAssetTx, nil).AnyTimes()
				return state
			},
			txFunc: func(require *require.Assertions) *txs.Tx {
				importTx := unsignedImportTx
				importTx.Ins = nil
				importTx.ImportedIns = []*lux.TransferableInput{
					&input,
				}
				tx := &txs.Tx{
					Unsigned: &importTx,
				}
				require.NoError(tx.SignSECP256K1Fx(
					codec,
					nil,
				))
				return tx
			},
			expectedErr: errIncompatibleFx,
		},
		{
			name: "unknown asset",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)
				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil).AnyTimes()
				state.EXPECT().GetTx(asset.ID).Return(nil, database.ErrNotFound)
				return state
			},
			txFunc: func(*require.Assertions) *txs.Tx {
				return importTx
			},
			expectedErr: database.ErrNotFound,
		},
		{
			name: "not an asset",
			stateFunc: func(ctrl *gomock.Controller) state.Chain {
				state := statemock.NewChain(ctrl)
				tx := txs.Tx{
					Unsigned: &baseTx,
				}
				state.EXPECT().GetUTXO(utxoID.InputID()).Return(&utxo, nil).AnyTimes()
				state.EXPECT().GetTx(asset.ID).Return(&tx, nil)
				return state
			},
			txFunc: func(*require.Assertions) *txs.Tx {
				return importTx
			},
			expectedErr: errNotAnAsset,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)

			state := test.stateFunc(ctrl)
			tx := test.txFunc(require)
			err := tx.Unsigned.Visit(&SemanticVerifier{
				Backend: backendObj,
				State:   state,
				Tx:      tx,
			})
			require.ErrorIs(err, test.expectedErr)
		})
	}
}
