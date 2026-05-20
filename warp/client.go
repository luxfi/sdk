// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"context"

	apiwarp "github.com/luxfi/api/warp"
	"github.com/luxfi/ids"
	"github.com/luxfi/rpc"
)

var _ Client = (*client)(nil)

// Client interface for interacting with the IPCS endpoint
//
// Deprecated: The IPCs API is deprecated. The Index API should be used instead.
type Client interface {
	// PublishBlockchain requests the node to begin publishing consensus and decision events
	PublishBlockchain(ctx context.Context, chainID string, options ...rpc.Option) (*apiwarp.PublishBlockchainReply, error)
	// UnpublishBlockchain requests the node to stop publishing consensus and decision events
	UnpublishBlockchain(ctx context.Context, chainID string, options ...rpc.Option) error
	// GetPublishedBlockchains requests the node to get blockchains being published
	GetPublishedBlockchains(ctx context.Context, options ...rpc.Option) ([]ids.ID, error)
}

// Client implementation for interacting with the IPCS endpoint
type client struct {
	requester rpc.EndpointRequester
}

// NewClient returns a Client for interacting with the IPCS endpoint
//
// Deprecated: The IPCs API is deprecated. The Index API should be used instead.
func NewClient(uri string) Client {
	return &client{requester: rpc.NewEndpointRequester(
		uri + "/ext/ipcs",
	)}
}

func (c *client) PublishBlockchain(ctx context.Context, blockchainID string, options ...rpc.Option) (*apiwarp.PublishBlockchainReply, error) {
	res := &apiwarp.PublishBlockchainReply{}
	err := c.requester.SendRequest(ctx, "ipcs.publishBlockchain", &apiwarp.PublishBlockchainArgs{
		BlockchainID: blockchainID,
	}, res, options...)
	return res, err
}

func (c *client) UnpublishBlockchain(ctx context.Context, blockchainID string, options ...rpc.Option) error {
	return c.requester.SendRequest(ctx, "ipcs.unpublishBlockchain", &apiwarp.UnpublishBlockchainArgs{
		BlockchainID: blockchainID,
	}, &apiwarp.EmptyReply{}, options...)
}

func (c *client) GetPublishedBlockchains(ctx context.Context, options ...rpc.Option) ([]ids.ID, error) {
	res := &apiwarp.GetPublishedBlockchainsReply{}
	err := c.requester.SendRequest(ctx, "ipcs.getPublishedBlockchains", nil, res, options...)
	return res.Chains, err
}
