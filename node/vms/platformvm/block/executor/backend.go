// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"errors"
	"sync"
	"time"

	"github.com/luxfi/ids"
	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/vms/platformvm/block"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/txs/mempool"
)

var errConflictingParentTxs = errors.New("block contains a transaction that conflicts with a transaction in a parent block")

// Shared fields used by visitors.
type backend struct {
	mempool.Mempool[*txs.Tx]
	// lastAccepted is the ID of the last block that had Accept() called on it.
	lastAccepted ids.ID

	// blkIDToState is a map from a block's ID to the state of the block.
	// Blocks are put into this map when they are verified.
	// Proposal blocks are removed from this map when they are rejected
	// or when a child is accepted.
	// All other blocks are removed when they are accepted/rejected.
	// Note that Genesis block is a commit block so no need to update
	// blkIDToState with it upon backend creation (Genesis is already accepted)
	blkIDToState     map[ids.ID]*blockState
	blkIDToStateLock sync.RWMutex // Protects concurrent access to blkIDToState
	state            state.State

	ctx *consensusctx.Context
}

// SharedMemory provides cross-chain atomic operations
type SharedMemory interface {
	Get(peerChainID ids.ID, keys [][]byte) ([][]byte, error)
	Apply(requests map[ids.ID]interface{}, batch ...interface{}) error
}

func (b *backend) GetState(blkID ids.ID) (state.Chain, bool) {
	b.blkIDToStateLock.RLock()
	defer b.blkIDToStateLock.RUnlock()

	// If the block is in the map, it is either processing or a proposal block
	// that was accepted without an accepted child.
	if state, ok := b.blkIDToState[blkID]; ok {
		if state.onAcceptState != nil {
			return state.onAcceptState, true
		}
		return nil, false
	}

	// Note: If the last accepted block is a proposal block, we will have
	//       returned in the above if statement.
	return b.state, blkID == b.state.GetLastAccepted()
}

func (b *backend) getOnAbortState(blkID ids.ID) (state.Diff, bool) {
	b.blkIDToStateLock.RLock()
	defer b.blkIDToStateLock.RUnlock()

	state, ok := b.blkIDToState[blkID]
	if !ok || state.onAbortState == nil {
		return nil, false
	}
	return state.onAbortState, true
}

func (b *backend) getOnCommitState(blkID ids.ID) (state.Diff, bool) {
	b.blkIDToStateLock.RLock()
	defer b.blkIDToStateLock.RUnlock()

	state, ok := b.blkIDToState[blkID]
	if !ok || state.onCommitState == nil {
		return nil, false
	}
	return state.onCommitState, true
}

func (b *backend) GetBlock(blkID ids.ID) (block.Block, error) {
	b.blkIDToStateLock.RLock()
	// See if the block is in memory.
	if blk, ok := b.blkIDToState[blkID]; ok {
		b.blkIDToStateLock.RUnlock()
		return blk.statelessBlock, nil
	}
	b.blkIDToStateLock.RUnlock()

	// The block isn't in memory. Check the database.
	return b.state.GetStatelessBlock(blkID)
}

func (b *backend) LastAccepted() ids.ID {
	return b.lastAccepted
}

func (b *backend) free(blkID ids.ID) {
	b.blkIDToStateLock.Lock()
	defer b.blkIDToStateLock.Unlock()
	delete(b.blkIDToState, blkID)
}

// getBlockState returns the block state for the given block ID.
// Returns nil and false if the block state doesn't exist.
func (b *backend) getBlockState(blkID ids.ID) (*blockState, bool) {
	b.blkIDToStateLock.RLock()
	defer b.blkIDToStateLock.RUnlock()
	state, ok := b.blkIDToState[blkID]
	return state, ok
}

// setBlockState sets the block state for the given block ID.
func (b *backend) setBlockState(blkID ids.ID, state *blockState) {
	b.blkIDToStateLock.Lock()
	defer b.blkIDToStateLock.Unlock()
	b.blkIDToState[blkID] = state
}

func (b *backend) getTimestamp(blkID ids.ID) time.Time {
	b.blkIDToStateLock.RLock()
	// Check if the block is processing.
	// If the block is processing, then we are guaranteed to have populated its
	// timestamp in its state.
	if blkState, ok := b.blkIDToState[blkID]; ok {
		b.blkIDToStateLock.RUnlock()
		return blkState.timestamp
	}
	b.blkIDToStateLock.RUnlock()

	// The block isn't processing.
	// According to the chain.Block interface, the last accepted
	// block is the only accepted block that must return a correct timestamp,
	// so we just return the chain time.
	return b.state.GetTimestamp()
}

// verifyUniqueInputs returns nil iff no blocks in the inclusive
// ancestry of [blkID] consume an input in [inputs].
func (b *backend) verifyUniqueInputs(blkID ids.ID, inputs set.Set[ids.ID]) error {
	if inputs.Len() == 0 {
		return nil
	}

	b.blkIDToStateLock.RLock()
	defer b.blkIDToStateLock.RUnlock()

	// Check for conflicts in ancestors.
	for {
		state, ok := b.blkIDToState[blkID]
		if !ok {
			// The parent state isn't pinned in memory.
			// This means the parent must be accepted already.
			return nil
		}

		if state.inputs.Overlaps(inputs) {
			return errConflictingParentTxs
		}

		blk := state.statelessBlock
		blkID = blk.Parent()
	}
}
