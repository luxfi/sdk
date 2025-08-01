// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"math"

	"github.com/luxfi/luxd/codec"
	"github.com/luxfi/luxd/codec/linearcodec"
	"github.com/luxfi/luxd/utils/wrappers"
)

const (
	codecVersion = uint16(0)
)

var vdrCodec codec.Manager

func init() {
	vdrCodec = codec.NewManager(math.MaxInt32)
	c := linearcodec.NewDefault()

	errs := wrappers.Errs{}
	errs.Add(
		c.RegisterType(validatorData{}),

		vdrCodec.RegisterCodec(codecVersion, c),
	)

	if errs.Errored() {
		panic(errs.Err)
	}
}
