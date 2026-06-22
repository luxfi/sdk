// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

import type { Hex } from "viem";
import { DOMAIN_MODELSPEC, bytes32, keccakConcat, u32be, u64be, utf8 } from "./bytesutil.js";

/**
 * ModelSpec is the SDK's canonical description of a model. On-chain a model is
 * referenced only by its 32-byte weight-commitment hash; this hashes down to
 * that key, byte-identical to the Go SDK's ModelSpec.Hash.
 */
export interface ModelSpec {
  name: string;
  version: bigint | number;
  weightCommit: Hex;
  quantization: string;
}

/**
 * modelSpecHash is the canonical model id:
 *
 *   keccak256( DOMAIN_MODELSPEC ||
 *     u32be(len(name)) || name ||
 *     u64be(version) ||
 *     weightCommit(32) ||
 *     u32be(len(quantization)) || quantization )
 */
export function modelSpecHash(spec: ModelSpec): Hex {
  const name = utf8(spec.name);
  const quant = utf8(spec.quantization);
  return keccakConcat(
    utf8(DOMAIN_MODELSPEC),
    u32be(name.length),
    name,
    u64be(spec.version),
    bytes32(spec.weightCommit),
    u32be(quant.length),
    quant,
  );
}

/** The registry name a model is keyed by (== modelSpecHash). */
export function registryName(spec: ModelSpec): Hex {
  return modelSpecHash(spec);
}
