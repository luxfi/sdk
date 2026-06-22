// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

import { describe, expect, it } from "vitest";
import type { Hex } from "viem";
import {
  AIChainClient,
  ReceiptStatus,
  computeIntentID,
  decodeGenerateResult,
  decodeGetApprovedResult,
  decodeProof,
  decodeReceipt,
  encodeGenerate,
  encodeProof,
  encodeReceipt,
  encodeRegisterModel,
  encodeSubmitInferenceIntent,
  encodeVerifyInferenceReceipt,
  isCompleted,
  modelSpecHash,
  promptHash,
  receiptHash,
  registryName,
  verifyReceiptInclusion,
  type InferenceReceipt,
  type ModelSpec,
  type MerkleProof,
} from "../src/index.js";

// GOLDEN vectors — these MUST equal the Go SDK's (github.com/luxfi/sdk/aichain
// wire_test.go / calldata_test.go) and the on-chain encoders (chains/aivm).
// If a value drifts, the cross-language wire broke.
const GOLDEN_INTENT_ID =
  "0x5e967be3e83750c25fb91887a125d67d2440fb41825d24d63a0c00e6fb2bfbde";
const GOLDEN_RECEIPT_HASH =
  "0xfe0a1e45baf5255e2461c5f8f38b8446a691ec8bf0ca260750259d8bb5677851";
const GOLDEN_MODELSPEC_HASH =
  "0xd8ab4fca51f36de6db2efd1a7a022fef6943de8d9986ae8ca3f4db70f318b4a7";

// The canonical fixture (identical to chains/aivm/quorum_wire_test.go).
const FIX = {
  cChainID: "0x1111111111111111111111111111111111111111111111111111111111111111" as Hex,
  aChainID: "0x2222222222222222222222222222222222222222222222222222222222222222" as Hex,
  cTxHash: "0x3333333333333333333333333333333333333333333333333333333333333333" as Hex,
  modelSpecHash: "0x4444444444444444444444444444444444444444444444444444444444444444" as Hex,
  promptHash: "0x5555555555555555555555555555555555555555555555555555555555555555" as Hex,
  callIndex: 7,
  caller: "0x00000000000000000000000000000000000000aa" as Hex,
  n: 5,
  threshold: 3,
  fee: 1_000_000n,
};

describe("intent id", () => {
  it("matches the Go/on-chain golden intent_id", () => {
    const id = computeIntentID(FIX);
    expect(id.toLowerCase()).toBe(GOLDEN_INTENT_ID);
  });
});

describe("modelSpec hash", () => {
  const spec: ModelSpec = {
    name: "zenlm/zen-omni",
    version: 3n,
    weightCommit: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890" as Hex,
    quantization: "int8",
  };
  it("matches the Go golden modelSpecHash", () => {
    expect(modelSpecHash(spec).toLowerCase()).toBe(GOLDEN_MODELSPEC_HASH);
    expect(registryName(spec).toLowerCase()).toBe(GOLDEN_MODELSPEC_HASH);
  });
  it("is injective (length-prefixed)", () => {
    const a: ModelSpec = { name: "ab", version: 0n, weightCommit: "0x0" as Hex, quantization: "cd" };
    const b: ModelSpec = { name: "abc", version: 0n, weightCommit: "0x0" as Hex, quantization: "d" };
    expect(modelSpecHash(a)).not.toBe(modelSpecHash(b));
  });
});

describe("receipt", () => {
  const rec: InferenceReceipt = {
    version: 1,
    intentId: GOLDEN_INTENT_ID as Hex,
    taskId: "0x6666666666666666666666666666666666666666666666666666666666666666" as Hex,
    cChainId: FIX.cChainID,
    aChainId: FIX.aChainID,
    requester: FIX.caller,
    modelSpecHash: FIX.modelSpecHash,
    promptHash: FIX.promptHash,
    canonicalOutputHash: "0x7777777777777777777777777777777777777777777777777777777777777777" as Hex,
    status: ReceiptStatus.Completed,
    n: 5,
    threshold: 3,
    winnersRoot: "0x8888888888888888888888888888888888888888888888888888888888888888" as Hex,
    operatorsRoot: "0x9999999999999999999999999999999999999999999999999999999999999999" as Hex,
    feePaid: 1_000_000n,
    settledAtHeight: 161n,
  };

  it("encodes to exactly 355 bytes and matches the golden receipt_hash", () => {
    const enc = encodeReceipt(rec);
    expect(enc.length).toBe(355);
    expect(receiptHash(rec).toLowerCase()).toBe(GOLDEN_RECEIPT_HASH);
  });

  it("round-trips through decodeReceipt", () => {
    const enc = encodeReceipt(rec);
    const back = decodeReceipt(enc);
    expect(receiptHash(back).toLowerCase()).toBe(GOLDEN_RECEIPT_HASH);
    expect(isCompleted(back)).toBe(true);
  });

  it("rejects a bad length", () => {
    expect(() => decodeReceipt(new Uint8Array(354))).toThrow();
    expect(() => decodeReceipt(new Uint8Array(356))).toThrow();
  });
});

describe("calldata golden hex (shared with Go)", () => {
  it("submitInferenceIntent", () => {
    const got = encodeSubmitInferenceIntent({
      modelSpecHash: FIX.modelSpecHash,
      promptHash: FIX.promptHash,
      n: 5,
      threshold: 3,
      fee: 1_000_000n,
      routing: "0x00000000000000000000000000000000000000000000000000000000deadbeef" as Hex,
    });
    const want =
      "0x10000000" +
      "4444444444444444444444444444444444444444444444444444444444444444" +
      "5555555555555555555555555555555555555555555555555555555555555555" +
      "0000000000000000000000000000000000000000000000000000000000000005" +
      "0000000000000000000000000000000000000000000000000000000000000003" +
      "00000000000000000000000000000000000000000000000000000000000f4240" +
      "00000000000000000000000000000000000000000000000000000000deadbeef";
    expect(got).toBe(want);
  });

  it("generate", () => {
    expect(encodeGenerate(10, [1, 7, 13, 2])).toBe(
      "0x01000000" + "0000000a" + "00000001" + "00000007" + "0000000d" + "00000002",
    );
    expect(decodeGenerateResult("0x00000001000000070000000d00000002")).toEqual([1, 7, 13, 2]);
  });

  it("registerModel", () => {
    const spec: ModelSpec = {
      name: "zenlm/zen-omni",
      version: 3n,
      weightCommit: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890" as Hex,
      quantization: "int8",
    };
    expect(encodeRegisterModel(spec)).toBe(
      "0x01000000" +
        "d8ab4fca51f36de6db2efd1a7a022fef6943de8d9986ae8ca3f4db70f318b4a7" +
        "0000000000000000000000000000000000000000000000000000000000000003" +
        "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
    );
  });

  it("getApproved decode", () => {
    const ret = ("0x" + "00".repeat(31) + "07" + "abc".padStart(64, "0")) as Hex;
    const am = decodeGetApprovedResult(ret);
    expect(am.version).toBe(7n);
  });
});

describe("verify frame + proof", () => {
  it("frames receipt+proof and round-trips the proof", () => {
    const rec: InferenceReceipt = {
      version: 1,
      intentId: GOLDEN_INTENT_ID as Hex,
      taskId: "0x00" as Hex,
      cChainId: FIX.cChainID,
      aChainId: FIX.aChainID,
      requester: FIX.caller,
      modelSpecHash: FIX.modelSpecHash,
      promptHash: FIX.promptHash,
      canonicalOutputHash: "0x01" as Hex,
      status: ReceiptStatus.Completed,
      n: 3,
      threshold: 2,
      winnersRoot: "0x00" as Hex,
      operatorsRoot: "0x00" as Hex,
      feePaid: 5n,
      settledAtHeight: 1n,
    };
    const proof: MerkleProof = {
      receiptRoot: "0xfeed" as Hex,
      index: 1n,
      siblings: ["0xaa" as Hex, "0xbb" as Hex],
    };
    const pb = encodeProof(proof);
    const cd = encodeVerifyInferenceReceipt(encodeReceipt(rec), pb);
    expect(cd.startsWith("0x11000000")).toBe(true);

    const back = decodeProof(pb);
    expect(back.index).toBe(1n);
    expect(back.siblings.length).toBe(2);
  });

  it("verifies a 5-leaf inclusion proof (chains/aivm merkle)", () => {
    // Build leaves, root, and a proof for each, then verify — mirrors the Go
    // cross-module merkle parity test.
    const leaves: Hex[] = ["0x01", "0x02", "0x03", "0x04", "0x05"].map((x) => x as Hex);
    const { root, proofs } = buildTree(leaves);
    leaves.forEach((leaf, i) => {
      expect(verifyReceiptInclusion(leaf, { receiptRoot: root, index: BigInt(i), siblings: proofs[i]! }, root)).toBe(
        true,
      );
    });
    expect(verifyReceiptInclusion("0xff" as Hex, { receiptRoot: root, index: 0n, siblings: proofs[0]! }, root)).toBe(
      false,
    );
  });
});

describe("client construction + Tier-1", () => {
  it("constructs read-only and runs generateDeterministic via a stub public client", async () => {
    // Minimal stub PublicClient: only call() + getChainId() are exercised.
    const stub = {
      getChainId: async () => 36911,
      call: async () => ({ data: "0x00000001000000070000000d000000020000000400000028" as Hex }),
    } as unknown as import("viem").PublicClient;
    const c = new AIChainClient({ publicClient: stub });
    expect(c.from()).toBeUndefined();
    const out = await c.generateDeterministic(2, [1, 7, 13, 2]);
    expect(out).toEqual([1, 7, 13, 2, 4, 40]);
  });

  it("rejects submit without a wallet", async () => {
    const stub = { getChainId: async () => 1, call: async () => ({ data: "0x" }) } as unknown as import("viem").PublicClient;
    const c = new AIChainClient({ publicClient: stub });
    await expect(
      c.submitInferenceIntent({ modelSpecHash: "0x44" as Hex, promptHash: "0x55" as Hex, n: 3, threshold: 2, fee: 0n }),
    ).rejects.toThrow();
  });
});

// --- helpers (mirror chains/aivm merkleRoot/merkleProof, duplicate-odd-tail) ---
import { merkleNode, leafHash } from "../src/index.js";

function buildTree(rawLeaves: Hex[]): { root: Hex; proofs: Hex[][] } {
  const hashed = rawLeaves.map((h) => leafHash(h));
  const root = foldRoot(hashed);
  const proofs = rawLeaves.map((_, i) => siblings(hashed, i));
  return { root, proofs };
}

function foldRoot(leaves: Hex[]): Hex {
  if (leaves.length === 0) return ("0x" + "00".repeat(32)) as Hex;
  let level = [...leaves];
  while (level.length > 1) {
    const next: Hex[] = [];
    for (let i = 0; i < level.length; i += 2) {
      next.push(i + 1 < level.length ? merkleNode(level[i]!, level[i + 1]!) : merkleNode(level[i]!, level[i]!));
    }
    level = next;
  }
  return level[0]!;
}

function siblings(leaves: Hex[], idx: number): Hex[] {
  const out: Hex[] = [];
  let level = [...leaves];
  let i = idx;
  while (level.length > 1) {
    let sib: Hex;
    if (i % 2 === 0) sib = i + 1 < level.length ? level[i + 1]! : level[i]!;
    else sib = level[i - 1]!;
    out.push(sib);
    const next: Hex[] = [];
    for (let j = 0; j < level.length; j += 2) {
      next.push(j + 1 < level.length ? merkleNode(level[j]!, level[j + 1]!) : merkleNode(level[j]!, level[j]!));
    }
    level = next;
    i = Math.floor(i / 2);
  }
  return out;
}
