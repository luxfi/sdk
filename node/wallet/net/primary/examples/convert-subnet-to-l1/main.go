// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"encoding/hex"
	"log"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/api/info"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/platformvm/warp/message"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/node/wallet/net/primary"
	"github.com/luxfi/sdk/node/wallet/net/primary/examples/keyutil"
)

func main() {
	key := keyutil.MustLoadKey()
	uri := primary.LocalAPIURI
	kc := primary.NewKeychainAdapter(secp256k1fx.NewKeychain(key))

	// Create adapter for the keychain
	netID := ids.FromStringOrPanic("2DeHa7Qb6sufPkmQcFWG2uCd4pBPv9WB6dkzroiMQhd1NSRtof")
	chainID := ids.FromStringOrPanic("E8nTR9TtRwfkS7XFjTYUYHENQ91mkPMtDUwwCeu7rNgBBtkqu")
	addressHex := ""
	weight := units.Schmeckle

	address, err := hex.DecodeString(addressHex)
	if err != nil {
		log.Fatalf("failed to decode address %q: %s\n", addressHex, err)
	}

	ctx := context.Background()
	infoClient := info.NewClient(uri)

	nodeInfoStartTime := time.Now()
	nodeID, nodePoP, err := infoClient.GetNodeID(ctx)
	if err != nil {
		log.Fatalf("failed to fetch node IDs: %s\n", err)
	}
	log.Printf("fetched node ID %s in %s\n", nodeID, time.Since(nodeInfoStartTime))

	validationID := netID.Append(0)
	conversionID, err := message.ChainToL1ConversionID(message.ChainToL1ConversionData{
		ChainID:        netID,
		ManagerChainID: chainID,
		ManagerAddress: address,
		Validators: []message.ChainToL1ConversionValidatorData{
			{
				NodeID:       nodeID.Bytes(),
				BLSPublicKey: nodePoP.PublicKey,
				Weight:       weight,
			},
		},
	})
	if err != nil {
		log.Fatalf("failed to calculate conversionID: %s\n", err)
	}

	// MakeWallet fetches the available UTXOs owned by [kc] on the P-chain that
	// [uri] is hosting and registers [netID].
	walletSyncStartTime := time.Now()
	wallet, err := primary.MakeWallet(
		ctx,
		&primary.WalletConfig{
			URI:              uri,
			LUXKeychain:      kc,
			EthKeychain:      kc,
			PChainTxsToFetch: set.Of(netID),
		},
	)
	if err != nil {
		log.Fatalf("failed to initialize wallet: %s\n", err)
	}
	log.Printf("synced wallet in %s\n", time.Since(walletSyncStartTime))

	convertNetToL1StartTime := time.Now()
	convertNetToL1Tx, err := wallet.P().IssueConvertChainToL1Tx(
		netID,
		chainID,
		address,
		[]*txs.ConvertChainToL1Validator{
			{
				NodeID:                nodeID[:],
				Weight:                weight,
				Balance:               units.Lux,
				Signer:                *nodePoP,
				RemainingBalanceOwner: message.PChainOwner{},
				DeactivationOwner:     message.PChainOwner{},
			},
		},
	)
	if err != nil {
		log.Fatalf("failed to issue net conversion transaction: %s\n", err)
	}
	log.Printf("converted net %s with transactionID %s, validationID %s, and conversionID %s in %s\n",
		netID,
		convertNetToL1Tx.ID(),
		validationID,
		conversionID,
		time.Since(convertNetToL1StartTime),
	)
}
