// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package keystore

import (
	"github.com/luxfi/sdk/node/codec"
	"github.com/luxfi/sdk/node/codec/linearcodec"
	"github.com/luxfi/sdk/utils/units"
)

const (
	CodecVersion = 0

	maxPackerSize = 1 * units.GiB // max size, in bytes, of something being marshalled by Marshal()
)

var Codec codec.Manager

func init() {
	lc := linearcodec.NewDefault()
	Codec = codec.NewManager(maxPackerSize)
	if err := Codec.RegisterCodec(CodecVersion, lc); err != nil {
		panic(err)
	}
}
