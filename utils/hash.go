// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"crypto/sha256"
	
	"github.com/luxfi/evm/interfaces"
)

// ComputeHash256Array computes SHA256 hash and returns it as an ID
func ComputeHash256Array(data []byte) interfaces.ID {
	hash := sha256.Sum256(data)
	var id interfaces.ID
	copy(id[:], hash[:])
	return id
}

// ComputeHash256 computes SHA256 hash and returns it as bytes
func ComputeHash256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}