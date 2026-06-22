// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package aichain

import (
	"fmt"
	"math/big"

	"github.com/luxfi/geth/common"
)

// calldata.go holds the PURE (no-chain) calldata encoders + return-data decoders
// for every op the SDK calls. Each encoder produces EXACTLY the bytes the target
// precompile decodes — the aivmbridge frames from LP-5301, the inference
// precompile's tight token frame, and the model registry's word-aligned ABI.
// calldata_test.go pins golden hex for each so an encoder drift fails the build.

// ---------------------------------------------------------------------------
// Selectors (first 4 bytes of calldata), pinned to the on-chain modules.
// ---------------------------------------------------------------------------
const (
	// aivmbridge (0x0300…0004), LP-5301.
	SelectorSubmitInferenceIntent  uint32 = 0x10000000
	SelectorVerifyInferenceReceipt uint32 = 0x11000000

	// inference precompile (0x0300…0003), LP-0303 — generate(uint32,uint32[]).
	SelectorGenerate uint32 = 0x01000000

	// model registry (0x0300…0002).
	SelectorAdopt       uint32 = 0x01000000
	SelectorGetApproved uint32 = 0x02000000
	SelectorIsAdmin     uint32 = 0x03000000
	SelectorSetAdmin    uint32 = 0x04000000
)

// Precompile addresses in the AI reserved range 0x0300…0000 .. 0x0300…00FF.
var (
	AIBridgeAddress      = common.HexToAddress("0x0300000000000000000000000000000000000004")
	InferenceAddress     = common.HexToAddress("0x0300000000000000000000000000000000000003")
	ModelRegistryAddress = common.HexToAddress("0x0300000000000000000000000000000000000002")
)

func selBytes(s uint32) []byte {
	return []byte{byte(s >> 24), byte(s >> 16), byte(s >> 8), byte(s)}
}

// word32 right-aligns b into a 32-byte EVM word (panics caller-side only on >32).
func word32(b []byte) []byte {
	w := make([]byte, 32)
	copy(w[32-len(b):], b)
	return w
}

// ---------------------------------------------------------------------------
// aivmbridge Pattern A — submitInferenceIntent
// ---------------------------------------------------------------------------

// submitIntentArgsLen is the fixed 6-word (192-byte) frame after the selector.
const submitIntentArgsLen = 6 * 32

// EncodeSubmitInferenceIntent builds calldata for the aivmbridge Pattern-A op
// (LP-5301). After the 4-byte selector, a fixed 6-word frame:
//
//	[0:32]    modelSpecHash (bytes32)
//	[32:64]   promptHash    (bytes32)
//	[64:96]   n             (uint16, right-aligned; high 30 bytes zero)
//	[96:128]  threshold     (uint16, right-aligned; high 30 bytes zero)
//	[128:160] fee           (uint256)
//	[160:192] routing       (bytes32, opaque transport hint)
//
// The precompile rejects a short OR oversized frame and dirty high bytes; this
// encoder always emits the exact 196-byte (4+192) well-formed frame.
func EncodeSubmitInferenceIntent(modelSpecHash, promptHash common.Hash, n, threshold uint16, fee *big.Int, routing common.Hash) []byte {
	out := make([]byte, 0, 4+submitIntentArgsLen)
	out = append(out, selBytes(SelectorSubmitInferenceIntent)...)
	out = append(out, modelSpecHash.Bytes()...)
	out = append(out, promptHash.Bytes()...)
	out = append(out, word32(u16be(n))...)
	out = append(out, word32(u16be(threshold))...)
	out = append(out, word32(u256be(fee))...)
	out = append(out, routing.Bytes()...)
	return out
}

// DecodeSubmitInferenceIntentResult decodes the aivmbridge Pattern-A return: a
// single bytes32 intentID.
func DecodeSubmitInferenceIntentResult(ret []byte) (common.Hash, error) {
	if len(ret) != 32 {
		return common.Hash{}, fmt.Errorf("aichain: submit return %d bytes, want 32", len(ret))
	}
	return common.BytesToHash(ret), nil
}

// ---------------------------------------------------------------------------
// aivmbridge Pattern B — verifyInferenceReceipt
// ---------------------------------------------------------------------------

// EncodeVerifyInferenceReceipt builds calldata for the aivmbridge Pattern-B op
// (LP-5301). After the 4-byte selector, a tight length-prefixed frame:
//
//	[0:2]      u16be receiptLen
//	[2:4]      u16be proofLen
//	[4:4+rl]   receipt bytes (canonical 355-byte AInferenceReceipt encoding)
//	[..]       proof bytes   (ReceiptRoot|Index|pathLen|path*32)
//
// `receipt` must be a canonical 355-byte encoding (InferenceReceipt.Encode) and
// `proof` the LP-5301 proof frame (MerkleProof.EncodeProof). Both are emitted
// verbatim; the precompile re-decodes them with the same exact-length discipline.
func EncodeVerifyInferenceReceipt(receipt, proof []byte) ([]byte, error) {
	if len(receipt) > 0xFFFF || len(proof) > 0xFFFF {
		return nil, fmt.Errorf("aichain: receipt/proof too long for u16 length prefix")
	}
	out := make([]byte, 0, 4+4+len(receipt)+len(proof))
	out = append(out, selBytes(SelectorVerifyInferenceReceipt)...)
	out = append(out, u16be(uint16(len(receipt)))...)
	out = append(out, u16be(uint16(len(proof)))...)
	out = append(out, receipt...)
	out = append(out, proof...)
	return out, nil
}

// VerifyResult is the decoded aivmbridge Pattern-B return:
// (bytes32 intentID, bytes32 canonicalOutputHash, uint8 status) packed as 3
// words (96 bytes).
type VerifyResult struct {
	IntentID            common.Hash
	CanonicalOutputHash common.Hash
	Status              uint8
}

// DecodeVerifyInferenceReceiptResult decodes the 96-byte Pattern-B return.
func DecodeVerifyInferenceReceiptResult(ret []byte) (VerifyResult, error) {
	var v VerifyResult
	if len(ret) != 96 {
		return v, fmt.Errorf("aichain: verify return %d bytes, want 96", len(ret))
	}
	v.IntentID = common.BytesToHash(ret[0:32])
	v.CanonicalOutputHash = common.BytesToHash(ret[32:64])
	v.Status = ret[95] // uint8 right-aligned in the third word
	return v, nil
}

// ---------------------------------------------------------------------------
// inference precompile (Tier-1) — generate(uint32 nNew, uint32[] promptTokens)
// ---------------------------------------------------------------------------

// EncodeGenerate builds calldata for the deterministic in-consensus inference
// precompile (LP-0303). The frame is tight (NOT Solidity-ABI): the 4-byte
// selector, then u32be(nNew), then each prompt token as u32be. This matches
// precompile/inference/module.go Run exactly (input[4:8] = nNew, input[8:] =
// big-endian uint32 tokens).
func EncodeGenerate(nNew uint32, promptTokens []uint32) []byte {
	out := make([]byte, 0, 4+4+len(promptTokens)*4)
	out = append(out, selBytes(SelectorGenerate)...)
	out = append(out, u32be(nNew)...)
	for _, t := range promptTokens {
		out = append(out, u32be(t)...)
	}
	return out
}

// DecodeGenerateResult decodes the inference precompile return: a tight
// big-endian uint32 array (prompt+generated tokens). Errors on a non-multiple-of-4
// length.
func DecodeGenerateResult(ret []byte) ([]uint32, error) {
	if len(ret)%4 != 0 {
		return nil, fmt.Errorf("aichain: generate return %d bytes, not a multiple of 4", len(ret))
	}
	out := make([]uint32, len(ret)/4)
	for i := range out {
		o := i * 4
		out[i] = uint32(ret[o])<<24 | uint32(ret[o+1])<<16 | uint32(ret[o+2])<<8 | uint32(ret[o+3])
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// model registry (0x0300…0002)
// ---------------------------------------------------------------------------

// EncodeRegisterModel builds calldata for the registry's
// adopt(bytes32 name, uint256 version, bytes32 weightHash) op. The name is the
// spec's canonical id (ModelSpec.RegistryName == Hash), version is the spec
// version, and weightHash is the spec's weight commitment. Layout after the
// selector: name(32) | version(32, uint256) | weightHash(32).
func EncodeRegisterModel(spec ModelSpec) []byte {
	out := make([]byte, 0, 4+96)
	out = append(out, selBytes(SelectorAdopt)...)
	out = append(out, spec.RegistryName().Bytes()...)
	out = append(out, word32(u64be(spec.Version))...)
	out = append(out, spec.WeightCommit.Bytes()...)
	return out
}

// EncodeGetApproved builds calldata for getApproved(bytes32 name) ->
// (uint256 version, bytes32 weightHash).
func EncodeGetApproved(name common.Hash) []byte {
	return append(selBytes(SelectorGetApproved), name.Bytes()...)
}

// ApprovedModel is the decoded getApproved return: the adopted version and weight
// commitment for a model name (weight == zero means none adopted).
type ApprovedModel struct {
	Version      uint64
	WeightCommit common.Hash
}

// DecodeGetApprovedResult decodes the 64-byte getApproved return:
// (uint256 version, bytes32 weightHash). Version is read from the low 8 bytes of
// the first word (matching modelregistry GetApproved, which reads v[24:32]).
func DecodeGetApprovedResult(ret []byte) (ApprovedModel, error) {
	var a ApprovedModel
	if len(ret) != 64 {
		return a, fmt.Errorf("aichain: getApproved return %d bytes, want 64", len(ret))
	}
	var v uint64
	for i := 24; i < 32; i++ {
		v = v<<8 | uint64(ret[i])
	}
	a.Version = v
	a.WeightCommit = common.BytesToHash(ret[32:64])
	return a, nil
}
