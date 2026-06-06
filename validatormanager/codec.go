// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validatormanager

import (
	"math"

	"github.com/luxfi/codec"
	"github.com/luxfi/codec/linearcodec"
	"github.com/luxfi/proto/p/warp/payload"
)

// PayloadCodec is the proto/p/warp/payload wire codec used by
// validatormanager.RootInitiateValidatorRegistration to construct an
// AddressedCall payload around a ChainToL1Conversion message.
//
// Constructed once at package init via newPayloadCodec — proto/p
// carries no github.com/luxfi/codec import after Wave 2A (#101); the
// linearcodec-backed codec.Manager instance lives here so the call
// site stays free of inline codec construction.
var PayloadCodec payload.Codec

// newPayloadCodec constructs the linearcodec-backed proto/p/warp/payload
// codec. payload.RegisterTypes seeds the canonical AddressedCall + Hash
// payload types in the wire-byte order required by proto/p/warp/payload.
func newPayloadCodec() (payload.Codec, error) {
	c := linearcodec.NewDefault()
	cm := codec.NewManager(math.MaxInt)
	if err := payload.RegisterTypes(c); err != nil {
		return nil, err
	}
	if err := cm.RegisterCodec(payload.CodecVersion, c); err != nil {
		return nil, err
	}
	return cm, nil
}

func init() {
	cm, err := newPayloadCodec()
	if err != nil {
		panic(err)
	}
	PayloadCodec = cm
}
