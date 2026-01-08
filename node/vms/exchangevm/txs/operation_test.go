// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/codec"
	"github.com/luxfi/sdk/node/codec/linearcodec"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/components/verify"
)

type testOperable struct {
	lux.TestTransferable `serialize:"true"`

	Outputs []verify.State `serialize:"true"`
}

func (*testOperable) InitCtx(context.Context) {}

func (*testOperable) InitializeContext(*consensusctx.Context) error { return nil }

func (o *testOperable) Outs() []verify.State {
	return o.Outputs
}

func TestOperationVerifyNil(t *testing.T) {
	op := (*Operation)(nil)
	err := op.Verify()
	require.ErrorIs(t, err, ErrNilOperation)
}

func TestOperationVerifyEmpty(t *testing.T) {
	op := &Operation{
		Asset: lux.Asset{ID: ids.Empty},
	}
	err := op.Verify()
	require.ErrorIs(t, err, ErrNilFxOperation)
}

func TestOperationVerifyUTXOIDsNotSorted(t *testing.T) {
	op := &Operation{
		Asset: lux.Asset{ID: ids.Empty},
		UTXOIDs: []*lux.UTXOID{
			{
				TxID:        ids.Empty,
				OutputIndex: 1,
			},
			{
				TxID:        ids.Empty,
				OutputIndex: 0,
			},
		},
		Op: &testOperable{},
	}
	err := op.Verify()
	require.ErrorIs(t, err, ErrNotSortedAndUniqueUTXOIDs)
}

func TestOperationVerify(t *testing.T) {
	assetID := ids.GenerateTestID()
	op := &Operation{
		Asset: lux.Asset{ID: assetID},
		UTXOIDs: []*lux.UTXOID{
			{
				TxID:        assetID,
				OutputIndex: 1,
			},
		},
		Op: &testOperable{},
	}
	require.NoError(t, op.Verify())
}

func TestOperationSorting(t *testing.T) {
	require := require.New(t)

	c := linearcodec.NewDefault()
	require.NoError(c.RegisterType(&testOperable{}))

	m := codec.NewDefaultManager()
	require.NoError(m.RegisterCodec(CodecVersion, c))

	ops := []*Operation{
		{
			Asset: lux.Asset{ID: ids.Empty},
			UTXOIDs: []*lux.UTXOID{
				{
					TxID:        ids.Empty,
					OutputIndex: 1,
				},
			},
			Op: &testOperable{},
		},
		{
			Asset: lux.Asset{ID: ids.Empty},
			UTXOIDs: []*lux.UTXOID{
				{
					TxID:        ids.Empty,
					OutputIndex: 0,
				},
			},
			Op: &testOperable{},
		},
	}
	require.False(IsSortedAndUniqueOperations(ops, m))
	SortOperations(ops, m)
	require.True(IsSortedAndUniqueOperations(ops, m))
	ops = append(ops, &Operation{
		Asset: lux.Asset{ID: ids.Empty},
		UTXOIDs: []*lux.UTXOID{
			{
				TxID:        ids.Empty,
				OutputIndex: 1,
			},
		},
		Op: &testOperable{},
	})
	require.False(IsSortedAndUniqueOperations(ops, m))
}

func TestOperationTxNotState(t *testing.T) {
	intf := interface{}(&OperationTx{})
	_, ok := intf.(verify.State)
	require.False(t, ok)
}
