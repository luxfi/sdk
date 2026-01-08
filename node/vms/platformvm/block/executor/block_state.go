// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/chains/atomic"
	"github.com/luxfi/sdk/node/vms/platformvm/block"
	"github.com/luxfi/sdk/node/vms/platformvm/metrics"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
)

type proposalBlockState struct {
	onDecisionState state.Diff
	onCommitState   state.Diff
	onAbortState    state.Diff
}

// The state of a block.
// Note that not all fields will be set for a given block.
type blockState struct {
	proposalBlockState
	statelessBlock block.Block

	onAcceptState state.Diff
	onAcceptFunc  func()

	inputs          set.Set[ids.ID]
	timestamp       time.Time
	atomicRequests  map[ids.ID]*atomic.Requests
	verifiedHeights set.Set[uint64]
	metrics         metrics.Block
}
