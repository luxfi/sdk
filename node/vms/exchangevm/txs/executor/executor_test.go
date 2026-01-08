// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/database"
	"github.com/luxfi/metric"

	"github.com/luxfi/database/memdb"

	"github.com/luxfi/database/versiondb"

	"github.com/luxfi/ids"

	"github.com/luxfi/const"

	"github.com/luxfi/crypto/secp256k1"

	"github.com/luxfi/sdk/utils/units"

	"github.com/luxfi/sdk/node/vms/exchangevm/block"

	"github.com/luxfi/sdk/node/vms/exchangevm/fxs"

	"github.com/luxfi/sdk/node/vms/exchangevm/state"

	"github.com/luxfi/sdk/node/vms/exchangevm/txs"

	"github.com/luxfi/sdk/node/vms/components/lux"

	"github.com/luxfi/sdk/node/vms/components/verify"

	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

const trackChecksums = false

var (
	chainID = ids.ID{5, 4, 3, 2, 1}
	assetID = ids.ID{1, 2, 3}
)

func TestBaseTxExecutor(t *testing.T) {
	require := require.New(t)

	secpFx := &secp256k1fx.Fx{}
	parser, err := block.NewParser(
		[]fxs.Fx{secpFx},
	)
	require.NoError(err)
	codec := parser.Codec()

	db := memdb.New()
	vdb := versiondb.New(db)
	registerer := metric.NewNoOp().Registry()
	state, err := state.New(vdb, parser, registerer, trackChecksums)
	require.NoError(err)

	utxoID := lux.UTXOID{
		TxID:        ids.GenerateTestID(),
		OutputIndex: 1,
	}

	addr := keys[0].Address()
	utxo := &lux.UTXO{
		UTXOID: utxoID,
		Asset:  lux.Asset{ID: assetID},
		Out: &secp256k1fx.TransferOutput{
			Amt: 20 * units.KiloLux,
			OutputOwners: secp256k1fx.OutputOwners{
				Threshold: 1,
				Addrs: []ids.ShortID{
					addr,
				},
			},
		},
	}

	// Populate the UTXO that we will be consuming
	state.AddUTXO(utxo)
	require.NoError(state.Commit())

	baseTx := &txs.Tx{Unsigned: &txs.BaseTx{BaseTx: lux.BaseTx{
		NetworkID:    constants.UnitTestID,
		BlockchainID: chainID,
		Ins: []*lux.TransferableInput{{
			UTXOID: utxoID,
			Asset:  lux.Asset{ID: assetID},
			In: &secp256k1fx.TransferInput{
				Amt: 20 * units.KiloLux,
				Input: secp256k1fx.Input{
					SigIndices: []uint32{
						0,
					},
				},
			},
		}},
		Outs: []*lux.TransferableOutput{{
			Asset: lux.Asset{ID: assetID},
			Out: &secp256k1fx.TransferOutput{
				Amt: 10 * units.KiloLux,
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     []ids.ShortID{addr},
				},
			},
		}},
	}}}
	require.NoError(baseTx.SignSECP256K1Fx(codec, [][]*secp256k1.PrivateKey{{keys[0]}}))

	executor := &Executor{
		Codec: codec,
		State: state,
		Tx:    baseTx,
	}

	// Execute baseTx
	require.NoError(baseTx.Unsigned.Visit(executor))

	// Verify the consumed UTXO was removed from the state
	_, err = executor.State.GetUTXO(utxoID.InputID())
	require.ErrorIs(err, database.ErrNotFound)

	// Verify the produced UTXO was added to the state
	expectedOutputUTXO := &lux.UTXO{
		UTXOID: lux.UTXOID{
			TxID:        baseTx.TxID,
			OutputIndex: 0,
		},
		Asset: lux.Asset{
			ID: assetID,
		},
		Out: &secp256k1fx.TransferOutput{
			Amt: 10 * units.KiloLux,
			OutputOwners: secp256k1fx.OutputOwners{
				Threshold: 1,
				Addrs:     []ids.ShortID{addr},
			},
		},
	}
	expectedOutputUTXOID := expectedOutputUTXO.InputID()
	outputUTXO, err := executor.State.GetUTXO(expectedOutputUTXOID)
	require.NoError(err)

	outputUTXOID := outputUTXO.InputID()
	require.Equal(expectedOutputUTXOID, outputUTXOID)
	require.Equal(expectedOutputUTXO, outputUTXO)
}

func TestCreateAssetTxExecutor(t *testing.T) {
	require := require.New(t)

	secpFx := &secp256k1fx.Fx{}
	parser, err := block.NewParser(
		[]fxs.Fx{secpFx},
	)
	require.NoError(err)
	codec := parser.Codec()

	db := memdb.New()
	vdb := versiondb.New(db)
	registerer := metric.NewNoOp().Registry()
	state, err := state.New(vdb, parser, registerer, trackChecksums)
	require.NoError(err)

	utxoID := lux.UTXOID{
		TxID:        ids.GenerateTestID(),
		OutputIndex: 1,
	}

	addr := keys[0].Address()
	utxo := &lux.UTXO{
		UTXOID: utxoID,
		Asset:  lux.Asset{ID: assetID},
		Out: &secp256k1fx.TransferOutput{
			Amt: 20 * units.KiloLux,
			OutputOwners: secp256k1fx.OutputOwners{
				Threshold: 1,
				Addrs: []ids.ShortID{
					addr,
				},
			},
		},
	}

	// Populate the UTXO that we will be consuming
	state.AddUTXO(utxo)
	require.NoError(state.Commit())

	createAssetTx := &txs.Tx{Unsigned: &txs.CreateAssetTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    constants.UnitTestID,
			BlockchainID: chainID,
			Ins: []*lux.TransferableInput{{
				UTXOID: utxoID,
				Asset:  lux.Asset{ID: assetID},
				In: &secp256k1fx.TransferInput{
					Amt: 20 * units.KiloLux,
					Input: secp256k1fx.Input{
						SigIndices: []uint32{
							0,
						},
					},
				},
			}},
			Outs: []*lux.TransferableOutput{{
				Asset: lux.Asset{ID: assetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 10 * units.KiloLux,
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: 1,
						Addrs:     []ids.ShortID{addr},
					},
				},
			}},
		}},
		Name:         "name",
		Symbol:       "symb",
		Denomination: 0,
		States: []*txs.InitialState{
			{
				FxIndex: 0,
				Outs: []verify.State{
					&secp256k1fx.MintOutput{
						OutputOwners: secp256k1fx.OutputOwners{
							Threshold: 1,
							Addrs:     []ids.ShortID{addr},
						},
					},
				},
			},
		},
	}}
	require.NoError(createAssetTx.SignSECP256K1Fx(codec, [][]*secp256k1.PrivateKey{{keys[0]}}))

	executor := &Executor{
		Codec: codec,
		State: state,
		Tx:    createAssetTx,
	}

	// Execute createAssetTx
	require.NoError(createAssetTx.Unsigned.Visit(executor))

	// Verify the consumed UTXO was removed from the state
	_, err = executor.State.GetUTXO(utxoID.InputID())
	require.ErrorIs(err, database.ErrNotFound)

	// Verify the produced UTXOs were added to the state
	txID := createAssetTx.ID()
	expectedOutputUTXOs := []*lux.UTXO{
		{
			UTXOID: lux.UTXOID{
				TxID:        txID,
				OutputIndex: 0,
			},
			Asset: lux.Asset{
				ID: assetID,
			},
			Out: &secp256k1fx.TransferOutput{
				Amt: 10 * units.KiloLux,
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     []ids.ShortID{addr},
				},
			},
		},
		{
			UTXOID: lux.UTXOID{
				TxID:        txID,
				OutputIndex: 1,
			},
			Asset: lux.Asset{
				ID: txID,
			},
			Out: &secp256k1fx.MintOutput{
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     []ids.ShortID{addr},
				},
			},
		},
	}
	for _, expectedOutputUTXO := range expectedOutputUTXOs {
		expectedOutputUTXOID := expectedOutputUTXO.InputID()
		outputUTXO, err := executor.State.GetUTXO(expectedOutputUTXOID)
		require.NoError(err)

		outputUTXOID := outputUTXO.InputID()
		require.Equal(expectedOutputUTXOID, outputUTXOID)
		require.Equal(expectedOutputUTXO, outputUTXO)
	}
}

func TestOperationTxExecutor(t *testing.T) {
	require := require.New(t)

	secpFx := &secp256k1fx.Fx{}
	parser, err := block.NewParser(
		[]fxs.Fx{secpFx},
	)
	require.NoError(err)
	codec := parser.Codec()

	db := memdb.New()
	vdb := versiondb.New(db)
	registerer := metric.NewNoOp().Registry()
	state, err := state.New(vdb, parser, registerer, trackChecksums)
	require.NoError(err)

	outputOwners := secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs: []ids.ShortID{
			keys[0].Address(),
		},
	}

	utxoID := lux.UTXOID{
		TxID:        ids.GenerateTestID(),
		OutputIndex: 1,
	}
	utxo := &lux.UTXO{
		UTXOID: utxoID,
		Asset:  lux.Asset{ID: assetID},
		Out: &secp256k1fx.TransferOutput{
			Amt:          20 * units.KiloLux,
			OutputOwners: outputOwners,
		},
	}

	opUTXOID := lux.UTXOID{
		TxID:        ids.GenerateTestID(),
		OutputIndex: 1,
	}
	opUTXO := &lux.UTXO{
		UTXOID: opUTXOID,
		Asset:  lux.Asset{ID: assetID},
		Out: &secp256k1fx.MintOutput{
			OutputOwners: outputOwners,
		},
	}

	// Populate the UTXOs that we will be consuming
	state.AddUTXO(utxo)
	state.AddUTXO(opUTXO)
	require.NoError(state.Commit())

	operationTx := &txs.Tx{Unsigned: &txs.OperationTx{
		BaseTx: txs.BaseTx{BaseTx: lux.BaseTx{
			NetworkID:    constants.UnitTestID,
			BlockchainID: chainID,
			Ins: []*lux.TransferableInput{{
				UTXOID: utxoID,
				Asset:  lux.Asset{ID: assetID},
				In: &secp256k1fx.TransferInput{
					Amt: 20 * units.KiloLux,
					Input: secp256k1fx.Input{
						SigIndices: []uint32{
							0,
						},
					},
				},
			}},
			Outs: []*lux.TransferableOutput{{
				Asset: lux.Asset{ID: assetID},
				Out: &secp256k1fx.TransferOutput{
					Amt:          10 * units.KiloLux,
					OutputOwners: outputOwners,
				},
			}},
		}},
		Ops: []*txs.Operation{{
			Asset: lux.Asset{ID: assetID},
			UTXOIDs: []*lux.UTXOID{
				&opUTXOID,
			},
			Op: &secp256k1fx.MintOperation{
				MintInput: secp256k1fx.Input{
					SigIndices: []uint32{0},
				},
				MintOutput: secp256k1fx.MintOutput{
					OutputOwners: outputOwners,
				},
				TransferOutput: secp256k1fx.TransferOutput{
					Amt:          12345,
					OutputOwners: outputOwners,
				},
			},
		}},
	}}
	require.NoError(operationTx.SignSECP256K1Fx(
		codec,
		[][]*secp256k1.PrivateKey{
			{keys[0]},
			{keys[0]},
		},
	))

	executor := &Executor{
		Codec: codec,
		State: state,
		Tx:    operationTx,
	}

	// Execute operationTx
	require.NoError(operationTx.Unsigned.Visit(executor))

	// Verify the consumed UTXOs were removed from the state
	_, err = executor.State.GetUTXO(utxo.InputID())
	require.ErrorIs(err, database.ErrNotFound)
	_, err = executor.State.GetUTXO(opUTXO.InputID())
	require.ErrorIs(err, database.ErrNotFound)

	// Verify the produced UTXOs were added to the state
	txID := operationTx.ID()
	expectedOutputUTXOs := []*lux.UTXO{
		{
			UTXOID: lux.UTXOID{
				TxID:        txID,
				OutputIndex: 0,
			},
			Asset: lux.Asset{
				ID: assetID,
			},
			Out: &secp256k1fx.TransferOutput{
				Amt:          10 * units.KiloLux,
				OutputOwners: outputOwners,
			},
		},
		{
			UTXOID: lux.UTXOID{
				TxID:        txID,
				OutputIndex: 1,
			},
			Asset: lux.Asset{
				ID: assetID,
			},
			Out: &secp256k1fx.MintOutput{
				OutputOwners: outputOwners,
			},
		},
		{
			UTXOID: lux.UTXOID{
				TxID:        txID,
				OutputIndex: 2,
			},
			Asset: lux.Asset{
				ID: assetID,
			},
			Out: &secp256k1fx.TransferOutput{
				Amt:          12345,
				OutputOwners: outputOwners,
			},
		},
	}
	for _, expectedOutputUTXO := range expectedOutputUTXOs {
		expectedOutputUTXOID := expectedOutputUTXO.InputID()
		outputUTXO, err := executor.State.GetUTXO(expectedOutputUTXOID)
		require.NoError(err)

		outputUTXOID := outputUTXO.InputID()
		require.Equal(expectedOutputUTXOID, outputUTXOID)
		require.Equal(expectedOutputUTXO, outputUTXO)
	}
}
