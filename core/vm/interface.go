// Copyright 2016 The go-ethereum Authors
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

package vm

import (
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/vm"
	"github.com/luxfi/geth/params"
	"github.com/luxfi/evm/core/state"
	luxparams "github.com/luxfi/evm/params"
)

// StateDB is an EVM database for full state querying.
type StateDB interface {
	vm.StateDB
}

// EVMLogger is used to collect execution traces from an EVM transaction
// execution. CaptureState is called for each step of the VM with the
// current VM state.
// Note that reference types are actual VM data structures; make copies
// if you need to retain them beyond the current call.
type EVMLogger interface {
	// Transaction level
	CaptureTxStart(gasLimit uint64)
	CaptureTxEnd(restGas uint64)
	// Top call frame
	CaptureStart(env *EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int)
	CaptureEnd(output []byte, gasUsed uint64, err error)
	// Rest of call frames
	CaptureEnter(typ OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int)
	CaptureExit(output []byte, gasUsed uint64, err error)
	// Opcode level
	CaptureState(pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, rData []byte, depth int, err error)
	CaptureFault(pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, depth int, err error)
}

// BlockContext provides the EVM with auxiliary information. Once provided
// it shouldn't be modified.
type BlockContext = vm.BlockContext

// TxContext provides the EVM with information about a transaction.
// All fields can change between transactions.
type TxContext = vm.TxContext

// EVM is the Ethereum Virtual Machine base object and provides
// the necessary tools to run a contract on the given state with
// the provided context. It should be noted that any error
// generated through any of the calls should be considered a
// revert-state-and-consume-all-gas operation, no checks on
// specific errors should ever be performed. The interpreter makes
// sure that any errors generated are to be considered faulty code.
//
// The EVM should never be reused and is not thread safe.
type EVM struct {
	*vm.EVM
}

// Config are the configuration options for the Interpreter
type Config = vm.Config

// ScopeContext contains the things that are per-call, such as stack and memory,
// but not transients like pc and gas
type ScopeContext = vm.ScopeContext

// OpCode is an EVM opcode
type OpCode = vm.OpCode

// ActivateableEips returns the list of EIPs that can be activated
func ActivateableEips() []string {
	return vm.ActivateableEips()
}

// NewEVM creates a new EVM instance.
func NewEVM(blockCtx BlockContext, statedb StateDB, chainConfig *params.ChainConfig, config Config) *EVM {
	// Create an empty TxContext for backward compatibility
	txCtx := vm.TxContext{}
	evm := vm.NewEVM(blockCtx, txCtx, statedb, chainConfig, config)
	return &EVM{EVM: evm}
}

// ConvertChainConfig converts luxfi chainConfig to ethereum chainConfig
func ConvertChainConfig(cfg *luxparams.ChainConfig) *params.ChainConfig {
	if cfg == nil {
		return nil
	}
	
	ethConfig := &params.ChainConfig{
		ChainID:             cfg.ChainID,
		HomesteadBlock:      cfg.HomesteadBlock,
		EIP150Block:         cfg.EIP150Block,
		EIP155Block:         cfg.EIP155Block,
		EIP158Block:         cfg.EIP158Block,
		ByzantiumBlock:      cfg.ByzantiumBlock,
		ConstantinopleBlock: cfg.ConstantinopleBlock,
		PetersburgBlock:     cfg.PetersburgBlock,
		IstanbulBlock:       cfg.IstanbulBlock,
		MuirGlacierBlock:    cfg.MuirGlacierBlock,
		BerlinBlock:         cfg.BerlinBlock,
		LondonBlock:         cfg.LondonBlock,
	}
	
	// Convert time-based forks to block-based for compatibility
	if cfg.ShanghaiTime != nil && *cfg.ShanghaiTime == 0 {
		ethConfig.ShanghaiTime = cfg.ShanghaiTime
	}
	if cfg.CancunTime != nil && *cfg.CancunTime == 0 {
		ethConfig.CancunTime = cfg.CancunTime
	}
	
	return ethConfig
}

// NewEVMWithStateDB creates a new EVM with luxfi chainConfig and statedb
func NewEVMWithStateDB(blockCtx BlockContext, txCtx TxContext, statedb *state.StateDB, chainConfig *luxparams.ChainConfig, config Config) *EVM {
	ethConfig := ConvertChainConfig(chainConfig)
	evm := vm.NewEVM(blockCtx, txCtx, statedb, ethConfig, config)
	return &EVM{EVM: evm}
}