// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package wallet

import (
	"math"

	"github.com/luxfi/codec"
	"github.com/luxfi/codec/linearcodec"
	"github.com/luxfi/proto/p/warp"
	"github.com/luxfi/proto/p/warp/message"
	"github.com/luxfi/proto/p/warp/payload"
)

// WarpCodec is the proto/p/warp wire codec used by the backend visitor
// to parse the embedded warp.Message blob on RegisterL1Validator txs
// that the wallet observes from the chain.
//
// Constructed once at package init via newPVMWalletWarpCodec —
// proto/p carries no github.com/luxfi/codec import after Wave 2A
// (#101); the linearcodec-backed codec.Manager instance lives here so
// the backend visitor stays free of inline codec construction at the
// ParseMessage call site.
var WarpCodec warp.Codec

// PayloadCodec is the proto/p/warp/payload wire codec used by the
// backend visitor to parse the AddressedCall inside the warp.Message
// observed on RegisterL1Validator txs.
var PayloadCodec payload.Codec

// MessageCodec is the proto/p/warp/message wire codec used by the
// backend visitor to parse the RegisterL1Validator message inside the
// AddressedCall payload.
var MessageCodec message.Codec

// newPVMWalletWarpCodec constructs the linearcodec-backed proto/p/warp
// codec. warp.RegisterTypes seeds the canonical signature + teleport
// types in the wire-byte order required by proto/p/warp.
func newPVMWalletWarpCodec() (warp.Codec, error) {
	c := linearcodec.NewDefault()
	cm := codec.NewManager(math.MaxInt)
	if err := warp.RegisterTypes(c); err != nil {
		return nil, err
	}
	if err := cm.RegisterCodec(warp.CodecVersion, c); err != nil {
		return nil, err
	}
	return cm, nil
}

// newPVMWalletPayloadCodec constructs the linearcodec-backed
// proto/p/warp/payload codec. payload.RegisterTypes seeds the canonical
// AddressedCall + Hash payload types.
func newPVMWalletPayloadCodec() (payload.Codec, error) {
	c := linearcodec.NewDefault()
	cm := codec.NewManager(payload.MaxMessageSize)
	if err := payload.RegisterTypes(c); err != nil {
		return nil, err
	}
	if err := cm.RegisterCodec(payload.CodecVersion, c); err != nil {
		return nil, err
	}
	return cm, nil
}

// newPVMWalletMessageCodec constructs the linearcodec-backed
// proto/p/warp/message codec. message.RegisterTypes seeds the canonical
// ChainToL1Conversion / RegisterL1Validator / L1ValidatorRegistration /
// L1ValidatorWeight types.
func newPVMWalletMessageCodec() (message.Codec, error) {
	c := linearcodec.NewDefault()
	cm := codec.NewManager(math.MaxInt)
	if err := message.RegisterTypes(c); err != nil {
		return nil, err
	}
	if err := cm.RegisterCodec(message.CodecVersion, c); err != nil {
		return nil, err
	}
	return cm, nil
}

func init() {
	wcm, err := newPVMWalletWarpCodec()
	if err != nil {
		panic(err)
	}
	WarpCodec = wcm

	pcm, err := newPVMWalletPayloadCodec()
	if err != nil {
		panic(err)
	}
	PayloadCodec = pcm

	mcm, err := newPVMWalletMessageCodec()
	if err != nil {
		panic(err)
	}
	MessageCodec = mcm
}
