// (c) 2019-2024, Lux Industries, Inc.
// All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"math"
	"math/big"
)

// BigMax returns the larger of two big.Int values
func BigMax(a, b *big.Int) *big.Int {
	if a.Cmp(b) > 0 {
		return a
	}
	return b
}

// MaxUint64 is the maximum value for a uint64
const MaxUint64 = math.MaxUint64