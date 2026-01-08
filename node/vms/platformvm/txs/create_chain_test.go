// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	consensusctx "github.com/luxfi/consensus/context"

	"testing"

	"github.com/stretchr/testify/require"


	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/ids"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

func TestUnsignedCreateChainTxVerify(t *testing.T) {
	testChainID := ids.GenerateTestID() // Use a test chain ID instead of empty
	ctx := &consensusctx.Context{
		NetworkID: constants.UnitTestID,
		
		
		ChainID:   ids.GenerateTestID(),
	}
	ctx = &consensusctx.Context{
		
		ChainID:   testChainID,
	}
	testNet1ID := ids.GenerateTestID()

	type test struct {
		description string
		netID       ids.ID
		genesisData []byte
		vmID        ids.ID
		fxIDs       []ids.ID
		chainName   string
		setup       func(*CreateChainTx) *CreateChainTx
		expectedErr error
	}

	tests := []test{
		{
			description: "tx is nil",
			netID:       testNet1ID,
			genesisData: nil,
			vmID:        constants.XVMID,
			fxIDs:       nil,
			chainName:   "yeet",
			setup: func(*CreateChainTx) *CreateChainTx {
				return nil
			},
			expectedErr: ErrNilTx,
		},
		{
			description: "vm ID is empty",
			netID:       testNet1ID,
			genesisData: nil,
			vmID:        constants.XVMID,
			fxIDs:       nil,
			chainName:   "yeet",
			setup: func(tx *CreateChainTx) *CreateChainTx {
				tx.VMID = ids.Empty
				return tx
			},
			expectedErr: errInvalidVMID,
		},
		{
			description: "subnet ID is primary network ID",
			netID:       testNet1ID,
			genesisData: nil,
			vmID:        constants.XVMID,
			fxIDs:       nil,
			chainName:   "yeet",
			setup: func(tx *CreateChainTx) *CreateChainTx {
				tx.ChainID = constants.PrimaryNetworkID
				return tx
			},
			expectedErr: ErrCantValidatePrimaryNetwork,
		},
		{
			description: "chain name is too long",
			netID:       testNet1ID,
			genesisData: nil,
			vmID:        constants.XVMID,
			fxIDs:       nil,
			chainName:   "yeet",
			setup: func(tx *CreateChainTx) *CreateChainTx {
				tx.BlockchainName = string(make([]byte, MaxNameLen+1))
				return tx
			},
			expectedErr: errNameTooLong,
		},
		{
			description: "chain name has invalid character",
			netID:       testNet1ID,
			genesisData: nil,
			vmID:        constants.XVMID,
			fxIDs:       nil,
			chainName:   "yeet",
			setup: func(tx *CreateChainTx) *CreateChainTx {
				tx.BlockchainName = "⌘"
				return tx
			},
			expectedErr: errIllegalNameCharacter,
		},
		{
			description: "genesis data is too long",
			netID:       testNet1ID,
			genesisData: nil,
			vmID:        constants.XVMID,
			fxIDs:       nil,
			chainName:   "yeet",
			setup: func(tx *CreateChainTx) *CreateChainTx {
				tx.GenesisData = make([]byte, MaxGenesisLen+1)
				return tx
			},
			expectedErr: errGenesisTooLong,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			require := require.New(t)

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

			createChainTx := &CreateChainTx{
				BaseTx: BaseTx{BaseTx: lux.BaseTx{
					NetworkID:    ctx.NetworkID,
					BlockchainID: ctx.ChainID,
					Ins:          inputs,
					Outs:         outputs,
				}},
				ChainID:       test.netID,
				BlockchainName:   test.chainName,
				VMID:        test.vmID,
				FxIDs:       test.fxIDs,
				GenesisData: test.genesisData,
				ChainAuth:     subnetAuth,
			}

			signers := [][]*secp256k1.PrivateKey{preFundedKeys}
			stx, err := NewSigned(createChainTx, Codec, signers)
			require.NoError(err)

			createChainTx.SyntacticallyVerified = false
			stx.Unsigned = test.setup(createChainTx)

			err = stx.SyntacticVerify(ctx)
			require.ErrorIs(err, test.expectedErr)
		})
	}
}
