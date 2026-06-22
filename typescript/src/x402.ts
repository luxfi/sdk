// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

// x402.ts is the x402 (HTTP 402 "Payment Required") machine-payment layer for
// pay-per-cognition, mirroring the Go SDK (github.com/luxfi/sdk/aichain x402.go)
// BYTE-FOR-BYTE. It joins an off-chain x402 settlement to an on-chain
// Proof-of-Thought receipt:
//
//   x402 flow: client wants inference -> server answers 402 with
//   PaymentRequirements -> client returns a SIGNED PaymentPayload (X-PAYMENT) ->
//   a Facilitator verifies + settles it -> server serves the result + a
//   SettlementReceipt (X-PAYMENT-RESPONSE).
//
// The load-bearing value is paymentHash: the EIP-712 digest of the signed
// authorization. It is the SAME value the on-chain ProofOfThoughtRegistry stores
// as ThoughtReceipt.paymentHash AND the SAME value the Go SDK computes (the
// golden-vector parity test in test/x402.test.ts asserts equality), so a paid
// x402 inference can be registered on-chain and proven payment<->thought.
//
// The fixed-width primitives live in bytesutil.ts (the one place width/endianness
// is decided); signing/recovery uses viem so the SDK adds no crypto dependency.

import { keccak256, recoverAddress, serializeSignature, type Hex } from "viem";
import { privateKeyToAccount, sign } from "viem/accounts";
import { bytes32, bytesToHex, hexToBytes, keccakConcat, u256be, utf8 } from "./bytesutil.js";
import { modelSpecHash, type ModelSpec } from "./modelspec.js";
import { receiptHash, type InferenceReceipt } from "./receipt.js";
import type { AIChainClient } from "./client.js";

// ---------------------------------------------------------------------------
// EIP-712 domain + type strings (the ONLY place the x402 signing preimage lives)
// ---------------------------------------------------------------------------

// EIP-712 digest (identical to Go x402.go):
//
//   paymentHash = keccak256( 0x19 || 0x01 || domainSeparator || hashStruct(msg) )
//
//   domainSeparator = keccak256(
//     keccak256(EIP712_DOMAIN_TYPE) || keccak256(name) || keccak256(version) ||
//     u256(chainId) )
//
//   hashStruct(msg) = keccak256(
//     keccak256(PAYMENT_AUTH_TYPE) ||
//     from(32) || to(32) || value(32) || token(32) ||
//     u256(validAfter) || u256(validBefore) || nonce(32) )
//
// Addresses are left-padded to 32 bytes (EIP-712 encodeData). This is the EXACT
// preimage the Go SDK reproduces (golden-vector parity).
const EIP712_DOMAIN_TYPE = "EIP712Domain(string name,string version,uint256 chainId)";
const PAYMENT_AUTH_TYPE =
  "PaymentAuthorization(address from,address to,uint256 value,address token,uint256 validAfter,uint256 validBefore,bytes32 nonce)";
const X402_DOMAIN_NAME = "x402PaymentAuthorization";
const X402_DOMAIN_VERSION = "1";

// ---------------------------------------------------------------------------
// x402 wire types
// ---------------------------------------------------------------------------

/**
 * X402Scheme: `exact` settles exactly maxAmount; `upto` settles at most it. The
 * signed authorization is identical; the scheme tells the facilitator how to
 * interpret maxAmount vs the settled amount.
 */
export type X402Scheme = "exact" | "upto";

/** PaymentRequirements is what a resource server returns in its HTTP 402. */
export interface PaymentRequirements {
  scheme: X402Scheme;
  /** EVM chain id the token/settlement lives on. */
  network: bigint;
  /** Amount (exact) or cap (upto), in token base units. */
  maxAmount: bigint;
  /** ERC-20 payment token (zero address == native). */
  token: Hex;
  /** Payee (resource server's receiving address). */
  payTo: Hex;
  /** The resource being paid for (e.g. model id / URL). */
  resource: string;
  /** Human-readable description. */
  description: string;
  /** Server-chosen unique nonce (replay protection). */
  nonce: Hex;
  /** Unix seconds the authorization is valid until (validBefore). */
  validUntil: bigint;
}

/**
 * PaymentPayload is the signed authorization sent in the X-PAYMENT header. An
 * EIP-3009-shaped transferWithAuthorization over (from, to, value, validAfter,
 * validBefore, nonce) bound to (token, chainId) via the EIP-712 domain.
 */
export interface PaymentPayload {
  scheme: X402Scheme;
  network: bigint;
  from: Hex;
  to: Hex;
  value: bigint;
  token: Hex;
  validAfter: bigint;
  validBefore: bigint;
  nonce: Hex;
  /** 65-byte secp256k1 [R||S||V] hex. */
  signature: Hex;
}

/** SettlementReceipt is the facilitator's X-PAYMENT-RESPONSE. */
export interface SettlementReceipt {
  /** EIP-712 digest of the payload (== on-chain PoT paymentHash). */
  paymentHash: Hex;
  /** Settlement tx (zero if off-chain). */
  txHash: Hex;
  settled: boolean;
  payer: Hex;
  payee: Hex;
  /** Amount actually settled (<= payload.value for `upto`). */
  amount: bigint;
}

const ZERO_HASH = "0x0000000000000000000000000000000000000000000000000000000000000000" as Hex;
const ZERO_ADDR = "0x0000000000000000000000000000000000000000" as Hex;

// ---------------------------------------------------------------------------
// EIP-712 digest (paymentHash)
// ---------------------------------------------------------------------------

/** EIP-712 domain separator for the x402 domain bound to chainId. */
export function x402DomainSeparator(chainId: bigint): Hex {
  return keccakConcat(
    hexToBytes(keccak256(bytesToHex(utf8(EIP712_DOMAIN_TYPE)))),
    hexToBytes(keccak256(bytesToHex(utf8(X402_DOMAIN_NAME)))),
    hexToBytes(keccak256(bytesToHex(utf8(X402_DOMAIN_VERSION)))),
    u256be(chainId),
  );
}

/** EIP-712 hashStruct of the PaymentAuthorization message. */
function hashStruct(p: PaymentPayload): Hex {
  return keccakConcat(
    hexToBytes(keccak256(bytesToHex(utf8(PAYMENT_AUTH_TYPE)))),
    bytes32(p.from),
    bytes32(p.to),
    u256be(p.value),
    bytes32(p.token),
    u256be(p.validAfter),
    u256be(p.validBefore),
    bytes32(p.nonce),
  );
}

/**
 * paymentHash is the EIP-712 signing digest of the authorization:
 * keccak256(0x1901 || domainSeparator(network) || hashStruct(msg)). This is the
 * value the signature is over AND the value the on-chain registry stores as
 * paymentHash — the canonical x402<->PoT join key (matches the Go SDK).
 */
export function paymentHash(p: PaymentPayload): Hex {
  return keccakConcat(
    Uint8Array.from([0x19, 0x01]),
    hexToBytes(x402DomainSeparator(p.network)),
    hexToBytes(hashStruct(p)),
  );
}

// ---------------------------------------------------------------------------
// build + sign
// ---------------------------------------------------------------------------

function validateRequirements(r: PaymentRequirements): void {
  if (r.scheme !== "exact" && r.scheme !== "upto") {
    throw new Error(`aichain/x402: unknown scheme ${String(r.scheme)}`);
  }
  if (r.network <= 0n) throw new Error("aichain/x402: missing/zero network (chain id)");
  if (r.maxAmount <= 0n) throw new Error("aichain/x402: missing/zero maxAmount");
  if (isZero(r.payTo)) throw new Error("aichain/x402: zero payTo");
  if (isZero(r.nonce)) throw new Error("aichain/x402: zero nonce");
  if (r.validUntil <= 0n) throw new Error("aichain/x402: zero validUntil");
}

/**
 * buildPaymentPayload constructs the x402 authorization from a server's
 * PaymentRequirements and EIP-712-signs it with `privateKey`. The returned
 * payload's paymentHash() is the EIP-712 digest the signature is over (and the
 * PoT join key). Deterministic: same (key, requirements) -> same payload +
 * paymentHash + signature. ASYNC because viem signing is async.
 */
export async function buildPaymentPayload(
  privateKey: Hex,
  reqs: PaymentRequirements,
): Promise<PaymentPayload> {
  validateRequirements(reqs);
  const account = privateKeyToAccount(privateKey);
  const p: PaymentPayload = {
    scheme: reqs.scheme,
    network: reqs.network,
    from: account.address,
    to: reqs.payTo,
    value: reqs.maxAmount,
    token: reqs.token,
    validAfter: 0n,
    validBefore: reqs.validUntil,
    nonce: reqs.nonce,
    signature: "0x" as Hex,
  };
  const sig = await sign({ hash: paymentHash(p), privateKey });
  p.signature = serializeSignature(sig);
  return p;
}

/**
 * recoverPayer recovers the address that signed the authorization from the
 * signature over paymentHash(). Throws on a recovered address that does not
 * equal payload.from (the facilitator's authenticity check).
 */
export async function recoverPayer(p: PaymentPayload): Promise<Hex> {
  const got = await recoverAddress({ hash: paymentHash(p), signature: p.signature });
  if (got.toLowerCase() !== p.from.toLowerCase()) {
    throw new Error(`aichain/x402: signer ${got} != payload.from ${p.from}`);
  }
  return got;
}

/** bindsTo asserts the payload faithfully encodes the requirements. */
function bindsTo(p: PaymentPayload, reqs: PaymentRequirements): void {
  if (p.scheme !== reqs.scheme) {
    throw new Error(`aichain/x402: payload scheme ${p.scheme} != required ${reqs.scheme}`);
  }
  if (p.network !== reqs.network) throw new Error("aichain/x402: payload network != required network");
  if (p.to.toLowerCase() !== reqs.payTo.toLowerCase()) {
    throw new Error(`aichain/x402: payload to ${p.to} != payTo ${reqs.payTo}`);
  }
  if (p.token.toLowerCase() !== reqs.token.toLowerCase()) {
    throw new Error("aichain/x402: payload token != required token");
  }
  if (p.nonce.toLowerCase() !== reqs.nonce.toLowerCase()) {
    throw new Error("aichain/x402: payload nonce != required nonce");
  }
  if (p.validBefore !== reqs.validUntil) {
    throw new Error("aichain/x402: payload validBefore != required validUntil");
  }
  if (reqs.scheme === "exact") {
    if (p.value !== reqs.maxAmount) throw new Error("aichain/x402: exact scheme requires value == maxAmount");
  } else {
    if (p.value <= 0n || p.value > reqs.maxAmount) {
      throw new Error("aichain/x402: upto scheme requires 0 < value <= maxAmount");
    }
  }
}

// ---------------------------------------------------------------------------
// Facilitator
// ---------------------------------------------------------------------------

/**
 * Facilitator is the x402 verify+settle service. A real implementation calls the
 * facilitator HTTP endpoint (or settles on-chain itself); MockFacilitator settles
 * deterministically in-process for tests.
 */
export interface Facilitator {
  verify(payload: PaymentPayload, reqs: PaymentRequirements): Promise<void>;
  settle(payload: PaymentPayload, reqs: PaymentRequirements): Promise<SettlementReceipt>;
}

/**
 * MockFacilitator runs the genuine x402 verification (signature recovery,
 * binding, expiry) then "settles" deterministically (no chain). `now` is
 * injectable (ms epoch) for deterministic expiry tests.
 */
export class MockFacilitator implements Facilitator {
  private readonly now: () => number;

  constructor(now?: () => number) {
    this.now = now ?? Date.now;
  }

  async verify(payload: PaymentPayload, reqs: PaymentRequirements): Promise<void> {
    validateRequirements(reqs);
    bindsTo(payload, reqs);
    await recoverPayer(payload);
    const nowSec = BigInt(Math.floor(this.now() / 1000));
    if (payload.validAfter !== 0n && nowSec < payload.validAfter) {
      throw new Error(`aichain/x402: authorization not yet valid (validAfter=${payload.validAfter} now=${nowSec})`);
    }
    if (payload.validBefore !== 0n && nowSec >= payload.validBefore) {
      throw new Error(`aichain/x402: authorization expired (validBefore=${payload.validBefore} now=${nowSec})`);
    }
  }

  async settle(payload: PaymentPayload, reqs: PaymentRequirements): Promise<SettlementReceipt> {
    await this.verify(payload, reqs);
    const payer = await recoverPayer(payload);
    const ph = paymentHash(payload);
    const txHash = keccakConcat(utf8("x402/mock-settle"), hexToBytes(ph));
    return {
      paymentHash: ph,
      txHash,
      settled: true,
      payer,
      payee: payload.to,
      amount: payload.value,
    };
  }
}

// ---------------------------------------------------------------------------
// Proof-of-Thought receiptId (matches ProofOfThoughtRegistry.computeReceiptId)
// ---------------------------------------------------------------------------

/**
 * computeReceiptId reproduces the on-chain
 * ProofOfThoughtRegistry.computeReceiptId for a paid thought:
 *
 *   keccak256( abi.encode(modelId, promptHash, outputHash, paymentHash, payer, operator) )
 *
 * abi.encode of (bytes32,bytes32,bytes32,bytes32,address,address) is six 32-byte
 * words (addresses left-padded). Matches the Go SDK + the Solidity contract.
 */
export function computeReceiptId(args: {
  modelId: Hex;
  promptHash: Hex;
  outputHash: Hex;
  paymentHash: Hex;
  payer: Hex;
  operator: Hex;
}): Hex {
  return keccakConcat(
    bytes32(args.modelId),
    bytes32(args.promptHash),
    bytes32(args.outputHash),
    bytes32(args.paymentHash),
    bytes32(args.payer), // address left-padded to 32 (abi.encode)
    bytes32(args.operator),
  );
}

/**
 * PoTReceipt is the SDK's view of a Proof-of-Thought record: the join of a paid
 * x402 settlement, a model output, and the producing proof, keyed by the on-chain
 * receiptId. What a caller registers on ProofOfThoughtRegistry and what
 * verifyPaidReceipt checks against a SettlementReceipt.
 */
export interface PoTReceipt {
  modelId: Hex;
  promptHash: Hex;
  outputHash: Hex;
  paymentHash: Hex;
  quorumProof: Hex;
  payer: Hex;
  operator: Hex;
  cost: bigint;
}

/** The deterministic on-chain id for a PoT record (== computeReceiptId). */
export function potReceiptId(r: PoTReceipt): Hex {
  return computeReceiptId({
    modelId: r.modelId,
    promptHash: r.promptHash,
    outputHash: r.outputHash,
    paymentHash: r.paymentHash,
    payer: r.payer,
    operator: r.operator,
  });
}

// ---------------------------------------------------------------------------
// paid inference flows
// ---------------------------------------------------------------------------

/** PaidResult joins an inference result to its x402 settlement + PoT record. */
export interface PaidResult {
  intentId: Hex;
  txHash: Hex;
  inference: InferenceReceipt;
  settlement: SettlementReceipt;
  pot: PoTReceipt;
}

/**
 * inferPaid runs the full x402 pay-per-cognition flow for a Tier-2 inference:
 * build+sign payload -> facilitator settles -> run quorum inference -> assemble
 * the PoT join. Payment happens BEFORE inference (pay-first); a settle error
 * aborts before any inference is requested. settlement.paymentHash is the join
 * key; potReceiptId(result.pot) is the on-chain id.
 */
export async function inferPaid(args: {
  client: AIChainClient;
  model: ModelSpec;
  prompt: Uint8Array | string;
  n: number;
  threshold: number;
  reqs: PaymentRequirements;
  facilitator: Facilitator;
  privateKey: Hex;
}): Promise<PaidResult> {
  const { client, model, prompt, n, threshold, reqs, facilitator, privateKey } = args;
  const payload = await buildPaymentPayload(privateKey, reqs);
  const settlement = await facilitator.settle(payload, reqs);
  if (!settlement.settled) throw new Error("aichain/x402: facilitator returned unsettled receipt");

  // Pay-first: only now request the cognition. reqs.maxAmount funds the A escrow.
  const res = await client.infer(model, prompt, n, threshold, reqs.maxAmount);
  const pot: PoTReceipt = {
    modelId: res.receipt.modelSpecHash,
    promptHash: res.receipt.promptHash,
    outputHash: res.receipt.canonicalOutputHash,
    paymentHash: settlement.paymentHash,
    quorumProof: receiptHash(res.receipt), // the settled receipt's commitment is the Tier-2 quorum proof (matches Go)
    payer: settlement.payer,
    operator: ZERO_ADDR,
    cost: settlement.amount,
  };
  return { intentId: res.intentId, txHash: res.txHash, inference: res.receipt, settlement, pot };
}

/**
 * verifyPaidReceipt proves the payment<->thought join: the PoT receipt's
 * paymentHash equals the settlement's paymentHash. true means "this recorded
 * thought was paid for by exactly this settlement". Throws on an unsettled
 * receipt or a zero paymentHash (which the on-chain registry rejects).
 */
export function verifyPaidReceipt(pot: PoTReceipt, settlement: SettlementReceipt): boolean {
  if (!settlement.settled) throw new Error("aichain/x402: settlement is not settled");
  if (isZero(pot.paymentHash) || isZero(settlement.paymentHash)) {
    throw new Error("aichain/x402: zero paymentHash");
  }
  return pot.paymentHash.toLowerCase() === settlement.paymentHash.toLowerCase();
}

// ---------------------------------------------------------------------------
// budget gate
// ---------------------------------------------------------------------------

/**
 * BudgetPolicy is a CLIENT-SIDE spend guard checked BEFORE any payment is made:
 * per-request cap, rolling per-hour cap, optional model allow-list, and an
 * optional require-PoT flag. Advisory (the client's own discipline), independent
 * of the on-chain escrow.
 */
export interface BudgetPolicy {
  /** Caps a single request's amount (undefined/0n == no per-request cap). */
  maxPerRequest?: bigint;
  /** Caps the rolling 1-hour spend (undefined/0n == no hourly cap). */
  maxPerHour?: bigint;
  /** If non-empty, whitelists model spec hashes that may be paid for. */
  allowedModels?: Hex[];
  /** If true, require the paid result to carry a valid PoT join. */
  requirePoT?: boolean;
}

/**
 * BudgetGuard enforces a BudgetPolicy across many generateWithBudget calls,
 * holding the rolling hourly spend (a one-hour window). `now` is injectable (ms
 * epoch) for tests. Separate from the client so one client can serve many budgets.
 */
export class BudgetGuard {
  private readonly windowMs = 3_600_000;
  private readonly events: { at: number; amount: bigint }[] = [];

  constructor(
    private readonly policy: BudgetPolicy,
    private readonly now: () => number = Date.now,
  ) {}

  get requirePoT(): boolean {
    return this.policy.requirePoT === true;
  }

  private allows(model: ModelSpec): void {
    const allow = this.policy.allowedModels;
    if (!allow || allow.length === 0) return;
    const h = modelSpecHash(model).toLowerCase();
    if (!allow.some((m) => m.toLowerCase() === h)) {
      throw new Error(`aichain/x402: model ${h} not in budget allow-list`);
    }
  }

  private spentInWindow(): bigint {
    const cutoff = this.now() - this.windowMs;
    let i = 0;
    let sum = 0n;
    for (const e of this.events) {
      if (e.at > cutoff) {
        this.events[i++] = e;
        sum += e.amount;
      }
    }
    this.events.length = i; // prune aged-out events
    return sum;
  }

  /**
   * check enforces the per-request + per-model caps and reserves the hourly
   * spend. Called BEFORE paying; on success the amount is recorded so an
   * over-budget request never reaches the facilitator. Throws on rejection.
   */
  check(model: ModelSpec, amount: bigint): void {
    if (amount <= 0n) throw new Error("aichain/x402: non-positive request amount");
    this.allows(model);
    const mp = this.policy.maxPerRequest;
    if (mp !== undefined && mp > 0n && amount > mp) {
      throw new Error(`aichain/x402: request amount ${amount} exceeds per-request cap ${mp}`);
    }
    const cap = this.policy.maxPerHour;
    if (cap !== undefined && cap > 0n) {
      const projected = this.spentInWindow() + amount;
      if (projected > cap) {
        throw new Error(`aichain/x402: hourly budget exceeded (would spend ${projected}, cap ${cap})`);
      }
    }
    this.events.push({ at: this.now(), amount });
  }
}

/**
 * generateWithBudget is the budget-gated inferPaid: it enforces `guard`'s policy
 * (per-request cap, hourly cap, model allow-list) BEFORE building or settling any
 * payment, then runs the full paid inference. An over-budget or disallowed
 * request is rejected with NO payment made. If the policy requires PoT, the
 * result is checked to carry a valid join before returning.
 */
export async function generateWithBudget(args: {
  client: AIChainClient;
  model: ModelSpec;
  prompt: Uint8Array | string;
  n: number;
  threshold: number;
  reqs: PaymentRequirements;
  guard: BudgetGuard;
  facilitator: Facilitator;
  privateKey: Hex;
}): Promise<PaidResult> {
  const { guard, model, reqs } = args;
  validateRequirements(reqs);
  // Gate on the amount the requirements will actually charge BEFORE paying.
  guard.check(model, reqs.maxAmount);
  const res = await inferPaid(args);
  if (guard.requirePoT) {
    if (!verifyPaidReceipt(res.pot, res.settlement)) {
      throw new Error("aichain/x402: PoT required but paymentHash join did not verify");
    }
  }
  return res;
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

function isZero(h: Hex): boolean {
  return /^0x0*$/.test(h);
}
