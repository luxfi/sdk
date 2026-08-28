// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// calldata.ts holds the pure calldata encoders + return decoders, byte-for-byte
// identical to the Go SDK and the on-chain precompiles (LP-5301 frames, the
// inference token frame, the modelregistry adopt ABI).

import type { Hex } from "viem";
import {
  RECEIPT_ENCODED_LEN,
  ReceiptStatus,
  bytes32,
  bytesToHex,
  concatBytes,
  u16be,
  u256be,
  u32be,
  u64be,
} from "./bytesutil.js";
import { modelSpecHash, registryName, type ModelSpec } from "./modelspec.js";

// Selectors (first 4 bytes), pinned to the on-chain modules.
export const SELECTOR_SUBMIT_INTENT = 0x10000000;
export const SELECTOR_VERIFY_RECEIPT = 0x11000000;
export const SELECTOR_GENERATE = 0x01000000;
export const SELECTOR_ADOPT = 0x01000000;
export const SELECTOR_GET_APPROVED = 0x02000000;

// Precompile addresses (AI reserved range 0x0300…00xx).
export const AI_BRIDGE_ADDRESS = "0x0300000000000000000000000000000000000004" as Hex;
export const INFERENCE_ADDRESS = "0x0300000000000000000000000000000000000003" as Hex;
export const MODEL_REGISTRY_ADDRESS = "0x0300000000000000000000000000000000000002" as Hex;

const sel = (s: number): Uint8Array => u32be(s);
const word32 = (b: Uint8Array): Uint8Array => {
  const w = new Uint8Array(32);
  w.set(b, 32 - b.length);
  return w;
};

// ---------------------------------------------------------------------------
// aivmbridge Pattern A — submitInferenceIntent (6-word frame)
// ---------------------------------------------------------------------------

export function encodeSubmitInferenceIntent(args: {
  modelSpecHash: Hex;
  promptHash: Hex;
  n: number;
  threshold: number;
  fee: bigint;
  routing: Hex;
}): Hex {
  return bytesToHex(
    concatBytes(
      sel(SELECTOR_SUBMIT_INTENT),
      bytes32(args.modelSpecHash),
      bytes32(args.promptHash),
      word32(u16be(args.n)),
      word32(u16be(args.threshold)),
      word32(u256be(args.fee)),
      bytes32(args.routing),
    ),
  );
}

export function decodeSubmitInferenceIntentResult(ret: Hex): Hex {
  const b = hexBytes(ret);
  if (b.length !== 32) throw new Error(`aichain: submit return ${b.length} bytes, want 32`);
  return bytesToHex(b);
}

// ---------------------------------------------------------------------------
// aivmbridge Pattern B — verifyInferenceReceipt (length-prefixed frame)
// ---------------------------------------------------------------------------

export function encodeVerifyInferenceReceipt(receipt: Uint8Array, proof: Uint8Array): Hex {
  if (receipt.length > 0xffff || proof.length > 0xffff) {
    throw new Error("aichain: receipt/proof too long for u16 length prefix");
  }
  return bytesToHex(
    concatBytes(
      sel(SELECTOR_VERIFY_RECEIPT),
      u16be(receipt.length),
      u16be(proof.length),
      receipt,
      proof,
    ),
  );
}

export interface VerifyResult {
  intentId: Hex;
  canonicalOutputHash: Hex;
  status: ReceiptStatus;
}

export function decodeVerifyInferenceReceiptResult(ret: Hex): VerifyResult {
  const b = hexBytes(ret);
  if (b.length !== 96) throw new Error(`aichain: verify return ${b.length} bytes, want 96`);
  return {
    intentId: bytesToHex(b.subarray(0, 32)),
    canonicalOutputHash: bytesToHex(b.subarray(32, 64)),
    status: b[95]! as ReceiptStatus,
  };
}

// ---------------------------------------------------------------------------
// inference precompile (Tier-1) — generate(uint32 nNew, uint32[] promptTokens)
// ---------------------------------------------------------------------------

export function encodeGenerate(nNew: number, promptTokens: number[]): Hex {
  return bytesToHex(
    concatBytes(sel(SELECTOR_GENERATE), u32be(nNew), ...promptTokens.map((t) => u32be(t))),
  );
}

export function decodeGenerateResult(ret: Hex): number[] {
  const b = hexBytes(ret);
  if (b.length % 4 !== 0) throw new Error(`aichain: generate return ${b.length} bytes, not multiple of 4`);
  const out: number[] = [];
  for (let i = 0; i < b.length; i += 4) {
    out.push((b[i]! << 24) | (b[i + 1]! << 16) | (b[i + 2]! << 8) | b[i + 3]!);
  }
  return out;
}

// ---------------------------------------------------------------------------
// model registry — adopt / getApproved
// ---------------------------------------------------------------------------

export function encodeRegisterModel(spec: ModelSpec): Hex {
  return bytesToHex(
    concatBytes(
      sel(SELECTOR_ADOPT),
      bytes32(registryName(spec)),
      word32(u64be(spec.version)),
      bytes32(spec.weightCommit),
    ),
  );
}

export function encodeGetApproved(name: Hex): Hex {
  return bytesToHex(concatBytes(sel(SELECTOR_GET_APPROVED), bytes32(name)));
}

export interface ApprovedModel {
  version: bigint;
  weightCommit: Hex;
}

export function decodeGetApprovedResult(ret: Hex): ApprovedModel {
  const b = hexBytes(ret);
  if (b.length !== 64) throw new Error(`aichain: getApproved return ${b.length} bytes, want 64`);
  let version = 0n;
  for (let i = 24; i < 32; i++) version = (version << 8n) | BigInt(b[i]!);
  return { version, weightCommit: bytesToHex(b.subarray(32, 64)) };
}

// modelSpecHash re-exported for convenience at the calldata layer.
export { modelSpecHash };

function hexBytes(hex: Hex): Uint8Array {
  const clean = hex.startsWith("0x") ? hex.slice(2) : hex;
  const padded = clean.length % 2 ? "0" + clean : clean;
  const n = padded.length / 2;
  const out = new Uint8Array(n);
  for (let i = 0; i < n; i++) out[i] = parseInt(padded.slice(i * 2, i * 2 + 2), 16);
  return out;
}

export { RECEIPT_ENCODED_LEN };
