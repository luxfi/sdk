# aichain — Lux A-Chain (Beluga) inference SDK

`github.com/luxfi/sdk/aichain`

The developer-facing client for **on-chain LLM inference + embeddings** on the
Lux A-Chain ("Thinking Chains" / Beluga). It wraps the three AI precompiles in
the `0x0300…00xx` range and pins the cross-chain wire spec byte-for-byte to the
A-Chain settlement engine (`github.com/luxfi/chains/aivm`).

| Precompile | Address | What it is |
|---|---|---|
| AI Bridge (LP-5301) | `0x0300…0004` | **Tier-2** large-model inference: submit a committed intent, settle by A-Chain quorum |
| Deterministic inference (LP-0303) | `0x0300…0003` | **Tier-1** small model, answered in-consensus inside one EVM call |
| Model Registry | `0x0300…0002` | Governance adoption of the canonical model |

## Tier 1 vs Tier 2 — the core distinction

- **Tier 1 — `GenerateDeterministic`** runs the small int8 transformer that
  *every validator computes identically*. It is answered **synchronously inside a
  single `eth_call`** — no quorum, no waiting, no gas (it's a call, not a tx).
  Deterministic token-for-token across CPU/Metal/CUDA/HIP. Use it for fast,
  cheap, small-model inference where the result is part of consensus.

- **Tier 2 — `SubmitInferenceIntent` / `WaitReceipt` / `Infer`** requests a
  **large** model that runs on the A-Chain and is settled by an **M-of-N quorum**.
  This is **asynchronous**: `SubmitInferenceIntent` writes a committed C-Chain
  intent and returns an `intent_id`; the result arrives **later** as a committed
  A-Chain receipt (`InferenceReceipt`) the bridge verifies via a Merkle proof
  against a committed `receipt_root`. The chain commits the **canonical output
  hash**, not the bytes — fetch the bytes from your provider/DA layer by that
  hash.

## Install

It's a package in the Lux SDK module:

```go
import "github.com/luxfi/sdk/aichain"
```

> The live transport constructor `aichain.Dial` (which imports
> `luxfi/evm/ethclient`) is behind the `livedial` build tag, so the core SDK
> builds with **no** heavy node dependency and `GOWORK=off go test ./aichain/...`
> stays green. Inside the lux workspace, or with `-tags livedial`, `Dial` is
> available. Everywhere else, build a `Client` from any `EVMBackend` (your node
> client already satisfies it).

## Quick start

### Tier 1 — deterministic, in one call

```go
c, _ := aichain.NewClient(backend, "" /* read-only, eth_call needs no key */)

// promptTokens are model token ids; returns prompt+generated tokens.
tokens, err := c.GenerateDeterministic(ctx, /*nNew*/ 10, []uint32{1, 7, 13, 2})
```

### Tier 2 — large model, async quorum settlement

```go
c, _ := aichain.Dial(rpcURL, privKeyHex, aichain.WithReceiptStore(store))

model := aichain.ModelSpec{
    Name:         "zenlm/zen-omni",
    Version:      3,
    WeightCommit: weightCommitHash, // the on-chain weight commitment
    Quantization: "int8",
}

// One-call convenience: submit -> wait for quorum -> return the canonical result.
res, err := c.Infer(ctx, model, []byte("Explain post-quantum signatures."),
    /*N*/ 5, /*threshold*/ 3, /*fee*/ big.NewInt(1e15))
// res.Receipt.CanonicalOutputHash is the agreed output digest (fetch bytes by it).
```

Or drive the two steps yourself:

```go
intentID, txHash, err := c.SubmitInferenceIntent(ctx, aichain.SubmitOptions{
    ModelSpecHash: model.Hash(),
    PromptHash:    aichain.PromptHash(prompt),
    N:             5,
    Threshold:     3,
    Fee:           big.NewInt(1e15),
})
// ...later, after the A-Chain settles and the A->C boundary commits the root...
receipt, err := c.WaitReceipt(ctx, intentID) // blocks until Completed or deadline
```

### Register a model (governance)

```go
txHash, err := c.RegisterModel(ctx, model) // caller must be a registry admin
approved, err := c.GetModel(ctx, model.RegistryName()) // (version, weightCommit)
```

## The `ReceiptStore` seam

`WaitReceipt` / `Infer` / `GetReceipt` read committed A-Chain receipts through a
`ReceiptStore` you supply (poll the A-Chain RPC, or watch the bridge
receipt-root checkpoint + A→C export). It is **separate** from the EVM tx path —
the SDK ships no live store (it depends on your node-local A-Chain transport),
keeping the SDK decoupled from any single A-Chain wire. Implement:

```go
type ReceiptStore interface {
    ReceiptByIntent(ctx, intentID) (InferenceReceipt, MerkleProof, found bool, err error)
}
```

An optional `AChainID() common.Hash` method lets the SDK reproduce the exact
cross-chain `intent_id`; otherwise the authoritative id is always
`receipt.IntentID`.

## Wire compatibility

Every encoder produces **exactly** the bytes the on-chain side hashes/decodes:

- `ComputeIntentID` — `keccak(DomainIntent || c_chain || a_chain || c_tx ||
  u32be(call_index) || caller || model_spec || prompt || u16be(N) ||
  u16be(threshold) || u256be(fee))` — identical to `chains/aivm.ComputeIntentID`.
- `InferenceReceipt.Encode` — the pinned 355-byte canonical encoding;
  `Hash() = keccak(DomainReceipt || Encode())`.
- `EncodeSubmitInferenceIntent` / `EncodeVerifyInferenceReceipt` — the LP-5301
  calldata frames.
- `EncodeGenerate` — the inference precompile's tight `u32be(nNew) || u32be tokens…`.
- `EncodeRegisterModel` — the registry `adopt(bytes32,uint256,bytes32)` ABI.

Golden vectors (shared with the TypeScript client `@luxfi/aichain`):

```
intent_id     = 0x5e967be3e83750c25fb91887a125d67d2440fb41825d24d63a0c00e6fb2bfbde
receipt_hash  = 0xfe0a1e45baf5255e2461c5f8f38b8446a691ec8bf0ca260750259d8bb5677851
modelSpecHash = 0xd8ab4fca51f36de6db2efd1a7a022fef6943de8d9986ae8ca3f4db70f318b4a7
```
(for the canonical fixture in `wire_test.go` / `calldata_test.go`).

## Tests

```bash
# Default lane (no live chain, no heavy deps) — CI-green:
SDKROOT=$(xcrun --show-sdk-path) GOWORK=off CGO_ENABLED=1 go test ./aichain/...

# Cross-spec parity vs the live chains/aivm wire (also CI-safe, no extra deps):
SDKROOT=$(xcrun --show-sdk-path) GOWORK=off CGO_ENABLED=1 go test -tags crossmodule ./aichain/

# Live transport build check (luxfi/evm; run in the lux workspace or with the tag):
go build -tags livedial ./aichain/
```

The `crossmodule` test asserts the SDK's encoders are byte-identical to the
A-Chain settlement wire. It does **not** import `chains/aivm` (the SDK module does
not `require` it, and that module's graph needs the lux `go.work` replace set, so
an import would break the `GOWORK=off` lane). Instead it hand-builds the chain's
exact keccak preimages and pins the live golden digests emitted by
`chains/aivm/quorum_wire_test.go`. It is build-tagged so the default
`go test ./...` lane carries **no** cross-module dependency, and it stays green in
both lanes.

## Status codes

`InferenceReceipt.Status`: `0` Unknown, `1` Pending, `2` Completed, `3` Failed,
`4` Challenged. Only `Completed` with a **non-zero** `CanonicalOutputHash` is
actionable (`InferenceReceipt.Completed()`); `WaitReceipt` enforces this.
