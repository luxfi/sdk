// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"log"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/node/wallet/net/primary"
	"github.com/luxfi/sdk/node/wallet/net/primary/examples/keyutil"
)

func main() {
	key := keyutil.MustLoadKey()
	uri := primary.LocalAPIURI
	kc := primary.NewKeychainAdapter(secp256k1fx.NewKeychain(key))

	// Create adapter for the keychain
	validationID := ids.FromStringOrPanic("9FAftNgNBrzHUMMApsSyV6RcFiL9UmCbvsCu28xdLV2mQ7CMo")

	ctx := context.Background()

	// MakeWallet fetches the available UTXOs owned by [kc] on the P-chain that
	// [uri] is hosting and registers [validationID].
	walletSyncStartTime := time.Now()
	wallet, err := primary.MakeWallet(
		ctx,
		&primary.WalletConfig{
			URI:              uri,
			LUXKeychain:      kc,
			EthKeychain:      kc,
			PChainTxsToFetch: set.Of(validationID),
		},
	)
	if err != nil {
		log.Fatalf("failed to initialize wallet: %s\n", err)
	}
	log.Printf("synced wallet in %s\n", time.Since(walletSyncStartTime))

	disableL1ValidatorStartTime := time.Now()
	disableL1ValidatorTx, err := wallet.P().IssueDisableL1ValidatorTx(
		validationID,
	)
	if err != nil {
		log.Fatalf("failed to issue disable L1 validator transaction: %s\n", err)
	}
	log.Printf("disabled %s with %s in %s\n",
		validationID,
		disableL1ValidatorTx.ID(),
		time.Since(disableL1ValidatorStartTime),
	)
}
