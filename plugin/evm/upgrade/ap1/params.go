// (c) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ap1

import (
	"math/big"
	"time"
)

// ApricotPhase1MinBaseFee is the minimum base fee that can be used for blocks
// accepted in ApricotPhase1
var ApricotPhase1MinBaseFee = big.NewInt(225_000_000_000)

// ApricotPhase1BlockTimestamp is the timestamp of the first block that is subject to the Apricot Phase 1 rules
var ApricotPhase1BlockTimestamp = big.NewInt(time.Date(2021, time.March, 31, 14, 0, 0, 0, time.UTC).Unix())