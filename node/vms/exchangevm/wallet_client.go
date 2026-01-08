// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package exchangevm

import (
	"context"
	"fmt"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/api"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/utils/formatting"
	"github.com/luxfi/sdk/utils/rpc"
)

// WalletClient for interacting with exchangevm managed wallet.
//
// Deprecated: Transactions should be issued using the
// `luxfi/node/wallet/chain/x.Wallet` utility.
type WalletClient struct {
	Requester rpc.EndpointRequester
}

// NewWalletClient returns an Exchange VM wallet client for interacting with
// exchangevm managed wallet
//
// Deprecated: Transactions should be issued using the
// `luxfi/node/wallet/chain/x.Wallet` utility.
func NewWalletClient(uri, chain string) *WalletClient {
	path := fmt.Sprintf(
		"%s/ext/%s/%s/wallet",
		uri,
		constants.ChainAliasPrefix,
		chain,
	)
	return &WalletClient{
		Requester: rpc.NewEndpointRequester(path),
	}
}

// IssueTx issues a transaction to a node and returns the TxID
func (c *WalletClient) IssueTx(ctx context.Context, txBytes []byte, options ...rpc.Option) (ids.ID, error) {
	txStr, err := formatting.Encode(formatting.Hex, txBytes)
	if err != nil {
		return ids.Empty, err
	}
	res := &api.JSONTxID{}
	err = c.Requester.SendRequest(ctx, "wallet.issueTx", &api.FormattedTx{
		Tx:       txStr,
		Encoding: formatting.Hex,
	}, res, options...)
	return res.TxID, err
}