// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package signer

import (
	"github.com/luxfi/codec"
	"github.com/luxfi/codec/linearcodec"
	"github.com/luxfi/proto/p/block"
	"github.com/luxfi/proto/p/txs"
)

// Codec is the canonical PVM runtime wire codec used by the signer for
// marshaling unsigned/signed PVM txs. Constructed once at package init
// via newPVMSignerCodec — proto/p carries no github.com/luxfi/codec
// import after Wave 2A (#101); the linearcodec-backed codec.Manager
// instance lives here so the signer stays free of inline codec
// construction at every Marshal call site.
//
// Mirrors sdk/wallet/chain/x/builder/constants.go::Parser.Codec() —
// each subpackage owns the wire codec it needs to keep the production
// API contract one-canonical-helper-per-need.
var Codec txs.Codec

// newPVMSignerCodec constructs the linearcodec-backed PVM runtime tx
// codec used by the signer. block.RegisterTypes pulls in the full
// Apricot/Banff/Durango/Quasar block + tx type set in the canonical
// wire-byte order required by proto/p.
func newPVMSignerCodec() (txs.Codec, error) {
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
	cm, err := newPVMSignerCodec()
	if err != nil {
		panic(err)
	}
	Codec = cm
}
