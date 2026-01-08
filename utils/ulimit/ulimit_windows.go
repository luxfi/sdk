// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build windows
// +build windows

package ulimit

import "github.com/luxfi/log"

const DefaultFDLimit = 16384

// Set is a no-op for windows and will warn if the default is not used.
func Set(limit uint64, log log.Logger) error {
	if limit != DefaultFDLimit {
		log.Warn("fd-limit is not supported for windows")
	}
	return nil
}
