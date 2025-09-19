// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package testutils

import (
	"github.com/holiman/uint256"
	"github.com/luxfi/evm/core/state"
	"github.com/luxfi/evm/precompile/contract"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/tracing"
)

// StateDBAdapter wraps *state.StateDB to implement contract.StateDB
type StateDBAdapter struct {
	*state.StateDB
}

// AddBalance adapts the AddBalance method to match the contract.StateDB interface
func (s *StateDBAdapter) AddBalance(address common.Address, amount *uint256.Int) {
	// Call the underlying StateDB.AddBalance with a default BalanceChangeReason
	s.StateDB.AddBalance(address, amount, tracing.BalanceChangeUnspecified)
}

// SetNonce implements the contract.StateDB interface
func (s *StateDBAdapter) SetNonce(address common.Address, nonce uint64) {
	s.StateDB.SetNonce(address, nonce, tracing.NonceChangeUnspecified)
}

// SetState implements the contract.StateDB interface
func (s *StateDBAdapter) SetState(address common.Address, key, value common.Hash) {
	// The underlying SetState returns the old value, but contract.StateDB expects no return
	_ = s.StateDB.SetState(address, key, value)
}

// GetPredicateStorageSlots implements the contract.StateDB interface
func (s *StateDBAdapter) GetPredicateStorageSlots(address common.Address, index int) ([]byte, bool) {
	// StateDB doesn't have this method, so we return nil, false
	// This should be overridden in test implementations if needed
	return nil, false
}

// WrapStateDB wraps an *state.StateDB to implement contract.StateDB
func WrapStateDB(stateDB *state.StateDB) contract.StateDB {
	return &StateDBAdapter{StateDB: stateDB}
}
