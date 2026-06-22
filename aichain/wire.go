// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

// Package aichain is the developer-facing client SDK for the Lux A-Chain
// (Beluga) "Thinking Chains" architecture: on-chain LLM inference and
// embeddings settled by quorum.
//
// It wraps the three on-chain entry points:
//
//   - the AI bridge precompile (LP-5301) at 0x0300…0004 — Tier-2 large-model
//     inference, submitted as a committed C-Chain intent and later settled by an
//     A-Chain quorum receipt (Pattern A submit + Pattern B verify);
//   - the deterministic in-consensus inference precompile (LP-0303) at
//     0x0300…0003 — Tier-1 small models run identically by every validator,
//     answered inside one EVM call;
//   - the model registry precompile at 0x0300…0002 — governance adoption of the
//     model the chain treats as canonical.
//
// wire.go pins the SHARED CROSS-CHAIN WIRE SPEC byte-for-byte. The encoders here
// MUST produce exactly the preimages the on-chain side hashes/decodes
// (chains/aivm/quorum_wire.go ComputeIntentID, chains/aivm/receipts.go
// AInferenceReceipt.Encode, and the LP-5301 aivmbridge calldata layout) or an
// intent submitted via this SDK will not re-derive to the same intent_id on
// A-Chain, and a receipt minted there will not verify against the SDK's
// recomputed hash. wire_test.go freezes golden vectors so any drift fails the
// build; the optional `crossmodule` build tag cross-checks against the live
// chains/aivm encoders.
//
// keccak256 is luxfi/crypto.Keccak256 — the canonical keccak in the Lux stack,
// the same primitive the geth-side precompile and the A-Chain settlement engine
// use — so a digest computed here equals one computed on either chain
// bit-for-bit.
package aichain

import (
	"math/big"

	"github.com/holiman/uint256"
	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/common"
)

// Domain separators (raw UTF-8 bytes, no length prefix — concatenated verbatim).
// They keep the intent-id, receipt-hash, and model-spec keyspaces disjoint and
// version the wire so a future v2 cannot collide with v1. The first two are the
// cross-chain contract pinned in chains/aivm/quorum_wire.go and MUST match
// byte-for-byte. DomainModelSpec is the SDK's canonical model-identity hash; it
// is the value carried as modelSpecHash in the intent (opaque to the chain,
// which only ever compares it for equality), so the SDK is its sole author.
const (
	DomainIntent    = "lux/aivmbridge/intent/v1"
	DomainReceipt   = "lux/aivmbridge/receipt/v1"
	DomainModelSpec = "lux/aivmbridge/modelspec/v1"
)

// ReceiptVersion is the wire version stamped into every receipt (the u16be
// Version field). Bumping it is a wire change shared with the settlement engine.
const ReceiptVersion uint16 = 1

// Receipt status codes (the AInferenceReceipt.Status byte). Pinned values shared
// with the bridge + settlement engine. Only StatusCompleted with a non-zero
// CanonicalOutputHash is actionable C-side.
const (
	StatusUnknown    uint8 = 0
	StatusPending    uint8 = 1
	StatusCompleted  uint8 = 2
	StatusFailed     uint8 = 3
	StatusChallenged uint8 = 4
)

// ReceiptEncodedLen is the exact byte length of the canonical AInferenceReceipt
// encoding (see AInferenceReceipt.Encode). Pinned so a drift in field widths or
// order is a compile/test failure, not a silent cross-chain mismatch.
//
//	u16(Version)2 + IntentID32 + TaskID32 + CChainID32 + AChainID32 +
//	Requester20 + ModelSpecHash32 + PromptHash32 + CanonicalOutputHash32 +
//	u8(Status)1 + u16(N)2 + u16(Threshold)2 + WinnersRoot32 + OperatorsRoot32 +
//	u256(FeePaid)32 + u64(SettledAtHeight)8
const ReceiptEncodedLen = 2 + 32 + 32 + 32 + 32 + 20 + 32 + 32 + 32 + 1 + 2 + 2 + 32 + 32 + 32 + 8 // = 355

// MaxFanout is the upper bound on the intent fan-out N (LP-5301). It also caps
// Threshold.
const MaxFanout uint16 = 256

// ---------------------------------------------------------------------------
// fixed-width encoders (the ONLY place width/endianness is decided)
// ---------------------------------------------------------------------------

func u16be(v uint16) []byte { return []byte{byte(v >> 8), byte(v)} }

func u32be(v uint32) []byte {
	return []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
}

func u64be(v uint64) []byte {
	return []byte{
		byte(v >> 56), byte(v >> 48), byte(v >> 40), byte(v >> 32),
		byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v),
	}
}

// u256be returns the 32-byte big-endian encoding of a non-negative integer.
// A nil pointer encodes as zero, exactly like chains/aivm's u256be, so an
// unspecified fee hashes identically on both sides. A negative big.Int is
// clamped to zero (a fee is unsigned on the wire).
func u256be(v *big.Int) []byte {
	if v == nil || v.Sign() <= 0 {
		return make([]byte, 32)
	}
	var u uint256.Int
	u.SetFromBig(v) // saturates >2^256-1 to max; fees never approach that
	b := u.Bytes32()
	return b[:]
}

// ComputeIntentID derives the cross-chain intent id from the COMMITTED C-Chain
// intent fields, exactly the preimage chains/aivm/quorum_wire.go ComputeIntentID
// hashes:
//
//	keccak256( DomainIntent ||
//	    c_chain_id(32) || a_chain_id(32) || c_tx_hash(32) || u32be(call_index) ||
//	    caller(20) || model_spec_hash(32) || prompt_hash(32) ||
//	    u16be(N) || u16be(threshold) || u256be(fee,32) )
//
// Every field is fixed-width in this precise order. The A-Chain importer
// recomputes this from the delivered fields and rejects the intent unless it
// equals the id the C side committed, so any altered field yields a different id.
//
// Note: the submitInferenceIntent CALLER does not pick the intent id — the
// precompile derives it from the landed tx hash and call index. The SDK exposes
// this so a client that already knows (cTxHash, callIndex) — e.g. after the tx
// lands — can reproduce the id locally and correlate the eventual receipt.
func ComputeIntentID(
	cChainID, aChainID, cTxHash common.Hash,
	callIndex uint32,
	caller common.Address,
	modelSpecHash, promptHash common.Hash,
	n, threshold uint16,
	fee *big.Int,
) common.Hash {
	buf := make([]byte, 0, len(DomainIntent)+32*3+4+20+32*2+2+2+32)
	buf = append(buf, []byte(DomainIntent)...)
	buf = append(buf, cChainID.Bytes()...)
	buf = append(buf, aChainID.Bytes()...)
	buf = append(buf, cTxHash.Bytes()...)
	buf = append(buf, u32be(callIndex)...)
	buf = append(buf, caller.Bytes()...)
	buf = append(buf, modelSpecHash.Bytes()...)
	buf = append(buf, promptHash.Bytes()...)
	buf = append(buf, u16be(n)...)
	buf = append(buf, u16be(threshold)...)
	buf = append(buf, u256be(fee)...)
	return common.BytesToHash(crypto.Keccak256(buf))
}

// PromptHash is the canonical commitment to a prompt/input: keccak256 of the raw
// prompt bytes. The chain treats promptHash as opaque (equality only), so any
// stable hash the requester and provider agree on works; this SDK standardises
// on keccak of the UTF-8/binary prompt so the same prompt always yields the same
// commitment.
func PromptHash(prompt []byte) common.Hash {
	return common.BytesToHash(crypto.Keccak256(prompt))
}
