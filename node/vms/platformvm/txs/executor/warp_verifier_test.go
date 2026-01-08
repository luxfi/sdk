// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	validators "github.com/luxfi/consensus/validator"
	validatorstest "github.com/luxfi/consensus/validator/validatorstest"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/codec"
	"github.com/luxfi/const"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/crypto/bls/signer/localsigner"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/platformvm/warp"
)

func TestVerifyWarpMessages(t *testing.T) {
	var (
		chainID      = ids.GenerateTestID()
		newValidator = func() (bls.Signer, *validators.GetValidatorOutput) {
			sk, err := localsigner.New()
			require.NoError(t, err)

			return sk, &validators.GetValidatorOutput{
				NodeID:    ids.GenerateTestNodeID(),
				PublicKey: bls.PublicKeyToUncompressedBytes(sk.PublicKey()),
				Weight:    1,
			}
		}
		sk0, vdr0 = newValidator()
		sk1, vdr1 = newValidator()
		vdrs      = map[ids.NodeID]*validators.GetValidatorOutput{
			vdr0.NodeID: vdr0,
			vdr1.NodeID: vdr1,
		}
		state = &validatorstest.State{
			GetValidatorSetF: func(context.Context, uint64, ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
				return vdrs, nil
			},
		}
	)

	validUnsignedWarpMessage, err := warp.NewUnsignedMessage(
		constants.UnitTestID,
		chainID,
		nil,
	)
	require.NoError(t, err)

	sig0, err := sk0.Sign(validUnsignedWarpMessage.Bytes())
	require.NoError(t, err)
	sig1, err := sk1.Sign(validUnsignedWarpMessage.Bytes())
	require.NoError(t, err)

	sig, err := bls.AggregateSignatures([]*bls.Signature{sig0, sig1})
	require.NoError(t, err)

	warpSignature := &warp.BitSetSignature{
		Signers:   set.NewBits(0, 1).Bytes(),
		Signature: [bls.SignatureLen]byte(bls.SignatureToBytes(sig)),
	}
	validWarpMessage, err := warp.NewMessage(
		validUnsignedWarpMessage,
		warpSignature,
	)
	require.NoError(t, err)

	invalidWarpMessage, err := warp.NewMessage(
		must[*warp.UnsignedMessage](t)(warp.NewUnsignedMessage(
			constants.UnitTestID+1,
			chainID,
			nil,
		)),
		warpSignature,
	)
	require.NoError(t, err)

	tests := []struct {
		name        string
		tx          txs.UnsignedTx
		expectedErr error
	}{
		{
			name: "AddValidatorTx",
			tx:   &txs.AddValidatorTx{},
		},
		{
			name: "AddChainValidatorTx",
			tx:   &txs.AddChainValidatorTx{},
		},
		{
			name: "AddDelegatorTx",
			tx:   &txs.AddDelegatorTx{},
		},
		{
			name: "CreateChainTx",
			tx:   &txs.CreateChainTx{},
		},
		{
			name: "CreateSubnetTx",
			tx:   &txs.CreateSubnetTx{},
		},
		{
			name: "ImportTx",
			tx:   &txs.ImportTx{},
		},
		{
			name: "ExportTx",
			tx:   &txs.ExportTx{},
		},
		{
			name: "AdvanceTimeTx",
			tx:   &txs.AdvanceTimeTx{},
		},
		{
			name: "RewardValidatorTx",
			tx:   &txs.RewardValidatorTx{},
		},
		{
			name: "RemoveChainValidatorTx",
			tx:   &txs.RemoveChainValidatorTx{},
		},
		{
			name: "TransformChainTx",
			tx:   &txs.TransformChainTx{},
		},
		{
			name: "AddPermissionlessValidatorTx",
			tx:   &txs.AddPermissionlessValidatorTx{},
		},
		{
			name: "AddPermissionlessDelegatorTx",
			tx:   &txs.AddPermissionlessDelegatorTx{},
		},
		{
			name: "TransferChainOwnershipTx",
			tx:   &txs.TransferChainOwnershipTx{},
		},
		{
			name: "BaseTx",
			tx:   &txs.BaseTx{},
		},
		{
			name: "ConvertChainToL1Tx",
			tx:   &txs.ConvertChainToL1Tx{},
		},
		{
			name:        "RegisterL1ValidatorTx with unparsable message",
			tx:          &txs.RegisterL1ValidatorTx{},
			expectedErr: codec.ErrCantUnpackVersion,
		},
		{
			name: "RegisterL1ValidatorTx with invalid message",
			tx: &txs.RegisterL1ValidatorTx{
				Message: invalidWarpMessage.Bytes(),
			},
			expectedErr: warp.ErrWrongNetworkID,
		},
		{
			name: "RegisterL1ValidatorTx with valid message",
			tx: &txs.RegisterL1ValidatorTx{
				Message: validWarpMessage.Bytes(),
			},
		},
		{
			name:        "SetL1ValidatorWeightTx with unparsable message",
			tx:          &txs.SetL1ValidatorWeightTx{},
			expectedErr: codec.ErrCantUnpackVersion,
		},
		{
			name: "SetL1ValidatorWeightTx with invalid message",
			tx: &txs.SetL1ValidatorWeightTx{
				Message: invalidWarpMessage.Bytes(),
			},
			expectedErr: warp.ErrWrongNetworkID,
		},
		{
			name: "SetL1ValidatorWeightTx with valid message",
			tx: &txs.SetL1ValidatorWeightTx{
				Message: validWarpMessage.Bytes(),
			},
		},
		{
			name: "IncreaseL1ValidatorBalanceTx",
			tx:   &txs.IncreaseL1ValidatorBalanceTx{},
		},
		{
			name: "DisableL1ValidatorTx",
			tx:   &txs.DisableL1ValidatorTx{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := VerifyWarpMessages(
				context.Background(),
				constants.UnitTestID,
				state,
				0,
				test.tx,
			)
			require.Equal(t, test.expectedErr, err)
		})
	}
}
