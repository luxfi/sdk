# @luxfi/aichain

Lux A-Chain (Beluga) inference SDK for TypeScript — **on-chain LLM inference +
embeddings** via the Thinking Chains architecture (LP-5300 / LP-5301).

Wire-compatible **byte-for-byte** with the Go SDK (`github.com/luxfi/sdk/aichain`)
and the on-chain encoders (`chains/aivm`). The same golden vectors are asserted in
both languages' test suites.

| Precompile | Address | Tier |
|---|---|---|
| AI Bridge (LP-5301) | `0x0300…0004` | **Tier-2** large-model, quorum-settled (async) |
| Deterministic inference (LP-0303) | `0x0300…0003` | **Tier-1** small model, in one `eth_call` (sync) |
| Model Registry | `0x0300…0002` | governance model adoption |

## Install

```bash
npm install @luxfi/aichain viem
```

`viem` is a peer-grade dependency (its `keccak256` is the only hash used, so
encoders are dependency-light).

## Tier 1 vs Tier 2

- **Tier 1 — `generateDeterministic`**: the small int8 transformer every
  validator computes identically, answered **synchronously inside one
  `eth_call`** (no quorum, no gas). Returns prompt + generated token ids.
- **Tier 2 — `submitInferenceIntent` / `waitReceipt` / `infer`**: a **large**
  model settled by an **M-of-N quorum**, **asynchronous**. `submit` returns an
  `intentId`; the result arrives later as a committed receipt whose
  `canonicalOutputHash` is the agreed output digest (fetch bytes by it).

## Usage

```ts
import { createPublicClient, createWalletClient, http } from "viem";
import { privateKeyToAccount } from "viem/accounts";
import { AIChainClient, type ModelSpec } from "@luxfi/aichain";

const transport = http("https://api.lux.network/ext/bc/C/rpc");
const publicClient = createPublicClient({ transport });
const account = privateKeyToAccount("0x…");
const walletClient = createWalletClient({ account, transport });

const c = new AIChainClient({ publicClient, walletClient, account, receiptStore });

// Tier 1 — deterministic, synchronous (read-only client is fine):
const tokens = await c.generateDeterministic(10, [1, 7, 13, 2]);

// Tier 2 — large model, async quorum settlement:
const model: ModelSpec = {
  name: "zenlm/zen-omni",
  version: 3n,
  weightCommit: "0x…", // on-chain weight commitment
  quantization: "int8",
};
const { receipt } = await c.infer(model, "Explain PQ signatures.", 5, 3, 10n ** 15n);
// receipt.canonicalOutputHash is the agreed output digest.
```

Or step-by-step:

```ts
const { intentId, txHash } = await c.submitInferenceIntent({
  modelSpecHash: modelSpecHash(model),
  promptHash: promptHash("…"),
  n: 5,
  threshold: 3,
  fee: 10n ** 15n,
});
const receipt = await c.waitReceipt(intentId); // blocks until Completed or deadline
```

### The `ReceiptStore` seam

`waitReceipt` / `infer` / `getReceipt` read committed A-Chain receipts through a
`ReceiptStore` you supply — it is separate from the EVM tx path, so the SDK stays
decoupled from any single A-Chain transport:

```ts
interface ReceiptStore {
  receiptByIntent(intentId: Hex): Promise<{ receipt: InferenceReceipt; proof: MerkleProof; found: boolean }>;
  aChainID?(): Hex; // optional: lets the client reproduce the exact cross-chain intent id
}
```

## Pure encoders (no chain)

Every wire function is exported for offline use (the calldata builders, the
355-byte receipt codec, the Merkle proof frame, `computeIntentID`,
`modelSpecHash`, `promptHash`). These produce the EXACT bytes the on-chain
precompiles decode.

## Golden vectors (shared Go ↔ TS)

For the canonical fixture (`test/wire.test.ts`):

```
intent_id     = 0x5e967be3e83750c25fb91887a125d67d2440fb41825d24d63a0c00e6fb2bfbde
receipt_hash  = 0xfe0a1e45baf5255e2461c5f8f38b8446a691ec8bf0ca260750259d8bb5677851
modelSpecHash = 0xd8ab4fca51f36de6db2efd1a7a022fef6943de8d9986ae8ca3f4db70f318b4a7
```

## Build & test

```bash
npm install
npm run typecheck
npm run build   # -> dist/
npm test        # vitest: golden vectors + client smoke
```
