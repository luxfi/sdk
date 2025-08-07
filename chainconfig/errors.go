// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chainconfig

import "errors"

var (
	// ErrInvalidChainConfig indicates the chain configuration is invalid
	ErrInvalidChainConfig = errors.New("invalid chain configuration")

	// ErrMissingChainID indicates the chain ID is missing
	ErrMissingChainID = errors.New("chain ID is required")

	// ErrInvalidGasLimit indicates the gas limit is invalid
	ErrInvalidGasLimit = errors.New("gas limit must be greater than 0")

	// ErrInvalidPrecompile indicates a precompile configuration is invalid
	ErrInvalidPrecompile = errors.New("invalid precompile configuration")

	// ErrInvalidAllocation indicates an allocation is invalid
	ErrInvalidAllocation = errors.New("invalid allocation")
)
