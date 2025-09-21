// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package gossip

import (
	"context"
	"time"

	"github.com/luxfi/consensus/engine/core"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/node/network/p2p"
	"github.com/luxfi/node/network/p2p/gossip"
	"github.com/prometheus/client_golang/prometheus"
)

var _ p2p.Handler = (*txGossipHandler)(nil)

func NewTxGossipHandler[T gossip.Gossipable](
	logger log.Logger,
	marshaller gossip.Marshaller[T],
	mempool gossip.Set[T],
	metrics gossip.Metrics,
	maxMessageSize int,
	throttlingPeriod time.Duration,
	requestsPerPeer float64,
	validators p2p.ValidatorSet,
	registerer prometheus.Registerer,
	namespace string,
) (*txGossipHandler, error) {
	// push gossip messages can be handled from any peer
	handler := gossip.NewHandler(
		logger,
		marshaller,
		mempool,
		metrics,
		maxMessageSize,
	)

	// Create a sliding window throttler for rate limiting
	throttler := p2p.NewSlidingWindowThrottler(
		throttlingPeriod,
		int(requestsPerPeer), // Convert float64 to int for limit
	)

	throttledHandler := p2p.NewThrottlerHandler(
		handler,
		throttler,
		logger,
	)

	// pull gossip requests are filtered by validators and are throttled
	// to prevent spamming
	validatorHandler := p2p.NewValidatorHandler(
		throttledHandler,
		validators,
		logger,
	)

	return &txGossipHandler{
		appGossipHandler:  handler,
		appRequestHandler: validatorHandler,
	}, nil
}

type txGossipHandler struct {
	appGossipHandler  p2p.Handler
	appRequestHandler p2p.Handler
}

func (t *txGossipHandler) AppGossip(ctx context.Context, nodeID ids.NodeID, gossipBytes []byte) {
	t.appGossipHandler.AppGossip(ctx, nodeID, gossipBytes)
}

func (t *txGossipHandler) AppRequest(ctx context.Context, nodeID ids.NodeID, deadline time.Time, requestBytes []byte) ([]byte, *core.AppError) {
	return t.appRequestHandler.AppRequest(ctx, nodeID, deadline, requestBytes)
}

func (t *txGossipHandler) CrossChainAppRequest(ctx context.Context, chainID ids.ID, deadline time.Time, requestBytes []byte) ([]byte, error) {
	// Cross-chain requests are not supported for transaction gossip
	return nil, nil
}
