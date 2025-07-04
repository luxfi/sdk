// Copyright (C) 2019-2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"math"

	"github.com/luxdefi/node/codec"
	"github.com/luxdefi/node/codec/linearcodec"
	"github.com/luxdefi/node/utils/wrappers"
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
