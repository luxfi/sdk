// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"testing"

	"github.com/luxfi/ids"
	"github.com/luxfi/node/utils/set"
	nodeCore "github.com/luxfi/node/consensus/engine/core"
)

// TestSender is a test implementation of the Sender interface
type TestSender struct {
	T *testing.T

	CantSendAppGossip         bool
	CantSendAppGossipSpecific bool
	CantSendAppRequest        bool
	CantSendAppResponse       bool
	CantSendAppError          bool

	SendAppGossipF             func(context.Context, set.Set[ids.NodeID], []byte) error
	SendAppGossipSpecificF     func(context.Context, set.Set[ids.NodeID], []byte) error
	SendAppRequestF            func(context.Context, set.Set[ids.NodeID], uint32, []byte) error
	SendAppResponseF           func(context.Context, ids.NodeID, uint32, []byte) error
	SendAppErrorF              func(context.Context, ids.NodeID, uint32, int32, string) error
	SendCrossChainAppRequestF  func(context.Context, ids.ID, uint32, []byte) error
	SendCrossChainAppResponseF func(context.Context, ids.ID, uint32, []byte) error
	SendCrossChainAppErrorF    func(context.Context, ids.ID, uint32, int32, string) error

	// Channel for capturing sent gossip messages
	SentAppGossip chan []byte
}

// SendAppGossip implements the consensus AppSender interface
func (s *TestSender) SendAppGossip(ctx context.Context, config nodeCore.SendConfig, msg []byte) error {
	// Convert SendConfig to node set for internal functions
	nodeSet := set.Set[ids.NodeID]{}
	for id := range config.NodeIDs {
		nodeSet.Add(id)
	}

	if s.SendAppGossipF != nil {
		return s.SendAppGossipF(ctx, nodeSet, msg)
	}
	if s.SentAppGossip != nil {
		// Send to channel if available
		s.SentAppGossip <- msg
	}
	if s.CantSendAppGossip && s.T != nil {
		s.T.Helper()
		s.T.Fatal("unexpectedly called SendAppGossip")
	}
	return nil
}

func (s *TestSender) SendAppGossipSpecific(ctx context.Context, nodeIDs set.Set[ids.NodeID], msg []byte) error {
	if s.SendAppGossipSpecificF != nil {
		return s.SendAppGossipSpecificF(ctx, nodeIDs, msg)
	}
	if s.CantSendAppGossipSpecific && s.T != nil {
		s.T.Helper()
		s.T.Fatal("unexpectedly called SendAppGossipSpecific")
	}
	return nil
}

func (s *TestSender) SendAppRequest(ctx context.Context, nodeIDs set.Set[ids.NodeID], requestID uint32, request []byte) error {
	if s.SendAppRequestF != nil {
		return s.SendAppRequestF(ctx, nodeIDs, requestID, request)
	}
	if s.CantSendAppRequest && s.T != nil {
		s.T.Helper()
		s.T.Fatal("unexpectedly called SendAppRequest")
	}
	return nil
}

func (s *TestSender) SendAppResponse(ctx context.Context, nodeID ids.NodeID, requestID uint32, response []byte) error {
	if s.SendAppResponseF != nil {
		return s.SendAppResponseF(ctx, nodeID, requestID, response)
	}
	if s.CantSendAppResponse && s.T != nil {
		s.T.Helper()
		s.T.Fatal("unexpectedly called SendAppResponse")
	}
	return nil
}

func (s *TestSender) SendAppError(ctx context.Context, nodeID ids.NodeID, requestID uint32, errorCode int32, errorMessage string) error {
	if s.SendAppErrorF != nil {
		return s.SendAppErrorF(ctx, nodeID, requestID, errorCode, errorMessage)
	}
	if s.CantSendAppError && s.T != nil {
		s.T.Helper()
		s.T.Fatal("unexpectedly called SendAppError")
	}
	return nil
}

func (s *TestSender) SendCrossChainAppRequest(ctx context.Context, chainID ids.ID, requestID uint32, appRequestBytes []byte) error {
	if s.SendCrossChainAppRequestF != nil {
		return s.SendCrossChainAppRequestF(ctx, chainID, requestID, appRequestBytes)
	}
	return nil
}

func (s *TestSender) SendCrossChainAppResponse(ctx context.Context, chainID ids.ID, requestID uint32, appResponseBytes []byte) error {
	if s.SendCrossChainAppResponseF != nil {
		return s.SendCrossChainAppResponseF(ctx, chainID, requestID, appResponseBytes)
	}
	return nil
}

func (s *TestSender) SendCrossChainAppError(ctx context.Context, chainID ids.ID, requestID uint32, errorCode int32, errorMessage string) error {
	if s.SendCrossChainAppErrorF != nil {
		return s.SendCrossChainAppErrorF(ctx, chainID, requestID, errorCode, errorMessage)
	}
	return nil
}
