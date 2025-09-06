// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// warptest exposes common functionality for testing the warp package.
package warptest

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/consensus/protocol/chain"
	"github.com/luxfi/node/consensus/choices"
)

var ErrNotFound = errors.New("not found")

// mockBlock implements chain.Block for testing
type mockBlock struct {
	id ids.ID
}

// ID returns the block ID as a string
func (b *mockBlock) ID() string {
	return b.id.String()
}

// Accept marks the block as accepted
func (b *mockBlock) Accept(context.Context) error {
	return nil
}

// Reject marks the block as rejected  
func (b *mockBlock) Reject(context.Context) error {
	return nil
}

// Status returns the block status
func (b *mockBlock) Status() choices.Status {
	return choices.Accepted
}

// Parent returns the parent block ID as a string
func (b *mockBlock) Parent() string {
	return ids.Empty.String()
}

// Verify verifies the block
func (b *mockBlock) Verify(context.Context) error {
	return nil
}

// Bytes returns the block bytes
func (b *mockBlock) Bytes() []byte {
	return nil
}

// Height returns the block height
func (b *mockBlock) Height() uint64 {
	return 0
}

// Timestamp returns the block timestamp
func (b *mockBlock) Timestamp() time.Time {
	return time.Time{}
}

// EmptyBlockClient returns an error if a block is requested
var EmptyBlockClient BlockClient = MakeBlockClient()

type BlockClient func(ctx context.Context, blockID ids.ID) (chain.Block, error)

func (f BlockClient) GetAcceptedBlock(ctx context.Context, blockID ids.ID) (chain.Block, error) {
	return f(ctx, blockID)
}

// MakeBlockClient returns a new BlockClient that returns the provided blocks.
// If a block is requested that isn't part of the provided blocks, an error is
// returned.
func MakeBlockClient(blkIDs ...ids.ID) BlockClient {
	return func(_ context.Context, blkID ids.ID) (chain.Block, error) {
		if !slices.Contains(blkIDs, blkID) {
			return nil, ErrNotFound
		}

		// Create a mock block with the requested ID
		return &mockBlock{
			id: blkID,
		}, nil
	}
}
