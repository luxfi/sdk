// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"log"
	"net/netip"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/protobuf/proto"

	"github.com/luxfi/luxd/api/info"
	"github.com/luxfi/luxd/ids"
	"github.com/luxfi/luxd/network/p2p"
	"github.com/luxfi/luxd/network/peer"
	"github.com/luxfi/luxd/proto/pb/sdk"
	"github.com/luxfi/luxd/snow/networking/router"
	"github.com/luxfi/luxd/utils/compression"
	"github.com/luxfi/luxd/vms/platformvm/warp"
	"github.com/luxfi/luxd/vms/platformvm/warp/payload"
	"github.com/luxfi/luxd/wallet/subnet/primary"
	"github.com/luxfi/evm/warp/messages"

	p2pmessage "github.com/luxfi/luxd/message"
)

// An example application demonstrating how to request a signature for
// an uptime message from a node running locally.
func main() {
	uri := primary.LocalAPIURI
	// The following IDs are placeholders and should be replaced with real values
	// before running the code.
	// The validationID is for the validation period that the uptime message is signed for.
	validationID := ids.FromStringOrPanic("p3NUAY4PbcAnyCyvUTjGVjezNEQCdnVdfAbJcZScvKpxP5tJr")
	// The sourceChainID is the ID of the chain.
	sourceChainID := ids.FromStringOrPanic("2UZWB4xjNadRcHSpXarQoCryiVdcGWoT5w1dUztNfMKkAd2hJX")
	reqUptime := uint64(3486)
	infoClient := info.NewClient(uri)
	networkID, err := infoClient.GetNetworkID(context.Background())
	if err != nil {
		log.Fatalf("failed to fetch network ID: %s\n", err)
	}

	validatorUptime, err := messages.NewValidatorUptime(validationID, reqUptime)
	if err != nil {
		log.Fatalf("failed to create validatorUptime message: %s\n", err)
	}

	addressedCall, err := payload.NewAddressedCall(
		nil,
		validatorUptime.Bytes(),
	)
	if err != nil {
		log.Fatalf("failed to create AddressedCall message: %s\n", err)
	}

	unsignedWarp, err := warp.NewUnsignedMessage(
		networkID,
		sourceChainID,
		addressedCall.Bytes(),
	)
	if err != nil {
		log.Fatalf("failed to create unsigned Warp message: %s\n", err)
	}

	p, err := peer.StartTestPeer(
		context.Background(),
		netip.AddrPortFrom(
			netip.AddrFrom4([4]byte{127, 0, 0, 1}),
			9651,
		),
		networkID,
		router.InboundHandlerFunc(func(_ context.Context, msg p2pmessage.InboundMessage) {
			log.Printf("received %s: %s", msg.Op(), msg.Message())
		}),
	)
	if err != nil {
		log.Fatalf("failed to start peer: %s\n", err)
	}

	messageBuilder, err := p2pmessage.NewCreator(
		prometheus.NewRegistry(),
		compression.TypeZstd,
		time.Hour,
	)
	if err != nil {
		log.Fatalf("failed to create message builder: %s\n", err)
	}

	appRequestPayload, err := proto.Marshal(&sdk.SignatureRequest{
		Message: unsignedWarp.Bytes(),
	})
	if err != nil {
		log.Fatalf("failed to marshal SignatureRequest: %s\n", err)
	}

	appRequest, err := messageBuilder.AppRequest(
		sourceChainID,
		0,
		time.Hour,
		p2p.PrefixMessage(
			p2p.ProtocolPrefix(p2p.SignatureRequestHandlerID),
			appRequestPayload,
		),
	)
	if err != nil {
		log.Fatalf("failed to create AppRequest: %s\n", err)
	}

	p.Send(context.Background(), appRequest)

	time.Sleep(5 * time.Second)

	p.StartClose()
	err = p.AwaitClosed(context.Background())
	if err != nil {
		log.Fatalf("failed to close peer: %s\n", err)
	}
}
