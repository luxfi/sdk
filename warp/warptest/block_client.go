// (c) 2024, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// warptest exposes common functionality for testing the warp package.
package warptest

import (
	"context"
	"slices"

	"github.com/luxfi/node/database"
	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/consensus/linear"
	lineartest "github.com/luxfi/node/consensus/linear/lineartest"
	consensustest "github.com/luxfi/node/consensus/consensustest"
)

// EmptyBlockClient returns an error if a block is requested
var EmptyBlockClient BlockClient = MakeBlockClient()

type BlockClient func(ctx context.Context, blockID ids.ID) (linear.Block, error)

func (f BlockClient) GetAcceptedBlock(ctx context.Context, blockID ids.ID) (linear.Block, error) {
	return f(ctx, blockID)
}

// MakeBlockClient returns a new BlockClient that returns the provided blocks.
// If a block is requested that isn't part of the provided blocks, an error is
// returned.
func MakeBlockClient(blkIDs ...ids.ID) BlockClient {
	return func(_ context.Context, blkID ids.ID) (linear.Block, error) {
		if !slices.Contains(blkIDs, blkID) {
			return nil, database.ErrNotFound
		}

		return &lineartest.Block{
			Decidable: consensustest.Decidable{
				IDV:    blkID,
				Status: consensustest.Accepted,
			},
		}, nil
	}
}
