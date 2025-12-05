// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package c

import (
	"context"

	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/node/api/info"
	"github.com/luxfi/node/utils/constants"
)

const Alias = "C"

type Context struct {
	NetworkID    uint32
	BlockchainID ids.ID
	XAssetID     ids.ID
}

func NewContextFromURI(ctx context.Context, uri string, luxAssetID ids.ID) (*Context, error) {
	infoClient := info.NewClient(uri)
	return NewContextFromClients(ctx, infoClient, luxAssetID)
}

func NewContextFromClients(
	ctx context.Context,
	infoClient *info.Client,
	luxAssetID ids.ID,
) (*Context, error) {
	networkID, err := infoClient.GetNetworkID(ctx)
	if err != nil {
		return nil, err
	}

	blockchainID, err := infoClient.GetBlockchainID(ctx, Alias)
	if err != nil {
		return nil, err
	}

	return &Context{
		NetworkID:    networkID,
		BlockchainID: blockchainID,
		XAssetID:     luxAssetID,
	}, nil
}

func newConsensusContext(c *Context) (*consensusctx.Context, error) {
	lookup := ids.NewAliaser()
	return &consensusctx.Context{
		NetworkID:   c.NetworkID,
		NetID:       constants.PrimaryNetworkID,
		ChainID:     c.BlockchainID,
		XAssetID:  c.XAssetID,
		Log:         log.NoLog{},
	}, lookup.Alias(c.BlockchainID, Alias)
}
