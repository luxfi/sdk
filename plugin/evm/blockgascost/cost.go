// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// blockgascost implements the block gas cost logic
package blockgascost

import (
	"math"

	"github.com/luxfi/evm/commontype"
	safemath "github.com/luxfi/node/utils/math"
)

// BlockGasCost calculates the required block gas cost.
//
// cost = parentCost + step * (TargetBlockRate - timeElapsed)
//
// The returned cost is clamped to [MinBlockGasCost, MaxBlockGasCost].
func BlockGasCost(
	feeConfig commontype.FeeConfig,
	parentCost uint64,
	step uint64,
	timeElapsed uint64,
) uint64 {
	deviation := safemath.AbsDiff(feeConfig.TargetBlockRate, timeElapsed)
	change, err := safemath.Mul64(step, deviation)
	if err != nil {
		change = math.MaxUint64
	}

	var (
		minBlockGasCost = feeConfig.MinBlockGasCost.Uint64()
		maxBlockGasCost = feeConfig.MaxBlockGasCost.Uint64()
	)

	var cost uint64
	if timeElapsed > feeConfig.TargetBlockRate {
		cost, err = safemath.Sub(parentCost, change)
		if err != nil {
			cost = minBlockGasCost
		}
	} else {
		cost, err = safemath.Add64(parentCost, change)
		if err != nil {
			cost = maxBlockGasCost
		}
	}

	switch {
	case cost < minBlockGasCost:
		// This is technically dead code because [MinBlockGasCost] is 0, but it
		// makes the code more clear.
		return minBlockGasCost
	case cost > maxBlockGasCost:
		return maxBlockGasCost
	default:
		return cost
	}
}
