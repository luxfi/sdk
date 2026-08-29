// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package exchangevm

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/rpc"
	"github.com/luxfi/sdk/constants"
)

// records the method string a client sends, and nothing else. What is under
// test is the ADDRESS — the name on the wire and the path it goes to — not what
// the node would answer.
type recorder struct {
	method string
}

func (r *recorder) SendRequest(_ context.Context, method string, _, _ interface{}, _ ...rpc.Option) error {
	r.method = method
	return errStop
}

var errStop = stopErr{}

type stopErr struct{}

func (stopErr) Error() string { return "stop" }

// TestTheServiceIsTheOneTheNodeRegisters is the defect this package shipped: it
// sent exchangevm.getBlock and friends, and the node registers the X-chain's
// service under "xvm" (vms/xvm/vm.go). Eleven methods addressed a service that
// does not exist, and nothing failed to compile — a JSON-RPC method name is a
// string, so a wrong one is a run-time 404 and not a build error.
//
// buildGenesis already said xvm, which is what made the disagreement visible.
func TestTheServiceIsTheOneTheNodeRegisters(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	for name, send := range map[string]func(*Client) error{
		"xvm.getBlock":            func(c *Client) error { _, err := c.GetBlock(ctx, ids.Empty); return err },
		"xvm.getBlockByHeight":    func(c *Client) error { _, err := c.GetBlockByHeight(ctx, 1); return err },
		"xvm.getHeight":           func(c *Client) error { _, err := c.GetHeight(ctx); return err },
		"xvm.issueTx":             func(c *Client) error { _, err := c.IssueTx(ctx, []byte{1}); return err },
		"xvm.getTxStatus":         func(c *Client) error { _, err := c.GetTxStatus(ctx, ids.Empty); return err },
		"xvm.getTx":               func(c *Client) error { _, err := c.GetTx(ctx, ids.Empty); return err },
		"xvm.getUTXOs":            func(c *Client) error { _, _, _, err := c.GetUTXOs(ctx, nil, 0, ids.ShortEmpty, ids.Empty); return err },
		"xvm.getAssetDescription": func(c *Client) error { _, err := c.GetAssetDescription(ctx, "LUX"); return err },
		"xvm.getBalance":          func(c *Client) error { _, err := c.GetBalance(ctx, ids.ShortEmpty, "LUX", false); return err },
		"xvm.getAllBalances":      func(c *Client) error { _, err := c.GetAllBalances(ctx, ids.ShortEmpty, false); return err },
		"xvm.getTxFee":            func(c *Client) error { _, _, err := c.GetTxFee(ctx); return err },
	} {
		r := &recorder{}
		_ = send(&Client{Requester: r})
		require.Equal(name, r.method)
	}
}

// TestTheAddressIsOneTheNodeServes pins the other half of the same defect. The
// node's only prefix is /v1 — "the legacy /ext prefix is gone", server/http/
// server.go — and this package asked for /ext/bc/X, so every call 404'd.
//
// It reads the composer rather than a literal because that is the point: one
// place spells a node address, so all of them move together.
func TestTheAddressIsOneTheNodeServes(t *testing.T) {
	require := require.New(t)

	addr := constants.Chain("http://localhost:9650", "X")
	require.True(strings.HasPrefix(addr, "http://localhost:9650/v1/"), addr)
	require.NotContains(addr, "/ext/")
	require.True(strings.HasSuffix(addr, "/X"), addr)

	// A trailing slash on the node's URI is not a second address.
	require.Equal(addr, constants.Chain("http://localhost:9650/", "X"))

	require.Equal(addr+"/wallet", constants.Chain("http://localhost:9650", "X")+"/wallet")
	require.NotContains(constants.VM("http://localhost:9650", "xvm"), "/ext/")
	require.NotContains(constants.Index("http://localhost:9650", "C", "block"), "/ext/")
}
