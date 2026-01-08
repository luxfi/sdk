// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package x

import (
	"context"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/wallet/chain/x/builder"
	"github.com/luxfi/sdk/api/info"
)

func NewContextFromURI(ctx context.Context, uri string, luxAssetID ids.ID, baseTxFee uint64, createAssetTxFee uint64) (*builder.Context, error) {
	infoClient := info.NewClient(uri)
	return NewContextFromClients(ctx, infoClient, luxAssetID, baseTxFee, createAssetTxFee)
}

func NewContextFromClients(
	ctx context.Context,
	infoClient *info.Client,
	luxAssetID ids.ID,
	baseTxFee uint64,
	createAssetTxFee uint64,
) (*builder.Context, error) {
	networkID, err := infoClient.GetNetworkID(ctx)
	if err != nil {
		return nil, err
	}

	chainID, err := infoClient.GetBlockchainID(ctx, builder.Alias)
	if err != nil {
		return nil, err
	}

	return &builder.Context{
		NetworkID:        networkID,
		BlockchainID:     chainID,
		XAssetID:         luxAssetID,
		BaseTxFee:        baseTxFee,
		CreateAssetTxFee: createAssetTxFee,
	}, nil
}
