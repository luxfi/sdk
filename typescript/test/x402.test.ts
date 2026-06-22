// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

import { describe, expect, it } from "vitest";
import type { Hex } from "viem";
import { privateKeyToAccount } from "viem/accounts";
import {
  AIChainClient,
  BudgetGuard,
  MockFacilitator,
  buildPaymentPayload,
  computeReceiptId,
  generateWithBudget,
  inferPaid,
  modelSpecHash,
  paymentHash,
  potReceiptId,
  recoverPayer,
  verifyPaidReceipt,
  x402DomainSeparator,
  ReceiptStatus,
  promptHash,
  receiptHash,
  type InferenceReceipt,
  type ModelSpec,
  type PaidResult,
  type PaymentRequirements,
} from "../src/index.js";

// GOLDEN x402 vectors — these MUST equal the Go SDK's (aichain/x402_test.go) and
// the on-chain ProofOfThoughtRegistry.computeReceiptId. If a value drifts the
// cross-language x402<->PoT wire broke.
const SIGNER_KEY = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" as Hex;
const SIGNER_ADDR = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266" as Hex;
const GOLDEN_PAYMENT_HASH = "0x99a3abd03c66497ebc597c616b7f3cd42b1a2b129af884d7607206a1408569d2";
const GOLDEN_RECEIPT_ID = "0x803dbf024c2f91bb115752b416145b571d204480c8c9e1919539a8b83a184762";

// The canonical fixture — byte-identical to the Go test's x402Fixture().
function fixture(): PaymentRequirements {
  return {
    scheme: "exact",
    network: 36911n,
    maxAmount: 1_000_000n,
    token: "0x00000000000000000000000000000000000000a0" as Hex,
    payTo: "0x00000000000000000000000000000000000000bb" as Hex,
    resource: "zenlm/zen-omni",
    description: "one cognition",
    nonce: "0x0000000000000000000000000000000000000000000000000000000000000077" as Hex,
    validUntil: 2_000_000_000n,
  };
}

const MODEL: ModelSpec = {
  name: "zenlm/zen-omni",
  version: 3n,
  weightCommit: "0x0000000000000000000000000000000000000000000000000000000000000abc" as Hex,
  quantization: "int8",
};

// A clock fixed before validUntil so authorizations are live (ms epoch).
const liveClock = () => 1_000_000_000_000;

// fakeClient exposes only infer(), which inferPaid/generateWithBudget call. It
// returns a completed receipt bound to the requested model/prompt (the Go tests'
// anyIDStore equivalent).
function fakeClient(opts?: { outputHash?: Hex; trackTx?: { count: number } }): AIChainClient {
  const outputHash = opts?.outputHash ?? ("0x000000000000000000000000000000000000000000000000000000000d0e0a0d" as Hex);
  return {
    async infer(model: ModelSpec, prompt: Uint8Array | string, n: number, threshold: number, _fee: bigint) {
      if (opts?.trackTx) opts.trackTx.count++;
      const receipt: InferenceReceipt = {
        version: 1,
        intentId: "0x00000000000000000000000000000000000000000000000000000000000000ab" as Hex,
        taskId: ("0x" + "00".repeat(32)) as Hex,
        cChainId: ("0x" + "00".repeat(32)) as Hex,
        aChainId: ("0x" + "00".repeat(32)) as Hex,
        requester: ("0x" + "00".repeat(20)) as Hex,
        modelSpecHash: modelSpecHash(model),
        promptHash: promptHash(prompt),
        canonicalOutputHash: outputHash,
        status: ReceiptStatus.Completed,
        n,
        threshold,
        winnersRoot: ("0x" + "00".repeat(32)) as Hex,
        operatorsRoot: ("0x" + "00".repeat(32)) as Hex,
        feePaid: 0n,
        settledAtHeight: 0n,
      };
      return {
        intentId: receipt.intentId,
        txHash: "0x00000000000000000000000000000000000000000000000000000000deadbeef" as Hex,
        receipt,
      };
    },
  } as unknown as AIChainClient;
}

describe("x402 paymentHash (Go parity)", () => {
  it("derives the GOLDEN domain separator + paymentHash and recovers the signer", async () => {
    const p = await buildPaymentPayload(SIGNER_KEY, fixture());

    expect(p.from).toBe(SIGNER_ADDR);
    expect(p.to.toLowerCase()).toBe(fixture().payTo.toLowerCase());
    expect(p.value).toBe(fixture().maxAmount);
    expect(p.validBefore).toBe(fixture().validUntil);
    expect(p.validAfter).toBe(0n);

    // GOLDEN paymentHash — equals the Go-computed value byte-for-byte.
    expect(paymentHash(p).toLowerCase()).toBe(GOLDEN_PAYMENT_HASH);

    // The signature recovers to the payer.
    expect((await recoverPayer(p)).toLowerCase()).toBe(SIGNER_ADDR.toLowerCase());
  });

  it("is deterministic (same key + requirements -> same hash + signature)", async () => {
    const a = await buildPaymentPayload(SIGNER_KEY, fixture());
    const b = await buildPaymentPayload(SIGNER_KEY, fixture());
    expect(paymentHash(a)).toBe(paymentHash(b));
    expect(a.signature).toBe(b.signature);
    expect(x402DomainSeparator(36911n).toLowerCase()).toBe(
      "0x7e24d39986b4e9f798919bfee1d74c4159835a944a6260ec7b43c17a61732ba5",
    );
  });

  it("binds token + network + nonce (changing any changes the hash)", async () => {
    const base = await buildPaymentPayload(SIGNER_KEY, fixture());

    const net = await buildPaymentPayload(SIGNER_KEY, { ...fixture(), network: 1n });
    expect(paymentHash(net)).not.toBe(paymentHash(base));

    const tok = await buildPaymentPayload(SIGNER_KEY, {
      ...fixture(),
      token: "0x00000000000000000000000000000000000000ff" as Hex,
    });
    expect(paymentHash(tok)).not.toBe(paymentHash(base));

    const non = await buildPaymentPayload(SIGNER_KEY, {
      ...fixture(),
      nonce: "0x0000000000000000000000000000000000000000000000000000000000000078" as Hex,
    });
    expect(paymentHash(non)).not.toBe(paymentHash(base));
  });

  it("rejects invalid requirements", async () => {
    await expect(buildPaymentPayload(SIGNER_KEY, { ...fixture(), scheme: "bogus" as never })).rejects.toThrow();
    await expect(buildPaymentPayload(SIGNER_KEY, { ...fixture(), network: 0n })).rejects.toThrow();
    await expect(buildPaymentPayload(SIGNER_KEY, { ...fixture(), maxAmount: 0n })).rejects.toThrow();
    await expect(
      buildPaymentPayload(SIGNER_KEY, { ...fixture(), payTo: ("0x" + "00".repeat(20)) as Hex }),
    ).rejects.toThrow();
    await expect(
      buildPaymentPayload(SIGNER_KEY, { ...fixture(), nonce: ("0x" + "00".repeat(32)) as Hex }),
    ).rejects.toThrow();
    await expect(buildPaymentPayload(SIGNER_KEY, { ...fixture(), validUntil: 0n })).rejects.toThrow();
  });
});

describe("computeReceiptId (Go + on-chain parity)", () => {
  it("matches the GOLDEN receiptId (abi.encode layout)", () => {
    const id = computeReceiptId({
      modelId: "0x4444444444444444444444444444444444444444444444444444444444444444" as Hex,
      promptHash: "0x5555555555555555555555555555555555555555555555555555555555555555" as Hex,
      outputHash: "0x6666666666666666666666666666666666666666666666666666666666666666" as Hex,
      paymentHash: "0x7777777777777777777777777777777777777777777777777777777777777777" as Hex,
      payer: "0x00000000000000000000000000000000000000aa" as Hex,
      operator: "0x00000000000000000000000000000000000000bb" as Hex,
    });
    expect(id.toLowerCase()).toBe(GOLDEN_RECEIPT_ID);
  });
});

describe("MockFacilitator", () => {
  it("verifies + settles, carrying the paymentHash", async () => {
    const reqs = fixture();
    const p = await buildPaymentPayload(SIGNER_KEY, reqs);
    const fac = new MockFacilitator(liveClock);

    await expect(fac.verify(p, reqs)).resolves.toBeUndefined();
    const st = await fac.settle(p, reqs);
    expect(st.settled).toBe(true);
    expect(st.paymentHash.toLowerCase()).toBe(GOLDEN_PAYMENT_HASH);
    expect(st.payer.toLowerCase()).toBe(SIGNER_ADDR.toLowerCase());
    expect(st.payee.toLowerCase()).toBe(reqs.payTo.toLowerCase());
    expect(st.amount).toBe(reqs.maxAmount);
  });

  it("rejects tampered/expired/mismatched payloads", async () => {
    const reqs = fixture();
    const good = await buildPaymentPayload(SIGNER_KEY, reqs);
    const fac = new MockFacilitator(liveClock);

    await expect(fac.verify({ ...good, value: 999n }, reqs)).rejects.toThrow();
    await expect(
      fac.verify({ ...good, to: "0x00000000000000000000000000000000000000cc" as Hex }, reqs),
    ).rejects.toThrow();

    // Corrupted signature byte -> recovery mismatch.
    const corrupt = ("0x" + good.signature.slice(2, 22) + "ff" + good.signature.slice(24)) as Hex;
    await expect(fac.verify({ ...good, signature: corrupt }, reqs)).rejects.toThrow();

    // Expired (clock at/after validBefore).
    const expired = new MockFacilitator(() => Number(reqs.validUntil) * 1000);
    await expect(expired.verify(good, reqs)).rejects.toThrow();

    // Nonce mismatch between payload + requirements.
    await expect(
      fac.verify(good, { ...reqs, nonce: ("0x" + "99".padStart(64, "0")) as Hex }),
    ).rejects.toThrow();
  });
});

describe("inferPaid + verifyPaidReceipt", () => {
  it("pays then infers, assembling a verifiable PoT join", async () => {
    const reqs = fixture();
    const client = fakeClient();
    const res = await inferPaid({
      client,
      model: MODEL,
      prompt: "think about beluga",
      n: 5,
      threshold: 3,
      reqs,
      facilitator: new MockFacilitator(liveClock),
      privateKey: SIGNER_KEY,
    });

    expect(res.settlement.settled).toBe(true);
    expect(res.settlement.paymentHash.toLowerCase()).toBe(GOLDEN_PAYMENT_HASH);
    expect(res.inference.status).toBe(ReceiptStatus.Completed);

    // PoT join.
    expect(res.pot.paymentHash).toBe(res.settlement.paymentHash);
    expect(res.pot.outputHash).toBe(res.inference.canonicalOutputHash);
    expect(res.pot.modelId).toBe(modelSpecHash(MODEL));
    expect(res.pot.promptHash).toBe(promptHash("think about beluga"));
    expect(res.pot.quorumProof).toBe(receiptHash(res.inference));
    expect(potReceiptId(res.pot)).not.toBe("0x" + "00".repeat(32));

    expect(verifyPaidReceipt(res.pot, res.settlement)).toBe(true);
  });

  it("verifyPaidReceipt matches + rejects", () => {
    const ph = GOLDEN_PAYMENT_HASH as Hex;
    const pot = { paymentHash: ph } as PaidResult["pot"];
    expect(verifyPaidReceipt(pot, { settled: true, paymentHash: ph } as PaidResult["settlement"])).toBe(true);
    expect(
      verifyPaidReceipt(pot, { settled: true, paymentHash: "0xdeadbeef" as Hex } as PaidResult["settlement"]),
    ).toBe(false);
    expect(() =>
      verifyPaidReceipt(pot, { settled: false, paymentHash: ph } as PaidResult["settlement"]),
    ).toThrow();
    expect(() =>
      verifyPaidReceipt({ paymentHash: ("0x" + "00".repeat(32)) as Hex } as PaidResult["pot"], {
        settled: true,
        paymentHash: ("0x" + "00".repeat(32)) as Hex,
      } as PaidResult["settlement"]),
    ).toThrow();
  });

  it("does not infer when settlement fails (pay-first)", async () => {
    const reqs = fixture();
    const tracker = { count: 0 };
    const client = fakeClient({ trackTx: tracker });
    const expired = new MockFacilitator(() => Number(reqs.validUntil) * 1000); // expired clock
    await expect(
      inferPaid({
        client,
        model: MODEL,
        prompt: "x",
        n: 3,
        threshold: 2,
        reqs,
        facilitator: expired,
        privateKey: SIGNER_KEY,
      }),
    ).rejects.toThrow();
    expect(tracker.count).toBe(0);
  });
});

describe("generateWithBudget", () => {
  it("allows within caps and rejects over the hourly cap (no infer)", async () => {
    const reqs = fixture(); // 1_000_000 per request
    const tracker = { count: 0 };
    const client = fakeClient({ trackTx: tracker });
    const fac = new MockFacilitator(liveClock);
    const guard = new BudgetGuard(
      {
        maxPerRequest: 2_000_000n,
        maxPerHour: 3_000_000n,
        allowedModels: [modelSpecHash(MODEL)],
        requirePoT: true,
      },
      liveClock,
    );

    const call = () =>
      generateWithBudget({
        client,
        model: MODEL,
        prompt: "hello",
        n: 5,
        threshold: 3,
        reqs,
        guard,
        facilitator: fac,
        privateKey: SIGNER_KEY,
      });

    await expect(call()).resolves.toBeDefined(); // 1M
    await expect(call()).resolves.toBeDefined(); // 2M
    await expect(call()).resolves.toBeDefined(); // 3M (== cap, inclusive)
    expect(tracker.count).toBe(3);

    await expect(call()).rejects.toThrow(); // 4M > 3M
    expect(tracker.count).toBe(3); // no extra infer
  });

  it("rejects over per-request cap before paying", async () => {
    const reqs = fixture(); // 1_000_000
    const tracker = { count: 0 };
    const client = fakeClient({ trackTx: tracker });
    const guard = new BudgetGuard({ maxPerRequest: 500_000n }, liveClock);
    await expect(
      generateWithBudget({
        client,
        model: MODEL,
        prompt: "x",
        n: 3,
        threshold: 2,
        reqs,
        guard,
        facilitator: new MockFacilitator(liveClock),
        privateKey: SIGNER_KEY,
      }),
    ).rejects.toThrow();
    expect(tracker.count).toBe(0);
  });

  it("rejects a model not in the allow-list before paying", async () => {
    const other: ModelSpec = { name: "other", version: 1n, weightCommit: "0x07" as Hex, quantization: "" };
    const tracker = { count: 0 };
    const client = fakeClient({ trackTx: tracker });
    const guard = new BudgetGuard({ allowedModels: [modelSpecHash(MODEL)] }, liveClock);
    await expect(
      generateWithBudget({
        client,
        model: other,
        prompt: "x",
        n: 3,
        threshold: 2,
        reqs: fixture(),
        guard,
        facilitator: new MockFacilitator(liveClock),
        privateKey: SIGNER_KEY,
      }),
    ).rejects.toThrow();
    expect(tracker.count).toBe(0);
  });

  it("rolling window frees capacity after an hour", async () => {
    let now = 1_000_000_000_000;
    const guard = new BudgetGuard({ maxPerHour: 1000n }, () => now);
    guard.check(MODEL, 600n);
    guard.check(MODEL, 400n); // 1000 == cap
    expect(() => guard.check(MODEL, 1n)).toThrow();
    now += 3_600_000 + 1000; // advance past the window
    expect(() => guard.check(MODEL, 1000n)).not.toThrow();
  });
});
