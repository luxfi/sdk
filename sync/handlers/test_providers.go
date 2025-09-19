// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package handlers

import (
	"github.com/luxfi/evm/core/state/snapshot"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
)

var (
	_ BlockProvider    = (*TestBlockProvider)(nil)
	_ SnapshotProvider = (*TestSnapshotProvider)(nil)
)

type TestBlockProvider struct {
	GetBlockFn func(common.Hash, uint64) *types.Block
}

func (t *TestBlockProvider) GetBlock(hash common.Hash, number uint64) *types.Block {
	return t.GetBlockFn(hash, number)
}

type TestSnapshotProvider struct {
	Snapshot *snapshot.Tree
}

func (t *TestSnapshotProvider) Snapshots() *snapshot.Tree {
	return t.Snapshot
}
