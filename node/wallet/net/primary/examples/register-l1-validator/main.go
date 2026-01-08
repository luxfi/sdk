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
	"github.com/luxfi/sdk/api/info"
	"github.com/luxfi/sdk/utils/units"
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
	netID := ids.FromStringOrPanic("2DeHa7Qb6sufPkmQcFWG2uCd4pBPv9WB6dkzroiMQhd1NSRtof")
	chainID := ids.FromStringOrPanic("2BMFrJ9xeh5JdwZEx6uuFcjfZC2SV2hdbMT8ee5HrvjtfJb5br")
	address := []byte{}
	weight := uint64(1)
	blsSKHex := "3f783929b295f16cd1172396acb23b20eed057b9afb1caa419e9915f92860b35"

	blsSKBytes, err := hex.DecodeString(blsSKHex)
	if err != nil {
		log.Fatalf("failed to decode secret key: %s\n", err)
	}

	sk, err := localsigner.FromBytes(blsSKBytes)
	if err != nil {
		log.Fatalf("failed to parse secret key: %s\n", err)
	}

	ctx := context.Background()
	infoClient := info.NewClient(uri)

	nodeInfoStartTime := time.Now()
	nodeID, nodePoP, err := infoClient.GetNodeID(ctx)
	if err != nil {
		log.Fatalf("failed to fetch node IDs: %s\n", err)
	}
	log.Printf("fetched node ID %s in %s\n", nodeID, time.Since(nodeInfoStartTime))

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

	// Get the chain context
	context := wallet.P().Builder().Context()

	expiry := uint64(time.Now().Add(5 * time.Minute).Unix()) // This message will expire in 5 minutes
	addressedCallPayload, err := message.NewRegisterL1Validator(
		netID,
		nodeID,
		nodePoP.PublicKey,
		expiry,
		message.PChainOwner{},
		message.PChainOwner{},
		weight,
	)
	if err != nil {
		log.Fatalf("failed to create RegisterL1Validator message: %s\n", err)
	}
	addressedCallPayloadJSON, err := json.MarshalIndent(addressedCallPayload, "", "\t")
	if err != nil {
		log.Fatalf("failed to marshal RegisterL1Validator message: %s\n", err)
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

	// This example assumes that the hard-coded BLS key is for the first
	// validator in the signature bit-set.
	signers := set.NewBits(0)

	unsignedBytes := unsignedWarp.Bytes()
	sig, err := sk.Sign(unsignedBytes)
	if err != nil {
		log.Fatalf("failed to sign message: %s\n", err)
	}
	sigBytes := [bls.SignatureLen]byte{}
	copy(sigBytes[:], bls.SignatureToBytes(sig))

	warp, err := warp.NewMessage(
		unsignedWarp,
		&warp.BitSetSignature{
			Signers:   signers.Bytes(),
			Signature: sigBytes,
		},
	)
	if err != nil {
		log.Fatalf("failed to create Warp message: %s\n", err)
	}

	registerL1ValidatorStartTime := time.Now()
	registerL1ValidatorTx, err := wallet.P().IssueRegisterL1ValidatorTx(
		units.Lux,
		nodePoP.ProofOfPossession,
		warp.Bytes(),
	)
	if err != nil {
		log.Fatalf("failed to issue register L1 validator transaction: %s\n", err)
	}

	validationID := addressedCallPayload.ValidationID()
	log.Printf("registered new L1 validator %s to netID %s with txID %s as validationID %s in %s\n", nodeID, netID, registerL1ValidatorTx.ID(), validationID, time.Since(registerL1ValidatorStartTime))
}
