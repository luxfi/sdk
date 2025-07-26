// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package client

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// Client is an Ethereum client for interacting with EVM chains
type Client struct {
	rpc *rpc.Client
}

// NewClient creates a new EVM client
func NewClient(endpoint string) (*Client, error) {
	rc, err := rpc.Dial(endpoint)
	if err != nil {
		return nil, err
	}
	return &Client{rpc: rc}, nil
}

// Close closes the client connection
func (c *Client) Close() {
	c.rpc.Close()
}

// ChainID retrieves the chain ID
func (c *Client) ChainID(ctx context.Context) (*big.Int, error) {
	var result hexutil.Big
	err := c.rpc.CallContext(ctx, &result, "eth_chainId")
	if err != nil {
		return nil, err
	}
	return (*big.Int)(&result), nil
}

// BlockNumber returns the current block number
func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	var result hexutil.Uint64
	err := c.rpc.CallContext(ctx, &result, "eth_blockNumber")
	return uint64(result), err
}

// BalanceAt returns the balance of an account at a specific block
func (c *Client) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	var result hexutil.Big
	err := c.rpc.CallContext(ctx, &result, "eth_getBalance", account, toBlockNumArg(blockNumber))
	return (*big.Int)(&result), err
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	return hexutil.EncodeBig(number)
}