// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"log"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/node/wallet/net/primary"
	"github.com/luxfi/sdk/node/wallet/net/primary/examples/keyutil"
)

func main() {
	key := keyutil.MustLoadKey()
	uri := primary.LocalAPIURI
	kc := primary.NewKeychainAdapter(secp256k1fx.NewKeychain(key))
	subnetOwner := key.Address()

	ctx := context.Background()

	// MakeWallet fetches the available UTXOs owned by [kc] on the network that
	// [uri] is hosting.
	walletSyncStartTime := time.Now()
	wallet, err := primary.MakeWallet(ctx, &primary.WalletConfig{
		URI:         uri,
		LUXKeychain: kc,
		EthKeychain: kc,
	})
	if err != nil {
		log.Fatalf("failed to initialize wallet: %s\n", err)
	}
	log.Printf("synced wallet in %s\n", time.Since(walletSyncStartTime))

	// Get the P-chain wallet
	pWallet := wallet.P()

	// Pull out useful constants to use when issuing transactions.
	owner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs: []ids.ShortID{
			subnetOwner,
		},
	}

	createSubnetStartTime := time.Now()
	createSubnetTx, err := pWallet.IssueCreateSubnetTx(owner)
	if err != nil {
		log.Fatalf("failed to issue create subnet transaction: %s\n", err)
	}
	log.Printf("created new net %s in %s\n", createSubnetTx.ID(), time.Since(createSubnetStartTime))
}
