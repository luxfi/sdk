// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// wire.ts derives the cross-chain intent id + prompt hash, byte-for-byte
// identical to the Go SDK (github.com/luxfi/sdk/aichain) and the on-chain
// encoders (chains/aivm/quorum_wire.go, LP-5301). The fixed-width primitives live
// in bytesutil.ts (the one place width/endianness is decided); the golden vectors
// in test/wire.test.ts assert the cross-language equality.

import type { Hex } from "viem";
import {
  DOMAIN_INTENT,
  address20,
  bytes32,
  bytesToHex,
  keccakConcat,
  u16be,
  u256be,
  u32be,
  utf8,
} from "./bytesutil.js";
import { keccak256 } from "viem";

/**
 * computeIntentID derives the cross-chain intent id from the committed C-Chain
 * intent fields:
 *
 *   keccak256( DOMAIN_INTENT ||
 *     c_chain_id(32) || a_chain_id(32) || c_tx_hash(32) || u32be(call_index) ||
 *     caller(20) || model_spec_hash(32) || prompt_hash(32) ||
 *     u16be(N) || u16be(threshold) || u256be(fee,32) )
 */
export function computeIntentID(args: {
  cChainID: Hex;
  aChainID: Hex;
  cTxHash: Hex;
  callIndex: number;
  caller: Hex;
  modelSpecHash: Hex;
  promptHash: Hex;
  n: number;
  threshold: number;
  fee: bigint;
}): Hex {
  return keccakConcat(
    utf8(DOMAIN_INTENT),
    bytes32(args.cChainID),
    bytes32(args.aChainID),
    bytes32(args.cTxHash),
    u32be(args.callIndex),
    address20(args.caller),
    bytes32(args.modelSpecHash),
    bytes32(args.promptHash),
    u16be(args.n),
    u16be(args.threshold),
    u256be(args.fee),
  );
}

/** promptHash = keccak256(prompt bytes). */
export function promptHash(prompt: Uint8Array | string): Hex {
  const bytes = typeof prompt === "string" ? utf8(prompt) : prompt;
  return keccak256(bytesToHex(bytes));
}
