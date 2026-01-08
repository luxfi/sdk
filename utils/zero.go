// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

// Zero returns the zero value of any type T.
func Zero[T any]() T {
	var zero T
	return zero
}
