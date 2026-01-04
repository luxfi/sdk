// Copyright (C) 2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

// Package flags provides command-line flag utilities.
package flags

func EnsureMutuallyExclusive(flags []bool) bool {
	set := 0
	for _, f := range flags {
		if !f {
			continue
		}
		set++
		if set > 1 {
			return false
		}
	}

	return true
}
