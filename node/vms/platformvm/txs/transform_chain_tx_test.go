// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"encoding/json"
	"testing"

	consensusctx "github.com/luxfi/consensus/context"

	"github.com/luxfi/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/components/verify/verifymock"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
	"github.com/luxfi/sdk/node/vms/platformvm/stakeable"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/node/vms/types"
)

func TestTransformChainTxSerialization(t *testing.T) {
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

	simpleTransformTx := &TransformChainTx{
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
							Amt: 10 * units.Lux,
							Input: secp256k1fx.Input{
								SigIndices: []uint32{5},
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
						In: &secp256k1fx.TransferInput{
							Amt: 0xefffffffffffffff,
							Input: secp256k1fx.Input{
								SigIndices: []uint32{0},
							},
						},
					},
				},
				Memo: types.JSONByteSlice{},
			},
		},
		Chain:                      netID,
		AssetID:                  customAssetID,
		InitialSupply:            0x1000000000000000,
		MaximumSupply:            0xffffffffffffffff,
		MinConsumptionRate:       1_000,
		MaxConsumptionRate:       1_000_000,
		MinValidatorStake:        1,
		MaxValidatorStake:        0xffffffffffffffff,
		MinStakeDuration:         1,
		MaxStakeDuration:         365 * 24 * 60 * 60,
		MinDelegationFee:         reward.PercentDenominator,
		MinDelegatorStake:        1,
		MaxValidatorWeightFactor: 1,
		UptimeRequirement:        .95 * reward.PercentDenominator,
		ChainAuth: &secp256k1fx.Input{
			SigIndices: []uint32{3},
		},
	}
	testChainID := ids.Empty // Use empty chain ID for serialization test to match expected bytes
	ctx := &consensusctx.Context{
		NetworkID:  constants.MainnetID, // Must match tx.NetworkID
		
		
		ChainID:    testChainID,
		XAssetID: luxAssetID,
	}
	require.NoError(simpleTransformTx.SyntacticVerify(ctx))

	expectedUnsignedSimpleTransformTxBytes := []byte{
		// Codec version
		0x00, 0x00,
		// TransformChainTx type ID
		0x00, 0x00, 0x00, 0x18,
		// Mainnet network ID
		0x00, 0x00, 0x00, 0x01,
		// P-chain blockchain ID (31 zeros + 'P')
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// number of outputs
		0x00, 0x00, 0x00, 0x00,
		// number of inputs
		0x00, 0x00, 0x00, 0x02,
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
		// input amount = 10 LUX (10 * 10^6 with 6 decimals)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x98, 0x96, 0x80,
		// number of signatures needed in input
		0x00, 0x00, 0x00, 0x01,
		// index of signer
		0x00, 0x00, 0x00, 0x05,
		// inputs[1]
		// TxID
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		// Tx output index
		0x00, 0x00, 0x00, 0x02,
		// custom asset ID
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		// secp256k1fx transfer input type ID
		0x00, 0x00, 0x00, 0x05,
		// input amount
		0xef, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		// number of signatures needed in input
		0x00, 0x00, 0x00, 0x01,
		// index of signer
		0x00, 0x00, 0x00, 0x00,
		// length of memo
		0x00, 0x00, 0x00, 0x00,
		// netID being transformed
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
		0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38,
		// staking asset ID
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		// initial supply
		0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// maximum supply
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		// minimum consumption rate
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xe8,
		// maximum consumption rate (1,000,000 = 0x0f4240)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x0f, 0x42, 0x40,
		// minimum staking amount
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		// maximum staking amount
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		// minimum staking duration
		0x00, 0x00, 0x00, 0x01,
		// maximum staking duration
		0x01, 0xe1, 0x33, 0x80,
		// minimum delegation fee
		0x00, 0x0f, 0x42, 0x40,
		// minimum delegation amount
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		// maximum validator weight factor
		0x01,
		// uptime requirement
		0x00, 0x0e, 0x7e, 0xf0,
		// secp256k1fx authorization type ID
		0x00, 0x00, 0x00, 0x0a,
		// number of signatures needed in authorization
		0x00, 0x00, 0x00, 0x01,
		// authorization signfature index
		0x00, 0x00, 0x00, 0x03,
	}
	var unsignedSimpleTransformTx UnsignedTx = simpleTransformTx
	unsignedSimpleTransformTxBytes, err := Codec.Marshal(CodecVersion, &unsignedSimpleTransformTx)
	require.NoError(err)
	require.Equal(expectedUnsignedSimpleTransformTxBytes, unsignedSimpleTransformTxBytes)

	complexTransformTx := &TransformChainTx{
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
							Amt: units.KiloLux,
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
		Chain:                      netID,
		AssetID:                  customAssetID,
		InitialSupply:            0x1000000000000000,
		MaximumSupply:            0x1000000000000000,
		MinConsumptionRate:       0,
		MaxConsumptionRate:       0,
		MinValidatorStake:        1,
		MaxValidatorStake:        0x1000000000000000,
		MinStakeDuration:         1,
		MaxStakeDuration:         1,
		MinDelegationFee:         0,
		MinDelegatorStake:        0xffffffffffffffff,
		MaxValidatorWeightFactor: 255,
		UptimeRequirement:        0,
		ChainAuth: &secp256k1fx.Input{
			SigIndices: []uint32{},
		},
	}
	lux.SortTransferableOutputs(complexTransformTx.Outs, Codec)
	utils.Sort(complexTransformTx.Ins)
	ctx2 := &consensusctx.Context{
		NetworkID:  constants.MainnetID, // Must match tx.NetworkID
		
		
		ChainID:    testChainID,
		XAssetID: luxAssetID,
	}
	require.NoError(complexTransformTx.SyntacticVerify(ctx2))

	expectedUnsignedComplexTransformTxBytes := []byte{
		// Codec version
		0x00, 0x00,
		// TransformChainTx type ID
		0x00, 0x00, 0x00, 0x18,
		// Mainnet network ID
		0x00, 0x00, 0x00, 0x01,
		// P-chain blockchain ID (31 zeros + 'P')
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// number of outputs
		0x00, 0x00, 0x00, 0x02,
		// outputs[0]
		// Mainnet LUX asset ID
		0x51, 0xc2, 0x4f, 0xe7, 0xee, 0x02, 0x01, 0xff,
		0x0f, 0x33, 0x5f, 0x51, 0x99, 0x28, 0xdb, 0x6e,
		0xef, 0x62, 0x24, 0x25, 0x45, 0x52, 0xc9, 0x6b,
		0x6b, 0x42, 0x5f, 0xbc, 0x18, 0xfa, 0x24, 0x3b,
		// Stakeable locked output type ID
		0x00, 0x00, 0x00, 0x16,
		// Locktime
		0x00, 0x00, 0x00, 0x00, 0x05, 0x39, 0x7f, 0xb1,
		// seck256k1fx transfer output type ID
		0x00, 0x00, 0x00, 0x07,
		// amount
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		// secp256k1fx locktime
		0x00, 0x00, 0x00, 0x00, 0x00, 0xbc, 0x61, 0x4e,
		// threshold
		0x00, 0x00, 0x00, 0x00,
		// number of addresses
		0x00, 0x00, 0x00, 0x00,
		// outputs[1]
		// custom assest ID
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		// Stakeable locked output type ID
		0x00, 0x00, 0x00, 0x16,
		// Locktime
		0x00, 0x00, 0x00, 0x00, 0x34, 0x3e, 0xfc, 0xea,
		// seck256k1fx transfer output type ID
		0x00, 0x00, 0x00, 0x07,
		// amount
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		// secp256k1fx locktime
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
		// Mainnet LUX asset ID
		0x51, 0xc2, 0x4f, 0xe7, 0xee, 0x02, 0x01, 0xff,
		0x0f, 0x33, 0x5f, 0x51, 0x99, 0x28, 0xdb, 0x6e,
		0xef, 0x62, 0x24, 0x25, 0x45, 0x52, 0xc9, 0x6b,
		0x6b, 0x42, 0x5f, 0xbc, 0x18, 0xfa, 0x24, 0x3b,
		// secp256k1fx transfer input type ID
		0x00, 0x00, 0x00, 0x05,
		// amount = 1 KiloLux (with 6 decimals = 1,000,000,000 microLUX)
		0x00, 0x00, 0x00, 0x00, 0x3b, 0x9a, 0xca, 0x00,
		// number of signatures indices
		0x00, 0x00, 0x00, 0x02,
		// first signature index
		0x00, 0x00, 0x00, 0x02,
		// second signature index
		0x00, 0x00, 0x00, 0x05,
		// inputs[1]
		// TxID
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		// Tx output index
		0x00, 0x00, 0x00, 0x02,
		// custom asset ID
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		// stakeable locked input type ID
		0x00, 0x00, 0x00, 0x15,
		// locktime
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
		// memo length
		0x00, 0x00, 0x00, 0x14,
		// memo
		0xf0, 0x9f, 0x98, 0x85, 0x0a, 0x77, 0x65, 0x6c,
		0x6c, 0x20, 0x74, 0x68, 0x61, 0x74, 0x27, 0x73,
		0x01, 0x23, 0x45, 0x21,
		// netID being transformed
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
		0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38,
		// staking asset ID
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		0x99, 0x77, 0x55, 0x77, 0x11, 0x33, 0x55, 0x31,
		// initial supply
		0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// maximum supply
		0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// minimum consumption rate
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// maximum consumption rate
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// minimum staking amount
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		// maximum staking amount
		0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// minimum staking duration
		0x00, 0x00, 0x00, 0x01,
		// maximum staking duration
		0x00, 0x00, 0x00, 0x01,
		// minimum delegation fee
		0x00, 0x00, 0x00, 0x00,
		// minimum delegation amount
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		// maximum validator weight factor
		0xff,
		// uptime requirement
		0x00, 0x00, 0x00, 0x00,
		// secp256k1fx authorization type ID
		0x00, 0x00, 0x00, 0x0a,
		// number of signatures needed in authorization
		0x00, 0x00, 0x00, 0x00,

	}
	var unsignedComplexTransformTx UnsignedTx = complexTransformTx
	unsignedComplexTransformTxBytes, err := Codec.Marshal(CodecVersion, &unsignedComplexTransformTx)
	require.NoError(err)
	require.Equal(expectedUnsignedComplexTransformTxBytes, unsignedComplexTransformTxBytes)

	// Remove aliaser as BCLookup field doesn't exist in consensus.Context
	// This functionality is now handled differently

	ctx3 := &consensusctx.Context{
		NetworkID:  constants.MainnetID, // Must match tx.NetworkID
		
		
		ChainID:    testChainID,
		XAssetID: luxAssetID,
	}
	unsignedComplexTransformTx.InitCtx(ctx3)

	unsignedComplexTransformTxJSONBytes, err := json.MarshalIndent(unsignedComplexTransformTx, "", "\t")
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
				"amount": 1000000000,
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
	"assetID": "2Ab62uWwJw1T6VvmKD36ufsiuGZuX1pGykXAvPX1LtjTRHxwcc",
	"initialSupply": 1152921504606846976,
	"maximumSupply": 1152921504606846976,
	"minConsumptionRate": 0,
	"maxConsumptionRate": 0,
	"minValidatorStake": 1,
	"maxValidatorStake": 1152921504606846976,
	"minStakeDuration": 1,
	"maxStakeDuration": 1,
	"minDelegationFee": 0,
	"minDelegatorStake": 18446744073709551615,
	"maxValidatorWeightFactor": 255,
	"uptimeRequirement": 0,
	"chainAuthorization": {
		"signatureIndices": []
	}
}`, string(unsignedComplexTransformTxJSONBytes))
}

func TestTransformChainTxSyntacticVerify(t *testing.T) {
	type test struct {
		name   string
		txFunc func(*gomock.Controller) *TransformChainTx
		err    error
	}

	var (
		networkID  = uint32(1337)
		chainID    = ids.GenerateTestID()
		luxAssetID = ids.GenerateTestID()
	)

	ctx := &consensusctx.Context{
		NetworkID:  networkID, // Must match tx.NetworkID
		
		
		ChainID:    chainID,
		XAssetID: luxAssetID,
	}

	// A BaseTx that already passed syntactic verification.
	verifiedBaseTx := BaseTx{
		SyntacticallyVerified: true,
	}

	// A BaseTx that passes syntactic verification.
	validBaseTx := BaseTx{
		BaseTx: lux.BaseTx{
			NetworkID:   networkID,
			BlockchainID: chainID,
		},
	}

	// A BaseTx that fails syntactic verification.
	invalidBaseTx := BaseTx{}

	tests := []test{
		{
			name: "nil tx",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return nil
			},
			err: ErrNilTx,
		},
		{
			name: "already verified",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx: verifiedBaseTx,
				}
			},
			err: nil,
		},
		{
			name: "invalid netID",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx: validBaseTx,
					Chain:    constants.PrimaryNetworkID,
				}
			},
			err: errCantTransformPrimaryNetwork,
		},
		{
			name: "empty assetID",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:  validBaseTx,
					Chain:     ids.GenerateTestID(),
					AssetID: ids.Empty,
				}
			},
			err: errEmptyAssetID,
		},
		{
			name: "LUX assetID",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:        validBaseTx,
					Chain:           ids.GenerateTestID(),
					AssetID:       luxAssetID,
					InitialSupply: 1, // Non-zero to hit LUX assetID error first
					MaximumSupply: 1,
				}
			},
			err: errAssetIDCantBeLUX,
		},
		{
			name: "initialSupply == 0",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:        validBaseTx,
					Chain:           ids.GenerateTestID(),
					AssetID:       ids.GenerateTestID(),
					InitialSupply: 0,
				}
			},
			err: errInitialSupplyZero,
		},
		{
			name: "initialSupply > maximumSupply",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:        validBaseTx,
					Chain:           ids.GenerateTestID(),
					AssetID:       ids.GenerateTestID(),
					InitialSupply: 2,
					MaximumSupply: 1,
				}
			},
			err: errInitialSupplyGreaterThanMaxSupply,
		},
		{
			name: "minConsumptionRate > maxConsumptionRate",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:             validBaseTx,
					Chain:                ids.GenerateTestID(),
					AssetID:            ids.GenerateTestID(),
					InitialSupply:      1,
					MaximumSupply:      1,
					MinConsumptionRate: 2,
					MaxConsumptionRate: 1,
				}
			},
			err: errMinConsumptionRateTooLarge,
		},
		{
			name: "maxConsumptionRate > 100%",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:             validBaseTx,
					Chain:                ids.GenerateTestID(),
					AssetID:            ids.GenerateTestID(),
					InitialSupply:      1,
					MaximumSupply:      1,
					MinConsumptionRate: 0,
					MaxConsumptionRate: reward.PercentDenominator + 1,
				}
			},
			err: errMaxConsumptionRateTooLarge,
		},
		{
			name: "minValidatorStake == 0",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:             validBaseTx,
					Chain:                ids.GenerateTestID(),
					AssetID:            ids.GenerateTestID(),
					InitialSupply:      1,
					MaximumSupply:      1,
					MinConsumptionRate: 0,
					MaxConsumptionRate: reward.PercentDenominator,
					MinValidatorStake:  0,
				}
			},
			err: errMinValidatorStakeZero,
		},
		{
			name: "minValidatorStake > initialSupply",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:             validBaseTx,
					Chain:                ids.GenerateTestID(),
					AssetID:            ids.GenerateTestID(),
					InitialSupply:      1,
					MaximumSupply:      1,
					MinConsumptionRate: 0,
					MaxConsumptionRate: reward.PercentDenominator,
					MinValidatorStake:  2,
				}
			},
			err: errMinValidatorStakeAboveSupply,
		},
		{
			name: "minValidatorStake > maxValidatorStake",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:             validBaseTx,
					Chain:                ids.GenerateTestID(),
					AssetID:            ids.GenerateTestID(),
					InitialSupply:      10,
					MaximumSupply:      10,
					MinConsumptionRate: 0,
					MaxConsumptionRate: reward.PercentDenominator,
					MinValidatorStake:  2,
					MaxValidatorStake:  1,
				}
			},
			err: errMinValidatorStakeAboveMax,
		},
		{
			name: "maxValidatorStake > maximumSupply",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:             validBaseTx,
					Chain:                ids.GenerateTestID(),
					AssetID:            ids.GenerateTestID(),
					InitialSupply:      10,
					MaximumSupply:      10,
					MinConsumptionRate: 0,
					MaxConsumptionRate: reward.PercentDenominator,
					MinValidatorStake:  2,
					MaxValidatorStake:  11,
				}
			},
			err: errMaxValidatorStakeTooLarge,
		},
		{
			name: "minStakeDuration == 0",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:             validBaseTx,
					Chain:                ids.GenerateTestID(),
					AssetID:            ids.GenerateTestID(),
					InitialSupply:      10,
					MaximumSupply:      10,
					MinConsumptionRate: 0,
					MaxConsumptionRate: reward.PercentDenominator,
					MinValidatorStake:  2,
					MaxValidatorStake:  10,
					MinStakeDuration:   0,
				}
			},
			err: errMinStakeDurationZero,
		},
		{
			name: "minStakeDuration > maxStakeDuration",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:             validBaseTx,
					Chain:                ids.GenerateTestID(),
					AssetID:            ids.GenerateTestID(),
					InitialSupply:      10,
					MaximumSupply:      10,
					MinConsumptionRate: 0,
					MaxConsumptionRate: reward.PercentDenominator,
					MinValidatorStake:  2,
					MaxValidatorStake:  10,
					MinStakeDuration:   2,
					MaxStakeDuration:   1,
				}
			},
			err: errMinStakeDurationTooLarge,
		},
		{
			name: "minDelegationFee > 100%",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:             validBaseTx,
					Chain:                ids.GenerateTestID(),
					AssetID:            ids.GenerateTestID(),
					InitialSupply:      10,
					MaximumSupply:      10,
					MinConsumptionRate: 0,
					MaxConsumptionRate: reward.PercentDenominator,
					MinValidatorStake:  2,
					MaxValidatorStake:  10,
					MinStakeDuration:   1,
					MaxStakeDuration:   2,
					MinDelegationFee:   reward.PercentDenominator + 1,
				}
			},
			err: errMinDelegationFeeTooLarge,
		},
		{
			name: "minDelegatorStake == 0",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:             validBaseTx,
					Chain:                ids.GenerateTestID(),
					AssetID:            ids.GenerateTestID(),
					InitialSupply:      10,
					MaximumSupply:      10,
					MinConsumptionRate: 0,
					MaxConsumptionRate: reward.PercentDenominator,
					MinValidatorStake:  2,
					MaxValidatorStake:  10,
					MinStakeDuration:   1,
					MaxStakeDuration:   2,
					MinDelegationFee:   reward.PercentDenominator,
					MinDelegatorStake:  0,
				}
			},
			err: errMinDelegatorStakeZero,
		},
		{
			name: "maxValidatorWeightFactor == 0",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:                   validBaseTx,
					Chain:                      ids.GenerateTestID(),
					AssetID:                  ids.GenerateTestID(),
					InitialSupply:            10,
					MaximumSupply:            10,
					MinConsumptionRate:       0,
					MaxConsumptionRate:       reward.PercentDenominator,
					MinValidatorStake:        2,
					MaxValidatorStake:        10,
					MinStakeDuration:         1,
					MaxStakeDuration:         2,
					MinDelegationFee:         reward.PercentDenominator,
					MinDelegatorStake:        1,
					MaxValidatorWeightFactor: 0,
				}
			},
			err: errMaxValidatorWeightFactorZero,
		},
		{
			name: "uptimeRequirement > 100%",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:                   validBaseTx,
					Chain:                      ids.GenerateTestID(),
					AssetID:                  ids.GenerateTestID(),
					InitialSupply:            10,
					MaximumSupply:            10,
					MinConsumptionRate:       0,
					MaxConsumptionRate:       reward.PercentDenominator,
					MinValidatorStake:        2,
					MaxValidatorStake:        10,
					MinStakeDuration:         1,
					MaxStakeDuration:         2,
					MinDelegationFee:         reward.PercentDenominator,
					MinDelegatorStake:        1,
					MaxValidatorWeightFactor: 1,
					UptimeRequirement:        reward.PercentDenominator + 1,
				}
			},
			err: errUptimeRequirementTooLarge,
		},
		{
			name: "invalid subnetAuth",
			txFunc: func(ctrl *gomock.Controller) *TransformChainTx {
				// This NetAuth fails verification.
				invalidNetAuth := verifymock.NewVerifiable(ctrl)
				invalidNetAuth.EXPECT().Verify().Return(errInvalidNetAuth)
				return &TransformChainTx{
					BaseTx:                   validBaseTx,
					Chain:                      ids.GenerateTestID(),
					AssetID:                  ids.GenerateTestID(),
					InitialSupply:            10,
					MaximumSupply:            10,
					MinConsumptionRate:       0,
					MaxConsumptionRate:       reward.PercentDenominator,
					MinValidatorStake:        2,
					MaxValidatorStake:        10,
					MinStakeDuration:         1,
					MaxStakeDuration:         2,
					MinDelegationFee:         reward.PercentDenominator,
					MinDelegatorStake:        1,
					MaxValidatorWeightFactor: 1,
					UptimeRequirement:        reward.PercentDenominator,
					ChainAuth:               invalidNetAuth,
				}
			},
			err: errInvalidNetAuth,
		},
		{
			name: "invalid BaseTx",
			txFunc: func(*gomock.Controller) *TransformChainTx {
				return &TransformChainTx{
					BaseTx:                   invalidBaseTx,
					Chain:                      ids.GenerateTestID(),
					AssetID:                  ids.GenerateTestID(),
					InitialSupply:            10,
					MaximumSupply:            10,
					MinConsumptionRate:       0,
					MaxConsumptionRate:       reward.PercentDenominator,
					MinValidatorStake:        2,
					MaxValidatorStake:        10,
					MinStakeDuration:         1,
					MaxStakeDuration:         2,
					MinDelegationFee:         reward.PercentDenominator,
					MinDelegatorStake:        1,
					MaxValidatorWeightFactor: 1,
					UptimeRequirement:        reward.PercentDenominator,
				}
			},
			err: lux.ErrWrongNetworkID,
		},
		{
			name: "passes verification",
			txFunc: func(ctrl *gomock.Controller) *TransformChainTx {
				// This NetAuth passes verification.
				validNetAuth := verifymock.NewVerifiable(ctrl)
				validNetAuth.EXPECT().Verify().Return(nil)
				return &TransformChainTx{
					BaseTx:                   validBaseTx,
					Chain:                      ids.GenerateTestID(),
					AssetID:                  ids.GenerateTestID(),
					InitialSupply:            10,
					MaximumSupply:            10,
					MinConsumptionRate:       0,
					MaxConsumptionRate:       reward.PercentDenominator,
					MinValidatorStake:        2,
					MaxValidatorStake:        10,
					MinStakeDuration:         1,
					MaxStakeDuration:         2,
					MinDelegationFee:         reward.PercentDenominator,
					MinDelegatorStake:        1,
					MaxValidatorWeightFactor: 1,
					UptimeRequirement:        reward.PercentDenominator,
					ChainAuth:               validNetAuth,
				}
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			tx := tt.txFunc(ctrl)
			err := tx.SyntacticVerify(ctx)
			require.ErrorIs(t, err, tt.err)
	})
	}
}
