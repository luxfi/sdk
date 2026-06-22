// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

import { keccak256, type Hex } from "viem";
import {
  DOMAIN_RECEIPT,
  RECEIPT_ENCODED_LEN,
  ReceiptStatus,
  address20,
  bytes32,
  bytesToHex,
  concatBytes,
  u16be,
  u256be,
  u64be,
  utf8,
} from "./bytesutil.js";

/**
 * InferenceReceipt is the cross-chain settlement receipt. Its canonical encoding
 * is the pinned 355-byte layout (encodeReceipt), byte-identical to
 * chains/aivm/receipts.go AInferenceReceipt and the Go SDK.
 */
export interface InferenceReceipt {
  version: number;
  intentId: Hex;
  taskId: Hex;
  cChainId: Hex;
  aChainId: Hex;
  requester: Hex;
  modelSpecHash: Hex;
  promptHash: Hex;
  canonicalOutputHash: Hex;
  status: ReceiptStatus;
  n: number;
  threshold: number;
  winnersRoot: Hex;
  operatorsRoot: Hex;
  feePaid: bigint;
  settledAtHeight: bigint | number;
}

/**
 * encodeReceipt produces the canonical 355-byte fixed-width encoding in the
 * pinned order:
 *
 *   u16be(version) || intentId(32) || taskId(32) || cChainId(32) || aChainId(32)
 *   || requester(20) || modelSpecHash(32) || promptHash(32) ||
 *   canonicalOutputHash(32) || u8(status) || u16be(n) || u16be(threshold) ||
 *   winnersRoot(32) || operatorsRoot(32) || u256be(feePaid,32) ||
 *   u64be(settledAtHeight)
 */
export function encodeReceipt(r: InferenceReceipt): Uint8Array {
  return concatBytes(
    u16be(r.version),
    bytes32(r.intentId),
    bytes32(r.taskId),
    bytes32(r.cChainId),
    bytes32(r.aChainId),
    address20(r.requester),
    bytes32(r.modelSpecHash),
    bytes32(r.promptHash),
    bytes32(r.canonicalOutputHash),
    Uint8Array.from([r.status & 0xff]),
    u16be(r.n),
    u16be(r.threshold),
    bytes32(r.winnersRoot),
    bytes32(r.operatorsRoot),
    u256be(r.feePaid),
    u64be(r.settledAtHeight),
  );
}

/** receiptHash = keccak256(DOMAIN_RECEIPT || encodeReceipt(r)). */
export function receiptHash(r: InferenceReceipt): Hex {
  return keccak256(bytesToHex(concatBytes(utf8(DOMAIN_RECEIPT), encodeReceipt(r))));
}

/** Only a Completed receipt with a non-zero output is actionable C-side. */
export function isCompleted(r: InferenceReceipt): boolean {
  return r.status === ReceiptStatus.Completed && !isZeroHash(r.canonicalOutputHash);
}

function isZeroHash(h: Hex): boolean {
  return /^0x0*$/.test(h);
}

/**
 * decodeReceipt parses a canonical 355-byte receipt encoding. Fail-secure:
 * rejects any input that is not exactly RECEIPT_ENCODED_LEN bytes.
 */
export function decodeReceipt(b: Uint8Array): InferenceReceipt {
  if (b.length !== RECEIPT_ENCODED_LEN) {
    throw new Error(`aichain: receipt length ${b.length}, want ${RECEIPT_ENCODED_LEN}`);
  }
  let o = 0;
  const rd = (n: number): Uint8Array => {
    const s = b.subarray(o, o + n);
    o += n;
    return s;
  };
  const hex = (n: number): Hex => bytesToHex(rd(n));
  const num = (n: number): bigint => {
    let v = 0n;
    for (const x of rd(n)) v = (v << 8n) | BigInt(x);
    return v;
  };
  const version = Number(num(2));
  const intentId = hex(32);
  const taskId = hex(32);
  const cChainId = hex(32);
  const aChainId = hex(32);
  const requester = hex(20);
  const modelSpecHash = hex(32);
  const promptHash = hex(32);
  const canonicalOutputHash = hex(32);
  const status = Number(num(1)) as ReceiptStatus;
  const n = Number(num(2));
  const threshold = Number(num(2));
  const winnersRoot = hex(32);
  const operatorsRoot = hex(32);
  const feePaid = num(32);
  const settledAtHeight = num(8);
  return {
    version,
    intentId,
    taskId,
    cChainId,
    aChainId,
    requester,
    modelSpecHash,
    promptHash,
    canonicalOutputHash,
    status,
    n,
    threshold,
    winnersRoot,
    operatorsRoot,
    feePaid,
    settledAtHeight,
  };
}
