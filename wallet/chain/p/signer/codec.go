// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package signer

import (
	"github.com/luxfi/proto/p/txs"
	"github.com/luxfi/sdk/wallet/chain/p/pcodecs"
)

// Codec is the canonical PVM runtime wire codec used by the signer for
// marshaling unsigned/signed PVM txs. Constructed once at package
// init via pcodecs.NewPVMRuntimeCodec — proto/p carries no
// github.com/luxfi/codec import after Wave 2A (#101); the
// linearcodec-backed codec.Manager instance is built in pcodecs and
// pinned here so the signer stays free of inline codec construction
// at every Marshal call site.
var Codec txs.Codec

func init() {
	cm, err := pcodecs.NewPVMRuntimeCodec()
	if err != nil {
		panic(err)
	}
	Codec = cm
}
