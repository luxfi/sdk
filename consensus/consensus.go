// (c) 2019-2020, Lux Industries, Inc.
// All rights reserved.
// See the file LICENSE for licensing terms.

package consensus

import (
   "math/big"

   "github.com/luxfi/evm/interfaces"
   "github.com/luxfi/node/ids"
)


// Block is the interface for linear blocks
type Block interface {
	ID() ids.ID
	ParentID() ids.ID
	Height() uint64
	Verify() error
	Accept() error
	Reject() error
	Status() Status
}

// Status represents the status of a block
type Status uint8

const (
	Unknown Status = iota
	Processing
	Rejected
	Accepted
)

// Consensus interface for linear consensus
type Consensus interface {
	Add(Block) error
	Finalized() bool
}

// Use interfaces from the interfaces package
type ChainHeaderReader = interfaces.ChainHeaderReader
type ChainReader = interfaces.ChainReader
type Engine = interfaces.Engine

// PoW is a consensus engine based on proof-of-work (deprecated).
type PoW interface {
	Engine

	// Hashrate returns the current mining hashrate of a PoW consensus engine.
	Hashrate() float64
}

// PoS is a consensus engine based on proof-of-stake (delegated to Lux).
type PoS interface {
	Engine
}

// FeeConfig defines the interface for chain fee configuration
type FeeConfig interface {
	GetBaseFee(timestamp uint64) *big.Int
	GetMaxGasLimit() *big.Int
}
