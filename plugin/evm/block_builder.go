// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"sync"
	"time"

	"github.com/holiman/uint256"
	commonEng "github.com/luxfi/consensus/core"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/evm/core/txpool"
	"github.com/luxfi/log"
)

const (
	// Minimum amount of time to wait after building a block before attempting to build a block
	// a second time without changing the contents of the mempool.
	minBlockBuildingRetryDelay = 500 * time.Millisecond
)

type blockBuilder struct {
	ctx context.Context

	txPool *txpool.TxPool

	shutdownChan <-chan struct{}
	shutdownWg   *sync.WaitGroup

	pendingSignal *sync.Cond

	buildBlockLock sync.Mutex

	// lastBuildTime is the time when the last block was built.
	// This is used to ensure that we don't build blocks too frequently,
	// but at least after a minimum delay of minBlockBuildingRetryDelay.
	lastBuildTime time.Time
}

func (vm *VM) NewBlockBuilder() *blockBuilder {
	b := &blockBuilder{
		ctx:          context.Background(),
		txPool:       vm.txPool,
		shutdownChan: vm.shutdownChan,
		shutdownWg:   &vm.shutdownWg,
	}
	b.pendingSignal = sync.NewCond(&b.buildBlockLock)
	return b
}

// handleGenerateBlock is called from the VM immediately after BuildBlock.
func (b *blockBuilder) handleGenerateBlock() {
	b.buildBlockLock.Lock()
	defer b.buildBlockLock.Unlock()
	b.lastBuildTime = time.Now()
}

// needToBuild returns true if there are outstanding transactions to be issued
// into a block.
func (b *blockBuilder) needToBuild() bool {
	size := b.txPool.PendingSize(txpool.PendingFilter{
		MinTip: uint256.MustFromBig(b.txPool.GasTip()),
	})
	return size > 0
}

// signalCanBuild signals that a new block can be built.
func (b *blockBuilder) signalCanBuild() {
	b.buildBlockLock.Lock()
	defer b.buildBlockLock.Unlock()
	b.pendingSignal.Broadcast()
}

// awaitSubmittedTxs waits for new transactions to be submitted
// and notifies the VM when the tx pool has transactions to be
// put into a new block.
func (b *blockBuilder) awaitSubmittedTxs() {
	// txSubmitChan is invoked when new transactions are issued as well as on re-orgs which
	// may orphan transactions that were previously in a preferred block.
	txSubmitChan := make(chan core.NewTxsEvent)
	b.txPool.SubscribeTransactions(txSubmitChan, true)

	b.shutdownWg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic in awaitSubmittedTxs", "error", r)
				panic(r)
			}
		}()
		defer b.shutdownWg.Done()

		for {
			select {
			case <-txSubmitChan:
				log.Trace("New tx detected, trying to generate a block")
				b.signalCanBuild()
			case <-b.shutdownChan:
				return
			}
		}
	}()
}

// waitForEvent waits until a block needs to be built.
// It returns only after at least [minBlockBuildingRetryDelay] passed from the last time a block was built.
func (b *blockBuilder) waitForEvent(ctx context.Context) (commonEng.Message, error) {
	lastBuildTime, err := b.waitForNeedToBuild(ctx)
	if err != nil {
		return commonEng.Message{}, err
	}
	timeSinceLastBuildTime := time.Since(lastBuildTime)
	if b.lastBuildTime.IsZero() || timeSinceLastBuildTime >= minBlockBuildingRetryDelay {
		log.Debug("Last time we built a block was long enough ago, no need to wait", "timeSinceLastBuildTime", timeSinceLastBuildTime)
		return commonEng.Message{Type: commonEng.PendingTxs}, nil
	}
	timeUntilNextBuild := minBlockBuildingRetryDelay - timeSinceLastBuildTime
	log.Debug("Last time we built a block was too recent, waiting", "timeUntilNextBuild", timeUntilNextBuild)
	select {
	case <-ctx.Done():
		return commonEng.Message{}, ctx.Err()
	case <-time.After(timeUntilNextBuild):
		return commonEng.Message{Type: commonEng.PendingTxs}, nil
	}
}

// waitForNeedToBuild waits until needToBuild returns true.
// It returns the last time a block was built.
func (b *blockBuilder) waitForNeedToBuild(ctx context.Context) (time.Time, error) {
	// Start a goroutine to broadcast when context is cancelled
	// This wakes up the Wait() call so we can check context cancellation
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			b.pendingSignal.Broadcast()
		case <-done:
		}
	}()
	defer close(done)

	b.buildBlockLock.Lock()
	defer b.buildBlockLock.Unlock()
	for !b.needToBuild() {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return time.Time{}, ctx.Err()
		default:
		}
		b.pendingSignal.Wait()
	}
	return b.lastBuildTime, nil
}
