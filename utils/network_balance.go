// Copyright (C) 2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"context"
	"fmt"

	"github.com/luxfi/codec/jsonrpc"
	"github.com/luxfi/ids"
	"github.com/luxfi/rpc"
	"github.com/luxfi/sdk/models"
)

// GetNetworkBalance returns the balance of an address on the P-chain
func GetNetworkBalance(address ids.ShortID, network models.Network) (uint64, error) {
	return GetAddressBalance(address, network.Endpoint())
}

// GetAddressBalance returns the LUX balance of an address using the given endpoint
func GetAddressBalance(address ids.ShortID, endpoint string) (uint64, error) {
	type getBalanceRequest struct {
		Addresses []string `json:"addresses"`
	}
	type getBalanceResponse struct {
		Unlocked  json.Uint64            `json:"unlocked"`
		Unlockeds map[ids.ID]json.Uint64 `json:"unlockeds"`
	}

	ctx := context.Background()
	requester := rpc.NewEndpointRequester(endpoint + "/ext/P")
	reply := &getBalanceResponse{}
	if err := requester.SendRequest(ctx, "platform.getBalance", &getBalanceRequest{
		Addresses: ids.ShortIDsToStrings([]ids.ShortID{address}),
	}, reply); err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}

	if reply.Unlocked > 0 {
		return uint64(reply.Unlocked), nil
	}

	var totalUnlocked uint64
	for _, balance := range reply.Unlockeds {
		totalUnlocked += uint64(balance)
	}
	return totalUnlocked, nil
}
