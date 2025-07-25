// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"time"

	commoneng "github.com/luxfi/node/consensus/engine/core"
	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/network/p2p"
	"github.com/luxfi/node/network/p2p/gossip"
	"github.com/luxfi/node/utils/logging"
)

// newTxGossipHandler creates a new transaction gossip handler
func newTxGossipHandler(
	log logging.Logger,
	marshaller gossip.Marshaller[*GossipEthTx],
	mempool gossip.Set[*GossipEthTx],
	metrics gossip.Metrics,
	maxSize int,
	throttlingPeriod time.Duration,
	throttlingLimit int,
	targetMessageSize int,
) p2p.Handler {
	// Create a gossip handler
	handler := gossip.NewHandler[*GossipEthTx](
		log,
		marshaller,
		mempool,
		metrics,
		maxSize,
	)

	// Wrap with throttling
	return &throttledHandler{
		Handler:          handler,
		throttlingPeriod: throttlingPeriod,
		throttlingLimit:  throttlingLimit,
	}
}

// throttledHandler wraps a handler with throttling
type throttledHandler[T any] struct {
	gossip.Handler[T]
	throttlingPeriod time.Duration
	throttlingLimit  int
}

func (h *throttledHandler[T]) AppRequest(ctx context.Context, nodeID ids.NodeID, deadline time.Time, request []byte) ([]byte, *commoneng.AppError) {
	// TODO: Implement throttling logic
	return h.Handler.AppRequest(ctx, nodeID, deadline, request)
}

func (h *throttledHandler[T]) AppGossip(ctx context.Context, nodeID ids.NodeID, gossipBytes []byte) {
	// TODO: Implement throttling logic
	h.Handler.AppGossip(ctx, nodeID, gossipBytes)
}
