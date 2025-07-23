// (c) 2023, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"context"
	"math"
	"testing"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/utils"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/node/consensus/validators/validatorsmock"
	"github.com/luxfi/evm/interfaces"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// This test copies the test coverage from https://github.com/luxfi/node/blob/v1.10.0/vms/platformvm/warp/signature_test.go#L137.
// These tests are only expected to fail if there is a breaking change in Lux that unexpectedly changes behavior.
type signatureTest struct {
	name         string
	stateF       func(*gomock.Controller) interfaces.State
	quorumNum    uint64
	quorumDen    uint64
	msgF         func(*require.Assertions) *interfaces.Message
	verifyErr    error
	canonicalErr error
}

// This test copies the test coverage from https://github.com/luxfi/node/blob/0117ab96/vms/platformvm/warp/signature_test.go#L137.
// These tests are only expected to fail if there is a breaking change in Lux that unexpectedly changes behavior.
func TestSignatureVerification(t *testing.T) {
	tests := []signatureTest{
		{
			name: "can't get subnetID",
			stateF: func(ctrl *gomock.Controller) interfaces.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, errTest)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *interfaces.Message {
				unsignedMsg, err := interfaces.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				msg, err := interfaces.NewMessage(
					unsignedMsg,
					&interfaces.BitSetSignature{},
				)
				require.NoError(err)
				return msg
			},
			canonicalErr: errTest,
		},
		{
			name: "can't get validator set",
			stateF: func(ctrl *gomock.Controller) interfaces.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, sourceSubnetID).Return(nil, errTest)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *interfaces.Message {
				unsignedMsg, err := interfaces.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				msg, err := interfaces.NewMessage(
					unsignedMsg,
					&interfaces.BitSetSignature{},
				)
				require.NoError(err)
				return msg
			},
			canonicalErr: errTest,
		},
		{
			name: "weight overflow",
			stateF: func(ctrl *gomock.Controller) interfaces.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, sourceSubnetID).Return(map[ids.NodeID]*interfaces.GetValidatorOutput{
					testVdrs[0].nodeID: {
						NodeID:    testVdrs[0].nodeID,
						PublicKey: testVdrs[0].vdr.PublicKey,
						Weight:    interfaces.MaxUint64,
					},
					testVdrs[1].nodeID: {
						NodeID:    testVdrs[1].nodeID,
						PublicKey: testVdrs[1].vdr.PublicKey,
						Weight:    interfaces.MaxUint64,
					},
				}, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *interfaces.Message {
				unsignedMsg, err := interfaces.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				msg, err := interfaces.NewMessage(
					unsignedMsg,
					&interfaces.BitSetSignature{
						Signers: make([]byte, 8),
					},
				)
				require.NoError(err)
				return msg
			},
			err: interfaces.ErrWeightOverflow,
			canonicalErr: interfaces.ErrWeightOverflow,
		},
		{
			name: "invalid bit set index",
			stateF: func(ctrl *gomock.Controller) interfaces.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *interfaces.Message {
				unsignedMsg, err := interfaces.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				msg, err := interfaces.NewMessage(
					unsignedMsg,
					&interfaces.BitSetSignature{
						Signers:   make([]byte, 1),
						Signature: [interfaces.SignatureLen]byte{},
					},
				)
				require.NoError(err)
				return msg
			},
			err: interfaces.ErrInvalidBitSet,
			verifyErr: interfaces.ErrInvalidBitSet,
		},
		{
			name: "unknown index",
			stateF: func(ctrl *gomock.Controller) interfaces.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *interfaces.Message {
				unsignedMsg, err := interfaces.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				signers := set.NewBits()
				signers.Add(3) // vdr oob

				msg, err := interfaces.NewMessage(
					unsignedMsg,
					&interfaces.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: [interfaces.SignatureLen]byte{},
					},
				)
				require.NoError(err)
				return msg
			},
			err: interfaces.ErrUnknownValidator,
			verifyErr: interfaces.ErrUnknownValidator,
		},
		{
			name: "insufficient weight",
			stateF: func(ctrl *gomock.Controller) interfaces.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 1,
			msgF: func(require *require.Assertions) *interfaces.Message {
				unsignedMsg, err := interfaces.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				// [signers] has weight from [vdr[0], vdr[1]],
				// which is 6, which is less than 9
				signers := set.NewBits()
				signers.Add(0)
				signers.Add(1)

				unsignedBytes := unsignedMsg.Bytes()
				vdr0Sig, err := testVdrs[0].sk.Sign(unsignedBytes)
				require.NoError(err)
				vdr1Sig, err := testVdrs[1].sk.Sign(unsignedBytes)
				require.NoError(err)
				aggSig, err := interfaces.AggregateSignatures([]*interfaces.Signature{vdr0Sig, vdr1Sig})
				require.NoError(err)
				aggSigBytes := [interfaces.SignatureLen]byte{}
				copy(aggSigBytes[:], interfaces.SignatureToBytes(aggSig))

				msg, err := interfaces.NewMessage(
					unsignedMsg,
					&interfaces.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			err: interfaces.ErrInsufficientWeight,
			verifyErr: interfaces.ErrInsufficientWeight,
		},
		{
			name: "can't parse sig",
			stateF: func(ctrl *gomock.Controller) interfaces.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *interfaces.Message {
				unsignedMsg, err := interfaces.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				signers := set.NewBits()
				signers.Add(0)
				signers.Add(1)

				msg, err := interfaces.NewMessage(
					unsignedMsg,
					&interfaces.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: [interfaces.SignatureLen]byte{},
					},
				)
				require.NoError(err)
				return msg
			},
			err: interfaces.ErrParseSignature,
			verifyErr: interfaces.ErrParseSignature,
		},
		{
			name: "no validators",
			stateF: func(ctrl *gomock.Controller) interfaces.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, sourceSubnetID).Return(nil, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *interfaces.Message {
				unsignedMsg, err := interfaces.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				unsignedBytes := unsignedMsg.Bytes()
				vdr0Sig, err := testVdrs[0].sk.Sign(unsignedBytes)
				require.NoError(err)
				aggSigBytes := [interfaces.SignatureLen]byte{}
				copy(aggSigBytes[:], interfaces.SignatureToBytes(vdr0Sig))

				msg, err := interfaces.NewMessage(
					unsignedMsg,
					&interfaces.BitSetSignature{
						Signers:   nil,
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			verifyErr: interfaces.ErrNoPublicKeys,
		},
		{
			name: "invalid signature (substitute)",
			stateF: func(ctrl *gomock.Controller) interfaces.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 3,
			quorumDen: 5,
			msgF: func(require *require.Assertions) *interfaces.Message {
				unsignedMsg, err := interfaces.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				signers := set.NewBits()
				signers.Add(0)
				signers.Add(1)

				unsignedBytes := unsignedMsg.Bytes()
				vdr0Sig, err := testVdrs[0].sk.Sign(unsignedBytes)
				require.NoError(err)
				// Give sig from vdr[2] even though the bit vector says it
				// should be from vdr[1]
				vdr2Sig, err := testVdrs[2].sk.Sign(unsignedBytes)
				require.NoError(err)
				aggSig, err := interfaces.AggregateSignatures([]*interfaces.Signature{vdr0Sig, vdr2Sig})
				require.NoError(err)
				aggSigBytes := [interfaces.SignatureLen]byte{}
				copy(aggSigBytes[:], interfaces.SignatureToBytes(aggSig))

				msg, err := interfaces.NewMessage(
					unsignedMsg,
					&interfaces.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			err: interfaces.ErrInvalidSignature,
			verifyErr: interfaces.ErrInvalidSignature,
		},
		{
			name: "invalid signature (missing one)",
			stateF: func(ctrl *gomock.Controller) interfaces.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 3,
			quorumDen: 5,
			msgF: func(require *require.Assertions) *interfaces.Message {
				unsignedMsg, err := interfaces.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				signers := set.NewBits()
				signers.Add(0)
				signers.Add(1)

				unsignedBytes := unsignedMsg.Bytes()
				vdr0Sig, err := testVdrs[0].sk.Sign(unsignedBytes)
				require.NoError(err)
				// Don't give the sig from vdr[1]
				aggSigBytes := [interfaces.SignatureLen]byte{}
				copy(aggSigBytes[:], interfaces.SignatureToBytes(vdr0Sig))

				msg, err := interfaces.NewMessage(
					unsignedMsg,
					&interfaces.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			err: interfaces.ErrInvalidSignature,
			verifyErr: interfaces.ErrInvalidSignature,
		},
		{
			name: "invalid signature (extra one)",
			stateF: func(ctrl *gomock.Controller) interfaces.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 3,
			quorumDen: 5,
			msgF: func(require *require.Assertions) *interfaces.Message {
				unsignedMsg, err := interfaces.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				signers := set.NewBits()
				signers.Add(0)
				signers.Add(1)

				unsignedBytes := unsignedMsg.Bytes()
				vdr0Sig, err := testVdrs[0].sk.Sign(unsignedBytes)
				require.NoError(err)
				vdr1Sig, err := testVdrs[1].sk.Sign(unsignedBytes)
				require.NoError(err)
				// Give sig from vdr[2] even though the bit vector doesn't have
				// it
				vdr2Sig, err := testVdrs[2].sk.Sign(unsignedBytes)
				require.NoError(err)
				aggSig, err := interfaces.AggregateSignatures([]*interfaces.Signature{vdr0Sig, vdr1Sig, vdr2Sig})
				require.NoError(err)
				aggSigBytes := [interfaces.SignatureLen]byte{}
				copy(aggSigBytes[:], interfaces.SignatureToBytes(aggSig))

				msg, err := interfaces.NewMessage(
					unsignedMsg,
					&interfaces.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			err: interfaces.ErrInvalidSignature,
			verifyErr: interfaces.ErrInvalidSignature,
		},
		{
			name: "valid signature",
			stateF: func(ctrl *gomock.Controller) interfaces.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 2,
			msgF: func(require *require.Assertions) *interfaces.Message {
				unsignedMsg, err := interfaces.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				// [signers] has weight from [vdr[1], vdr[2]],
				// which is 6, which is greater than 4.5
				signers := set.NewBits()
				signers.Add(1)
				signers.Add(2)

				unsignedBytes := unsignedMsg.Bytes()
				vdr1Sig, err := testVdrs[1].sk.Sign(unsignedBytes)
				require.NoError(err)
				vdr2Sig, err := testVdrs[2].sk.Sign(unsignedBytes)
				require.NoError(err)
				aggSig, err := interfaces.AggregateSignatures([]*interfaces.Signature{vdr1Sig, vdr2Sig})
				require.NoError(err)
				aggSigBytes := [interfaces.SignatureLen]byte{}
				copy(aggSigBytes[:], interfaces.SignatureToBytes(aggSig))

				msg, err := interfaces.NewMessage(
					unsignedMsg,
					&interfaces.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			verifyErr: nil,
		},
		{
			name: "valid signature (boundary)",
			stateF: func(ctrl *gomock.Controller) interfaces.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, sourceSubnetID).Return(vdrs, nil)
				return state
			},
			quorumNum: 2,
			quorumDen: 3,
			msgF: func(require *require.Assertions) *interfaces.Message {
				unsignedMsg, err := interfaces.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				// [signers] has weight from [vdr[1], vdr[2]],
				// which is 6, which meets the minimum 6
				signers := set.NewBits()
				signers.Add(1)
				signers.Add(2)

				unsignedBytes := unsignedMsg.Bytes()
				vdr1Sig, err := testVdrs[1].sk.Sign(unsignedBytes)
				require.NoError(err)
				vdr2Sig, err := testVdrs[2].sk.Sign(unsignedBytes)
				require.NoError(err)
				aggSig, err := interfaces.AggregateSignatures([]*interfaces.Signature{vdr1Sig, vdr2Sig})
				require.NoError(err)
				aggSigBytes := [interfaces.SignatureLen]byte{}
				copy(aggSigBytes[:], interfaces.SignatureToBytes(aggSig))

				msg, err := interfaces.NewMessage(
					unsignedMsg,
					&interfaces.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			verifyErr: nil,
		},
		{
			name: "valid signature (missing key)",
			stateF: func(ctrl *gomock.Controller) interfaces.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, sourceSubnetID).Return(map[ids.NodeID]*interfaces.GetValidatorOutput{
					testVdrs[0].nodeID: {
						NodeID:    testVdrs[0].nodeID,
						PublicKey: nil,
						Weight:    testVdrs[0].vdr.Weight,
					},
					testVdrs[1].nodeID: {
						NodeID:    testVdrs[1].nodeID,
						PublicKey: testVdrs[1].vdr.PublicKey,
						Weight:    testVdrs[1].vdr.Weight,
					},
					testVdrs[2].nodeID: {
						NodeID:    testVdrs[2].nodeID,
						PublicKey: testVdrs[2].vdr.PublicKey,
						Weight:    testVdrs[2].vdr.Weight,
					},
				}, nil)
				return state
			},
			quorumNum: 1,
			quorumDen: 3,
			msgF: func(require *require.Assertions) *interfaces.Message {
				unsignedMsg, err := interfaces.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				// [signers] has weight from [vdr2, vdr3],
				// which is 6, which is greater than 3
				signers := set.NewBits()
				// Note: the bits are shifted because vdr[0]'s key was zeroed
				signers.Add(0) // vdr[1]
				signers.Add(1) // vdr[2]

				unsignedBytes := unsignedMsg.Bytes()
				vdr1Sig, err := testVdrs[1].sk.Sign(unsignedBytes)
				require.NoError(err)
				vdr2Sig, err := testVdrs[2].sk.Sign(unsignedBytes)
				require.NoError(err)
				aggSig, err := interfaces.AggregateSignatures([]*interfaces.Signature{vdr1Sig, vdr2Sig})
				require.NoError(err)
				aggSigBytes := [interfaces.SignatureLen]byte{}
				copy(aggSigBytes[:], interfaces.SignatureToBytes(aggSig))

				msg, err := interfaces.NewMessage(
					unsignedMsg,
					&interfaces.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			verifyErr: nil,
		},
		{
			name: "valid signature (duplicate key)",
			stateF: func(ctrl *gomock.Controller) interfaces.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetSubnetID(gomock.Any(), sourceChainID).Return(sourceSubnetID, nil)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, sourceSubnetID).Return(map[ids.NodeID]*interfaces.GetValidatorOutput{
					testVdrs[0].nodeID: {
						NodeID:    testVdrs[0].nodeID,
						PublicKey: nil,
						Weight:    testVdrs[0].vdr.Weight,
					},
					testVdrs[1].nodeID: {
						NodeID:    testVdrs[1].nodeID,
						PublicKey: testVdrs[2].vdr.PublicKey,
						Weight:    testVdrs[1].vdr.Weight,
					},
					testVdrs[2].nodeID: {
						NodeID:    testVdrs[2].nodeID,
						PublicKey: testVdrs[2].vdr.PublicKey,
						Weight:    testVdrs[2].vdr.Weight,
					},
				}, nil)
				return state
			},
			quorumNum: 2,
			quorumDen: 3,
			msgF: func(require *require.Assertions) *interfaces.Message {
				unsignedMsg, err := interfaces.NewUnsignedMessage(
					networkID,
					sourceChainID,
					addressedPayloadBytes,
				)
				require.NoError(err)

				// [signers] has weight from [vdr2, vdr3],
				// which is 6, which meets the minimum 6
				signers := set.NewBits()
				// Note: the bits are shifted because vdr[0]'s key was zeroed
				// Note: vdr[1] and vdr[2] were combined because of a shared pk
				signers.Add(0) // vdr[1] + vdr[2]

				unsignedBytes := unsignedMsg.Bytes()
				// Because vdr[1] and vdr[2] share a key, only one of them sign.
				vdr2Sig, err := testVdrs[2].sk.Sign(unsignedBytes)
				require.NoError(err)
				aggSigBytes := [interfaces.SignatureLen]byte{}
				copy(aggSigBytes[:], interfaces.SignatureToBytes(vdr2Sig))

				msg, err := interfaces.NewMessage(
					unsignedMsg,
					&interfaces.BitSetSignature{
						Signers:   signers.Bytes(),
						Signature: aggSigBytes,
					},
				)
				require.NoError(err)
				return msg
			},
			verifyErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			msg := tt.msgF(require)
			pChainState := tt.stateF(ctrl)

			validatorSet, err := interfaces.GetCanonicalValidatorSetFromChainID(
				context.Background(),
				pChainState,
				pChainHeight,
				msg.UnsignedMessage.SourceChainID,
			)
			require.ErrorIs(err, tt.canonicalErr)
			if err != nil {
				return
			}
			err = msg.Signature.Verify(
				&msg.UnsignedMessage,
				networkID,
				validatorSet,
				tt.quorumNum,
				tt.quorumDen,
			)
			require.ErrorIs(err, tt.verifyErr)
		})
	}
}
