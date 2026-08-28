// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package aichain

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/luxfi/geth/common"
	"github.com/stretchr/testify/require"
)

// calldata_test.go pins GOLDEN calldata hex for every encoder. The submit /
// generate / register vectors are the exact bytes the on-chain precompiles
// decode (LP-5301 frame, inference module.go token frame, modelregistry adopt
// ABI). The TS client (@luxfi/aichain) asserts the SAME vectors, so the two
// implementations are byte-for-byte equivalent.

// Shared Go<->TS golden inputs (also used in the TS test).
var (
	gMS      = common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444")
	gPrompt  = common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555")
	gRouting = common.HexToHash("0x00000000000000000000000000000000000000000000000000000000deadbeef")
)

func TestGoldenSubmitCalldata(t *testing.T) {
	r := require.New(t)
	got := EncodeSubmitInferenceIntent(gMS, gPrompt, 5, 3, big.NewInt(1_000_000), gRouting)
	r.Len(got, 4+submitIntentArgsLen, "selector + 6 words")
	const want = "10000000" + // selector
		"4444444444444444444444444444444444444444444444444444444444444444" + // modelSpecHash
		"5555555555555555555555555555555555555555555555555555555555555555" + // promptHash
		"0000000000000000000000000000000000000000000000000000000000000005" + // n=5
		"0000000000000000000000000000000000000000000000000000000000000003" + // threshold=3
		"00000000000000000000000000000000000000000000000000000000000f4240" + // fee=1_000_000
		"00000000000000000000000000000000000000000000000000000000deadbeef" // routing
	r.Equal(want, hex.EncodeToString(got))
}

func TestGoldenGenerateCalldata(t *testing.T) {
	r := require.New(t)
	got := EncodeGenerate(10, []uint32{1, 7, 13, 2})
	// selector(0x01000000) | u32be(10) | u32be each token.
	const want = "01000000" + "0000000a" + "00000001" + "00000007" + "0000000d" + "00000002"
	r.Equal(want, hex.EncodeToString(got))

	back, err := DecodeGenerateResult([]byte{0, 0, 0, 1, 0, 0, 0, 7, 0, 0, 0, 0xd, 0, 0, 0, 2})
	r.NoError(err)
	r.Equal([]uint32{1, 7, 13, 2}, back)
	_, err = DecodeGenerateResult([]byte{0, 0, 0}) // not multiple of 4
	r.Error(err)
}

func TestGoldenRegisterCalldata(t *testing.T) {
	r := require.New(t)
	spec := ModelSpec{
		Name:         "zenlm/zen-omni",
		Version:      3,
		WeightCommit: common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
		Quantization: "int8",
	}
	got := EncodeRegisterModel(spec)
	// selector(adopt) | name(=spec hash) | u256(version=3) | weightCommit.
	const want = "01000000" +
		"d8ab4fca51f36de6db2efd1a7a022fef6943de8d9986ae8ca3f4db70f318b4a7" + // RegistryName()==Hash()
		"0000000000000000000000000000000000000000000000000000000000000003" + // version
		"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890" // weightCommit
	r.Equal(want, hex.EncodeToString(got))
}

func TestVerifyCalldataRoundTrip(t *testing.T) {
	r := require.New(t)
	// Build a real receipt + a 2-node proof, frame them, and decode back.
	rec := InferenceReceipt{
		Version: ReceiptVersion, Status: StatusCompleted, N: 3, Threshold: 2,
		CanonicalOutputHash: common.HexToHash("0x01"), FeePaid: big.NewInt(5),
	}
	proof := MerkleProof{
		ReceiptRoot: common.HexToHash("0xfeed"),
		Index:       1,
		Siblings:    []common.Hash{common.HexToHash("0xaa"), common.HexToHash("0xbb")},
	}
	pb, err := proof.EncodeProof()
	r.NoError(err)
	r.Len(pb, proofFrameHeader+2*32)

	cd, err := EncodeVerifyInferenceReceipt(rec.Encode(), pb)
	r.NoError(err)
	// selector + u16 receiptLen + u16 proofLen + bodies.
	r.Equal(SelectorVerifyInferenceReceipt, uint32(cd[0])<<24|uint32(cd[1])<<16|uint32(cd[2])<<8|uint32(cd[3]))
	rl := int(cd[4])<<8 | int(cd[5])
	pl := int(cd[6])<<8 | int(cd[7])
	r.Equal(ReceiptEncodedLen, rl)
	r.Equal(len(pb), pl)

	// Decode the framed bodies back out and confirm they reproduce inputs.
	gotRec, err := DecodeReceipt(cd[8 : 8+rl])
	r.NoError(err)
	r.Equal(rec.Encode(), gotRec.Encode())
	gotProof, err := DecodeProof(cd[8+rl : 8+rl+pl])
	r.NoError(err)
	r.Equal(proof.ReceiptRoot, gotProof.ReceiptRoot)
	r.Equal(proof.Index, gotProof.Index)
	r.Equal(proof.Siblings, gotProof.Siblings)
}

func TestDecodeResults(t *testing.T) {
	r := require.New(t)

	// submit result: bytes32 intent id.
	id := common.HexToHash("0xdead")
	gotID, err := DecodeSubmitInferenceIntentResult(id.Bytes())
	r.NoError(err)
	r.Equal(id, gotID)
	_, err = DecodeSubmitInferenceIntentResult([]byte{1, 2, 3})
	r.Error(err)

	// verify result: 3 words.
	ret := make([]byte, 96)
	copy(ret[0:32], common.HexToHash("0x11").Bytes())
	copy(ret[32:64], common.HexToHash("0x22").Bytes())
	ret[95] = StatusCompleted
	vr, err := DecodeVerifyInferenceReceiptResult(ret)
	r.NoError(err)
	r.Equal(common.HexToHash("0x11"), vr.IntentID)
	r.Equal(common.HexToHash("0x22"), vr.CanonicalOutputHash)
	r.Equal(StatusCompleted, vr.Status)

	// getApproved result: (uint256 version, bytes32 weight).
	gar := make([]byte, 64)
	gar[31] = 7 // version in low byte of first word
	copy(gar[32:64], common.HexToHash("0xabc").Bytes())
	am, err := DecodeGetApprovedResult(gar)
	r.NoError(err)
	r.Equal(uint64(7), am.Version)
	r.Equal(common.HexToHash("0xabc"), am.WeightCommit)
}

func TestProofDecodeHardening(t *testing.T) {
	r := require.New(t)
	_, err := DecodeProof(make([]byte, proofFrameHeader-1)) // too short for header
	r.Error(err)

	// declare pathLen=1 but provide no path bytes -> length mismatch.
	bad := make([]byte, proofFrameHeader)
	bad[41] = 1 // pathLen low byte = 1
	_, err = DecodeProof(bad)
	r.Error(err)

	// declared depth over MaxProofDepth.
	over := make([]byte, proofFrameHeader)
	over[40] = byte((MaxProofDepth + 1) >> 8)
	over[41] = byte(MaxProofDepth + 1)
	_, err = DecodeProof(over)
	r.Error(err)
}
