// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// bytesutil.ts is the ONE place fixed-width / endianness encoding lives for the
// TS SDK (the analogue of the Go SDK's u16be/u32be/u64be/u256be). Everything
// else builds on these so width/endianness is decided exactly once — byte-for-
// byte identical to the Go encoders the golden vectors pin.

import { keccak256, type Hex } from "viem";

export const DOMAIN_INTENT = "lux/aivmbridge/intent/v1";
export const DOMAIN_RECEIPT = "lux/aivmbridge/receipt/v1";
export const DOMAIN_MODELSPEC = "lux/aivmbridge/modelspec/v1";

export const RECEIPT_VERSION = 1;
export const RECEIPT_ENCODED_LEN = 355;
export const MAX_FANOUT = 256;
export const MAX_PROOF_DEPTH = 64;

export enum ReceiptStatus {
  Unknown = 0,
  Pending = 1,
  Completed = 2,
  Failed = 3,
  Challenged = 4,
}

const textEncoder = new TextEncoder();

/** UTF-8 bytes of a string. */
export function utf8(s: string): Uint8Array {
  return textEncoder.encode(s);
}

/** Big-endian fixed-width encoding of a non-negative integer into `len` bytes. */
export function uintBE(value: bigint | number, len: number): Uint8Array {
  let v = typeof value === "bigint" ? value : BigInt(value);
  if (v < 0n) v = 0n; // unsigned on the wire
  const out = new Uint8Array(len);
  for (let i = len - 1; i >= 0 && v > 0n; i--) {
    out[i] = Number(v & 0xffn);
    v >>= 8n;
  }
  return out;
}

export const u16be = (v: number | bigint): Uint8Array => uintBE(v, 2);
export const u32be = (v: number | bigint): Uint8Array => uintBE(v, 4);
export const u64be = (v: number | bigint): Uint8Array => uintBE(v, 8);
export const u256be = (v: bigint | number): Uint8Array => uintBE(v, 32);

/** A 0x-hex string to a byte array, optionally right-aligned into `len` bytes. */
export function hexToBytes(hex: Hex | string, len?: number): Uint8Array {
  let clean = hex.startsWith("0x") ? hex.slice(2) : hex;
  if (len !== undefined) {
    if (clean.length > len * 2) clean = clean.slice(clean.length - len * 2); // truncate high bytes
    clean = clean.padStart(len * 2, "0");
  } else if (clean.length % 2) {
    clean = "0" + clean;
  }
  const n = clean.length / 2;
  const out = new Uint8Array(n);
  for (let i = 0; i < n; i++) out[i] = parseInt(clean.slice(i * 2, i * 2 + 2), 16);
  return out;
}

/** A byte array to a 0x-hex string. */
export function bytesToHex(b: Uint8Array): Hex {
  let s = "0x";
  for (const x of b) s += x.toString(16).padStart(2, "0");
  return s as Hex;
}

/** Right-aligned 32-byte word. */
export function bytes32(hex: Hex | string): Uint8Array {
  return hexToBytes(hex, 32);
}

/** Right-aligned 20-byte address. */
export function address20(hex: Hex | string): Uint8Array {
  return hexToBytes(hex, 20);
}

/** Concatenate byte chunks. */
export function concatBytes(...chunks: Uint8Array[]): Uint8Array {
  let total = 0;
  for (const c of chunks) total += c.length;
  const out = new Uint8Array(total);
  let o = 0;
  for (const c of chunks) {
    out.set(c, o);
    o += c.length;
  }
  return out;
}

/** keccak256 over the concatenation of byte chunks, returned as 0x-hex. */
export function keccakConcat(...chunks: Uint8Array[]): Hex {
  return keccak256(bytesToHex(concatBytes(...chunks)));
}
