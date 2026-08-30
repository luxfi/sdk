// Copyright (C) 2019-2026, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package primary

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
)

// fakeInfoServer is a minimal JSON-RPC 2.0 server impersonating an
// /v1/info endpoint exposing GetNetworkID, GetBlockchainID, GetTxFee.
// Behavior is controlled by xChainServed: when false, info.getBlockchainID
// for alias "X" responds with the canonical "no ID with alias: X" error.
type fakeInfoServer struct {
	mu           sync.Mutex
	xChainServed bool
	pChainAlias  string
}

func (f *fakeInfoServer) handleInfo(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var req struct {
		ID     int             `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	type rpcErr struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	type rpcResp struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      int         `json:"id"`
		Result  interface{} `json:"result,omitempty"`
		Error   *rpcErr     `json:"error,omitempty"`
	}

	resp := rpcResp{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "info.getNetworkID":
		resp.Result = map[string]interface{}{"networkID": 1}
	case "info.getBlockchainID":
		var args struct {
			Alias string `json:"alias"`
		}
		_ = json.Unmarshal(req.Params, &args)
		f.mu.Lock()
		served := f.xChainServed
		f.mu.Unlock()
		if args.Alias == "X" && !served {
			resp.Error = &rpcErr{
				Code:    -32000,
				Message: fmt.Sprintf("there is no ID with alias: %s", args.Alias),
			}
		} else {
			// arbitrary deterministic ID for tests
			resp.Result = map[string]interface{}{
				"blockchainID": "2oYMBNV4eNHyqk2fjjV5nVQLDbtmNJzq5s3qs3Lo6ftnC6FByM",
			}
		}
	case "info.getTxFee":
		resp.Result = map[string]interface{}{
			"createAssetTxFee":              "10000000",
			"createNetworkTxFee":            "1000000000",
			"createChainTxFee":              "1000000000",
			"addPrimaryNetworkValidatorFee": "0",
			"addPrimaryNetworkDelegatorFee": "0",
			"addNetworkValidatorFee":        "1000000",
			"addNetworkDelegatorFee":        "1000000",
			"txFee":                         "1000000",
		}
	default:
		resp.Error = &rpcErr{Code: -32601, Message: fmt.Sprintf("method %s not found", req.Method)}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (f *fakeInfoServer) handlePChain(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	var req struct {
		ID     int             `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	type rpcErr struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	type rpcResp struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      int         `json:"id"`
		Result  interface{} `json:"result,omitempty"`
		Error   *rpcErr     `json:"error,omitempty"`
	}
	resp := rpcResp{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "platform.getUTXOs":
		// no UTXOs for the test wallet. EndIndex must hold a parseable
		// bech32 address; we return the zero address under the "lux" HRP
		// (computed offline via address.Format("P", "lux", 20-byte zero)).
		// numFetched < fetchLimit so the wallet's pagination loop exits
		// on the first reply.
		resp.Result = map[string]interface{}{
			"numFetched": "0",
			"utxos":      []string{},
			"endIndex": map[string]string{
				"address": "P-lux1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqquc4vdn",
				"utxo":    "11111111111111111111111111111111LpoYY",
			},
			"encoding": "hex",
		}
	case "platform.getStakingAssetID":
		// Quasar staking asset (arbitrary deterministic ID for tests)
		resp.Result = map[string]interface{}{
			"assetID": "2oYMBNV4eNHyqk2fjjV5nVQLDbtmNJzq5s3qs3Lo6ftnC6FByM",
		}
	case "platform.getFeeConfig":
		// LP-103 dynamic fee config — minimal values are fine for FetchState.
		// gas.Gas / gas.Price are uint64 so values are emitted as numbers.
		resp.Result = map[string]interface{}{
			"weights":                  []uint64{1, 1, 1, 1},
			"maxCapacity":              uint64(1000000),
			"maxPerSecond":             uint64(100000),
			"targetPerSecond":          uint64(10000),
			"minPrice":                 uint64(1),
			"excessConversionConstant": uint64(1),
		}
	case "platform.getFeeState":
		resp.Result = map[string]interface{}{
			"capacity":  uint64(1000000),
			"excess":    uint64(0),
			"price":     uint64(1),
			"timestamp": "2026-06-06T00:00:00Z",
		}
	default:
		resp.Error = &rpcErr{Code: -32601, Message: fmt.Sprintf("method %s not found", req.Method)}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// TestFetchState_XChainNotRegistered verifies the canonical fix: when
// the node's info service reports no X-Chain alias (the Quasar-era
// mainnet P+C topology), FetchState succeeds with an X-Chain context
// whose BlockchainID is the ids.Empty sentinel. Previously this errored
// out with `failed to decode client response: there is no ID with alias: X`.
func TestFetchState_XChainNotRegistered(t *testing.T) {
	require := require.New(t)

	fake := &fakeInfoServer{xChainServed: false}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/info", fake.handleInfo)
	mux.HandleFunc("/v1/chain/P", fake.handlePChain)
	mux.HandleFunc("/v1/P", fake.handlePChain)
	// Reject any non-P-Chain call: the SDK fallback MUST NOT touch
	// platform.getUTXOs against the X-Chain alias when XCTX.BlockchainID
	// is the sentinel. If anything else gets called the test fails fast.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/v1/chain/X") {
			t.Errorf("SDK called X-Chain endpoint %s in P-only mode", r.URL.Path)
		}
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	addrs := set.Set[ids.ShortID]{}
	addrs.Add(ids.ShortEmpty)

	state, err := FetchState(context.Background(), srv.URL, addrs)
	require.NoError(err, "FetchState must succeed in P-only mode")
	require.NotNil(state)
	require.Nil(state.XCTX, "XCTX must be nil when X-Chain is not registered (no sentinel kludge)")
	require.Nil(state.XClient, "XClient must be nil when X-Chain is not registered")
	require.NotNil(state.PCTX)
	require.NotNil(state.PClient)
}

// handleXChain answers the single X-Chain read the wallet makes: xvm.getUTXOs.
// The X wallet is built over this set, so a state fetch that never calls it
// would leave the wallet unable to spend.
func (f *fakeInfoServer) handleXChain(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	var req struct {
		ID     int    `json:"id"`
		Method string `json:"method"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
		"result": map[string]interface{}{
			"numFetched": "0",
			"utxos":      []string{},
			"endIndex": map[string]string{
				"address": "X-lux1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqzu5uf3",
				"utxo":    "11111111111111111111111111111111LpoYY",
			},
			"encoding": "hex",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// TestFetchState_XChainRegistered verifies the happy path is unchanged:
// when the X alias resolves, XCTX.BlockchainID is non-empty.
func TestFetchState_XChainRegistered(t *testing.T) {
	require := require.New(t)

	fake := &fakeInfoServer{xChainServed: true}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/info", fake.handleInfo)
	mux.HandleFunc("/v1/chain/P", fake.handlePChain)
	mux.HandleFunc("/v1/P", fake.handlePChain)
	mux.HandleFunc("/v1/chain/X", fake.handleXChain)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	addrs := set.Set[ids.ShortID]{}
	addrs.Add(ids.ShortEmpty)

	state, err := FetchState(context.Background(), srv.URL, addrs)
	require.NoError(err)
	require.NotEqual(ids.Empty, state.XCTX.BlockchainID, "BlockchainID must be set when X-Chain is registered")
	require.NotNil(state.XClient, "XClient must be set when X-Chain is registered")
}
