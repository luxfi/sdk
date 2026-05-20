// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package exchangevm

import (
	"context"

	"github.com/luxfi/rpc"
)

var _ StaticClient = (*staticClient)(nil)

// StaticClient for interacting with the XVM static api
type StaticClient interface {
	BuildGenesis(ctx context.Context, args *BuildGenesisArgs, options ...rpc.Option) (*BuildGenesisReply, error)
}

// staticClient is an implementation of an XVM client for interacting with the
// xvm static api
type staticClient struct {
	requester rpc.EndpointRequester
}

// NewClient returns an XVM client for interacting with the xvm static api
func NewStaticClient(uri string) StaticClient {
	return &staticClient{requester: rpc.NewEndpointRequester(
		uri + "/ext/vm/xvm",
	)}
}

func (c *staticClient) BuildGenesis(ctx context.Context, args *BuildGenesisArgs, options ...rpc.Option) (resp *BuildGenesisReply, err error) {
	resp = &BuildGenesisReply{}
	err = c.requester.SendRequest(ctx, "xvm.buildGenesis", args, resp, options...)
	return resp, err
}
