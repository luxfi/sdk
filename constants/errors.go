// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package constants

import "errors"

var (
	// Network errors
	ErrInvalidNetworkID = errors.New("invalid network ID")
	
	// Chain errors
	ErrUnknownChain = errors.New("unknown chain")
	ErrInvalidChainID = errors.New("invalid chain ID")
	
	// Configuration errors
	ErrInvalidConfiguration = errors.New("invalid configuration")
	ErrMissingConfiguration = errors.New("missing configuration")
)