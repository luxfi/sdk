// Copyright (C) 2019-2024, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"math"

	"github.com/luxfi/node/codec"
	"github.com/luxfi/node/codec/linearcodec"
)

const (
	codecVersion = uint16(0)
)

var vdrCodec codec.Manager

func init() {
	vdrCodec = codec.NewManager(math.MaxInt32)
	c := linearcodec.NewDefault()

	err := c.RegisterType(validatorData{})
	if err != nil {
		panic(err)
	}
	err = vdrCodec.RegisterCodec(codecVersion, c)
	if err != nil {
		panic(err)
	}
}
