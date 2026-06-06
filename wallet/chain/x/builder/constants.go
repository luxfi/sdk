// Copyright (C) 2019-2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"github.com/luxfi/proto/x/block"
	"github.com/luxfi/proto/x/fxs"
	xtxs "github.com/luxfi/proto/x/txs"
	"github.com/luxfi/proto/zap_codec"
	"github.com/luxfi/utxo/nftfx"
	"github.com/luxfi/utxo/propertyfx"
	"github.com/luxfi/utxo/secp256k1fx"
)

const (
	SECP256K1FxIndex = 0
	NFTFxIndex       = 1
	PropertyFxIndex  = 2
)

// Parser to support serialization and deserialization
var Parser block.Parser

// newXVMParserCodecs is the canonical wiring for the proto/x XVM wire
// codecs in the builder package. proto/x carries no
// github.com/luxfi/codec import after the Wave 1A rip (#101);
// construction is delegated to proto/zap_codec — the single canonical
// site where the wire codec backend is chosen.
//
// Wave 2G-Wallet (#101): the linearcodec/codec.Manager construction
// that previously lived here is gone. proto/zap_codec.NewXVMParser
// returns ZAP-native little-endian codec managers — see that package's
// doc comment for the wire-format break vs the legacy linearcodec wire
// bytes (LP-023 ZAP-native activation, forward-only).
//
// This file mirrors sdk/wallet/chain/x/constants.go — the two consumers
// (wallet vs wallet-builder) intentionally share ONE construction
// expression via proto/zap_codec so the wire codec is bound in EXACTLY
// ONE place.
//
// Tx-level and fx-owned wire payload types are registered when this
// bundle is handed to txs.NewParser — see parser.go fxOwnedTypes.
func newXVMParserCodecs() xtxs.ParserCodecs {
	runtime, genesis := zap_codec.NewXVMParser(xtxs.CodecVersion)
	return xtxs.ParserCodecs{
		Codec:           runtime,
		GenesisCodec:    genesis,
		Registry:        runtime,
		GenesisRegistry: genesis,
	}
}

func init() {
	var err error
	Parser, err = block.NewParser(
		newXVMParserCodecs(),
		[]fxs.Fx{
			&secp256k1fx.Fx{},
			&nftfx.Fx{},
			&propertyfx.Fx{},
		},
	)
	if err != nil {
		panic(err)
	}
}
