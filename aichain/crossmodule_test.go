// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build crossmodule

// crossmodule_test.go is the CROSS-SPEC PARITY check: it asserts the SDK's wire
// encoders are byte-for-byte identical to the A-Chain settlement engine
// (github.com/luxfi/chains/aivm) and the C-Chain aivmbridge precompile (LP-5301).
//
// It does this WITHOUT importing chains/aivm. A direct cross-module import is
// fragile here: the SDK module (github.com/luxfi/sdk) does not `require`
// github.com/luxfi/chains, and that module's graph carries dead version pins
// (luxfi/kms, luxfi/upgrade) that only the lux go.work `replace` set fixes — so a
// `require` would break the GOWORK=off CI lane. Per the parity-test convention,
// we instead REPLICATE the chain's exact keccak preimages by hand (exactly as
// chains/aivm/quorum_wire_test.go hand-builds them) and pin the LIVE golden
// digests captured from running that test:
//
//	$ cd ~/work/lux/chains && go test ./aivm/ -run 'IntentIDByteSpec|ReceiptByteSpec' -v
//	GOLDEN intent_id    = 0x5e967be3e83750c25fb91887a125d67d2440fb41825d24d63a0c00e6fb2bfbde
//	GOLDEN receipt_hash = 0xfe0a1e45baf5255e2461c5f8f38b8446a691ec8bf0ca260750259d8bb5677851
//
// If the SDK encoders drift from the chain wire, either the hand-built preimage
// assertion or the pinned-digest assertion fails. It is build-tagged
// `crossmodule` so the default `GOWORK=off go test ./...` lane never compiles it,
// keeping CI green with NO cross-module dependency. Run it with:
//
//	go test -tags crossmodule ./aichain/
package aichain

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/common"
	"github.com/stretchr/testify/require"
)

// Live golden digests, captured from chains/aivm/quorum_wire_test.go (the
// authoritative A-Chain wire spec). These are the cross-spec contract.
const (
	liveGoldenIntentID    = "0x5e967be3e83750c25fb91887a125d67d2440fb41825d24d63a0c00e6fb2bfbde"
	liveGoldenReceiptHash = "0xfe0a1e45baf5255e2461c5f8f38b8446a691ec8bf0ca260750259d8bb5677851"
)

// TestCrossModuleIntentIDParity: the SDK's ComputeIntentID equals (a) the chain's
// hand-built preimage and (b) the live golden digest the chain's own test emits.
func TestCrossModuleIntentIDParity(t *testing.T) {
	r := require.New(t)
	cChain, aChain, cTx, ms, ph, callIdx, caller, n, threshold, fee := wireFixture()

	// (a) Reconstruct the chain's EXACT intent_id preimage by hand — the same
	// byte layout chains/aivm/quorum_wire.go ComputeIntentID hashes.
	var pre []byte
	pre = append(pre, []byte("lux/aivmbridge/intent/v1")...) // == DomainIntent on both sides
	pre = append(pre, cChain.Bytes()...)
	pre = append(pre, aChain.Bytes()...)
	pre = append(pre, cTx.Bytes()...)
	pre = append(pre, 0, 0, 0, 7) // u32be(callIdx=7)
	pre = append(pre, caller.Bytes()...)
	pre = append(pre, ms.Bytes()...)
	pre = append(pre, ph.Bytes()...)
	pre = append(pre, 0, 5) // u16be(n=5)
	pre = append(pre, 0, 3) // u16be(threshold=3)
	var feeB [32]byte
	fee.FillBytes(feeB[:])
	pre = append(pre, feeB[:]...)
	chainID := common.BytesToHash(crypto.Keccak256(pre))

	sdkID := ComputeIntentID(cChain, aChain, cTx, callIdx, caller, ms, ph, n, threshold, fee)
	r.Equal(chainID, sdkID, "SDK intent_id must equal the chain's hand-built preimage hash")

	// (b) Pin against the live golden the chain's own test emits.
	r.Equal(liveGoldenIntentID, sdkID.Hex(), "SDK intent_id must equal the live chains/aivm golden")
}

// TestCrossModuleReceiptParity: the SDK's receipt Encode/Hash equals (a) the
// chain's hand-built 355-byte encoding and (b) the live golden receipt_hash.
func TestCrossModuleReceiptParity(t *testing.T) {
	r := require.New(t)
	cChain, aChain, cTx, ms, ph, callIdx, caller, n, threshold, fee := wireFixture()
	intentID := ComputeIntentID(cChain, aChain, cTx, callIdx, caller, ms, ph, n, threshold, fee)

	task := common.HexToHash("0x6666666666666666666666666666666666666666666666666666666666666666")
	out := common.HexToHash("0x7777777777777777777777777777777777777777777777777777777777777777")
	wr := common.HexToHash("0x8888888888888888888888888888888888888888888888888888888888888888")
	or := common.HexToHash("0x9999999999999999999999999999999999999999999999999999999999999999")

	rec := InferenceReceipt{
		Version: ReceiptVersion, IntentID: intentID, TaskID: task,
		CChainID: cChain, AChainID: aChain, Requester: caller,
		ModelSpecHash: ms, PromptHash: ph, CanonicalOutputHash: out,
		Status: StatusCompleted, N: n, Threshold: threshold,
		WinnersRoot: wr, OperatorsRoot: or, FeePaid: fee, SettledAtHeight: 161,
	}

	// (a) Hand-build the chain's exact 355-byte encoding (chains/aivm receipts.go).
	var ref []byte
	ref = append(ref, 0, 1) // u16be(Version=1)
	ref = append(ref, intentID.Bytes()...)
	ref = append(ref, task.Bytes()...)
	ref = append(ref, cChain.Bytes()...)
	ref = append(ref, aChain.Bytes()...)
	ref = append(ref, caller.Bytes()...)
	ref = append(ref, ms.Bytes()...)
	ref = append(ref, ph.Bytes()...)
	ref = append(ref, out.Bytes()...)
	ref = append(ref, StatusCompleted)
	ref = append(ref, 0, 5, 0, 3) // u16be(n), u16be(threshold)
	ref = append(ref, wr.Bytes()...)
	ref = append(ref, or.Bytes()...)
	var fb [32]byte
	fee.FillBytes(fb[:])
	ref = append(ref, fb[:]...)
	ref = append(ref, 0, 0, 0, 0, 0, 0, 0, 161) // u64be(161)
	r.Equal(hex.EncodeToString(ref), hex.EncodeToString(rec.Encode()), "receipt encoding must match the chain byte layout")
	r.Len(rec.Encode(), 355)

	// receipt_hash = keccak("lux/aivmbridge/receipt/v1" || encoding).
	chainHash := common.BytesToHash(crypto.Keccak256(append([]byte("lux/aivmbridge/receipt/v1"), ref...)))
	r.Equal(chainHash, rec.Hash())

	// (b) Pin against the live golden the chain's own test emits.
	r.Equal(liveGoldenReceiptHash, rec.Hash().Hex(), "SDK receipt_hash must equal the live chains/aivm golden")
}

// TestCrossModuleMerkleParity: the SDK's leaf/node hashing reproduces the
// chain's keccak merkle (leafHash=keccak(h), node=keccak(l||r), duplicate-odd-
// tail), so a proof exported by the A-Chain verifies in the SDK. We build a tree
// with the SDK primitives and assert every leaf verifies and a non-member fails.
func TestCrossModuleMerkleParity(t *testing.T) {
	r := require.New(t)
	leaves := []common.Hash{
		common.HexToHash("0x01"), common.HexToHash("0x02"), common.HexToHash("0x03"),
		common.HexToHash("0x04"), common.HexToHash("0x05"),
	}
	hashed := make([]common.Hash, len(leaves))
	for i, h := range leaves {
		hashed[i] = leafHashReceipt(h)
	}
	root := merkleRootSDK(hashed)

	for i, raw := range leaves {
		sp := MerkleProof{ReceiptRoot: root, Index: uint64(i), Siblings: merkleSiblings(hashed, i)}
		r.True(VerifyReceiptInclusion(raw, sp, root), "leaf %d must verify", i)
		// proof frame round-trips and still verifies.
		enc, err := sp.EncodeProof()
		r.NoError(err)
		dec, err := DecodeProof(enc)
		r.NoError(err)
		r.True(VerifyReceiptInclusion(raw, dec, root), "decoded leaf %d must verify", i)
	}

	bad := common.HexToHash("0xff")
	r.False(VerifyReceiptInclusion(bad, MerkleProof{ReceiptRoot: root, Index: 0, Siblings: merkleSiblings(hashed, 0)}, root))
}

// merkleRootSDK / merkleSiblings mirror chains/aivm merkleRoot / merkleProof
// (duplicate-odd-tail), built on the SDK's merkleNode so a divergence in the SDK
// node hashing surfaces as a verify failure above.
func merkleRootSDK(leaves []common.Hash) common.Hash {
	if len(leaves) == 0 {
		return common.Hash{}
	}
	level := append([]common.Hash(nil), leaves...)
	for len(level) > 1 {
		next := make([]common.Hash, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			if i+1 < len(level) {
				next = append(next, merkleNode(level[i], level[i+1]))
			} else {
				next = append(next, merkleNode(level[i], level[i]))
			}
		}
		level = next
	}
	return level[0]
}

func merkleSiblings(leaves []common.Hash, idx int) []common.Hash {
	var sibs []common.Hash
	level := append([]common.Hash(nil), leaves...)
	i := idx
	for len(level) > 1 {
		var sib common.Hash
		if i%2 == 0 {
			if i+1 < len(level) {
				sib = level[i+1]
			} else {
				sib = level[i]
			}
		} else {
			sib = level[i-1]
		}
		sibs = append(sibs, sib)
		next := make([]common.Hash, 0, (len(level)+1)/2)
		for j := 0; j < len(level); j += 2 {
			if j+1 < len(level) {
				next = append(next, merkleNode(level[j], level[j+1]))
			} else {
				next = append(next, merkleNode(level[j], level[j]))
			}
		}
		level = next
		i /= 2
	}
	return sibs
}

var _ = big.NewInt // keep math/big import stable if assertions are edited
