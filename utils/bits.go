// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"github.com/luxfi/evm/interfaces"
)

// BitSet implements interfaces.Bits
type BitSet struct {
	bits map[int]bool
}

// NewBits creates a new bit set
func NewBits() interfaces.Bits {
	return &BitSet{
		bits: make(map[int]bool),
	}
}

// Add adds a bit to the set
func (b *BitSet) Add(i int) {
	b.bits[i] = true
}

// Contains checks if a bit is in the set
func (b *BitSet) Contains(i int) bool {
	return b.bits[i]
}

// Remove removes a bit from the set
func (b *BitSet) Remove(i int) {
	delete(b.bits, i)
}

// Clear clears all bits
func (b *BitSet) Clear() {
	b.bits = make(map[int]bool)
}

// Len returns the number of bits set
func (b *BitSet) Len() int {
	return len(b.bits)
}

// Bytes returns the byte representation
func (b *BitSet) Bytes() []byte {
	if len(b.bits) == 0 {
		return []byte{}
	}
	
	// Find the maximum bit index
	max := 0
	for i := range b.bits {
		if i > max {
			max = i
		}
	}
	
	// Calculate the number of bytes needed
	numBytes := (max / 8) + 1
	bytes := make([]byte, numBytes)
	
	// Set the bits
	for i := range b.bits {
		byteIndex := i / 8
		bitIndex := uint(i % 8)
		bytes[byteIndex] |= 1 << bitIndex
	}
	
	return bytes
}