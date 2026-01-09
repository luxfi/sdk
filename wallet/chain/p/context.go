// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package p

import (
	"context"

	"github.com/luxfi/constants"
	"github.com/luxfi/vm/vms/platformvm"
	"github.com/luxfi/sdk/api/info"
	"github.com/luxfi/sdk/wallet/chain/p/builder"
)

// gasPriceMultiplier increases the gas price to support multiple transactions
// to be issued.
//
// TODO: Handle this better. Either here or in the mempool.
const gasPriceMultiplier = 2

func NewContextFromURI(ctx context.Context, uri string) (*builder.Context, error) {
	infoClient := info.NewClient(uri)
	chainClient := platformvm.NewClient(uri)
	return NewContextFromClients(ctx, infoClient, chainClient)
}

func NewContextFromClients(
	ctx context.Context,
	infoClient *info.Client,
	chainClient *platformvm.Client,
) (*builder.Context, error) {
	networkID, err := infoClient.GetNetworkID(ctx)
	if err != nil {
		return nil, err
	}

	// Fetch the P-Chain blockchain ID from the node
	// This is critical for transaction verification - the node validates that
	// tx.BlockchainID matches ctx.ChainID in its consensus context
	pChainID, err := infoClient.GetBlockchainID(ctx, "P")
	if err != nil {
		// Fall back to default PlatformChainID if the API call fails
		pChainID = constants.PlatformChainID
	}

	luxAssetID, err := chainClient.GetStakingAssetID(ctx, constants.PrimaryNetworkID)
	if err != nil {
		return nil, err
	}

	dynamicFeeConfig, err := chainClient.GetFeeConfig(ctx)
	if err != nil {
		return nil, err
	}

	_, gasPrice, _, err := chainClient.GetFeeState(ctx)
	if err != nil {
		return nil, err
	}

	return &builder.Context{
		NetworkID:         networkID,
		ChainID:           pChainID,
		XAssetID:          luxAssetID,
		ComplexityWeights: dynamicFeeConfig.Weights,
		GasPrice:          gasPriceMultiplier * gasPrice,
	}, nil
}
