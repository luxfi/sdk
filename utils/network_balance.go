// Copyright (C) 2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"context"
	"fmt"

	"github.com/luxfi/const"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/node/wallet/chain/p"
	"github.com/luxfi/node/wallet/chain/p/builder"
	"github.com/luxfi/node/wallet/net/primary"
	"github.com/luxfi/node/wallet/net/primary/common"
	"github.com/luxfi/sdk/models"
)

// GetNetworkBalance returns the balance of an address on the P-chain
func GetNetworkBalance(address ids.ShortID, network models.Network) (uint64, error) {
	return GetAddressBalance(address, network.Endpoint())
}

// GetAddressBalance returns the LUX balance of an address using the given endpoint
func GetAddressBalance(address ids.ShortID, endpoint string) (uint64, error) {
	ctx := context.Background()
	addresses := set.Of(address)

	// Fetch state including UTXOs
	state, err := primary.FetchState(ctx, endpoint, addresses)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch state: %w", err)
	}

	// Create chain UTXOs wrapper and backend
	pUTXOs := common.NewChainUTXOs(constants.PlatformChainID, state.UTXOs)
	pBackend := p.NewBackend(state.PCTX, pUTXOs, nil)
	pBuilder := builder.New(addresses, state.PCTX, pBackend)

	// Get balance from UTXOs
	currentBalances, err := pBuilder.GetBalance()
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}

	// Return the LUX balance
	luxID := state.PCTX.XAssetID
	return currentBalances[luxID], nil
}
