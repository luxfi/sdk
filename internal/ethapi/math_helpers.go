// (c) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ethapi

import (
	"math"
	"math/big"
)

// BigMin returns the smaller of two big.Int values
func BigMin(a, b *big.Int) *big.Int {
	if a.Cmp(b) < 0 {
		return new(big.Int).Set(a)
	}
	return new(big.Int).Set(b)
}

// BigMax returns the larger of two big.Int values
func BigMax(a, b *big.Int) *big.Int {
	if a.Cmp(b) > 0 {
		return new(big.Int).Set(a)
	}
	return new(big.Int).Set(b)
}

// MaxInt64 is the maximum value for int64
const MaxInt64 = math.MaxInt64