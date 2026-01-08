// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utxo

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/sdk/utils/timer/mockable"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/components/verify"
	"github.com/luxfi/sdk/node/vms/platformvm/stakeable"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"

	safemath "github.com/luxfi/math"
)

var _ txs.UnsignedTx = (*dummyUnsignedTx)(nil)

type dummyUnsignedTx struct {
	txs.BaseTx
}

func (*dummyUnsignedTx) Visit(txs.Visitor) error {
	return nil
}

func TestVerifySpendUTXOs(t *testing.T) {
	fx := &secp256k1fx.Fx{}

	require.NoError(t, fx.Initialize(&secp256k1fx.TestVM{}))
	require.NoError(t, fx.Bootstrapped())

	luxAssetID := ids.GenerateTestID()

	h := &handler{
		ctx: context.Background(),
		clk: &mockable.Clock{},
		fx:  fx,
	}

	// The handler time during a test, unless [chainTimestamp] is set
	now := time.Unix(1607133207, 0)

	unsignedTx := dummyUnsignedTx{
		BaseTx: txs.BaseTx{},
	}
	unsignedTx.SetBytes([]byte{0})

	customAssetID := ids.GenerateTestID()

	// Note that setting [chainTimestamp] also set's the handler's clock.
	// Adjust input/output locktimes accordingly.
	tests := []struct {
		description     string
		utxos           []*lux.UTXO
		ins             []*lux.TransferableInput
		outs            []*lux.TransferableOutput
		creds           []verify.Verifiable
		producedAmounts map[ids.ID]uint64
		expectedErr     error
	}{
		{
			description:     "no inputs, no outputs, no fee",
			utxos:           []*lux.UTXO{},
			ins:             []*lux.TransferableInput{},
			outs:            []*lux.TransferableOutput{},
			creds:           []verify.Verifiable{},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     nil,
		},
		{
			description: "no inputs, no outputs, positive fee",
			utxos:       []*lux.UTXO{},
			ins:         []*lux.TransferableInput{},
			outs:        []*lux.TransferableOutput{},
			creds:       []verify.Verifiable{},
			producedAmounts: map[ids.ID]uint64{
				luxAssetID: 1,
			},
			expectedErr: ErrInsufficientUnlockedFunds,
		},
		{
			description: "wrong utxo assetID, one input, no outputs, no fee",
			utxos: []*lux.UTXO{{
				Asset: lux.Asset{ID: customAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
				},
			}},
			ins: []*lux.TransferableInput{{
				Asset: lux.Asset{ID: luxAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     errAssetIDMismatch,
		},
		{
			description: "one wrong assetID input, no outputs, no fee",
			utxos: []*lux.UTXO{{
				Asset: lux.Asset{ID: luxAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
				},
			}},
			ins: []*lux.TransferableInput{{
				Asset: lux.Asset{ID: customAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     errAssetIDMismatch,
		},
		{
			description: "one input, one wrong assetID output, no fee",
			utxos: []*lux.UTXO{{
				Asset: lux.Asset{ID: luxAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
				},
			}},
			ins: []*lux.TransferableInput{{
				Asset: lux.Asset{ID: luxAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: customAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     ErrInsufficientUnlockedFunds,
		},
		{
			description: "attempt to consume locked output as unlocked",
			utxos: []*lux.UTXO{{
				Asset: lux.Asset{ID: luxAssetID},
				Out: &stakeable.LockOut{
					Locktime: uint64(now.Add(time.Second).Unix()),
					TransferableOut: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			}},
			ins: []*lux.TransferableInput{{
				Asset: lux.Asset{ID: luxAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     errLockedFundsNotMarkedAsLocked,
		},
		{
			description: "attempt to modify locktime",
			utxos: []*lux.UTXO{{
				Asset: lux.Asset{ID: luxAssetID},
				Out: &stakeable.LockOut{
					Locktime: uint64(now.Add(time.Second).Unix()),
					TransferableOut: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			}},
			ins: []*lux.TransferableInput{{
				Asset: lux.Asset{ID: luxAssetID},
				In: &stakeable.LockIn{
					Locktime: uint64(now.Unix()),
					TransferableIn: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			}},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     errLocktimeMismatch,
		},
		{
			description: "one input, no outputs, positive fee",
			utxos: []*lux.UTXO{{
				Asset: lux.Asset{ID: luxAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
				},
			}},
			ins: []*lux.TransferableInput{{
				Asset: lux.Asset{ID: luxAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{
				luxAssetID: 1,
			},
			expectedErr: nil,
		},
		{
			description: "wrong number of credentials",
			utxos: []*lux.UTXO{{
				Asset: lux.Asset{ID: luxAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
				},
			}},
			ins: []*lux.TransferableInput{{
				Asset: lux.Asset{ID: luxAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs:  []*lux.TransferableOutput{},
			creds: []verify.Verifiable{},
			producedAmounts: map[ids.ID]uint64{
				luxAssetID: 1,
			},
			expectedErr: errWrongNumberCredentials,
		},
		{
			description: "wrong number of UTXOs",
			utxos:       []*lux.UTXO{},
			ins: []*lux.TransferableInput{{
				Asset: lux.Asset{ID: luxAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{
				luxAssetID: 1,
			},
			expectedErr: errWrongNumberUTXOs,
		},
		{
			description: "invalid credential",
			utxos: []*lux.UTXO{{
				Asset: lux.Asset{ID: luxAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
				},
			}},
			ins: []*lux.TransferableInput{{
				Asset: lux.Asset{ID: luxAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				(*secp256k1fx.Credential)(nil),
			},
			producedAmounts: map[ids.ID]uint64{
				luxAssetID: 1,
			},
			expectedErr: secp256k1fx.ErrNilCredential,
		},
		{
			description: "invalid signature",
			utxos: []*lux.UTXO{{
				Asset: lux.Asset{ID: luxAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: 1,
						Addrs: []ids.ShortID{
							ids.GenerateTestShortID(),
						},
					},
				},
			}},
			ins: []*lux.TransferableInput{{
				Asset: lux.Asset{ID: luxAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
					Input: secp256k1fx.Input{
						SigIndices: []uint32{0},
					},
				},
			}},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{
					Sigs: [][secp256k1.SignatureLen]byte{
						{},
					},
				},
			},
			producedAmounts: map[ids.ID]uint64{
				luxAssetID: 1,
			},
			expectedErr: secp256k1.ErrRecoverFailed,
		},
		{
			description: "one input, no outputs, positive fee",
			utxos: []*lux.UTXO{{
				Asset: lux.Asset{ID: luxAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: 1,
				},
			}},
			ins: []*lux.TransferableInput{{
				Asset: lux.Asset{ID: luxAssetID},
				In: &secp256k1fx.TransferInput{
					Amt: 1,
				},
			}},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{
				luxAssetID: 1,
			},
			expectedErr: nil,
		},
		{
			description: "locked one input, no outputs, no fee",
			utxos: []*lux.UTXO{{
				Asset: lux.Asset{ID: luxAssetID},
				Out: &stakeable.LockOut{
					Locktime: uint64(now.Unix()) + 1,
					TransferableOut: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			}},
			ins: []*lux.TransferableInput{{
				Asset: lux.Asset{ID: luxAssetID},
				In: &stakeable.LockIn{
					Locktime: uint64(now.Unix()) + 1,
					TransferableIn: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			}},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     nil,
		},
		{
			description: "locked one input, no outputs, positive fee",
			utxos: []*lux.UTXO{{
				Asset: lux.Asset{ID: luxAssetID},
				Out: &stakeable.LockOut{
					Locktime: uint64(now.Unix()) + 1,
					TransferableOut: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			}},
			ins: []*lux.TransferableInput{{
				Asset: lux.Asset{ID: luxAssetID},
				In: &stakeable.LockIn{
					Locktime: uint64(now.Unix()) + 1,
					TransferableIn: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			}},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{
				luxAssetID: 1,
			},
			expectedErr: ErrInsufficientUnlockedFunds,
		},
		{
			description: "one locked and one unlocked input, one locked output, positive fee",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &stakeable.LockOut{
						Locktime: uint64(now.Unix()) + 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 1,
						},
					},
				},
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					In: &stakeable.LockIn{
						Locktime: uint64(now.Unix()) + 1,
						TransferableIn: &secp256k1fx.TransferInput{
							Amt: 1,
						},
					},
				},
				{
					Asset: lux.Asset{ID: luxAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &stakeable.LockOut{
						Locktime: uint64(now.Unix()) + 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 1,
						},
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{
				luxAssetID: 1,
			},
			expectedErr: nil,
		},
		{
			description: "one locked and one unlocked input, one locked output, positive fee, partially locked",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &stakeable.LockOut{
						Locktime: uint64(now.Unix()) + 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 1,
						},
					},
				},
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 2,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					In: &stakeable.LockIn{
						Locktime: uint64(now.Unix()) + 1,
						TransferableIn: &secp256k1fx.TransferInput{
							Amt: 1,
						},
					},
				},
				{
					Asset: lux.Asset{ID: luxAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 2,
					},
				},
			},
			outs: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &stakeable.LockOut{
						Locktime: uint64(now.Unix()) + 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 2,
						},
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{
				luxAssetID: 1,
			},
			expectedErr: nil,
		},
		{
			description: "one unlocked input, one locked output, zero fee",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &stakeable.LockOut{
						Locktime: uint64(now.Unix()) - 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 1,
						},
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     nil,
		},
		{
			description: "attempted overflow",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 2,
					},
				},
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: math.MaxUint64,
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     safemath.ErrOverflow,
		},
		{
			description: "attempted mint",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &stakeable.LockOut{
						Locktime: 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 2,
						},
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     ErrInsufficientLockedFunds,
		},
		{
			description: "attempted mint through locking",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &stakeable.LockOut{
						Locktime: 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 2,
						},
					},
				},
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &stakeable.LockOut{
						Locktime: 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: math.MaxUint64,
						},
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     safemath.ErrOverflow,
		},
		{
			description: "attempted mint through mixed locking (low then high)",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 2,
					},
				},
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &stakeable.LockOut{
						Locktime: 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: math.MaxUint64,
						},
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     ErrInsufficientLockedFunds,
		},
		{
			description: "attempted mint through mixed locking (high then low)",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: math.MaxUint64,
					},
				},
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &stakeable.LockOut{
						Locktime: 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 2,
						},
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     ErrInsufficientLockedFunds,
		},
		{
			description: "transfer non-lux asset",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: customAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: customAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: customAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     nil,
		},
		{
			description: "lock non-lux asset",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: customAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: customAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: customAssetID},
					Out: &stakeable.LockOut{
						Locktime: uint64(now.Add(time.Second).Unix()),
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 1,
						},
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     nil,
		},
		{
			description: "attempted asset conversion",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: customAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{},
			expectedErr:     ErrInsufficientUnlockedFunds,
		},
		{
			description: "attempted asset conversion with burn",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: customAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: customAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{
				luxAssetID: 1,
			},
			expectedErr: ErrInsufficientUnlockedFunds,
		},
		{
			description: "two inputs, one output with custom asset, with fee",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
				{
					Asset: lux.Asset{ID: customAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
				{
					Asset: lux.Asset{ID: customAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: customAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{
				luxAssetID: 1,
			},
			expectedErr: nil,
		},
		{
			description: "one input, fee, custom asset",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: customAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: customAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{
				luxAssetID: 1,
			},
			expectedErr: ErrInsufficientUnlockedFunds,
		},
		{
			description: "one input, custom fee",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: customAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: customAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{
				customAssetID: 1,
			},
			expectedErr: nil,
		},
		{
			description: "one input, custom fee, wrong burn",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{
				customAssetID: 1,
			},
			expectedErr: ErrInsufficientUnlockedFunds,
		},
		{
			description: "two inputs, multiple fee",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: luxAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
				{
					Asset: lux.Asset{ID: customAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: luxAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
				{
					Asset: lux.Asset{ID: customAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
				&secp256k1fx.Credential{},
			},
			producedAmounts: map[ids.ID]uint64{
				luxAssetID:    1,
				customAssetID: 1,
			},
			expectedErr: nil,
		},
		{
			description: "one unlock input, one locked output, zero fee, unlocked, custom asset",
			utxos: []*lux.UTXO{
				{
					Asset: lux.Asset{ID: customAssetID},
					Out: &stakeable.LockOut{
						Locktime: uint64(now.Unix()) - 1,
						TransferableOut: &secp256k1fx.TransferOutput{
							Amt: 1,
						},
					},
				},
			},
			ins: []*lux.TransferableInput{
				{
					Asset: lux.Asset{ID: customAssetID},
					In: &secp256k1fx.TransferInput{
						Amt: 1,
					},
				},
			},
			outs: []*lux.TransferableOutput{
				{
					Asset: lux.Asset{ID: customAssetID},
					Out: &secp256k1fx.TransferOutput{
						Amt: 1,
					},
				},
			},
			creds: []verify.Verifiable{
				&secp256k1fx.Credential{},
			},
			producedAmounts: make(map[ids.ID]uint64),
			expectedErr:     nil,
		},
	}

	for _, test := range tests {
		h.clk.Set(now)

		t.Run(test.description, func(t *testing.T) {
			err := h.VerifySpendUTXOs(
				&unsignedTx,
				test.utxos,
				test.ins,
				test.outs,
				test.creds,
				test.producedAmounts,
			)
			require.ErrorIs(t, err, test.expectedErr)
		})
	}
}
