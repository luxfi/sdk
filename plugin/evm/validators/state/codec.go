// Copyright (C) 2019-2024, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"math"

	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
)

const (
	codecVersion = uint16(0)
)

var vdrCodec interfaces.Codec

func init() {
	vdrCodec = interfaces.NewManager(interfaces.MaxInt32)
	c := linearinterfaces.NewDefault()

	errs := interfaces.Errs{}
	errs.Add(
		c.RegisterType(validatorData{}),

		vdrCodec.RegisterCodec(codecVersion, c),
	)

	if errs.Errored() {
		panic(errs.Err)
	}
}
