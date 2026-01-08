// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package reward

import (
	"fmt"
	"time"

	"github.com/luxfi/sdk/utils/units"
)

func ExampleNewCalculator() {
	const (
		day             = 24 * time.Hour
		week            = 7 * day
		stakingDuration = 4 * week

		stakeAmount = 100_000 * units.Lux // 100k LUX

		// The current supply can be fetched with the platform.getCurrentSupply API
		// With 6 decimal precision, values are in microLUX (μLUX)
		currentSupply = 447_903_490 * units.Lux // ~448m LUX
	)
	var (
		mainnetRewardConfig = Config{
			MaxConsumptionRate: .12 * PercentDenominator,
			MinConsumptionRate: .10 * PercentDenominator,
			MintingPeriod:      365 * 24 * time.Hour,
			SupplyCap:          720 * units.MegaLux,
		}
		mainnetCalculator = NewCalculator(mainnetRewardConfig)
	)

	potentialReward := mainnetCalculator.Calculate(stakingDuration, stakeAmount, currentSupply)

	fmt.Printf("Staking %d μLUX for %s with the current supply of %d μLUX would have a potential reward of %d μLUX",
		stakeAmount,
		stakingDuration,
		currentSupply,
		potentialReward,
	)
	// Output: Staking 100000000000 μLUX for 672h0m0s with the current supply of 447903490000000 μLUX would have a potential reward of 473168954 μLUX
}
