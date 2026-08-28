// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build livedial

// dial.go wires the LIVE EVM transport (luxfi/evm/ethclient) into the SDK.
//
// It is behind the `livedial` build tag because luxfi/evm is a heavy node-side
// module: with GOWORK=off (the SDK's CI lane) its published version drags in a
// luxfi/upgrade pin that does not resolve standalone — the same reason the lux
// workspace force-replaces luxfi/upgrade in go.work. The CORE SDK (encoders,
// receipt/proof codecs, the Client over the EVMBackend interface, all tests)
// therefore compiles and tests with NO luxfi/evm dependency, so default
// `GOWORK=off go test ./...` stays green. Build/import this file only inside the
// lux workspace (where evm resolves) or with `-tags livedial`:
//
//	c, err := aichain.Dial("https://api.lux.network/v1/bc/C/rpc", privKeyHex)
//
// Everywhere else, construct the Client directly from any EVMBackend
// (aichain.NewClient) — luxfi/evm/ethclient.Client satisfies that interface, so
// callers already holding a node client pass it straight through with no tag.
package aichain

import (
	"fmt"

	"github.com/luxfi/evm/ethclient"
)

// Dial connects to an EVM JSON-RPC endpoint (the C-Chain RPC) and returns a
// ready Client signing with privKeyHex. Wire a ReceiptStore via WithReceiptStore
// to enable the Tier-2 WaitReceipt / Infer convenience. luxfi/evm/ethclient.Client
// satisfies EVMBackend directly, so it is passed straight through.
func Dial(rpcURL, privKeyHex string, opts ...Option) (*Client, error) {
	ec, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("aichain: dial %s: %w", rpcURL, err)
	}
	return NewClient(ec, privKeyHex, opts...)
}
