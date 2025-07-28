// (c) 2023, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package iface

import (
	"github.com/luxfi/geth/common"
)

// UnsignedMessage represents an unsigned warp message
type UnsignedMessage struct {
	NetworkID     uint32
	SourceChainID common.Hash
	Payload       []byte
}

// ID returns the hash of the unsigned message
func (m *UnsignedMessage) ID() common.Hash {
	// TODO: Implement proper hashing
	return common.Hash{}
}

// Bytes returns the byte representation of the unsigned message
func (m *UnsignedMessage) Bytes() []byte {
	// TODO: Implement proper serialization
	return append(append([]byte{}, m.SourceChainID[:]...), m.Payload...)
}

// WarpSignedMessage represents a signed warp message
type WarpSignedMessage struct {
	UnsignedMessage UnsignedMessage
	Signature       *BLSSignature
	SourceChainID   common.Hash
}

// ID returns the ID of the unsigned message
func (m *WarpSignedMessage) ID() common.Hash {
	return m.UnsignedMessage.ID()
}

// Payload returns the payload from the unsigned message
func (m *WarpSignedMessage) Payload() []byte {
	return m.UnsignedMessage.Payload
}