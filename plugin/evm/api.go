// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/log"
)

// ChainAPI introduces linear specific functionality to the evm
type ChainAPI struct{ vm *VM }

// GetAcceptedFrontReply defines the reply that will be sent from the
// GetAcceptedFront API call
type GetAcceptedFrontReply struct {
	Hash   common.Hash `json:"hash"`
	Number *big.Int    `json:"number"`
}

// GetAcceptedFront returns the last accepted block's hash and height
func (api *ChainAPI) GetAcceptedFront(ctx context.Context) (*GetAcceptedFrontReply, error) {
	blk := api.vm.blockChain.LastConsensusAcceptedBlock()
	return &GetAcceptedFrontReply{
		Hash:   blk.Hash(),
		Number: blk.Number(),
	}, nil
}

// IssueBlock to the chain
func (api *ChainAPI) IssueBlock(ctx context.Context) error {
	log.Info("Issuing a new block")
	api.vm.builder.signalTxsReady()
	return nil
}
