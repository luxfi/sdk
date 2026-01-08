// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package network

import (
	"context"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/warp"
)

var _ Atomic = (*atomic)(nil)

type Atomic interface {
	warp.Handler

	Set(warp.Handler)
}

type atomic struct {
	handler utils.Atomic[warp.Handler]
}

func NewAtomic(h warp.Handler) Atomic {
	a := &atomic{}
	a.handler.Set(h)
	return a
}

func (a *atomic) Request(
	ctx context.Context,
	nodeID ids.NodeID,
	requestID uint32,
	deadline time.Time,
	msg []byte,
) ([]byte, *warp.Error) {
	h := a.handler.Get()
	return h.Request(
		ctx,
		nodeID,
		requestID,
		deadline,
		msg,
	)
}

func (a *atomic) RequestFailed(
	ctx context.Context,
	nodeID ids.NodeID,
	requestID uint32,
	appErr *warp.Error,
) error {
	h := a.handler.Get()
	return h.RequestFailed(
		ctx,
		nodeID,
		requestID,
		appErr,
	)
}

func (a *atomic) Response(
	ctx context.Context,
	nodeID ids.NodeID,
	requestID uint32,
	msg []byte,
) error {
	h := a.handler.Get()
	return h.Response(
		ctx,
		nodeID,
		requestID,
		msg,
	)
}

func (a *atomic) Gossip(
	ctx context.Context,
	nodeID ids.NodeID,
	msg []byte,
) error {
	h := a.handler.Get()
	return h.Gossip(
		ctx,
		nodeID,
		msg,
	)
}

func (a *atomic) Set(h warp.Handler) {
	a.handler.Set(h)
}
