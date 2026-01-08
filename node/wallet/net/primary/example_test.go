// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package primary

import (
	"context"
	"log"
	"time"

	"github.com/luxfi/const"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/components/verify"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
	"github.com/luxfi/sdk/node/vms/platformvm/signer"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/node/wallet/net/primary/examples/keyutil"
)

func ExampleWallet() {
	ctx := context.Background()
	// Load key from local key storage (~/.lux/keys/) or environment variables
	// Priority: LUX_MNEMONIC > LUX_PRIVATE_KEY > LUX_KEY_NAME > CLI arg > default key
	localKey, err := keyutil.LoadKey()
	if err != nil {
		log.Printf("no local key available, skipping example: %v", err)
		return
	}
	kc := secp256k1fx.NewKeychain(localKey)
	wkc := NewKeychainAdapter(kc)

	// MakeWallet fetches the available UTXOs owned by [kc] on the network that
	// [LocalAPIURI] is hosting.
	walletSyncStartTime := time.Now()
	wallet, err := MakeWallet(ctx, &WalletConfig{
		URI:         LocalAPIURI,
		LUXKeychain: wkc,
		EthKeychain: wkc,
	})
	if err != nil {
		log.Fatalf("failed to initialize wallet with: %s\n", err)
		return
	}
	log.Printf("synced wallet in %s\n", time.Since(walletSyncStartTime))

	// Get the P-chain and the X-chain wallets
	pWallet := wallet.P()
	xWallet := wallet.X()
	xBuilder := xWallet.Builder()
	xContext := xBuilder.Context()

	// Pull out useful constants to use when issuing transactions.
	xChainID := xContext.BlockchainID
	owner := &secp256k1fx.OutputOwners{
		Threshold: 1,
		Addrs: []ids.ShortID{
			localKey.PublicKey().Address(),
		},
	}

	// Create a custom asset to send to the P-chain.
	createAssetStartTime := time.Now()
	createAssetTx, err := xWallet.IssueCreateAssetTx(
		"RnM",
		"RNM",
		9,
		map[uint32][]verify.State{
			0: {
				&secp256k1fx.TransferOutput{
					Amt:          100 * units.MegaLux,
					OutputOwners: *owner,
				},
			},
		},
	)
	if err != nil {
		log.Fatalf("failed to create new X-chain asset with: %s\n", err)
		return
	}
	createAssetTxID := createAssetTx.ID()
	log.Printf("created X-chain asset %s in %s\n", createAssetTxID, time.Since(createAssetStartTime))

	// Send 100 MegaLux to the P-chain.
	exportStartTime := time.Now()
	exportTx, err := xWallet.IssueExportTx(
		constants.PlatformChainID,
		[]*lux.TransferableOutput{
			{
				Asset: lux.Asset{
					ID: createAssetTxID,
				},
				Out: &secp256k1fx.TransferOutput{
					Amt:          100 * units.MegaLux,
					OutputOwners: *owner,
				},
			},
		},
	)
	if err != nil {
		log.Fatalf("failed to issue X->P export transaction with: %s\n", err)
		return
	}
	exportTxID := exportTx.ID()
	log.Printf("issued X->P export %s in %s\n", exportTxID, time.Since(exportStartTime))

	// Import the 100 MegaLux from the X-chain into the P-chain.
	importStartTime := time.Now()
	importTx, err := pWallet.IssueImportTx(xChainID, owner)
	if err != nil {
		log.Fatalf("failed to issue X->P import transaction with: %s\n", err)
		return
	}
	importTxID := importTx.ID()
	log.Printf("issued X->P import %s in %s\n", importTxID, time.Since(importStartTime))

	createSubnetStartTime := time.Now()
	createSubnetTx, err := pWallet.IssueCreateSubnetTx(owner)
	if err != nil {
		log.Fatalf("failed to issue create subnet transaction with: %s\n", err)
		return
	}
	createSubnetTxID := createSubnetTx.ID()
	log.Printf("issued create subnet transaction %s in %s\n", createSubnetTxID, time.Since(createSubnetStartTime))

	transformSubnetStartTime := time.Now()
	transformSubnetTx, err := pWallet.IssueTransformChainTx(
		createSubnetTxID,
		createAssetTxID,
		50*units.MegaLux,
		100*units.MegaLux,
		reward.PercentDenominator,
		reward.PercentDenominator,
		1,
		100*units.MegaLux,
		time.Second,
		365*24*time.Hour,
		0,
		1,
		5,
		.80*reward.PercentDenominator,
	)
	if err != nil {
		log.Fatalf("failed to issue transform net transaction with: %s\n", err)
		return
	}
	transformSubnetTxID := transformSubnetTx.ID()
	log.Printf("issued transform net transaction %s in %s\n", transformSubnetTxID, time.Since(transformSubnetStartTime))

	addPermissionlessValidatorStartTime := time.Now()
	startTime := time.Now().Add(time.Minute)
	// Generate a test node ID for this example
	testNodeID := ids.GenerateTestNodeID()
	addNetValidatorTx, err := pWallet.IssueAddPermissionlessValidatorTx(
		&txs.ChainValidator{
			Validator: txs.Validator{
				NodeID: testNodeID,
				Start:  uint64(startTime.Unix()),
				End:    uint64(startTime.Add(5 * time.Second).Unix()),
				Wght:   25 * units.MegaLux,
			},
			Chain: createSubnetTxID,
		},
		&signer.Empty{},
		createAssetTx.ID(),
		&secp256k1fx.OutputOwners{},
		&secp256k1fx.OutputOwners{},
		reward.PercentDenominator,
	)
	if err != nil {
		log.Fatalf("failed to issue add net validator with: %s\n", err)
		return
	}
	addNetValidatorTxID := addNetValidatorTx.ID()
	log.Printf("issued add net validator transaction %s in %s\n", addNetValidatorTxID, time.Since(addPermissionlessValidatorStartTime))

	addPermissionlessDelegatorStartTime := time.Now()
	addNetDelegatorTx, err := pWallet.IssueAddPermissionlessDelegatorTx(
		&txs.ChainValidator{
			Validator: txs.Validator{
				NodeID: testNodeID,
				Start:  uint64(startTime.Unix()),
				End:    uint64(startTime.Add(5 * time.Second).Unix()),
				Wght:   25 * units.MegaLux,
			},
			Chain: createSubnetTxID,
		},
		createAssetTxID,
		&secp256k1fx.OutputOwners{},
	)
	if err != nil {
		log.Fatalf("failed to issue add net delegator with: %s\n", err)
		return
	}
	addNetDelegatorTxID := addNetDelegatorTx.ID()
	log.Printf("issued add net validator delegator %s in %s\n", addNetDelegatorTxID, time.Since(addPermissionlessDelegatorStartTime))
}
