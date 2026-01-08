// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"log"
	"time"

	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/crypto/bls/signer/localsigner"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/vms/platformvm/warp"
	"github.com/luxfi/sdk/node/vms/platformvm/warp/message"
	"github.com/luxfi/sdk/node/vms/platformvm/warp/payload"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/node/wallet/net/primary"
	"github.com/luxfi/sdk/node/wallet/net/primary/examples/keyutil"
)

func main() {
	key := keyutil.MustLoadKey()
	uri := primary.LocalAPIURI
	kc := primary.NewKeychainAdapter(secp256k1fx.NewKeychain(key))

	// Create adapter for the keychain
	chainID := ids.FromStringOrPanic("2BMFrJ9xeh5JdwZEx6uuFcjfZC2SV2hdbMT8ee5HrvjtfJb5br")
	address := []byte{}
	validationID := ids.FromStringOrPanic("2Y3ZZZXxpzm46geqVuqFXeSFVbeKihgrfeXRDaiF4ds6R2N8M5")
	nonce := uint64(1)
	weight := uint64(2)
	blsSKHex := "3f783929b295f16cd1172396acb23b20eed057b9afb1caa419e9915f92860b35"

	blsSKBytes, err := hex.DecodeString(blsSKHex)
	if err != nil {
		log.Fatalf("failed to decode secret key: %s\n", err)
	}

	sk, err := localsigner.FromBytes(blsSKBytes)
	if err != nil {
		log.Fatalf("failed to parse secret key: %s\n", err)
	}

	// MakeWallet fetches the available UTXOs owned by [kc] on the P-chain that
	// [uri] is hosting.
	walletSyncStartTime := time.Now()
	ctx := context.Background()
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

	// Get the chain context
	context := wallet.P().Builder().Context()

	addressedCallPayload, err := message.NewL1ValidatorWeight(
		validationID,
		nonce,
		weight,
	)
	if err != nil {
		log.Fatalf("failed to create L1ValidatorWeight message: %s\n", err)
	}
	addressedCallPayloadJSON, err := json.MarshalIndent(addressedCallPayload, "", "\t")
	if err != nil {
		log.Fatalf("failed to marshal L1ValidatorWeight message: %s\n", err)
	}
	log.Println(string(addressedCallPayloadJSON))

	addressedCall, err := payload.NewAddressedCall(
		address,
		addressedCallPayload.Bytes(),
	)
	if err != nil {
		log.Fatalf("failed to create AddressedCall message: %s\n", err)
	}

	unsignedWarp, err := warp.NewUnsignedMessage(
		context.NetworkID,
		chainID,
		addressedCall.Bytes(),
	)
	if err != nil {
		log.Fatalf("failed to create unsigned Warp message: %s\n", err)
	}
	signedWarp, err := sk.Sign(unsignedWarp.Bytes())
	if err != nil {
		log.Fatalf("failed to sign Warp message: %s\n", err)
	}

	warp, err := warp.NewMessage(
		unsignedWarp,
		&warp.BitSetSignature{
			Signers: set.NewBits(0).Bytes(),
			Signature: ([bls.SignatureLen]byte)(
				bls.SignatureToBytes(signedWarp),
			),
		},
	)
	if err != nil {
		log.Fatalf("failed to create Warp message: %s\n", err)
	}

	setWeightStartTime := time.Now()
	setWeightTx, err := wallet.P().IssueSetL1ValidatorWeightTx(
		warp.Bytes(),
	)
	if err != nil {
		log.Fatalf("failed to issue set L1 validator weight transaction: %s\n", err)
	}
	log.Printf("issued set weight of validationID %s to %d with nonce %d and txID %s in %s\n", validationID, weight, nonce, setWeightTx.ID(), time.Since(setWeightStartTime))
}
