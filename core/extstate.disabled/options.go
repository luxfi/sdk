// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package extstate

import (
	"github.com/luxfi/geth/core/state"

	"github.com/luxfi/evm/utils"
)

type workerPool struct {
	*utils.BoundedWorkers
}

func (wp *workerPool) Done() {
	// Done is guaranteed to only be called after all work is already complete,
	// so we call Wait for goroutines to finish before returning.
	wp.BoundedWorkers.Wait()
}

func WithConcurrentWorkers(prefetchers int) state.PrefetcherOption {
	pool := &workerPool{
		BoundedWorkers: utils.NewBoundedWorkers(prefetchers),
	}
	return state.WithWorkerPools(func() state.WorkerPool { return pool })
}
