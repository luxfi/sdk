// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txheap

import (
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/utils/heap"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
)

var _ TimedHeap = (*byEndTime)(nil)

type TimedHeap interface {
	Heap

	Timestamp() time.Time
}

type byEndTime struct {
	txHeap
}

func NewByEndTime() TimedHeap {
	return &byEndTime{
		txHeap: txHeap{
			heap: heap.NewMap[ids.ID, *txs.Tx](func(a, b *txs.Tx) bool {
				aTime := a.Unsigned.(txs.Staker).EndTime()
				bTime := b.Unsigned.(txs.Staker).EndTime()
				return aTime.Before(bTime)
			}),
		},
	}
}

func (h *byEndTime) Timestamp() time.Time {
	return h.Peek().Unsigned.(txs.Staker).EndTime()
}
