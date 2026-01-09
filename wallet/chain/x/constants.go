// Copyright (C) 2019-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package x

import (
	"github.com/luxfi/vm/vms/exchangevm/block"
	"github.com/luxfi/vm/vms/exchangevm/fxs"
	"github.com/luxfi/vm/nftfx"
	"github.com/luxfi/vm/propertyfx"
	"github.com/luxfi/vm/secp256k1fx"
)

const (
	SECP256K1FxIndex = 0
	NFTFxIndex       = 1
	PropertyFxIndex  = 2
)

// Parser to support serialization and deserialization
var Parser block.Parser

func init() {
	var err error
	Parser, err = block.NewParser([]fxs.Fx{
		&secp256k1fx.Fx{},
		&nftfx.Fx{},
		&propertyfx.Fx{},
	})
	if err != nil {
		panic(err)
	}
}
