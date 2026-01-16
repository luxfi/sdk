// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validatormanager

func pointersSlice[T any](values []T) []*T {
	pointers := make([]*T, len(values))
	for i := range values {
		pointers[i] = &values[i]
	}
	return pointers
}
