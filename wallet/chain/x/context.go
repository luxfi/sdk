// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package x

import (
	"context"

	"github.com/luxfi/ids"
	sdkinfo "github.com/luxfi/sdk/info"
	"github.com/luxfi/sdk/wallet/chain/x/builder"
)

func NewContextFromURI(ctx context.Context, uri string, utxoAssetID ids.ID, baseTxFee uint64, createAssetTxFee uint64) (*builder.Context, error) {
	infoClient := sdkinfo.NewClient(uri)
	return NewContextFromClients(ctx, infoClient, utxoAssetID, baseTxFee, createAssetTxFee)
}

func NewContextFromClients(
	ctx context.Context,
	infoClient *sdkinfo.Client,
	utxoAssetID ids.ID,
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
		UTXOAssetID:      utxoAssetID,
		BaseTxFee:        baseTxFee,
		CreateAssetTxFee: createAssetTxFee,
	}, nil
}
