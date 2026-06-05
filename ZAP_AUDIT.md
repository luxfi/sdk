# SDK ZAP Wire Audit

Activation: 2025-12-25T16:20:00-08:00 (unix 1766708400) — the new final Lux network. The Lux Go SDK is the public face of that wire to external clients (wallets, tooling, exchanges, indexers).

## Modules

```
~/work/lux/sdk/go.mod                                module github.com/luxfi/sdk
~/work/lux/sdk/examples/boot-mainnet/go.mod          example program, replaces sdk via replace directive
```

The SDK ships as ONE Go module (`github.com/luxfi/sdk`) plus an in-tree example. The example consumes the SDK via a local replace.

## Direct codec dep

`go.mod` line 220 has the only direct require:

```
github.com/luxfi/codec v1.1.4
```

Removing this require is the goal. Everything else (`luxfi/proto`, `luxfi/vm`, `luxfi/utxo`, `luxfi/warp`, `luxfi/node`) drags `luxfi/codec` in transitively — those flips land in LP-023/LP-184/LP-185/LP-186, not in this PR.

## Per-chain inventory

The SDK does NOT hand-roll P-chain or X-chain wire — it consumes the upstream proto modules. It DOES hand-roll C-chain atomic-tx wire (`wallet/chain/c/`).

### P-chain (`wallet/chain/p/`)

| Concern | File | Mechanism | Disposition |
|---|---|---|---|
| Tx construction | `builder/builder.go` | Builds `proto/p/txs` structs (typed Go values). No codec marshalling here. | Stays — typed Go values feed the upstream codec call in the signer. |
| Tx signing (wire bytes) | `signer/visitor.go`, `signer_visitor.go` | `proto/p/txs.Codec.Marshal(txs.CodecVersion, &tx.Unsigned)` then sign hash; second `Codec.Marshal(tx)` for signed bytes. | Blocked on LP-023 SDK API: P-chain ZAP-native lives at `~/work/lux/node/vms/platformvm/txs/zap_native/` but is not yet exposed via `proto/p/txs`. Switch to `zap_native.NewXxxTx(...)`+`Bytes()` once the parallel agent finishes wiring through proto. |
| Multisig blob | `multisig/multisig.go` | `txs.Codec.Marshal/Unmarshal` on `txs.Tx`. | Same upstream blocker. |
| Blockchain-tx parse | `contract/blockchain_tx.go` | `txs.Codec.Unmarshal(txBytes, &tx)`. | Same upstream blocker. |
| JSON-RPC types | `platformvm/api_types.go`, `platformvm/client.go` | `luxfi/codec/jsonrpc` (package name `json`) for `json.Uint64`. | **Switch to `luxfi/api/types` (`types.Uint64`).** Already a SDK dep. JSON-RPC is the external transport boundary — not wire — but the package path is `luxfi/codec/...` and that's what's removing-blocking the codec require. |

### X-chain (`wallet/chain/x/` + `exchangevm/`)

| Concern | File | Mechanism | Disposition |
|---|---|---|---|
| Parser/codec singleton | `constants.go`, `builder/constants.go` | `block.NewParser(...)` from `luxfi/proto/x/block`. | Upstream — leaves the SDK once `proto/x` migrates. |
| Tx construction | `builder/builder.go`, `builder.go` | Builds `proto/x/txs` structs. Calls `Parser.Codec()` only for `state.Sort(codec)` on `InitialState`. | Sort takes a codec to canonicalize bytes — same upstream blocker as P-chain. |
| Tx signing (wire bytes) | `signer/visitor.go`, `signer_visitor.go` | `Parser.Codec().Marshal(txs.CodecVersion, ...)` on `&tx.Unsigned` and `tx`. | LP-184 lands `~/work/lux/node/vms/xvm/wire_*.go` schemas (per task description). Once `proto/x/txs` exposes ZAP `Wrap*`/`New*`, the SDK signer switches. |
| JSON-RPC types | `exchangevm/api_types.go`, `exchangevm/client.go` | `luxfi/codec/jsonrpc`. | **Switch to `luxfi/api/types`.** |

### EVM C-chain (`wallet/chain/c/`)

| Concern | File | Mechanism | Disposition |
|---|---|---|---|
| Atomic-tx codec | `types.go` (`init()`) | `codec.NewDefaultManager()` + `linearcodec.NewDefault()`, both directly imported. The SDK constructs ITS OWN codec for C-chain atomic txs. | LP-185 declares Lux-internal EVM wire = ZAP, but the atomic-tx path lives at `~/work/lux/node/plugin/evm/atomic/*` (or its successor under `vms/evm/wire`). No `NewImportTx`/`WrapExportTx` API ships yet. **Scaffold-with-TODO** per the task — keep the existing local codec, file an issue. |
| Atomic-tx sign | `signer.go`, `types.go` | `Codec.Marshal(version, ...)`. | Same blocker. |
| Atomic-tx fee math | `types.go` (`CalculateDynamicFee`) | Pure arithmetic, not wire. | Stays. |

### EVM external surface

External Ethereum txs (`SendRawTransaction`, etc.) flow through `luxfi/rpc` / `luxfi/geth` — RLP, not codec. Out of scope per LP-185.

### Other chains (A=aivm, B=bridgevm, D=dexvm, G=graphvm, I=identityvm, K=keyvm, O=oraclevm, Q=quantumvm, R=relayvm)

The SDK exposes wallet primitives for the primary network (P, X, C-EVM) only. None of `chains/*` (`aivm`, `bridgevm`, `dexvm`, `graphvm`, `identityvm`, `keyvm`, `oraclevm`, `quantumvm`, `relayvm`) appear as separate sub-packages in `~/work/lux/sdk` — their tx surface comes through generic JSON-RPC against the chain RPC endpoint. No SDK-owned wire to migrate. LP-186 covers those VMs; the SDK consumes their RPC as already-JSON.

## JSON-RPC boundary (external transport — keep)

`luxfi/codec/jsonrpc` is misnamed — it's pure JSON-as-string wrappers (`type Uint64 uint64` with `MarshalJSON: "12345"`), not a wire codec. It is the EXTERNAL transport boundary and the task says to leave it alone. The only reason to touch it: its package path is under `luxfi/codec`, so the SDK importing it forces `luxfi/codec v1.1.4` into the direct `require` block.

`luxfi/api v1.0.11` (already a SDK direct require) ships an identical `types.Uint64` etc. Swapping the import path removes the last direct `luxfi/codec` dependency from `go.mod` without changing JSON-RPC wire semantics one bit.

## Wallet derivation / address formatting (out of scope)

`crypto/`, `key/`, `keychain/`, `address`, `multisig/` (the `multisig` package, not the file inside it): math + text formatting. No codec calls. No change.

## What this PR does

1. Switch every SDK-owned API arg/reply type from `luxfi/codec/jsonrpc.UintXX` to `luxfi/api/types.UintXX`. Same type names (`Uint64`, `Uint32`, `Float64`), same JSON wire (string-encoded numerics). Files: `exchangevm/api_types.go`, `indexer/{client,client_test,types}.go`, `ledger/ledger.go`, `platformvm/api_types.go`, `validator/validator.go`.
2. Add a local `Uint16` shim in `platformvm/api_types.go` because `luxfi/api/types` ships `Uint32/Uint64/Float64` only. Same wire format.
3. Keep `avajson "github.com/luxfi/codec/jsonrpc"` in `platformvm/client.go` and `exchangevm/client.go` for assignment to `luxfi/vm/api.GetUTXOsArgs.Limit` and `GetBlockByHeightArgs.Height` — those fields are typed by upstream `luxfi/vm/api`; this dep only moves when upstream does.
4. Replace ad-hoc `codec.Manager` parameter in `wallet/primary/api.go::AddAllUTXOs` with a local `UTXOUnmarshaler` interface (just `Unmarshal`). No more direct `luxfi/codec` reference in this file.
5. Keep `wallet/chain/c/types.go` as the residual `codec.Manager` scaffold — `lux.SortTransferableOutputs` (upstream `luxfi/utxo`) takes `codec.Manager` so the SDK cannot escape this until either (a) upstream `luxfi/utxo` ships a codec-free sort, or (b) the `wallet/chain/c/` package is deleted (it is currently dead code: `wallet/primary/wallet.go:170-180` wires `c.Wallet` as `nil`).

`luxfi/codec` stays in `go.mod` as a direct require because of points 3 and 5. Removing it requires upstream coordination, not SDK-local changes.

## Verify (after migration)

```
grep -rn "luxfi/codec\b\|linearcodec\|codec\.Manager" ~/work/lux/sdk \
  --include="*.go" | grep -v _test.go
```

Current residual hits (post-migration state, all annotated with TODO):

| File | Hit | Reason | Blocker |
|---|---|---|---|
| `platformvm/client.go:12` | `avajson "github.com/luxfi/codec/jsonrpc"` | `Limit avajson.Uint32`, `Height avajson.Uint64` on `luxfi/vm/api.GetUTXOsArgs` / `GetBlockByHeightArgs` are typed by upstream. | Upstream `luxfi/vm/api` migrates off `luxfi/codec/jsonrpc` (or off `Uint64`-named-types entirely). |
| `exchangevm/client.go:13` | same | same as platformvm/client.go | same |
| `wallet/chain/c/types.go:10-11` | `luxfi/codec` + `luxfi/codec/linearcodec` | Dead-code scaffold: C-chain atomic-tx wire. The whole `wallet/chain/c/` package is not wired at runtime (see `wallet/primary/wallet.go:170-180`; `c` arg to `NewWallet` is `nil`). `lux.SortTransferableOutputs(outs, Codec)` in `wallet/chain/c/builder.go:290` ties us to `codec.Manager`. | LP-185 follow-on shipping a `vms/evm/wire` ZAP-native atomic-tx API (`NewImport/ExportTx`, `Wrap*`). At that point delete `wallet/chain/c/types.go` Codec block + Sign param. |
| `wallet/primary/api.go:38-45` | comment-only references in `UTXOUnmarshaler` docstring | No code dep — interface satisfaction only. | LP-023/LP-184 ZAP-native UTXO accessors in `proto/p/txs` and `proto/x/txs`. |

`luxfi/codec` will remain in `go.mod` direct require until:
1. `wallet/chain/c/types.go` is deleted (when atomic-tx ZAP API ships), AND
2. `luxfi/vm/api` migrates off `luxfi/codec/jsonrpc` for `GetUTXOsArgs.Limit` and `GetBlockByHeightArgs.Height`.

Until then the dep is real, not stale. The SDK no longer constructs codec managers directly (the `init()` in `wallet/chain/c/types.go` is the only site, and it is for the unused dead path).

## Post-migration JSON-RPC boundary state

All SDK-owned API arg/reply types (`platformvm/api_types.go`, `exchangevm/api_types.go`, `indexer/types.go`, `validator/validator.go`, `ledger/ledger.go`) now use `apitypes.Uint32`/`apitypes.Uint64` from `github.com/luxfi/api/types`. JSON wire is byte-identical to the legacy `luxfi/codec/jsonrpc.UintXX` (both are string-encoded numerics).

`platformvm/api_types.go` defines a local `Uint16` because `luxfi/api/types` ships `Uint32/Uint64/Float64` only. Wire is identical.
