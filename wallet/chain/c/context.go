// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package c

import (
	"context"

	"github.com/luxfi/ids"
	log "github.com/luxfi/log"
	"github.com/luxfi/runtime"
	sdkinfo "github.com/luxfi/sdk/info"
)

const Alias = "C"

type Context struct {
	NetworkID    uint32
	BlockchainID ids.ID
	UTXOAssetID  ids.ID
}

func NewContextFromURI(ctx context.Context, uri string, luxAssetID ids.ID) (*Context, error) {
	infoClient := sdkinfo.NewClient(uri)
	return NewContextFromClients(ctx, infoClient, luxAssetID)
}

func NewContextFromClients(
	ctx context.Context,
	infoClient *sdkinfo.Client,
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
		UTXOAssetID:  luxAssetID,
	}, nil
}

func newConsensusRuntime(c *Context) (*runtime.Runtime, error) {
	lookup := ids.NewAliaser()
	rt := &runtime.Runtime{
		NetworkID:   c.NetworkID,
		ChainID:     c.BlockchainID,
		UTXOAssetID: c.UTXOAssetID,
		Log:         log.NoLog{},
	}
	return rt, lookup.Alias(c.BlockchainID, Alias)
}
