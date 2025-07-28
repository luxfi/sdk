// Copyright 2025 Lux Industries, Inc.
// This file contains the Block interface.

package iface

import "context"

// Block represents a block in the blockchain
type Block interface {
	// ID returns the block's ID
	ID() ID
	
	// Accept marks the block as accepted
	Accept(context.Context) error
	
	// Reject marks the block as rejected
	Reject(context.Context) error
	
	// Status returns the block's status
	Status() Status
	
	// Parent returns the parent block ID
	Parent() ID
	
	// Height returns the block height
	Height() uint64
	
	// Timestamp returns the block timestamp
	Timestamp() Timestamp
	
	// Verify verifies the block
	Verify(context.Context) error
	
	// Bytes returns the block bytes
	Bytes() []byte
}