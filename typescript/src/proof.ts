// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

import { keccak256, type Hex } from "viem";
import {
  MAX_PROOF_DEPTH,
  bytes32,
  bytesToHex,
  concatBytes,
  u16be,
  u64be,
} from "./bytesutil.js";
import type { InferenceReceipt } from "./receipt.js";
import { receiptHash } from "./receipt.js";

const PROOF_FRAME_HEADER = 32 + 8 + 2; // ReceiptRoot(32) | u64be Index(8) | u16be pathLen(2)

/**
 * MerkleProof is an inclusion proof that a receipt_hash leaf is committed under a
 * receiptRoot. Mirrors chains/aivm MerkleProof and the LP-5301 proofBytes frame.
 */
export interface MerkleProof {
  receiptRoot: Hex;
  index: bigint | number;
  siblings: Hex[];
}

/** leafHash = keccak256(receipt_hash) — identical to chains/aivm leafHash. */
export function leafHash(h: Hex): Hex {
  return keccak256(bytesToHex(bytes32(h)));
}

/** merkleNode = keccak256(l || r) — identical to chains/aivm merkleNode. */
export function merkleNode(l: Hex, r: Hex): Hex {
  return keccak256(bytesToHex(concatBytes(bytes32(l), bytes32(r))));
}

/**
 * verifyReceiptInclusion checks that receiptHash is included under root at the
 * proof's index, exactly inverting the chains/aivm merkle construction.
 */
export function verifyReceiptInclusion(receiptHashHex: Hex, proof: MerkleProof, root: Hex): boolean {
  let cur = leafHash(receiptHashHex);
  let idx = BigInt(proof.index);
  for (const sib of proof.siblings) {
    cur = idx % 2n === 0n ? merkleNode(cur, sib) : merkleNode(sib, cur);
    idx /= 2n;
  }
  return eqHex(cur, root);
}

/** verifyReceipt confirms a receipt's hash is included under the proof's root. */
export function verifyReceipt(proof: MerkleProof, r: InferenceReceipt): boolean {
  return verifyReceiptInclusion(receiptHash(r), proof, proof.receiptRoot);
}

/**
 * encodeProof serializes a proof into the LP-5301 proofBytes wire frame:
 *   ReceiptRoot(32) | u64be Index(8) | u16be pathLen(2) | pathLen*32
 */
export function encodeProof(p: MerkleProof): Uint8Array {
  if (p.siblings.length > MAX_PROOF_DEPTH) {
    throw new Error(`aichain: proof depth ${p.siblings.length} exceeds max ${MAX_PROOF_DEPTH}`);
  }
  return concatBytes(
    bytes32(p.receiptRoot),
    u64be(p.index),
    u16be(p.siblings.length),
    ...p.siblings.map((s) => bytes32(s)),
  );
}

/**
 * decodeProof parses an LP-5301 proofBytes frame. Exact-length: rejects short
 * frames, over-depth, and trailing junk — the same hardening the precompile
 * applies.
 */
export function decodeProof(b: Uint8Array): MerkleProof {
  if (b.length < PROOF_FRAME_HEADER) {
    throw new Error(`aichain: proof frame ${b.length} bytes, need >= ${PROOF_FRAME_HEADER}`);
  }
  const receiptRoot = bytesToHex(b.subarray(0, 32));
  let index = 0n;
  for (let i = 0; i < 8; i++) index = (index << 8n) | BigInt(b[32 + i]!);
  const pathLen = (b[40]! << 8) | b[41]!;
  if (pathLen > MAX_PROOF_DEPTH) {
    throw new Error(`aichain: proof depth ${pathLen} exceeds max ${MAX_PROOF_DEPTH}`);
  }
  const want = PROOF_FRAME_HEADER + pathLen * 32;
  if (b.length !== want) {
    throw new Error(`aichain: proof frame ${b.length} bytes, want ${want} for pathLen ${pathLen}`);
  }
  const siblings: Hex[] = [];
  for (let i = 0; i < pathLen; i++) {
    const off = PROOF_FRAME_HEADER + i * 32;
    siblings.push(bytesToHex(b.subarray(off, off + 32)));
  }
  return { receiptRoot, index, siblings };
}

function eqHex(a: Hex, b: Hex): boolean {
  return a.toLowerCase() === b.toLowerCase();
}
