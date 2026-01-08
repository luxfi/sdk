// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"log"
	"net/netip"
	"time"

	"github.com/luxfi/metric"
	"google.golang.org/protobuf/proto"

	"github.com/luxfi/const"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/api/info"
	p2psdk "github.com/luxfi/p2p"
	"github.com/luxfi/p2p/peer"
	"github.com/luxfi/sdk/node/proto/pb/platformvm"
	"github.com/luxfi/sdk/node/proto/pb/sdk"
	"github.com/luxfi/sdk/utils/compression"
	"github.com/luxfi/sdk/node/vms/platformvm/warp"
	"github.com/luxfi/sdk/node/vms/platformvm/warp/payload"
	"github.com/luxfi/sdk/node/wallet/net/primary"

	p2pmessage "github.com/luxfi/p2p/message"
	warpmessage "github.com/luxfi/sdk/node/vms/platformvm/warp/message"
)

type simpleInboundHandler struct{}

func (h *simpleInboundHandler) HandleInbound(_ context.Context, msg p2pmessage.InboundMessage) {
	log.Printf("received %s: %s", msg.Op(), msg.Message())
}

func main() {
	uri := primary.LocalAPIURI
	netID := ids.FromStringOrPanic("2DeHa7Qb6sufPkmQcFWG2uCd4pBPv9WB6dkzroiMQhd1NSRtof")
	validationIndex := uint32(0)
	infoClient := info.NewClient(uri)
	networkID, err := infoClient.GetNetworkID(context.Background())
	if err != nil {
		log.Fatalf("failed to fetch network ID: %s\n", err)
	}

	validationID := netID.Append(validationIndex)
	l1ValidatorRegistration, err := warpmessage.NewL1ValidatorRegistration(
		validationID,
		false,
	)
	if err != nil {
		log.Fatalf("failed to create L1ValidatorRegistration message: %s\n", err)
	}

	addressedCall, err := payload.NewAddressedCall(
		nil,
		l1ValidatorRegistration.Bytes(),
	)
	if err != nil {
		log.Fatalf("failed to create AddressedCall message: %s\n", err)
	}

	unsignedWarp, err := warp.NewUnsignedMessage(
		networkID,
		constants.PlatformChainID,
		addressedCall.Bytes(),
	)
	if err != nil {
		log.Fatalf("failed to create unsigned Warp message: %s\n", err)
	}

	justification := platformvm.L1ValidatorRegistrationJustification{
		Preimage: &platformvm.L1ValidatorRegistrationJustification_ConvertChainToL1TxData{
			ConvertChainToL1TxData: &platformvm.NetIDIndex{
				NetId: netID[:],
				Index: validationIndex,
			},
		},
	}
	// TODO: Fix proto marshal issue with L1ValidatorRegistrationJustification
	// justificationBytes, err := proto.Marshal(&justification)
	// if err != nil {
	// 	log.Fatalf("failed to create justification: %s\n", err)
	// }
	justificationBytes := []byte{} // placeholder
	_ = justification

	p, err := peer.StartTestPeer(
		context.Background(),
		netip.AddrPortFrom(
			netip.AddrFrom4([4]byte{127, 0, 0, 1}),
			9651,
		),
		networkID,
		&simpleInboundHandler{},
	)
	if err != nil {
		log.Fatalf("failed to start peer: %s\n", err)
	}

	messageBuilder, err := p2pmessage.NewCreator(

		metric.NewNoOpRegistry(),
		compression.TypeZstd,
		time.Hour,
	)
	if err != nil {
		log.Fatalf("failed to create message builder: %s\n", err)
	}

	appRequestPayload, err := proto.Marshal(&sdk.SignatureRequest{
		Message:       unsignedWarp.Bytes(),
		Justification: justificationBytes,
	})
	if err != nil {
		log.Fatalf("failed to marshal SignatureRequest: %s\n", err)
	}

	appRequest, err := messageBuilder.AppRequest(
		constants.PlatformChainID,
		0,
		time.Hour,
		p2psdk.PrefixMessage(
			p2psdk.ProtocolPrefix(0), // SignatureRequestHandlerID placeholder
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
