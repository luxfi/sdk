// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package aichain

import (
	"fmt"

	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/common"
)

// MaxProofDepth caps the Merkle path length the verifier (and the precompile)
// will walk — a 2^64-leaf tree is never reached in practice, but an unbounded
// pathLen is a calldata-DoS, so it is hard-capped (LP-5301 MaxProofDepth).
const MaxProofDepth = 64

// proofFrameHeader is the fixed prefix of the LP-5301 proof wire frame:
// ReceiptRoot(32) | u64be Index(8) | u16be pathLen(2).
const proofFrameHeader = 32 + 8 + 2

// MerkleProof is an inclusion proof that a receipt_hash leaf is committed under a
// ReceiptRoot: the sibling hashes from leaf to root plus the leaf Index (which
// fixes left/right at each level). It mirrors chains/aivm MerkleProof and the
// LP-5301 proofBytes frame exactly so a proof exported by the A-Chain
// (Engine.ExportReceipt) verifies here and vice-versa.
type MerkleProof struct {
	ReceiptRoot common.Hash   `json:"receiptRoot"` // the committed root this proof is against
	Index       uint64        `json:"index"`       // leaf index in the tree
	Siblings    []common.Hash `json:"siblings"`    // sibling at each level, leaf->root
}

// leafHashReceipt is the receipt-hash leaf hash: keccak256 over the raw
// receipt_hash digest (one extra keccak so a leaf can never be confused with an
// internal node). Identical to chains/aivm leafHash.
func leafHashReceipt(h common.Hash) common.Hash {
	return common.BytesToHash(crypto.Keccak256(h.Bytes()))
}

// merkleNode combines two child hashes: keccak256(l || r). Identical to
// chains/aivm merkleNode.
func merkleNode(l, r common.Hash) common.Hash {
	return common.BytesToHash(crypto.Keccak256(l.Bytes(), r.Bytes()))
}

// VerifyReceiptInclusion checks that receiptHash is included under root at the
// proof's Index, re-applying the leaf hash and folding with the siblings,
// choosing left/right by the index bit at each level — exactly inverting the
// chains/aivm merkleProof/merkleRoot construction. Returns true iff the
// recomputed root equals root.
func VerifyReceiptInclusion(receiptHash common.Hash, proof MerkleProof, root common.Hash) bool {
	cur := leafHashReceipt(receiptHash)
	idx := proof.Index
	for _, sib := range proof.Siblings {
		if idx%2 == 0 {
			cur = merkleNode(cur, sib)
		} else {
			cur = merkleNode(sib, cur)
		}
		idx /= 2
	}
	return cur == root
}

// Verify is the receipt-aware check: it confirms the proof is against the root
// the proof itself carries (proof.ReceiptRoot) and that the receipt's Hash() is
// included under it. The CALLER must still confirm proof.ReceiptRoot is a root
// the C-Chain holds committed (the precompile does this against its receipt-root
// checkpoint; off-chain you compare against a known-good root).
func (p MerkleProof) Verify(r InferenceReceipt) bool {
	return VerifyReceiptInclusion(r.Hash(), p, p.ReceiptRoot)
}

// EncodeProof serializes a proof into the LP-5301 proofBytes wire frame:
//
//	ReceiptRoot(32) | u64be Index(8) | u16be pathLen(2) | pathLen*32
//
// This is the exact bytes the aivmbridge verifyInferenceReceipt op decodes.
func (p MerkleProof) EncodeProof() ([]byte, error) {
	if len(p.Siblings) > MaxProofDepth {
		return nil, fmt.Errorf("aichain: proof depth %d exceeds max %d", len(p.Siblings), MaxProofDepth)
	}
	out := make([]byte, 0, proofFrameHeader+len(p.Siblings)*32)
	out = append(out, p.ReceiptRoot.Bytes()...)
	out = append(out, u64be(p.Index)...)
	out = append(out, u16be(uint16(len(p.Siblings)))...)
	for _, s := range p.Siblings {
		out = append(out, s.Bytes()...)
	}
	return out, nil
}

// DecodeProof parses an LP-5301 proofBytes frame back into a MerkleProof. It is
// the inverse of EncodeProof and is exact-length: it rejects a short frame, a
// frame whose declared pathLen exceeds MaxProofDepth, and a frame with trailing
// junk — the same calldata-hardening the precompile applies.
func DecodeProof(b []byte) (MerkleProof, error) {
	var p MerkleProof
	if len(b) < proofFrameHeader {
		return p, fmt.Errorf("aichain: proof frame %d bytes, need >= %d", len(b), proofFrameHeader)
	}
	p.ReceiptRoot = common.BytesToHash(b[0:32])
	var idx uint64
	for i := 0; i < 8; i++ {
		idx = idx<<8 | uint64(b[32+i])
	}
	p.Index = idx
	pathLen := int(uint16(b[40])<<8 | uint16(b[41]))
	if pathLen > MaxProofDepth {
		return p, fmt.Errorf("aichain: proof depth %d exceeds max %d", pathLen, MaxProofDepth)
	}
	want := proofFrameHeader + pathLen*32
	if len(b) != want {
		return p, fmt.Errorf("aichain: proof frame %d bytes, want %d for pathLen %d", len(b), want, pathLen)
	}
	p.Siblings = make([]common.Hash, pathLen)
	for i := 0; i < pathLen; i++ {
		off := proofFrameHeader + i*32
		p.Siblings[i] = common.BytesToHash(b[off : off+32])
	}
	return p, nil
}
