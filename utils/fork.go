// (c) 2024 Hanzo Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import "math/big"

// IsBlockForked returns whether a fork scheduled at block s is active at the
// given head block. Whilst this method is the same as isTimestampForked, they
// are explicitly separate for clearer reading.
func IsBlockForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}
	return s.Cmp(head) <= 0
}

// isTimestampForked returns whether a fork scheduled at timestamp s is active
// at the given head timestamp. Whilst this method is the same as isBlockForked,
// they are explicitly separate for clearer reading.
func isTimestampForked(s *uint64, head uint64) bool {
	if s == nil {
		return false
	}
	return *s <= head
}

// IsTimestampForked returns whether a fork scheduled at timestamp s is active  
// at the given head timestamp.
func IsTimestampForked(s *uint64, head uint64) bool {
	return isTimestampForked(s, head)
}