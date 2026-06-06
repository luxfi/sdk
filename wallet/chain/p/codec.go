// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package p

import (
	"github.com/luxfi/codec"
	"github.com/luxfi/codec/linearcodec"
	"github.com/luxfi/proto/p/block"
	"github.com/luxfi/proto/p/txs"
)

// Codec is the canonical PVM runtime wire codec used by the top-level
// p package's signer for marshaling unsigned/signed PVM txs.
// Constructed once at package init via newPVMCodec — proto/p carries
// no github.com/luxfi/codec import after Wave 2A (#101); the
// linearcodec-backed codec.Manager instance lives here so the package
// stays free of inline codec construction at every Marshal call site.
//
// Mirrors sdk/wallet/chain/p/signer/codec.go::Codec — both subpackages
// expose the same logical PVM runtime codec; production code uses the
// subpackage signer/, the top-level signer_visitor.go is retained for
// historical wiring and shares the same codec layout.
var Codec txs.Codec

// newPVMCodec constructs the linearcodec-backed PVM runtime tx codec.
// block.RegisterTypes pulls in the full Apricot/Banff/Durango/Quasar
// block + tx type set in the canonical wire-byte order required by
// proto/p.
func newPVMCodec() (txs.Codec, error) {
	c := linearcodec.NewDefault()
	cm := codec.NewDefaultManager()
	if err := block.RegisterTypes(c); err != nil {
		return nil, err
	}
	if err := cm.RegisterCodec(txs.CodecVersion, c); err != nil {
		return nil, err
	}
	return cm, nil
}

func init() {
	cm, err := newPVMCodec()
	if err != nil {
		panic(err)
	}
	Codec = cm
}
