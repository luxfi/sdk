// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package exchangevm

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	core "github.com/luxfi/consensus/core"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/ids"
	"github.com/luxfi/upgrade/upgradetest"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/node/vms/exchangevm/txs"
)

func TestSetsAndGets(t *testing.T) {
	require := require.New(t)

	// secp256k1fx is now included by default, so we only need the custom Fx
	env := setup(t, &envConfig{
		fork: upgradetest.GetConfig(upgradetest.Latest),
		additionalFxs: []interface{}{
			&core.Fx{
				ID: ids.GenerateTestID(),
				Fx: &FxTest{
					InitializeF: func(vmIntf interface{}) error {
						// The VM passed here is actually a txs.fxVM which implements secp256k1fx.VM
						vm, ok := vmIntf.(secp256k1fx.VM)
						if !ok {
							return fmt.Errorf("unexpected VM type: %T", vmIntf)
						}
						return vm.CodecRegistry().RegisterType(&lux.TestState{})
					},
				},
			},
		},
	})
	defer env.testLock.Unlock()

	utxo := &lux.UTXO{
		UTXOID: lux.UTXOID{
			TxID:        ids.Empty,
			OutputIndex: 1,
		},
		Asset: lux.Asset{ID: ids.Empty},
		Out:   &lux.TestState{},
	}
	utxoID := utxo.InputID()

	tx := &txs.Tx{Unsigned: &txs.BaseTx{BaseTx: lux.BaseTx{
		NetworkID:    constants.UnitTestID,
		BlockchainID: env.consensusCtx.XChainID,
		Ins: []*lux.TransferableInput{{
			UTXOID: lux.UTXOID{
				TxID:        ids.Empty,
				OutputIndex: 0,
			},
			Asset: lux.Asset{ID: assetID},
			In: &secp256k1fx.TransferInput{
				Amt: 20 * units.KiloLux,
				Input: secp256k1fx.Input{
					SigIndices: []uint32{
						0,
					},
				},
			},
		}},
	}}}
	require.NoError(tx.SignSECP256K1Fx(env.vm.parser.Codec(), [][]*secp256k1.PrivateKey{{keys[0]}}))

	txID := tx.ID()

	env.vm.state.AddUTXO(utxo)
	env.vm.state.AddTx(tx)

	resultUTXO, err := env.vm.state.GetUTXO(utxoID)
	require.NoError(err)
	resultTx, err := env.vm.state.GetTx(txID)
	require.NoError(err)

	require.Equal(uint32(1), resultUTXO.OutputIndex)
	require.Equal(tx.ID(), resultTx.ID())
}

func TestFundingNoAddresses(t *testing.T) {
	// secp256k1fx is now included by default, so we only need the custom Fx
	env := setup(t, &envConfig{
		fork: upgradetest.GetConfig(upgradetest.Latest),
		additionalFxs: []interface{}{
			&core.Fx{
				ID: ids.GenerateTestID(),
				Fx: &FxTest{
					InitializeF: func(vmIntf interface{}) error {
						// The VM passed here is actually a txs.fxVM which implements secp256k1fx.VM
						vm, ok := vmIntf.(secp256k1fx.VM)
						if !ok {
							return fmt.Errorf("unexpected VM type: %T", vmIntf)
						}
						return vm.CodecRegistry().RegisterType(&lux.TestState{})
					},
				},
			},
		},
	})
	defer env.testLock.Unlock()

	utxo := &lux.UTXO{
		UTXOID: lux.UTXOID{
			TxID:        ids.Empty,
			OutputIndex: 1,
		},
		Asset: lux.Asset{ID: ids.Empty},
		Out:   &lux.TestState{},
	}

	env.vm.state.AddUTXO(utxo)
	env.vm.state.DeleteUTXO(utxo.InputID())
}

func TestFundingAddresses(t *testing.T) {
	require := require.New(t)

	// secp256k1fx is now included by default, so we only need the custom Fx
	env := setup(t, &envConfig{
		fork: upgradetest.GetConfig(upgradetest.Latest),
		additionalFxs: []interface{}{
			&core.Fx{
				ID: ids.GenerateTestID(),
				Fx: &FxTest{
					InitializeF: func(vmIntf interface{}) error {
						vm := vmIntf.(secp256k1fx.VM)
						return vm.CodecRegistry().RegisterType(&lux.TestAddressable{})
					},
				},
			},
		},
	})
	defer env.testLock.Unlock()

	utxo := &lux.UTXO{
		UTXOID: lux.UTXOID{
			TxID:        ids.Empty,
			OutputIndex: 1,
		},
		Asset: lux.Asset{ID: ids.Empty},
		Out: &lux.TestAddressable{
			Addrs: [][]byte{{0}},
		},
	}

	env.vm.state.AddUTXO(utxo)
	require.NoError(env.vm.state.Commit())

	utxos, err := env.vm.state.UTXOIDs([]byte{0}, ids.Empty, math.MaxInt32)
	require.NoError(err)
	require.Len(utxos, 1)
	require.Equal(utxo.InputID(), utxos[0])

	env.vm.state.DeleteUTXO(utxo.InputID())
	require.NoError(env.vm.state.Commit())

	utxos, err = env.vm.state.UTXOIDs([]byte{0}, ids.Empty, math.MaxInt32)
	require.NoError(err)
	require.Empty(utxos)
}
