// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	consensusctx "github.com/luxfi/consensus/context"
	
	"encoding/json"
	"testing"

	"github.com/luxfi/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/components/verify/verifymock"
	"github.com/luxfi/sdk/node/vms/platformvm/fx/fxmock"
	"github.com/luxfi/sdk/node/vms/platformvm/stakeable"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/node/vms/types"
)

func TestTransferChainOwnershipTxSerialization(t *testing.T) {
	require := require.New(t)

	addr := ids.ShortID{
		0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb,
		0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb,
		0x44, 0x55, 0x66, 0x77,
	}

	luxAssetID, err := ids.FromString("d1Rdokz7Vq8H5aczkwgkiPCCa6JME7yT2xpqgWTfFKWYVsGbG")
	require.NoError(err)

	customAssetID := ids.ID{
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
	}

	txID := ids.ID{
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
	}
	netID := ids.ID{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
		0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38,
	}

	simpleTransferChainOwnershipTx := &TransferChainOwnershipTx{
		BaseTx: BaseTx{
			BaseTx: lux.BaseTx{
				NetworkID:   constants.MainnetID,
				BlockchainID: ids.Empty, // Use empty for serialization test
				Outs:         []*lux.TransferableOutput{},
				Ins: []*lux.TransferableInput{
					{
						UTXOID: lux.UTXOID{
							TxID:        txID,
							OutputIndex: 1,
						},
						Asset: lux.Asset{
							ID: luxAssetID,
						},
						In: &secp256k1fx.TransferInput{
							Amt: units.MilliLux,
							Input: secp256k1fx.Input{
								SigIndices: []uint32{5},
							},
						},
					},
				},
				Memo: types.JSONByteSlice{},
			},
		},
		Chain: netID,
		ChainAuth: &secp256k1fx.Input{
			SigIndices: []uint32{3},
		},
		Owner: &secp256k1fx.OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	testChainID := ids.Empty // Use empty chain ID for serialization test to match expected bytes
	ctx := &consensusctx.Context{
		NetworkID:  constants.MainnetID, // Must match tx.NetworkID
		
		ChainID:    testChainID,
		XAssetID: luxAssetID,
	}
	require.NoError(simpleTransferChainOwnershipTx.SyntacticVerify(ctx))

	expectedUnsignedSimpleTransferChainOwnershipTxBytes := []byte{
		// Codec version
		0x00, 0x00,
		// TransferChainOwnershipTx Type ID
		0x00, 0x00, 0x00, 0x21,
		// Mainnet network ID
		0x00, 0x00, 0x00, 0x01,
		// P-chain blockchain ID (31 zeros + 'P')
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Number of outputs
		0x00, 0x00, 0x00, 0x00,
		// Number of inputs
		0x00, 0x00, 0x00, 0x01,
		// Inputs[0]
		// TxID
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		// Tx output index
		0x00, 0x00, 0x00, 0x01,
		// Mainnet LUX assetID
		0x51, 0xc2, 0x4f, 0xe7, 0xee, 0x02, 0x01, 0xff,
		0x0f, 0x33, 0x5f, 0x51, 0x99, 0x28, 0xdb, 0x6e,
		0xef, 0x62, 0x24, 0x25, 0x45, 0x52, 0xc9, 0x6b,
		0x6b, 0x42, 0x5f, 0xbc, 0x18, 0xfa, 0x24, 0x3b,
		// secp256k1fx transfer input type ID
		0x00, 0x00, 0x00, 0x05,
		// input amount = 1 MilliLux
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xe8,
		// number of signatures needed in input
		0x00, 0x00, 0x00, 0x01,
		// index of signer
		0x00, 0x00, 0x00, 0x05,
		// length of memo
		0x00, 0x00, 0x00, 0x00,
		// netID to modify
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
		0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38,
		// secp256k1fx authorization type ID
		0x00, 0x00, 0x00, 0x0a,
		// number of signatures needed in authorization
		0x00, 0x00, 0x00, 0x01,
		// index of signer
		0x00, 0x00, 0x00, 0x03,
		// secp256k1fx output owners type ID
		0x00, 0x00, 0x00, 0x0b,
		// locktime
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// threshold
		0x00, 0x00, 0x00, 0x01,
		// number of addrs
		0x00, 0x00, 0x00, 0x01,
		// Addrs[0]
		0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb,
		0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb,
		0x44, 0x55, 0x66, 0x77,
	}
	var unsignedSimpleTransferChainOwnershipTx UnsignedTx = simpleTransferChainOwnershipTx
	unsignedSimpleTransferChainOwnershipTxBytes, err := Codec.Marshal(CodecVersion, &unsignedSimpleTransferChainOwnershipTx)
	require.NoError(err)
	require.Equal(expectedUnsignedSimpleTransferChainOwnershipTxBytes, unsignedSimpleTransferChainOwnershipTxBytes)

	complexTransferChainOwnershipTx := &TransferChainOwnershipTx{
		BaseTx: BaseTx{
			BaseTx: lux.BaseTx{
				NetworkID:   constants.MainnetID,
				BlockchainID: ids.Empty, // Use empty for serialization test
				Outs: []*lux.TransferableOutput{
					{
						Asset: lux.Asset{
							ID: luxAssetID,
						},
						Out: &stakeable.LockOut{
							Locktime: 87654321,
							TransferableOut: &secp256k1fx.TransferOutput{
								Amt: 1,
								OutputOwners: secp256k1fx.OutputOwners{
									Locktime:  12345678,
									Threshold: 0,
									Addrs:     []ids.ShortID{},
								},
							},
						},
					},
					{
						Asset: lux.Asset{
							ID: customAssetID,
						},
						Out: &stakeable.LockOut{
							Locktime: 876543210,
							TransferableOut: &secp256k1fx.TransferOutput{
								Amt: 0xffffffffffffffff,
								OutputOwners: secp256k1fx.OutputOwners{
									Locktime:  0,
									Threshold: 1,
									Addrs: []ids.ShortID{
										addr,
									},
								},
							},
						},
					},
				},
				Ins: []*lux.TransferableInput{
					{
						UTXOID: lux.UTXOID{
							TxID:        txID,
							OutputIndex: 1,
						},
						Asset: lux.Asset{
							ID: luxAssetID,
						},
						In: &secp256k1fx.TransferInput{
							Amt: units.Lux,
							Input: secp256k1fx.Input{
								SigIndices: []uint32{2, 5},
							},
						},
					},
					{
						UTXOID: lux.UTXOID{
							TxID:        txID,
							OutputIndex: 2,
						},
						Asset: lux.Asset{
							ID: customAssetID,
						},
						In: &stakeable.LockIn{
							Locktime: 876543210,
							TransferableIn: &secp256k1fx.TransferInput{
								Amt: 0xefffffffffffffff,
								Input: secp256k1fx.Input{
									SigIndices: []uint32{0},
								},
							},
						},
					},
					{
						UTXOID: lux.UTXOID{
							TxID:        txID,
							OutputIndex: 3,
						},
						Asset: lux.Asset{
							ID: customAssetID,
						},
						In: &secp256k1fx.TransferInput{
							Amt: 0x1000000000000000,
							Input: secp256k1fx.Input{
								SigIndices: []uint32{},
							},
						},
					},
				},
				Memo: types.JSONByteSlice("😅\nwell that's\x01\x23\x45!"),
			},
		},
		Chain: netID,
		ChainAuth: &secp256k1fx.Input{
			SigIndices: []uint32{},
		},
		Owner: &secp256k1fx.OutputOwners{
			Locktime:  876543210,
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	lux.SortTransferableOutputs(complexTransferChainOwnershipTx.Outs, Codec)
	utils.Sort(complexTransferChainOwnershipTx.Ins)
	ctx2 := &consensusctx.Context{
		NetworkID:  constants.MainnetID,
		
		ChainID:    testChainID,
		XAssetID: luxAssetID,
	}
	require.NoError(complexTransferChainOwnershipTx.SyntacticVerify(ctx2))

	expectedUnsignedComplexTransferChainOwnershipTxBytes := []byte{
		// Codec version
		0x00, 0x00,
		// TransferChainOwnershipTx Type ID
		0x00, 0x00, 0x00, 0x21,
		// Mainnet network ID
		0x00, 0x00, 0x00, 0x01,
		// P-chain blockchain ID (31 zeros + 'P')
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Number of outputs
		0x00, 0x00, 0x00, 0x02,
		// Outputs[0]
		// Mainnet LUX assetID
		0x51, 0xc2, 0x4f, 0xe7, 0xee, 0x02, 0x01, 0xff,
		0x0f, 0x33, 0x5f, 0x51, 0x99, 0x28, 0xdb, 0x6e,
		0xef, 0x62, 0x24, 0x25, 0x45, 0x52, 0xc9, 0x6b,
		0x6b, 0x42, 0x5f, 0xbc, 0x18, 0xfa, 0x24, 0x3b,
		// Stakeable locked output type ID
		0x00, 0x00, 0x00, 0x16,
		// Locktime
		0x00, 0x00, 0x00, 0x00, 0x05, 0x39, 0x7f, 0xb1,
		// secp256k1fx transfer output type ID
		0x00, 0x00, 0x00, 0x07,
		// amount
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		// secp256k1fx output locktime
		0x00, 0x00, 0x00, 0x00, 0x00, 0xbc, 0x61, 0x4e,
		// threshold
		0x00, 0x00, 0x00, 0x00,
		// number of addresses
		0x00, 0x00, 0x00, 0x00,
		// Outputs[1]
		// custom asset ID
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		// Stakeable locked output type ID
		0x00, 0x00, 0x00, 0x16,
		// Locktime
		0x00, 0x00, 0x00, 0x00, 0x34, 0x3e, 0xfc, 0xea,
		// secp256k1fx transfer output type ID
		0x00, 0x00, 0x00, 0x07,
		// amount
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		// secp256k1fx output locktime
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// threshold
		0x00, 0x00, 0x00, 0x01,
		// number of addresses
		0x00, 0x00, 0x00, 0x01,
		// address[0]
		0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb,
		0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb,
		0x44, 0x55, 0x66, 0x77,
		// number of inputs
		0x00, 0x00, 0x00, 0x03,
		// inputs[0]
		// TxID
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		// Tx output index
		0x00, 0x00, 0x00, 0x01,
		// Mainnet LUX assetID
		0x51, 0xc2, 0x4f, 0xe7, 0xee, 0x02, 0x01, 0xff,
		0x0f, 0x33, 0x5f, 0x51, 0x99, 0x28, 0xdb, 0x6e,
		0xef, 0x62, 0x24, 0x25, 0x45, 0x52, 0xc9, 0x6b,
		0x6b, 0x42, 0x5f, 0xbc, 0x18, 0xfa, 0x24, 0x3b,
		// secp256k1fx transfer input type ID
		0x00, 0x00, 0x00, 0x05,
		// input amount = 1 Lux
		0x00, 0x00, 0x00, 0x00, 0x00, 0x0f, 0x42, 0x40,
		// number of signatures needed in input
		0x00, 0x00, 0x00, 0x02,
		// index of first signer
		0x00, 0x00, 0x00, 0x02,
		// index of second signer
		0x00, 0x00, 0x00, 0x05,
		// inputs[1]
		// TxID
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		// Tx output index
		0x00, 0x00, 0x00, 0x02,
		// Custom asset ID
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		// Stakeable locked input type ID
		0x00, 0x00, 0x00, 0x15,
		// Locktime
		0x00, 0x00, 0x00, 0x00, 0x34, 0x3e, 0xfc, 0xea,
		// secp256k1fx transfer input type ID
		0x00, 0x00, 0x00, 0x05,
		// input amount
		0xef, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		// number of signatures needed in input
		0x00, 0x00, 0x00, 0x01,
		// index of signer
		0x00, 0x00, 0x00, 0x00,
		// inputs[2]
		// TxID
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		// Tx output index
		0x00, 0x00, 0x00, 0x03,
		// custom asset ID
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		// secp256k1fx transfer input type ID
		0x00, 0x00, 0x00, 0x05,
		// input amount
		0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// number of signatures needed in input
		0x00, 0x00, 0x00, 0x00,
		// length of memo
		0x00, 0x00, 0x00, 0x14,
		// memo
		0xf0, 0x9f, 0x98, 0x85, 0x0a, 0x77, 0x65, 0x6c,
		0x6c, 0x20, 0x74, 0x68, 0x61, 0x74, 0x27, 0x73,
		0x01, 0x23, 0x45, 0x21,
		// netID to modify
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
		0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38,
		// secp256k1fx authorization type ID
		0x00, 0x00, 0x00, 0x0a,
		// number of signatures needed in authorization
		0x00, 0x00, 0x00, 0x00,
		// secp256k1fx output owners type ID
		0x00, 0x00, 0x00, 0x0b,
		// locktime
		0x00, 0x00, 0x00, 0x00, 0x34, 0x3e, 0xfc, 0xea,
		// threshold
		0x00, 0x00, 0x00, 0x01,
		// number of addrs
		0x00, 0x00, 0x00, 0x01,
		// Addrs[0]
		0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb,
		0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb,
		0x44, 0x55, 0x66, 0x77,
	
	}
	var unsignedComplexTransferChainOwnershipTx UnsignedTx = complexTransferChainOwnershipTx
	unsignedComplexTransferChainOwnershipTxBytes, err := Codec.Marshal(CodecVersion, &unsignedComplexTransferChainOwnershipTx)
	require.NoError(err)
	require.Equal(expectedUnsignedComplexTransferChainOwnershipTxBytes, unsignedComplexTransferChainOwnershipTxBytes)

	// Remove aliaser as BCLookup field doesn't exist in consensus.Context
	// This functionality is now handled differently

	ctx3 := &consensusctx.Context{
		NetworkID:  constants.MainnetID, // Must match tx.NetworkID for "P-lux1..." address encoding
		
		ChainID:    testChainID,
		XAssetID: luxAssetID,
	}
	unsignedComplexTransferChainOwnershipTx.InitCtx(ctx3)

	unsignedComplexTransferChainOwnershipTxJSONBytes, err := json.MarshalIndent(unsignedComplexTransferChainOwnershipTx, "", "\t")
	require.NoError(err)
	require.JSONEq(`{
	"networkID": 1,
	"blockchainID": "11111111111111111111111111111111LpoYY",
	"outputs": [
		{
			"assetID": "d1Rdokz7Vq8H5aczkwgkiPCCa6JME7yT2xpqgWTfFKWYVsGbG",
			"fxID": "spdxUxVJQbX85MGxMHbKw1sHxMnSqJ3QBzDyDYEP3h6TLuxqQ",
			"output": {
				"locktime": 87654321,
				"output": {
					"addresses": [],
					"amount": 1,
					"locktime": 12345678,
					"threshold": 0
				}
			}
		},
		{
			"assetID": "2Ab62uWwJw1T6VvmKD36ufsiuGZuX1pGykXAvPX1LtjTRHxwcc",
			"fxID": "spdxUxVJQbX85MGxMHbKw1sHxMnSqJ3QBzDyDYEP3h6TLuxqQ",
			"output": {
				"locktime": 876543210,
				"output": {
					"addresses": [
						"7EKFm18KvWqcxMCNgpBSN51pJnEr1cVUb"
					],
					"amount": 18446744073709551615,
					"locktime": 0,
					"threshold": 1
				}
			}
		}
	],
	"inputs": [
		{
			"txID": "2wiU5PnFTjTmoAXGZutHAsPF36qGGyLHYHj9G1Aucfmb3JFFGN",
			"outputIndex": 1,
			"assetID": "d1Rdokz7Vq8H5aczkwgkiPCCa6JME7yT2xpqgWTfFKWYVsGbG",
			"fxID": "spdxUxVJQbX85MGxMHbKw1sHxMnSqJ3QBzDyDYEP3h6TLuxqQ",
			"input": {
				"amount": 1000000,
				"signatureIndices": [
					2,
					5
				]
			}
		},
		{
			"txID": "2wiU5PnFTjTmoAXGZutHAsPF36qGGyLHYHj9G1Aucfmb3JFFGN",
			"outputIndex": 2,
			"assetID": "2Ab62uWwJw1T6VvmKD36ufsiuGZuX1pGykXAvPX1LtjTRHxwcc",
			"fxID": "spdxUxVJQbX85MGxMHbKw1sHxMnSqJ3QBzDyDYEP3h6TLuxqQ",
			"input": {
				"locktime": 876543210,
				"input": {
					"amount": 17293822569102704639,
					"signatureIndices": [
						0
					]
				}
			}
		},
		{
			"txID": "2wiU5PnFTjTmoAXGZutHAsPF36qGGyLHYHj9G1Aucfmb3JFFGN",
			"outputIndex": 3,
			"assetID": "2Ab62uWwJw1T6VvmKD36ufsiuGZuX1pGykXAvPX1LtjTRHxwcc",
			"fxID": "spdxUxVJQbX85MGxMHbKw1sHxMnSqJ3QBzDyDYEP3h6TLuxqQ",
			"input": {
				"amount": 1152921504606846976,
				"signatureIndices": []
			}
		}
	],
	"memo": "0xf09f98850a77656c6c2074686174277301234521",
	"chainID": "SkB92YpWm4UpburLz9tEKZw2i67H3FF6YkjaU4BkFUDTG9Xm",
	"chainAuthorization": {
		"signatureIndices": []
	},
	"newOwner": {
		"addresses": [
			"7EKFm18KvWqcxMCNgpBSN51pJnEr1cVUb"
		],
		"locktime": 876543210,
		"threshold": 1
	}
}`, string(unsignedComplexTransferChainOwnershipTxJSONBytes))
}

func TestTransferChainOwnershipTxSyntacticVerify(t *testing.T) {
	type test struct {
		name        string
		txFunc      func(*gomock.Controller) *TransferChainOwnershipTx
		expectedErr error
	}

	var (
		networkID = uint32(1337)
		chainID   = ids.GenerateTestID()
	)

	ctx := &consensusctx.Context{
		NetworkID: networkID,
		
		
		ChainID:   chainID,
	}

	// A BaseTx that already passed syntactic verification.
	verifiedBaseTx := BaseTx{
		SyntacticallyVerified: true,
	}
	// Sanity check.
	require.NoError(t, verifiedBaseTx.SyntacticVerify(ctx))

	// A BaseTx that passes syntactic verification.
	validBaseTx := BaseTx{
		BaseTx: lux.BaseTx{
			NetworkID:   networkID,
			BlockchainID: chainID,
		},
	}
	// Sanity check.
	require.NoError(t, validBaseTx.SyntacticVerify(ctx))
	// Make sure we're not caching the verification result.
	require.False(t, validBaseTx.SyntacticallyVerified)

	// A BaseTx that fails syntactic verification.
	invalidBaseTx := BaseTx{}

	tests := []test{
		{
			name: "nil tx",
			txFunc: func(*gomock.Controller) *TransferChainOwnershipTx {
				return nil
			},
			expectedErr: ErrNilTx,
		},
		{
			name: "already verified",
			txFunc: func(*gomock.Controller) *TransferChainOwnershipTx {
				return &TransferChainOwnershipTx{BaseTx: verifiedBaseTx}
			},
			expectedErr: nil,
		},
		{
			name: "invalid BaseTx",
			txFunc: func(*gomock.Controller) *TransferChainOwnershipTx {
				return &TransferChainOwnershipTx{
					// Set netID so we don't error on that check.
					Chain:    ids.GenerateTestID(),
					BaseTx: invalidBaseTx,
				}
			},
			expectedErr: lux.ErrWrongNetworkID,
		},
		{
			name: "invalid netID",
			txFunc: func(*gomock.Controller) *TransferChainOwnershipTx {
				return &TransferChainOwnershipTx{
					BaseTx: validBaseTx,
					Chain:    constants.PrimaryNetworkID,
				}
			},
			expectedErr: ErrTransferPermissionlessChain,
		},
		{
			name: "invalid subnetAuth",
			txFunc: func(ctrl *gomock.Controller) *TransferChainOwnershipTx {
				// This NetAuth fails verification.
				invalidNetAuth := verifymock.NewVerifiable(ctrl)
				invalidNetAuth.EXPECT().Verify().Return(errInvalidNetAuth)
				return &TransferChainOwnershipTx{
					// Set netID so we don't error on that check.
					Chain:        ids.GenerateTestID(),
					BaseTx:     validBaseTx,
					ChainAuth: invalidNetAuth,
				}
			},
			expectedErr: errInvalidNetAuth,
		},
		{
			name: "passes verification",
			txFunc: func(ctrl *gomock.Controller) *TransferChainOwnershipTx {
				// This NetAuth passes verification.
				validNetAuth := verifymock.NewVerifiable(ctrl)
				validNetAuth.EXPECT().Verify().Return(nil)
				mockOwner := fxmock.NewOwner(ctrl)
				mockOwner.EXPECT().Verify().Return(nil)
				return &TransferChainOwnershipTx{
					// Set netID so we don't error on that check.
					Chain:        ids.GenerateTestID(),
					BaseTx:     validBaseTx,
					ChainAuth: validNetAuth,
					Owner:      mockOwner,
				}
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)

			tx := tt.txFunc(ctrl)
			err := tx.SyntacticVerify(ctx)
			require.ErrorIs(err, tt.expectedErr)
			if tt.expectedErr != nil {
				return
			}
			require.True(tx.SyntacticallyVerified)
	})
	}
}
