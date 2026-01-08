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

	validationID := ids.FromStringOrPanic("9FAftNgNBrzHUMMApsSyV6RcFiL9UmCbvsCu28xdLV2mQ7CMo")
	balance := uint64(2)

	ctx := context.Background()

	// MakeWallet fetches the available UTXOs owned by [kc] on the P-chain that
	// [uri] is hosting.
	walletSyncStartTime := time.Now()
	wallet, err := primary.MakeWallet(
		ctx,
		&primary.WalletConfig{
			URI:         uri,
			LUXKeychain: kc,
			EthKeychain: kc, // Empty ETH keychain
		},
	)
	if err != nil {
		log.Fatalf("failed to initialize wallet: %s\n", err)
	}
	log.Printf("synced wallet in %s\n", time.Since(walletSyncStartTime))

	increaseL1ValidatorBalanceStartTime := time.Now()
	increaseL1ValidatorBalanceTx, err := wallet.P().IssueIncreaseL1ValidatorBalanceTx(
		validationID,
		balance,
	)
	if err != nil {
		log.Fatalf("failed to issue increase balance transaction: %s\n", err)
	}
	log.Printf("increased balance of validationID %s by %d with %s in %s\n",
		validationID,
		balance,
		increaseL1ValidatorBalanceTx.ID(),
		time.Since(increaseL1ValidatorBalanceStartTime),
	)
}
