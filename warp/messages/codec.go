// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package messages

import (
	"errors"

	"github.com/luxfi/luxd/codec"
	"github.com/luxfi/luxd/codec/linearcodec"
	"github.com/luxfi/luxd/utils/units"
)

const (
	CodecVersion = 0

	MaxMessageSize = 24 * units.KiB
)

var Codec codec.Manager

func init() {
	Codec = codec.NewManager(MaxMessageSize)
	lc := linearcodec.NewDefault()

	err := errors.Join(
		lc.RegisterType(&ValidatorUptime{}),
		Codec.RegisterCodec(CodecVersion, lc),
	)
	if err != nil {
		panic(err)
	}
}
