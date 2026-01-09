// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package dynamicip

import (
	"context"
	"net/netip"
	"time"

	"github.com/luxfi/log"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/vm/utils"
)

const ipResolutionTimeout = 10 * time.Second

var _ Updater = (*updater)(nil)

// Updater periodically updates this node's public IP.
// Dispatch() and Stop() should only be called once.
type Updater interface {
	// Start periodically resolving and updating our public IP.
	// Doesn't return until after Stop() is called.
	// Should be called in a goroutine.
	Dispatch(log log.Logger)
	// Stop resolving and updating our public IP.
	Stop()
}

type updater struct {
	// The IP we periodically modify.
	dynamicIP *utils.Atomic[netip.AddrPort]
	// Used to find out what our public IP is.
	resolver Resolver
	// The parent of all contexts passed into resolver.Resolve().
	// Cancelling causes Dispatch() to eventually return.
	rootCtx context.Context
	// Cancelling causes Dispatch() to eventually return.
	// All in-flight calls to resolver.Resolve() will be cancelled.
	rootCtxCancel context.CancelFunc
	// Closed when Dispatch() has returned.
	doneChan chan struct{}
	// How often we update the public IP.
	updateFreq time.Duration
}

// Returns a new Updater that updates [dynamicIP]
// every [updateFreq]. Uses [resolver] to find
// out what our public IP is.
func NewUpdater(
	dynamicIP *utils.Atomic[netip.AddrPort],
	resolver Resolver,
	updateFreq time.Duration,
) Updater {
	ctx, cancel := context.WithCancel(context.Background())
	return &updater{
		dynamicIP:     dynamicIP,
		resolver:      resolver,
		rootCtx:       ctx,
		rootCtxCancel: cancel,
		doneChan:      make(chan struct{}),
		updateFreq:    updateFreq,
	}
}

// Start updating [u.dynamicIP] every [u.updateFreq].
// Stops when [dynamicIP.stopChan] is closed.
func (u *updater) Dispatch(log log.Logger) {
	ticker := time.NewTicker(u.updateFreq)
	defer func() {
		ticker.Stop()
		close(u.doneChan)
	}()

	var (
		initialAddrPort = u.dynamicIP.Get()
		oldAddr         = initialAddrPort.Addr()
		port            = initialAddrPort.Port()
	)
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(u.rootCtx, ipResolutionTimeout)
			newAddr, err := u.resolver.Resolve(ctx)
			cancel()
			if err != nil {
				log.Warn("couldn't resolve public IP. If this machine's IP recently changed, it may be sharing the wrong public IP with peers",
					luxlog.Err(err),
				)
				continue
			}

			if newAddr != oldAddr {
				u.dynamicIP.Set(netip.AddrPortFrom(newAddr, port))
				log.Info("updated public IP",
					luxlog.String("oldIP", oldAddr.String()),
					luxlog.String("newIP", newAddr.String()),
				)
				oldAddr = newAddr
			}
		case <-u.rootCtx.Done():
			return
		}
	}
}

func (u *updater) Stop() {
	// Cause Dispatch() to return and cancel all
	// in-flight calls to resolver.Resolve().
	u.rootCtxCancel()
	// Wait until Dispatch() has returned.
	<-u.doneChan
}
