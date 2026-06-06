// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"math"

	"github.com/luxfi/codec"
	"github.com/luxfi/codec/linearcodec"
	"github.com/luxfi/proto/p/warp"
)

// WarpCodec is the proto/p/warp wire codec used by fee.WarpComplexity
// when computing dynamic-fee gas dimensions for L1-validator txs
// (RegisterL1Validator / SetL1ValidatorWeight / IncreaseL1ValidatorBalance
// / DisableL1Validator) whose embedded warp message contributes to
// transaction complexity.
//
// Constructed once at package init via newPVMWarpCodec — proto/p
// carries no github.com/luxfi/codec import after Wave 2A (#101); the
// linearcodec-backed codec.Manager instance lives here so the builder
// stays free of inline codec construction at every WarpComplexity call
// site.
var WarpCodec warp.Codec

// newPVMWarpCodec constructs the linearcodec-backed proto/p/warp
// codec. warp.RegisterTypes seeds the canonical signature + teleport
// types in the wire-byte order required by proto/p/warp.
func newPVMWarpCodec() (warp.Codec, error) {
	c := linearcodec.NewDefault()
	cm := codec.NewManager(math.MaxInt)
	if err := warp.RegisterTypes(c); err != nil {
		return nil, err
	}
	if err := cm.RegisterCodec(warp.CodecVersion, c); err != nil {
		return nil, err
	}
	return cm, nil
}

func init() {
	cm, err := newPVMWarpCodec()
	if err != nil {
		panic(err)
	}
	WarpCodec = cm
}
