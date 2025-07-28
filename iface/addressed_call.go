// (c) 2023, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package iface

import (
	"encoding/binary"
	"errors"
)

var (
	ErrInvalidAddressLength = errors.New("invalid address length")
)

// AddressedCall represents a warp message payload for contract calls
type AddressedCall struct {
	SourceAddress []byte
	Payload       []byte
}

// Verify validates the addressed call
func (a *AddressedCall) Verify() error {
	if len(a.SourceAddress) != 20 { // Ethereum addresses are 20 bytes
		return ErrInvalidAddressLength
	}
	return nil
}

// Bytes returns the byte representation of the addressed call
func (a *AddressedCall) Bytes() []byte {
	// Format: [address length (4 bytes)][address][payload]
	result := make([]byte, 4+len(a.SourceAddress)+len(a.Payload))
	binary.BigEndian.PutUint32(result[:4], uint32(len(a.SourceAddress)))
	copy(result[4:], a.SourceAddress)
	copy(result[4+len(a.SourceAddress):], a.Payload)
	return result
}

// Unmarshal deserializes bytes into an AddressedCall
func (a *AddressedCall) Unmarshal(data []byte) error {
	if len(data) < 4 {
		return errors.New("data too short")
	}
	
	addrLen := binary.BigEndian.Uint32(data[:4])
	if len(data) < int(4+addrLen) {
		return errors.New("invalid address length")
	}
	
	a.SourceAddress = make([]byte, addrLen)
	copy(a.SourceAddress, data[4:4+addrLen])
	
	if len(data) > int(4+addrLen) {
		a.Payload = make([]byte, len(data)-int(4+addrLen))
		copy(a.Payload, data[4+addrLen:])
	}
	
	return a.Verify()
}