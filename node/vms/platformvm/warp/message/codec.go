// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package message

import (
	"errors"
	"math"

	"github.com/luxfi/sdk/node/codec"
	"github.com/luxfi/sdk/node/codec/linearcodec"
)

const CodecVersion = 0

var Codec codec.Manager

func init() {
	Codec = codec.NewManager(math.MaxInt)
	lc := linearcodec.NewDefault()

	err := errors.Join(
		lc.RegisterType(&ChainToL1Conversion{}),
		lc.RegisterType(&RegisterL1Validator{}),
		lc.RegisterType(&L1ValidatorRegistration{}),
		lc.RegisterType(&L1ValidatorWeight{}),
		Codec.RegisterCodec(CodecVersion, lc),
	)
	if err != nil {
		panic(err)
	}
}
