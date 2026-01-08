// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"errors"
	"math"

	"github.com/luxfi/sdk/node/codec"
	"github.com/luxfi/sdk/node/codec/linearcodec"
)

const CodecVersion = 0

var Codec codec.Manager

func init() {
	Codec = codec.NewManager(math.MaxInt)
	lc := linearcodec.NewDefault()

	err := errors.Join(
		// Warp 1.0: Classical BLS signatures
		lc.RegisterType(&BitSetSignature{}),
		// Warp 1.5: Quantum-safe signatures
		lc.RegisterType(&CoronaSignature{}),         // Recommended: RT-only (LWE-based threshold)
		lc.RegisterType(&EncryptedWarpPayload{}),      // ML-KEM + AES-256-GCM encryption
		lc.RegisterType(&HybridBLSRTSignature{}),      // Deprecated: BLS+RT hybrid
		// Teleport: Cross-chain bridging protocol
		lc.RegisterType(&TeleportMessage{}),           // High-level bridge message wrapper
		lc.RegisterType(&TeleportTransferPayload{}),   // Asset transfer payload
		lc.RegisterType(&TeleportAttestPayload{}),     // Attestation payload
		Codec.RegisterCodec(CodecVersion, lc),
	)
	if err != nil {
		panic(err)
	}
}
