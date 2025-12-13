// Copyright (C) 2019-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

// xfer is a simple tool to transfer LUX from X-Chain to P-Chain
// Usage: xfer -uri http://127.0.0.1:9630 -key ~/.lux/keys/mainnet-deployer.pk -amount 1000000000
// Or set LUX_PRIVATE_KEY env var (hex format) or LUX_MNEMONIC env var (24 words)
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/node/utils/constants"
	"github.com/luxfi/node/vms/components/lux"
	"github.com/luxfi/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/keys"
	"github.com/luxfi/sdk/wallet/primary"
)

func main() {
	uri := flag.String("uri", "http://127.0.0.1:9630", "Node URI")
	keyPath := flag.String("key", "", "Path to private key file (.pk hex format)")
	amount := flag.Uint64("amount", 100000000, "Amount in nLUX to transfer (default 0.1 LUX)")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Load key from file or environment
	keyBytes, err := keys.LoadKey(*keyPath)
	if err != nil {
		log.Fatalf("Failed to load key: %v", err)
	}

	// Create keychain
	kc, err := keys.NewKeychain(keyBytes)
	if err != nil {
		log.Fatalf("Failed to create keychain: %v", err)
	}
	kcAdapter := primary.NewKeychainAdapter(kc)

	log.Printf("Using address: %s", kc.Addrs)
	log.Printf("Node URI: %s", *uri)
	log.Printf("Amount to transfer: %d nLUX", *amount)

	// Create wallet
	log.Println("Creating wallet...")
	wallet, err := primary.MakeWallet(ctx, &primary.WalletConfig{
		URI:         *uri,
		LUXKeychain: kcAdapter,
		EthKeychain: kcAdapter,
	})
	if err != nil {
		log.Fatalf("Failed to create wallet: %v", err)
	}

	// Get P-Chain ID for export
	pChainID := constants.PlatformChainID

	// Get one of our addresses for the P-Chain output
	addrs := kc.Addrs.List()
	if len(addrs) == 0 {
		log.Fatal("No addresses in keychain")
	}
	pAddr := addrs[0]

	// Step 1: Export from X-Chain to P-Chain
	log.Printf("Step 1: Exporting %d nLUX from X-Chain to P-Chain...", *amount)

	// Get LUX asset ID from context
	luxState, err := primary.FetchState(ctx, *uri, kc.Addrs)
	if err != nil {
		log.Fatalf("Failed to fetch state: %v", err)
	}
	luxAssetID := luxState.XCTX.XAssetID

	xWallet := wallet.X()
	exportTx, err := xWallet.IssueExportTx(
		pChainID,
		[]*lux.TransferableOutput{
			{
				Asset: lux.Asset{ID: luxAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt: *amount,
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: 1,
						Addrs:     []ids.ShortID{pAddr},
					},
				},
			},
		},
	)
	if err != nil {
		log.Fatalf("Failed to issue export tx: %v", err)
	}
	log.Printf("Export TX ID: %s", exportTx.ID())

	// Wait for export to be accepted
	log.Println("Waiting for export to be accepted...")
	time.Sleep(2 * time.Second)

	// Step 2: Import on P-Chain
	log.Println("Step 2: Importing on P-Chain...")

	// Recreate wallet to pick up new UTXOs
	wallet, err = primary.MakeWallet(ctx, &primary.WalletConfig{
		URI:         *uri,
		LUXKeychain: kcAdapter,
		EthKeychain: kcAdapter,
	})
	if err != nil {
		log.Fatalf("Failed to recreate wallet: %v", err)
	}

	pWallet := wallet.P()

	// Get X-Chain ID for import source (from already-fetched state)
	xChainID := luxState.XCTX.BlockchainID

	importTx, err := pWallet.IssueImportTx(
		xChainID,
		&secp256k1fx.OutputOwners{
			Threshold: 1,
			Addrs:     []ids.ShortID{pAddr},
		},
	)
	if err != nil {
		log.Fatalf("Failed to issue import tx: %v", err)
	}
	log.Printf("Import TX ID: %s", importTx.ID())

	log.Println("Transfer complete!")
	log.Printf("Transferred %d nLUX from X-Chain to P-Chain", *amount)
}
