// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package wallet

import (
	"github.com/luxfi/proto/p/warp"
	"github.com/luxfi/proto/p/warp/message"
	"github.com/luxfi/proto/p/warp/payload"
	"github.com/luxfi/sdk/wallet/chain/p/pcodecs"
)

// WarpCodec is the proto/p/warp wire codec used by the backend visitor
// to parse the embedded warp.Message blob on RegisterL1Validator txs
// that the wallet observes from the chain.
//
// Constructed once at package init via pcodecs.NewWarpCodec —
// proto/p carries no github.com/luxfi/codec import after Wave 2A
// (#101); the linearcodec-backed codec.Manager instance is built in
// pcodecs and pinned here so the backend visitor stays free of inline
// codec construction at the ParseMessage call site.
var WarpCodec warp.Codec

// PayloadCodec is the proto/p/warp/payload wire codec used by the
// backend visitor to parse the AddressedCall inside the warp.Message
// observed on RegisterL1Validator txs.
var PayloadCodec payload.Codec

// MessageCodec is the proto/p/warp/message wire codec used by the
// backend visitor to parse the RegisterL1Validator message inside the
// AddressedCall payload.
var MessageCodec message.Codec

func init() {
	wcm, err := pcodecs.NewWarpCodec()
	if err != nil {
		panic(err)
	}
	WarpCodec = wcm

	pcm, err := pcodecs.NewPayloadCodec()
	if err != nil {
		panic(err)
	}
	PayloadCodec = pcm

	mcm, err := pcodecs.NewMessageCodec()
	if err != nil {
		panic(err)
	}
	MessageCodec = mcm
}
