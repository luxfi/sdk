// (c) 2019-2024, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package types

// MergeBloom merges the blooms from the given receipts into a single bloom.
func MergeBloom(receipts []*Receipt) Bloom {
	var bloom Bloom
	for _, receipt := range receipts {
		// Bloom is a fixed-size byte array, so we need to OR manually
		for i := 0; i < len(bloom); i++ {
			bloom[i] |= receipt.Bloom[i]
		}
	}
	return bloom
}