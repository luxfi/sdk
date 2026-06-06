// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validatormanager

import (
	"github.com/luxfi/proto/p/warp/payload"
	"github.com/luxfi/sdk/wallet/chain/p/pcodecs"
)

// PayloadCodec is the proto/p/warp/payload wire codec used by
// validatormanager.RootInitiateValidatorRegistration to construct an
// AddressedCall payload around a ChainToL1Conversion message.
//
// Constructed once at package init via pcodecs.NewPayloadCodec —
// proto/p carries no github.com/luxfi/codec import after Wave 2A
// (#101); the linearcodec-backed codec.Manager instance is built in
// pcodecs and pinned here so the call site stays free of inline
// codec construction.
var PayloadCodec payload.Codec

func init() {
	cm, err := pcodecs.NewPayloadCodec()
	if err != nil {
		panic(err)
	}
	PayloadCodec = cm
}
