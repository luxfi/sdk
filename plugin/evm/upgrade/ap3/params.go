// (c) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ap3

import (
	"math/big"
	"time"
)

// ApricotPhase3MinBaseFee is the minimum base fee that can be used for blocks
// accepted in ApricotPhase3
var ApricotPhase3MinBaseFee = big.NewInt(75_000_000_000)

// ApricotPhase3InitialBaseFee is the initial base fee that can be used for blocks
// accepted in ApricotPhase3
var ApricotPhase3InitialBaseFee = big.NewInt(225_000_000_000)

// ApricotPhase3BlockTimestamp is the timestamp of the first block that is subject to the Apricot Phase 3 rules
var ApricotPhase3BlockTimestamp = big.NewInt(time.Date(2021, time.August, 24, 14, 0, 0, 0, time.UTC).Unix())