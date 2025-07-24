// Copyright (C) 2019-2024, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package params

// Lux-specific gas costs for precompiled contracts
const (
	// AssetBalanceApricot is the gas cost for querying native asset balance
	AssetBalanceApricot uint64 = 2474

	// AssetCallApricot is the gas cost for calling native asset transfer
	AssetCallApricot uint64 = 9000
)
