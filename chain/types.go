// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/luxfi/node/ids"
)

// Transaction represents a blockchain transaction
type Transaction interface {
	// ID returns the transaction ID
	ID() ids.ID

	// Bytes returns the serialized transaction
	Bytes() []byte

	// Sign signs the transaction with the given signers
	Sign(signers []ids.ShortID) error

	// Verify verifies the transaction signatures
	Verify() error
}

// Action represents an action within a transaction
type Action interface {
	// TypeID returns the type ID of the action
	TypeID() uint8

	// Execute executes the action
	Execute() error
}

// Result represents the result of transaction execution
type Result struct {
	Success bool
	Error   error
	Outputs []Output
}

// Output represents a transaction output
type Output struct {
	AssetID ids.ID
	Amount  uint64
	Owner   ids.ShortID
}

// Block represents a blockchain block
type Block struct {
	Height       uint64
	ID           ids.ID
	ParentID     ids.ID
	Timestamp    int64
	Transactions []Transaction
}
