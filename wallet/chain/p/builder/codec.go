// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"github.com/luxfi/proto/p/warp"
	"github.com/luxfi/sdk/wallet/chain/p/pcodecs"
)

// WarpCodec is the proto/p/warp wire codec used by fee.WarpComplexity
// when computing dynamic-fee gas dimensions for L1-validator txs
// (RegisterL1Validator / SetL1ValidatorWeight / IncreaseL1ValidatorBalance
// / DisableL1Validator) whose embedded warp message contributes to
// transaction complexity.
//
// Constructed once at package init via pcodecs.NewWarpCodec —
// proto/p carries no github.com/luxfi/codec import after Wave 2A
// (#101); the linearcodec-backed codec.Manager instance is built in
// pcodecs and pinned here so the builder stays free of inline codec
// construction at every WarpComplexity call site.
var WarpCodec warp.Codec

func init() {
	cm, err := pcodecs.NewWarpCodec()
	if err != nil {
		panic(err)
	}
	WarpCodec = cm
}
