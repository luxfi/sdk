// (c) 2019-2020, Lux Industries, Inc.
//
// This file is a derived work, based on the go-ethereum library whose original
// notices appear below.
//
// It is distributed under a license compatible with the licensing terms of the
// original code from which it is derived.
//
// Much love to the original authors for their work.
// **********
// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package backends

import (
	"context"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/luxfi/evm/vmerrs"
	"github.com/luxfi/evm/accounts/abi"
	"github.com/luxfi/evm/accounts/abi/bind"
	"github.com/luxfi/evm/consensus/dummy"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/evm/core/bloombits"
	"github.com/luxfi/evm/core/rawdb"
	"github.com/luxfi/evm/core/state"
	"github.com/luxfi/evm/core/types"
	"github.com/luxfi/evm/core/vm"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/evm/rpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

// Verify that SimulatedBackend implements required interfaces
var (
	_ bind.AcceptedContractCaller = (*SimulatedBackend)(nil)
	_ bind.ContractBackend        = (*SimulatedBackend)(nil)
	_ bind.DeployBackend          = (*SimulatedBackend)(nil)

	_ ethereum.ChainReader              = (*SimulatedBackend)(nil)
	_ ethereum.ChainStateReader         = (*SimulatedBackend)(nil)
	_ ethereum.TransactionReader        = (*SimulatedBackend)(nil)
	_ ethereum.TransactionSender        = (*SimulatedBackend)(nil)
	_ ethereum.ContractCaller           = (*SimulatedBackend)(nil)
	_ ethereum.GasEstimator             = (*SimulatedBackend)(nil)
	_ ethereum.GasPricer                = (*SimulatedBackend)(nil)
	_ ethereum.LogFilterer              = (*SimulatedBackend)(nil)
	_ interfaces.AcceptedStateReader    = (*SimulatedBackend)(nil)
	_ interfaces.AcceptedContractCaller = (*SimulatedBackend)(nil)
)

// SimulatedBackend is a simulated blockchain.
// Deprecated: use package github.com/luxfi/evm/ethclient/simulated instead.
type SimulatedBackend struct {
	*simulated.Backend
	simulated.Client
}

// Fork sets the head to a new block, which is based on the provided parentHash.
func (b *SimulatedBackend) Fork(ctx context.Context, parentHash common.Hash) error {
	return b.Backend.Fork(parentHash)
}

// NewSimulatedBackend creates a new binding backend using a simulated blockchain
// for testing purposes.
//
// A simulated backend always uses chainID 1337.
//
// Deprecated: please use simulated.Backend from package
// github.com/luxfi/evm/ethclient/simulated instead.
func NewSimulatedBackend(alloc types.GenesisAlloc, gasLimit uint64) *SimulatedBackend {
	b := simulated.NewBackend(alloc, simulated.WithBlockGasLimit(gasLimit))
	return &SimulatedBackend{
		Backend: b,
		Client:  b.Client(),
	}
}
