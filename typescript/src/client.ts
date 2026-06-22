// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

// client.ts is the high-level AIChainClient over viem. It signs/sends txs to the
// AI precompiles and reads results. Like the Go SDK it distinguishes:
//   - Tier-1 generateDeterministic — synchronous, in one eth_call (no quorum);
//   - Tier-2 submitInferenceIntent / waitReceipt / infer — async quorum settlement.

import {
  type Account,
  type Address,
  type Hex,
  type PublicClient,
  type WalletClient,
} from "viem";
import {
  AI_BRIDGE_ADDRESS,
  INFERENCE_ADDRESS,
  MODEL_REGISTRY_ADDRESS,
  decodeGenerateResult,
  decodeGetApprovedResult,
  encodeGenerate,
  encodeGetApproved,
  encodeRegisterModel,
  encodeSubmitInferenceIntent,
  type ApprovedModel,
} from "./calldata.js";
import { MAX_FANOUT, ReceiptStatus } from "./bytesutil.js";
import { modelSpecHash, type ModelSpec } from "./modelspec.js";
import { computeIntentID, promptHash } from "./wire.js";
import type { InferenceReceipt } from "./receipt.js";
import type { MerkleProof } from "./proof.js";

/**
 * ReceiptStore reads committed A-Chain receipts. Separate from the EVM tx path
 * (the settlement read path is not the EVM path); callers wire their own (poll
 * the A-Chain RPC, or watch the bridge checkpoint + A->C export). `aChainID`, if
 * present, lets the client reproduce the exact cross-chain intent id.
 */
export interface ReceiptStore {
  receiptByIntent(
    intentId: Hex,
  ): Promise<{ receipt: InferenceReceipt; proof: MerkleProof; found: boolean }>;
  aChainID?(): Hex;
}

export interface SubmitOptions {
  modelSpecHash: Hex;
  promptHash: Hex;
  n: number;
  threshold: number;
  fee: bigint;
  routing?: Hex;
}

export interface AIChainClientConfig {
  publicClient: PublicClient;
  /** Required only for txs (submit / register). */
  walletClient?: WalletClient;
  /** The signing account (required with walletClient for txs). */
  account?: Account;
  /** Enables waitReceipt / infer / getReceipt. */
  receiptStore?: ReceiptStore;
  /** Poll cadence + deadline for Tier-2 settlement (ms). Defaults 2s / 5min. */
  pollIntervalMs?: number;
  pollTimeoutMs?: number;
}

const ZERO_HASH = "0x0000000000000000000000000000000000000000000000000000000000000000" as Hex;

export class AIChainClient {
  private readonly pub: PublicClient;
  private readonly wallet?: WalletClient;
  private readonly account?: Account;
  private readonly store?: ReceiptStore;
  private readonly pollIntervalMs: number;
  private readonly pollTimeoutMs: number;
  private chainId?: bigint;

  constructor(cfg: AIChainClientConfig) {
    this.pub = cfg.publicClient;
    this.wallet = cfg.walletClient;
    this.account = cfg.account;
    this.store = cfg.receiptStore;
    this.pollIntervalMs = cfg.pollIntervalMs ?? 2000;
    this.pollTimeoutMs = cfg.pollTimeoutMs ?? 5 * 60_000;
  }

  /** The signer address (undefined for a read-only client). */
  from(): Address | undefined {
    return this.account?.address;
  }

  private async chainIdOf(): Promise<bigint> {
    if (this.chainId === undefined) this.chainId = BigInt(await this.pub.getChainId());
    return this.chainId;
  }

  private requireWallet(): { wallet: WalletClient; account: Account } {
    if (!this.wallet || !this.account) {
      throw new Error("aichain: this operation needs a walletClient + account");
    }
    return { wallet: this.wallet, account: this.account };
  }

  // -------------------------------------------------------------------------
  // Tier-1 — deterministic, in one eth_call
  // -------------------------------------------------------------------------

  /**
   * generateDeterministic runs the Tier-1 in-consensus inference precompile via
   * eth_call and returns the full token sequence (prompt + generated).
   * Synchronous; no quorum, no gas.
   */
  async generateDeterministic(nNew: number, promptTokens: number[]): Promise<number[]> {
    if (promptTokens.length === 0) throw new Error("aichain: empty prompt");
    const { data } = await this.pub.call({
      to: INFERENCE_ADDRESS,
      data: encodeGenerate(nNew, promptTokens),
    });
    if (!data) throw new Error("aichain: empty generate return");
    return decodeGenerateResult(data);
  }

  // -------------------------------------------------------------------------
  // Tier-2 — async quorum settlement
  // -------------------------------------------------------------------------

  /**
   * submitInferenceIntent sends a Tier-2 intent tx to the aivmbridge precompile
   * (Pattern A) and returns the deterministic intent id (re-derived from the
   * landed tx hash + call index 0) and the tx hash. The START of async quorum
   * settlement; use waitReceipt / infer for the result.
   */
  async submitInferenceIntent(opts: SubmitOptions): Promise<{ intentId: Hex; txHash: Hex }> {
    validateSubmit(opts);
    const { wallet, account } = this.requireWallet();
    const data = encodeSubmitInferenceIntent({
      modelSpecHash: opts.modelSpecHash,
      promptHash: opts.promptHash,
      n: opts.n,
      threshold: opts.threshold,
      fee: opts.fee,
      routing: opts.routing ?? ZERO_HASH,
    });
    const txHash = await wallet.sendTransaction({
      account,
      chain: wallet.chain ?? null,
      to: AI_BRIDGE_ADDRESS,
      data,
      value: 0n,
    });
    const rcpt = await this.pub.waitForTransactionReceipt({ hash: txHash });
    if (rcpt.status !== "success") throw new Error(`aichain: tx ${txHash} reverted`);

    const cChainID = toBytes32(await this.chainIdOf());
    const aChainID = this.store?.aChainID?.() ?? cChainID;
    const intentId = computeIntentID({
      cChainID,
      aChainID,
      cTxHash: txHash,
      callIndex: 0,
      caller: account.address,
      modelSpecHash: opts.modelSpecHash,
      promptHash: opts.promptHash,
      n: opts.n,
      threshold: opts.threshold,
      fee: opts.fee,
    });
    return { intentId, txHash };
  }

  /**
   * waitReceipt polls the receipt store until the intent settles Completed (or
   * the deadline). Errors on a Failed / Challenged terminal receipt, on a
   * Completed-but-zero-output receipt, and on timeout.
   */
  async waitReceipt(intentId: Hex): Promise<InferenceReceipt> {
    if (!this.store) throw new Error("aichain: waitReceipt needs a receiptStore");
    const deadline = Date.now() + this.pollTimeoutMs;
    for (;;) {
      const { receipt, found } = await this.store.receiptByIntent(intentId);
      if (found) {
        switch (receipt.status) {
          case ReceiptStatus.Completed:
            if (isZero(receipt.canonicalOutputHash)) {
              throw new Error(`aichain: receipt completed but output is zero (${intentId})`);
            }
            return receipt;
          case ReceiptStatus.Failed:
            throw new Error(`aichain: inference failed (${intentId})`);
          case ReceiptStatus.Challenged:
            throw new Error(`aichain: inference challenged (${intentId})`);
        }
      }
      if (Date.now() > deadline) throw new Error(`aichain: timed out waiting for receipt (${intentId})`);
      await sleep(this.pollIntervalMs);
    }
  }

  /** getReceipt does a single non-blocking read of an intent's committed receipt. */
  async getReceipt(intentId: Hex): Promise<{ receipt: InferenceReceipt; proof: MerkleProof; found: boolean }> {
    if (!this.store) throw new Error("aichain: getReceipt needs a receiptStore");
    return this.store.receiptByIntent(intentId);
  }

  /**
   * infer is the one-call Tier-2 convenience: submit -> wait for quorum ->
   * return the canonical result. ASYNC: blocks polling the A-Chain until an
   * M-of-N quorum settles. receipt.canonicalOutputHash is the agreed output
   * digest (fetch bytes from your provider/DA layer by that hash).
   */
  async infer(
    model: ModelSpec,
    prompt: Uint8Array | string,
    n: number,
    threshold: number,
    fee: bigint,
  ): Promise<{ intentId: Hex; txHash: Hex; receipt: InferenceReceipt }> {
    const msh = modelSpecHash(model);
    const ph = promptHash(prompt);
    const { intentId, txHash } = await this.submitInferenceIntent({
      modelSpecHash: msh,
      promptHash: ph,
      n,
      threshold,
      fee,
    });
    const receipt = await this.waitReceipt(intentId);
    // Defense in depth: the store is untrusted transport.
    if (
      receipt.modelSpecHash.toLowerCase() !== msh.toLowerCase() ||
      receipt.promptHash.toLowerCase() !== ph.toLowerCase()
    ) {
      throw new Error("aichain: receipt binds different model/prompt than requested");
    }
    return { intentId, txHash, receipt };
  }

  // -------------------------------------------------------------------------
  // model registry
  // -------------------------------------------------------------------------

  /** registerModel adopts a model version (caller must be a registry admin). */
  async registerModel(spec: ModelSpec): Promise<Hex> {
    if (isZero(spec.weightCommit)) throw new Error("aichain: zero weight commitment");
    const { wallet, account } = this.requireWallet();
    return wallet.sendTransaction({
      account,
      chain: wallet.chain ?? null,
      to: MODEL_REGISTRY_ADDRESS,
      data: encodeRegisterModel(spec),
      value: 0n,
    });
  }

  /** getModel reads the adopted (version, weightCommit) for a model name. */
  async getModel(name: Hex): Promise<ApprovedModel> {
    const { data } = await this.pub.call({ to: MODEL_REGISTRY_ADDRESS, data: encodeGetApproved(name) });
    if (!data) throw new Error("aichain: empty getApproved return");
    return decodeGetApprovedResult(data);
  }
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

function validateSubmit(o: SubmitOptions): void {
  if (isZero(o.modelSpecHash)) throw new Error("aichain: zero modelSpecHash");
  if (isZero(o.promptHash)) throw new Error("aichain: zero promptHash");
  if (o.n < 1 || o.n > MAX_FANOUT) throw new Error(`aichain: N=${o.n} out of [1,${MAX_FANOUT}]`);
  if (o.threshold < 1 || o.threshold > o.n) throw new Error(`aichain: threshold=${o.threshold} out of [1,${o.n}]`);
  const floor = Math.floor(o.n / 2) + 1;
  if (o.threshold < floor) throw new Error(`aichain: threshold=${o.threshold} below quorum floor ${floor}`);
}

function toBytes32(v: bigint): Hex {
  return ("0x" + v.toString(16).padStart(64, "0")) as Hex;
}

function isZero(h: Hex): boolean {
  return /^0x0*$/.test(h);
}

function sleep(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}
