// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package types

import (
	"crypto/rand"
	"encoding/hex"
)

// ID represents a 32-byte identifier
type ID [32]byte

// NodeID represents a node identifier
type NodeID [20]byte

// ShortID represents a 20-byte identifier
type ShortID [20]byte

// Empty returns an empty ID
var Empty = ID{}

// String returns the string representation of an ID
func (id ID) String() string {
	return hex.EncodeToString(id[:])
}

// String returns the string representation of a NodeID
func (id NodeID) String() string {
	return "NodeID-" + hex.EncodeToString(id[:])
}

// String returns the string representation of a ShortID
func (id ShortID) String() string {
	return hex.EncodeToString(id[:])
}

// GenerateTestID generates a random ID for testing
func GenerateTestID() ID {
	var id ID
	rand.Read(id[:])
	return id
}

// GenerateTestNodeID generates a random NodeID for testing
func GenerateTestNodeID() NodeID {
	var id NodeID
	rand.Read(id[:])
	return id
}

// ShortFromString creates a ShortID from a string
func ShortFromString(s string) (ShortID, error) {
	var id ShortID
	// Simple implementation for testing
	copy(id[:], []byte(s))
	return id, nil
}

// NodeIDFromString creates a NodeID from a string
func NodeIDFromString(s string) (NodeID, error) {
	var id NodeID
	// Simple implementation for testing
	if len(s) > 7 && s[:7] == "NodeID-" {
		s = s[7:]
	}
	decoded, err := hex.DecodeString(s)
	if err != nil {
		return id, err
	}
	copy(id[:], decoded)
	return id, nil
}