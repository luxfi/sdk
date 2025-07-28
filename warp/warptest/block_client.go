// (c) 2024, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// warptest exposes common functionality for testing the warp package.
package warptest

import (
	"context"
	"slices"
	"time"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/evm/interfaces"
)

// Block status constants
const (
	Unknown = iota
	Processing
	Rejected
	Accepted
)

// TestBlock is a simple test implementation of a block
type TestBlock struct {
	id     interfaces.ID
	status int
	height uint64
	parent interfaces.ID
	bytes  []byte
}

// ID returns the block ID
func (b *TestBlock) ID() common.Hash {
	// Convert interfaces.ID to common.Hash
	return common.BytesToHash(b.id[:])
}

// Accept marks the block as accepted
func (b *TestBlock) Accept(context.Context) error {
	b.status = Accepted
	return nil
}

// Reject marks the block as rejected
func (b *TestBlock) Reject(context.Context) error {
	b.status = Rejected
	return nil
}

// Status returns the block's status
func (b *TestBlock) Status() interfaces.Status {
	return interfaces.Status(b.status)
}

// Parent returns the parent block ID
func (b *TestBlock) Parent() common.Hash {
	// Convert interfaces.ID to common.Hash
	return common.BytesToHash(b.parent[:])
}

// Height returns the block height
func (b *TestBlock) Height() uint64 {
	return b.height
}

// Timestamp returns the block timestamp
func (b *TestBlock) Timestamp() time.Time {
	return time.Now() // Return current time for test blocks
}

// Verify verifies the block
func (b *TestBlock) Verify(context.Context) error {
	return nil
}

// Bytes returns the block bytes
func (b *TestBlock) Bytes() []byte {
	return b.bytes
}

// EmptyBlockClient returns an error if a block is requested
var EmptyBlockClient BlockClient = MakeBlockClient()

type BlockClient func(ctx context.Context, blockID interfaces.ID) (interfaces.Block, error)

func (f BlockClient) GetAcceptedBlock(ctx context.Context, blockID interfaces.ID) (interfaces.Block, error) {
	return f(ctx, blockID)
}

// MakeBlockClient returns a new BlockClient that returns the provided blocks.
// If a block is requested that isn't part of the provided blocks, an error is
// returned.
func MakeBlockClient(blkIDs ...interfaces.ID) BlockClient {
	return func(_ context.Context, blkID interfaces.ID) (interfaces.Block, error) {
		if !slices.Contains(blkIDs, blkID) {
			return nil, interfaces.ErrNotFound
		}

		return &TestBlock{
			id:     blkID,
			status: Accepted,
		}, nil
	}
}
