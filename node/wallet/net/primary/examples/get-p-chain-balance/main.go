// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"log"
	"time"

	"github.com/luxfi/const"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/utils/formatting/address"
	"github.com/luxfi/sdk/node/wallet/chain/p"
	"github.com/luxfi/sdk/node/wallet/chain/p/builder"
	"github.com/luxfi/sdk/node/wallet/net/primary"
	"github.com/luxfi/sdk/node/wallet/net/primary/common"
)

func main() {
	uri := primary.LocalAPIURI
	addrStr := "P-local18jma8ppw3nhx5r4ap8clazz0dps7rv5u00z96u"

	addr, err := address.ParseToID(addrStr)
	if err != nil {
		log.Fatalf("failed to parse address: %s\n", err)
	}

	addresses := set.Of(addr)

	ctx := context.Background()

	fetchStartTime := time.Now()
	state, err := primary.FetchState(ctx, uri, addresses)
	if err != nil {
		log.Fatalf("failed to fetch state: %s\n", err)
	}
	log.Printf("fetched state of %s in %s\n", addrStr, time.Since(fetchStartTime))

	pUTXOs := common.NewChainUTXOs(constants.PlatformChainID, state.UTXOs)
	pBackend := p.NewBackend(state.PCTX, pUTXOs, nil)
	pBuilder := builder.New(addresses, state.PCTX, pBackend)

	currentBalances, err := pBuilder.GetBalance()
	if err != nil {
		log.Fatalf("failed to get the balance: %s\n", err)
	}

	luxID := state.PCTX.XAssetID
	luxBalance := currentBalances[luxID]
	log.Printf("current LUX balance of %s is %d nLUX\n", addrStr, luxBalance)
}
