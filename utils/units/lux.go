// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package units

// Denominations of value
// LUX uses 6 decimals (like USDC), allowing max supply of ~18.4 trillion LUX in uint64
const (
	MicroLux uint64 = 1               // Base unit (6 decimals) - 0.000001 LUX
	MilliLux uint64 = 1000 * MicroLux // 0.001 LUX
	Lux      uint64 = 1000 * MilliLux // 1 LUX = 10^6 microLUX
	KiloLux  uint64 = 1000 * Lux      // 1,000 LUX
	MegaLux  uint64 = 1000 * KiloLux  // 1,000,000 LUX
	GigaLux  uint64 = 1000 * MegaLux  // 1,000,000,000 LUX (1 billion)
	TeraLux  uint64 = 1000 * GigaLux  // 1,000,000,000,000 LUX (1 trillion)

	// Schmeckle preserved for compatibility (≈49.463 milliLUX)
	Schmeckle uint64 = 49*MilliLux + 463*MicroLux

	// NanoLux deprecated - use MicroLux as base unit
	// Kept for backward compatibility but represents same as MicroLux
	NanoLux uint64 = MicroLux
)
