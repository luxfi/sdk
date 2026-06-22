// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

// @luxfi/aichain — Lux A-Chain (Beluga) inference SDK.
//
// On-chain LLM inference + embeddings via the Thinking Chains architecture
// (LP-5300 / LP-5301). Wire-compatible byte-for-byte with the Go SDK
// (github.com/luxfi/sdk/aichain) and the on-chain encoders.

export {
  // domains + constants
  DOMAIN_INTENT,
  DOMAIN_RECEIPT,
  DOMAIN_MODELSPEC,
  RECEIPT_VERSION,
  RECEIPT_ENCODED_LEN,
  MAX_FANOUT,
  MAX_PROOF_DEPTH,
  ReceiptStatus,
  // low-level byte helpers (exported for advanced/interop use)
  utf8,
  uintBE,
  u16be,
  u32be,
  u64be,
  u256be,
  hexToBytes,
  bytesToHex,
  bytes32,
  address20,
  concatBytes,
  keccakConcat,
} from "./bytesutil.js";

export { computeIntentID, promptHash } from "./wire.js";

export { modelSpecHash, registryName, type ModelSpec } from "./modelspec.js";

export {
  encodeReceipt,
  decodeReceipt,
  receiptHash,
  isCompleted,
  type InferenceReceipt,
} from "./receipt.js";

export {
  leafHash,
  merkleNode,
  verifyReceiptInclusion,
  verifyReceipt,
  encodeProof,
  decodeProof,
  type MerkleProof,
} from "./proof.js";

export {
  // selectors + addresses
  SELECTOR_SUBMIT_INTENT,
  SELECTOR_VERIFY_RECEIPT,
  SELECTOR_GENERATE,
  SELECTOR_ADOPT,
  SELECTOR_GET_APPROVED,
  AI_BRIDGE_ADDRESS,
  INFERENCE_ADDRESS,
  MODEL_REGISTRY_ADDRESS,
  // encoders / decoders
  encodeSubmitInferenceIntent,
  decodeSubmitInferenceIntentResult,
  encodeVerifyInferenceReceipt,
  decodeVerifyInferenceReceiptResult,
  encodeGenerate,
  decodeGenerateResult,
  encodeRegisterModel,
  encodeGetApproved,
  decodeGetApprovedResult,
  type VerifyResult,
  type ApprovedModel,
} from "./calldata.js";

export {
  AIChainClient,
  type AIChainClientConfig,
  type ReceiptStore,
  type SubmitOptions,
} from "./client.js";

export {
  // x402 payment adapter (pay-per-cognition + Proof-of-Thought join)
  x402DomainSeparator,
  paymentHash,
  buildPaymentPayload,
  recoverPayer,
  computeReceiptId,
  potReceiptId,
  inferPaid,
  verifyPaidReceipt,
  generateWithBudget,
  MockFacilitator,
  BudgetGuard,
  type X402Scheme,
  type PaymentRequirements,
  type PaymentPayload,
  type SettlementReceipt,
  type Facilitator,
  type PoTReceipt,
  type PaidResult,
  type BudgetPolicy,
} from "./x402.js";
