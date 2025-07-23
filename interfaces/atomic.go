// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package interfaces

import (
	"math/big"
)

// SharedMemory provides atomic operations across chains
type SharedMemory interface {
	// Get retrieves data from shared memory
	Get(key []byte) ([]byte, error)
	
	// Put stores data in shared memory
	Put(key []byte, value []byte) error
	
	// Remove deletes data from shared memory
	Remove(key []byte) error
	
	// NewBatch creates a new batch
	NewBatch() Batch
}

// AtomicMemory manages shared memory across chains
type AtomicMemory interface {
	// NewSharedMemory creates a new shared memory instance for a chain
	NewSharedMemory(chainID ID) SharedMemory
}

// AtomicTx represents an atomic transaction
type AtomicTx interface {
	// BlockNumber returns the block number
	BlockNumber() *big.Int
	
	// UTXOs returns the UTXOs
	UTXOs() [][]byte
}

// NewMemory creates a new atomic memory instance
func NewMemory(db Database) AtomicMemory {
	return &atomicMemory{
		db: db,
	}
}

// atomicMemory implements AtomicMemory
type atomicMemory struct {
	db Database
}

// NewSharedMemory implements AtomicMemory
func (m *atomicMemory) NewSharedMemory(chainID ID) SharedMemory {
	return &sharedMemory{
		db:      m.db,
		chainID: chainID,
	}
}

// sharedMemory implements SharedMemory
type sharedMemory struct {
	db      Database
	chainID ID
}

// Get implements SharedMemory
func (sm *sharedMemory) Get(key []byte) ([]byte, error) {
	return sm.db.Get(append(sm.chainID[:], key...))
}

// Put implements SharedMemory
func (sm *sharedMemory) Put(key []byte, value []byte) error {
	return sm.db.Put(append(sm.chainID[:], key...), value)
}

// Remove implements SharedMemory
func (sm *sharedMemory) Remove(key []byte) error {
	return sm.db.Delete(append(sm.chainID[:], key...))
}

// NewBatch implements SharedMemory
func (sm *sharedMemory) NewBatch() Batch {
	return sm.db.NewBatch()
}