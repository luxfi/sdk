// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package warp

import (
	"net/http"
	"sync"

	"github.com/gorilla/rpc/v2"

	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/sdk/node/api"
	"github.com/luxfi/sdk/node/chains"
	"github.com/luxfi/sdk/utils/json"
	"github.com/luxfi/sdk/node/warp"
)

type Service struct {
	log          log.Logger
	chainManager chains.Manager
	lock         sync.RWMutex
	ipcs         *warp.ChainIPCs
}

func NewService(log log.Logger, chainManager chains.Manager, ipcs *warp.ChainIPCs) (http.Handler, error) {
	server := rpc.NewServer()
	codec := json.NewCodec()
	server.RegisterCodec(codec, "application/json")
	server.RegisterCodec(codec, "application/json;charset=UTF-8")
	return server, server.RegisterService(
		&Service{
			log:          log,
			chainManager: chainManager,
			ipcs:         ipcs,
		},
		"ipcs",
	)
}

type PublishBlockchainArgs struct {
	BlockchainID string `json:"blockchainID"`
}

type PublishBlockchainReply struct {
	ConsensusURL string `json:"consensusURL"`
	DecisionsURL string `json:"decisionsURL"`
}

// PublishBlockchain publishes the finalized accepted transactions from the
// blockchainID over the IPC
func (s *Service) PublishBlockchain(_ *http.Request, args *PublishBlockchainArgs, reply *PublishBlockchainReply) error {
	s.log.Warn("deprecated API called",
		log.UserString("service", "ipcs"),
		log.UserString("method", "publishBlockchain"),
		log.UserString("blockchainID", args.BlockchainID),
	)

	chainID, err := s.chainManager.Lookup(args.BlockchainID)
	if err != nil {
		s.log.Error("chain lookup failed",
			log.UserString("blockchainID", args.BlockchainID),
			log.Reflect("error", err),
		)
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	ipcs, err := s.ipcs.Publish(chainID)
	if err != nil {
		s.log.Error("couldn't publish chain",
			log.UserString("blockchainID", args.BlockchainID),
			log.Reflect("error", err),
		)
		return err
	}

	reply.ConsensusURL = ipcs.ConsensusURL()
	reply.DecisionsURL = ipcs.DecisionsURL()

	return nil
}

type UnpublishBlockchainArgs struct {
	BlockchainID string `json:"blockchainID"`
}

// UnpublishBlockchain closes publishing of a blockchainID
func (s *Service) UnpublishBlockchain(_ *http.Request, args *UnpublishBlockchainArgs, _ *api.EmptyReply) error {
	s.log.Warn("deprecated API called",
		log.UserString("service", "ipcs"),
		log.UserString("method", "unpublishBlockchain"),
		log.UserString("blockchainID", args.BlockchainID),
	)

	chainID, err := s.chainManager.Lookup(args.BlockchainID)
	if err != nil {
		s.log.Error("chain lookup failed",
			log.UserString("blockchainID", args.BlockchainID),
			log.Reflect("error", err),
		)
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	ok, err := s.ipcs.Unpublish(chainID)
	if !ok {
		s.log.Error("couldn't publish chain",
			log.UserString("blockchainID", args.BlockchainID),
			log.Reflect("error", err),
		)
	}

	return err
}

type GetPublishedBlockchainsReply struct {
	Chains []ids.ID `json:"chains"`
}

// GetPublishedBlockchains returns blockchains being published
func (s *Service) GetPublishedBlockchains(_ *http.Request, _ *struct{}, reply *GetPublishedBlockchainsReply) error {
	s.log.Warn("deprecated API called",
		log.UserString("service", "ipcs"),
		log.UserString("method", "getPublishedBlockchains"),
	)

	s.lock.RLock()
	defer s.lock.RUnlock()

	reply.Chains = s.ipcs.GetPublishedBlockchains()
	return nil
}
