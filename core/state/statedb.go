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
// Copyright 2014 The go-ethereum Authors
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

// Package state provides a caching layer atop the Ethereum state trie.
package state

import (
	"math/big"
	
	ethstate "github.com/luxfi/geth/core/state"
	"github.com/luxfi/geth/core/state/snapshot"
	"github.com/luxfi/geth/core/tracing"
	ethtypes "github.com/luxfi/geth/core/types"
	ethparams "github.com/luxfi/geth/params"
	"github.com/luxfi/geth/common"
	"github.com/holiman/uint256"
	"github.com/luxfi/evm/core/types"
	"github.com/luxfi/evm/params"
)

// StateDB wraps go-ethereum's StateDB with minimal extensions
type StateDB struct {
	*ethstate.StateDB
	thash common.Hash
}

// New creates a new state from a given trie.
func New(root common.Hash, db Database, snaps *snapshot.Tree) (*StateDB, error) {
	// Note: geth's New function doesn't take snapshot.Tree anymore
	ethStateDB, err := ethstate.New(root, db)
	if err != nil {
		return nil, err
	}
	return &StateDB{StateDB: ethStateDB}, nil
}

// GetTxHash returns the current transaction hash
func (s *StateDB) GetTxHash() common.Hash {
	return s.thash
}

// SetTxContext sets the current transaction hash and index
func (s *StateDB) SetTxContext(thash common.Hash, ti int) {
	s.thash = thash
	s.StateDB.SetTxContext(thash, ti)
}

// AddBalance adds amount to the account associated with addr.
// This satisfies the vm.StateDB interface which expects 2 parameters
func (s *StateDB) AddBalance(addr common.Address, amount *uint256.Int) {
	// Use the default balance change reason
	s.StateDB.AddBalance(addr, amount, tracing.BalanceChangeUnspecified)
}

// SubBalance subtracts amount from the account associated with addr.
// This satisfies the vm.StateDB interface which expects 2 parameters
func (s *StateDB) SubBalance(addr common.Address, amount *uint256.Int) {
	// Use the default balance change reason
	s.StateDB.SubBalance(addr, amount, tracing.BalanceChangeUnspecified)
}

// AddBalanceMultiCoin adds amount to the account's balance for the specified coinID
func (s *StateDB) AddBalanceMultiCoin(addr common.Address, coinID common.Hash, amount *big.Int) {
	// This is a placeholder implementation
	// Multi-coin functionality is not implemented in go-ethereum
	// For now, we'll just log a warning
}

// SubBalanceMultiCoin subtracts amount from the account's balance for the specified coinID
func (s *StateDB) SubBalanceMultiCoin(addr common.Address, coinID common.Hash, amount *big.Int) {
	// This is a placeholder implementation
	// Multi-coin functionality is not implemented in go-ethereum
	// For now, we'll just log a warning
}

// GetBalanceMultiCoin retrieves the balance of account for the specified coinID
func (s *StateDB) GetBalanceMultiCoin(addr common.Address, coinID common.Hash) *big.Int {
	// This is a placeholder implementation
	// Multi-coin functionality is not implemented in go-ethereum
	// For now, return zero
	return new(big.Int)
}

// GetCommittedStateAP1 retrieves the committed state value for the given address and hash
// AP1 stands for Apricot Phase 1 - this is an Avalanche-specific method
func (s *StateDB) GetCommittedStateAP1(addr common.Address, hash common.Hash) common.Hash {
	// For now, delegate to GetCommittedState
	// This may need special handling for Apricot Phase 1 compatibility
	return s.StateDB.GetCommittedState(addr, hash)
}

// SetBalance sets the balance of the account associated with addr.
func (s *StateDB) SetBalance(addr common.Address, amount *uint256.Int) {
	// Forward to ethereum's StateDB with BalanceChangeUnspecified reason
	s.StateDB.SetBalance(addr, amount, tracing.BalanceChangeUnspecified)
}

// SetNonce sets the nonce of the account
// This wrapper provides compatibility with the vm.StateDB interface
func (s *StateDB) SetNonce(addr common.Address, nonce uint64) {
	// Use the default nonce change reason
	s.StateDB.SetNonce(addr, nonce, tracing.NonceChangeUnspecified)
}

// AddLog adds a log to the state
// This wrapper provides compatibility with the vm.StateDB interface
func (s *StateDB) AddLog(addr common.Address, topics []common.Hash, data []byte, blockNumber uint64) {
	// Create an ethereum Log type
	log := &ethtypes.Log{
		Address:     addr,
		Topics:      topics,
		Data:        data,
		BlockNumber: blockNumber,
	}
	s.StateDB.AddLog(log)
}

// GetLogs returns the logs generated by the current transaction
// This wrapper provides compatibility with luxfi/evm types
func (s *StateDB) GetLogs(txHash common.Hash, blockNumber uint64, blockHash common.Hash) []*types.Log {
	// Get ethereum logs
	ethLogs := s.StateDB.GetLogs(txHash, blockNumber, blockHash, 0)
	
	// Convert to luxfi/evm logs
	logs := make([]*types.Log, len(ethLogs))
	for i, ethLog := range ethLogs {
		logs[i] = &types.Log{
			Address:     ethLog.Address,
			Topics:      ethLog.Topics,
			Data:        ethLog.Data,
			BlockNumber: ethLog.BlockNumber,
			TxHash:      ethLog.TxHash,
			TxIndex:     ethLog.TxIndex,
			BlockHash:   ethLog.BlockHash,
			Index:       ethLog.Index,
			Removed:     ethLog.Removed,
		}
	}
	return logs
}

// GetLogData returns the raw log data in the format expected by vm.StateDB
func (s *StateDB) GetLogData() (topics [][]common.Hash, data [][]byte) {
	// Get all logs from ethereum StateDB
	ethLogs := s.StateDB.GetLogs(s.thash, 0, common.Hash{}, 0)
	
	topics = make([][]common.Hash, len(ethLogs))
	data = make([][]byte, len(ethLogs))
	
	for i, log := range ethLogs {
		topics[i] = log.Topics
		data[i] = log.Data
	}
	
	return topics, data
}

// GetPredicateStorageSlots returns the predicate storage slots for the given address and index
// This is an Avalanche-specific feature for stateful precompiles
func (s *StateDB) GetPredicateStorageSlots(address common.Address, index int) ([]byte, bool) {
	// This is a placeholder implementation
	// Predicate storage is not implemented in go-ethereum
	return nil, false
}

// SetPredicateStorageSlots sets the predicate storage slots for the given address
// This is an Avalanche-specific feature for stateful precompiles
func (s *StateDB) SetPredicateStorageSlots(address common.Address, predicates [][]byte) {
	// This is a placeholder implementation
	// Predicate storage is not implemented in go-ethereum
}

// Prepare implements vm.StateDB
// This wrapper converts luxfi params.Rules to ethereum params.Rules
func (s *StateDB) Prepare(rules params.Rules, sender, coinbase common.Address, dst *common.Address, precompiles []common.Address, list types.AccessList) {
	// Convert luxfi AccessList to ethereum AccessList
	ethAccessList := make(ethtypes.AccessList, len(list))
	for i, item := range list {
		ethAccessList[i] = ethtypes.AccessTuple{
			Address:     item.Address,
			StorageKeys: item.StorageKeys,
		}
	}
	
	// Convert luxfi params.Rules to ethereum params.Rules
	ethRules := ethparams.Rules{
		ChainID:                     rules.ChainID,
		IsHomestead:                 rules.IsHomestead,
		IsEIP150:                   rules.IsEIP150,
		IsEIP155:                   rules.IsEIP155,
		IsEIP158:                   rules.IsEIP158,
		IsByzantium:                rules.IsByzantium,
		IsConstantinople:           rules.IsConstantinople,
		IsPetersburg:               rules.IsPetersburg,
		IsIstanbul:                 rules.IsIstanbul,
		IsBerlin:                   rules.IsBerlin,
		IsLondon:                   rules.IsLondon,
		IsMerge:                    rules.IsMerge,
		IsShanghai:                 rules.IsShanghai,
		IsCancun:                   rules.IsCancun,
		IsPrague:                   rules.IsPrague,
		IsVerkle:                   rules.IsVerkle,
		// Note: EIP-specific fields may not exist in ethereum's Rules struct
		// They are typically derived from the network upgrade flags
	}
	
	s.StateDB.Prepare(ethRules, sender, coinbase, dst, precompiles, ethAccessList)
}

// Logs returns all logs that have been added to the state
func (s *StateDB) Logs() []*types.Log {
	// Get ethereum logs from underlying StateDB
	ethLogs := s.StateDB.Logs()
	
	// Convert to luxfi/evm logs
	logs := make([]*types.Log, len(ethLogs))
	for i, ethLog := range ethLogs {
		logs[i] = &types.Log{
			Address:     ethLog.Address,
			Topics:      ethLog.Topics,
			Data:        ethLog.Data,
			BlockNumber: ethLog.BlockNumber,
			TxHash:      ethLog.TxHash,
			TxIndex:     ethLog.TxIndex,
			BlockHash:   ethLog.BlockHash,
			Index:       ethLog.Index,
			Removed:     ethLog.Removed,
		}
	}
	return logs
}

// SelfDestruct marks the given account as self-destructed.
// This method satisfies the vm.StateDB interface which expects no return value
func (s *StateDB) SelfDestruct(addr common.Address) {
	// ethereum's SelfDestruct returns a balance, but we ignore it
	// to match the vm.StateDB interface
	_ = s.StateDB.SelfDestruct(addr)
}

// Selfdestruct6780 marks the given account as self-destructed per EIP-6780.
// This method satisfies the vm.StateDB interface which expects no return value
func (s *StateDB) Selfdestruct6780(addr common.Address) {
	// ethereum's SelfDestruct6780 returns (balance, wasDestroyed), but we ignore them
	// to match the vm.StateDB interface
	_, _ = s.StateDB.SelfDestruct6780(addr)
}

// SetCode sets the code for the given account address.
// This method satisfies the vm.StateDB interface which expects no return value
func (s *StateDB) SetCode(addr common.Address, code []byte) {
	// ethereum's SetCode returns the old code, but we ignore it
	// to match the vm.StateDB interface
	_ = s.StateDB.SetCode(addr, code)
}

// SetState sets the state value for the given address and key.
// This method satisfies the vm.StateDB interface which expects no return value
func (s *StateDB) SetState(addr common.Address, key, value common.Hash) {
	// ethereum's SetState returns the old value, but we ignore it
	// to match the vm.StateDB interface
	_ = s.StateDB.SetState(addr, key, value)
}

// Commit writes the state to the underlying storage trie.
// This wrapper handles both 2 and 3 parameter versions
func (s *StateDB) Commit(num uint64, deleteEmptyObjects bool) (common.Hash, error) {
	// ethereum's StateDB.Commit now takes 3 parameters
	return s.StateDB.Commit(num, deleteEmptyObjects, false)
}

