// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package info

import (
	"context"
	"net/netip"
	"time"

	apiinfo "github.com/luxfi/api/info"
	"github.com/luxfi/ids"
	"github.com/luxfi/rpc"
	"github.com/luxfi/upgrade"
)

type Client struct {
	Requester rpc.EndpointRequester
}

func NewClient(uri string) *Client {
	return &Client{Requester: rpc.NewEndpointRequester(
		uri + "/v1/info",
	)}
}

func (c *Client) GetNodeVersion(ctx context.Context, options ...rpc.Option) (*apiinfo.GetNodeVersionReply, error) {
	res := &apiinfo.GetNodeVersionReply{}
	err := c.Requester.SendRequest(ctx, apiinfo.MethodGetNodeVersion, struct{}{}, res, options...)
	return res, err
}

func (c *Client) GetNodeID(ctx context.Context, options ...rpc.Option) (ids.NodeID, *apiinfo.ProofOfPossession, error) {
	res := &apiinfo.GetNodeIDReply{}
	err := c.Requester.SendRequest(ctx, apiinfo.MethodGetNodeID, struct{}{}, res, options...)
	return res.NodeID, res.NodePOP, err
}

func (c *Client) GetNodeIP(ctx context.Context, options ...rpc.Option) (netip.AddrPort, error) {
	res := &apiinfo.GetNodeIPReply{}
	err := c.Requester.SendRequest(ctx, apiinfo.MethodGetNodeIP, struct{}{}, res, options...)
	// The reply carries the address as the text it has always been on the JSON
	// wire. A netip.AddrPort cannot cross a wire whose fields are offsets:
	// every one of its own fields is unexported, so it derived an empty layout
	// and arrived blank with no error to say so.
	return res.IP.AddrPort(), err
}

func (c *Client) GetNetworkID(ctx context.Context, options ...rpc.Option) (uint32, error) {
	res := &apiinfo.GetNetworkIDReply{}
	err := c.Requester.SendRequest(ctx, apiinfo.MethodGetNetworkID, struct{}{}, res, options...)
	return uint32(res.NetworkID), err
}

func (c *Client) GetNetworkName(ctx context.Context, options ...rpc.Option) (string, error) {
	res := &apiinfo.GetNetworkNameReply{}
	err := c.Requester.SendRequest(ctx, apiinfo.MethodGetNetworkName, struct{}{}, res, options...)
	return res.NetworkName, err
}

func (c *Client) GetBlockchainID(ctx context.Context, alias string, options ...rpc.Option) (ids.ID, error) {
	res := &apiinfo.GetBlockchainIDReply{}
	err := c.Requester.SendRequest(ctx, apiinfo.MethodGetBlockchainID, &apiinfo.GetBlockchainIDArgs{
		Alias: alias,
	}, res, options...)
	return res.BlockchainID, err
}

func (c *Client) Peers(ctx context.Context, nodeIDs []ids.NodeID, options ...rpc.Option) ([]apiinfo.Peer, error) {
	res := &apiinfo.PeersReply{}
	err := c.Requester.SendRequest(ctx, apiinfo.MethodPeers, &apiinfo.PeersArgs{
		NodeIDs: nodeIDs,
	}, res, options...)
	return res.Peers, err
}

func (c *Client) IsBootstrapped(ctx context.Context, chainID string, options ...rpc.Option) (bool, error) {
	res := &apiinfo.IsBootstrappedResponse{}
	err := c.Requester.SendRequest(ctx, apiinfo.MethodIsBootstrapped, &apiinfo.IsBootstrappedArgs{
		Chain: chainID,
	}, res, options...)
	return res.IsBootstrapped, err
}

func (c *Client) Upgrades(ctx context.Context, options ...rpc.Option) (*upgrade.Config, error) {
	res := &upgrade.Config{}
	err := c.Requester.SendRequest(ctx, apiinfo.MethodUpgrades, struct{}{}, res, options...)
	return res, err
}

func (c *Client) Uptime(ctx context.Context, options ...rpc.Option) (*apiinfo.UptimeResponse, error) {
	res := &apiinfo.UptimeResponse{}
	err := c.Requester.SendRequest(ctx, apiinfo.MethodUptime, struct{}{}, res, options...)
	return res, err
}

// GetVMs are the VMs installed on the node, in id order. A list rather than a
// map because that is what the node answers with — a map has no layout on a
// wire whose fields are offsets, and no order for two reads to agree on.
func (c *Client) GetVMs(ctx context.Context, options ...rpc.Option) (apiinfo.VMAliases, error) {
	res := &apiinfo.GetVMsReply{}
	err := c.Requester.SendRequest(ctx, apiinfo.MethodGetVMs, struct{}{}, res, options...)
	return res.VMs, err
}

// AwaitBootstrapped polls the node every [freq] to check if [chainID] has
// finished bootstrapping. Returns true once [chainID] reports that it has
// finished bootstrapping.
// Only returns an error if [ctx] returns an error.
func AwaitBootstrapped(ctx context.Context, c *Client, chainID string, freq time.Duration, options ...rpc.Option) (bool, error) {
	ticker := time.NewTicker(freq)
	defer ticker.Stop()

	for {
		res, err := c.IsBootstrapped(ctx, chainID, options...)
		if err == nil && res {
			return true, nil
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
}
