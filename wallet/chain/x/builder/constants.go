// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"math"

	"github.com/luxfi/codec"
	"github.com/luxfi/codec/linearcodec"
	"github.com/luxfi/proto/x/block"
	"github.com/luxfi/proto/x/fxs"
	xtxs "github.com/luxfi/proto/x/txs"
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
// codecs. proto/x carries no github.com/luxfi/codec import after the
// Wave 1A rip (#101); construction of the linearcodec-backed managers
// lives in the SDK so consumers can pick their codec implementation
// (linearcodec today, zapcodec in a future wave) without touching
// proto/x.
//
// STRUCTURAL KEEP (Wave 2E, #101). Mirror of
// sdk/wallet/chain/x/constants.go for the builder package. The
// luxfi/codec + luxfi/codec/linearcodec imports below are deliberate
// and remain after the codec rip — they are the place where the wire
// codec implementation is chosen and bound. proto/x's ParserCodecs is
// a structural seam (four local interfaces, zero codec import) and
// the wiring lives in exactly one place per consumer. The two
// constants.go copies (this file + ../constants.go) differ only in
// package name; they are kept in lockstep by hand because the wallet
// builder and the wallet itself are distinct consumers that may
// diverge if one migrates off linearcodec ahead of the other.
//
// Tx-level and fx-owned wire payload types are registered when this
// bundle is handed to txs.NewParser — see parser.go fxOwnedTypes.
func newXVMParserCodecs() (xtxs.ParserCodecs, error) {
	c := linearcodec.NewDefault()
	gc := linearcodec.NewDefault()
	cm := codec.NewDefaultManager()
	gcm := codec.NewManager(math.MaxInt32)
	if err := cm.RegisterCodec(xtxs.CodecVersion, c); err != nil {
		return xtxs.ParserCodecs{}, err
	}
	if err := gcm.RegisterCodec(xtxs.CodecVersion, gc); err != nil {
		return xtxs.ParserCodecs{}, err
	}
	return xtxs.ParserCodecs{
		Codec:           cm,
		GenesisCodec:    gcm,
		Registry:        c,
		GenesisRegistry: gc,
	}, nil
}

func init() {
	codecs, err := newXVMParserCodecs()
	if err != nil {
		panic(err)
	}
	Parser, err = block.NewParser(
		codecs,
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
