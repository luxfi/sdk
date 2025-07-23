// (c) 2021-2022, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"github.com/luxfi/evm/core/state/snapshot"
	"github.com/luxfi/evm/core/types"
	"github.com/luxfi/geth/common"
)

type BlockProvider interface {
	GetBlock(common.Hash, uint64) *types.Block
}

type SnapshotProvider interface {
	Snapshots() *snapshot.Tree
}

type SyncDataProvider interface {
	BlockProvider
	SnapshotProvider
}
