// (c) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ap4

import (
	"math/big"
	"time"
)

// ApricotPhase4MinBaseFee is the minimum base fee that can be used for blocks
// accepted in ApricotPhase4
var ApricotPhase4MinBaseFee = big.NewInt(25_000_000_000)

// ApricotPhase4GasLimit is the gas limit that can be used for blocks
// accepted in ApricotPhase4
var ApricotPhase4GasLimit = big.NewInt(8_000_000)

// ApricotPhase4BlockTimestamp is the timestamp of the first block that is subject to the Apricot Phase 4 rules
var ApricotPhase4BlockTimestamp = big.NewInt(time.Date(2021, time.November, 24, 15, 0, 0, 0, time.UTC).Unix())