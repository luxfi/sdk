// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"fmt"
	"time"

	"github.com/luxfi/consensus/core"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/network/p2p"
	"github.com/luxfi/node/network/p2p/lp118"
)

// lp118HandlerAdapter adapts lp118.Handler to p2p.Handler
type lp118HandlerAdapter struct {
	handler lp118.Handler
}

// newLP118HandlerAdapter creates a new adapter
func newLP118HandlerAdapter(handler lp118.Handler) p2p.Handler {
	return &lp118HandlerAdapter{
		handler: handler,
	}
}

// AppGossip handles gossip messages
func (a *lp118HandlerAdapter) AppGossip(ctx context.Context, nodeID ids.NodeID, gossipBytes []byte) {
	// lp118 doesn't use gossip, so this is a no-op
}

// AppRequest handles request messages
func (a *lp118HandlerAdapter) AppRequest(ctx context.Context, nodeID ids.NodeID, deadline time.Time, requestBytes []byte) ([]byte, *core.AppError) {
	// Forward to lp118 handler
	response, err := a.handler.AppRequest(ctx, nodeID, deadline, requestBytes)
	if err != nil {
		return nil, &core.AppError{
			Code:    -1,
			Message: err.Error(),
		}
	}
	return response, nil
}

// CrossChainAppRequest handles cross-chain requests
func (a *lp118HandlerAdapter) CrossChainAppRequest(ctx context.Context, chainID ids.ID, deadline time.Time, requestBytes []byte) ([]byte, error) {
	// lp118 doesn't support cross-chain requests
	return nil, fmt.Errorf("cross-chain requests not supported")
}
