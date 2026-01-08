// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"context"

	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/p2p"
	"github.com/luxfi/warp"
)

// TestAppSender is a test implementation of warp.Sender (p2p.Sender) for platformvm tests
type TestAppSender struct{}

var _ warp.Sender = (*TestAppSender)(nil)

// SendRequest sends a request to the specified nodes (no-op for tests)
func (t *TestAppSender) SendRequest(ctx context.Context, nodeIDs set.Set[ids.NodeID], requestID uint32, request []byte) error {
	return nil
}

// SendResponse sends a response to a previous request (no-op for tests)
func (t *TestAppSender) SendResponse(ctx context.Context, nodeID ids.NodeID, requestID uint32, response []byte) error {
	return nil
}

// SendError sends an error response to a previous request (no-op for tests)
func (t *TestAppSender) SendError(ctx context.Context, nodeID ids.NodeID, requestID uint32, errorCode int32, errorMessage string) error {
	return nil
}

// SendGossip sends a gossip message (no-op for tests)
func (t *TestAppSender) SendGossip(ctx context.Context, config p2p.SendConfig, msg []byte) error {
	return nil
}
