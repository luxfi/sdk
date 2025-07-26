// (c) 2019-2020, Lux Industries, Inc.
// All rights reserved.
// See the file LICENSE for licensing terms.

package headerutil

import (
	"github.com/luxfi/evm/params"
)

// WindowSize is the size of the rolling window
const WindowSize = 32

// PredicateBytesFromExtra returns the predicate bytes from the extra field.
func PredicateBytesFromExtra(rules params.Rules, extra []byte) []byte {
	// For now, always process if we have enough data
	// This check can be refined based on specific chain rules
	
	offset := WindowSize
	// Prior to Durango, the VM enforces the extra data is smaller than or equal
	// to `offset`.
	// After Durango, the VM pre-verifies the extra data past `offset` is valid.
	if len(extra) <= offset {
		return nil
	}
	return extra[offset:]
}

// SetPredicateBytesInExtra sets the predicate bytes in the extra field.
func SetPredicateBytesInExtra(extra []byte, predicateBytes []byte) []byte {
	// Ensure extra is at least WindowSize long
	if len(extra) < WindowSize {
		newExtra := make([]byte, WindowSize)
		copy(newExtra, extra)
		extra = newExtra
	}
	
	// Truncate extra to WindowSize and append predicate bytes
	result := make([]byte, WindowSize+len(predicateBytes))
	copy(result, extra[:WindowSize])
	copy(result[WindowSize:], predicateBytes)
	return result
}