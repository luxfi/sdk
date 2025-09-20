// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"github.com/holiman/uint256"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/tracing"
)

// StateUpgradeAdapter adapts the new geth StateDB interface (with BalanceChangeReason)
// to the legacy stateupgrade.StateDB interface that doesn't use BalanceChangeReason.
type StateUpgradeAdapter struct {
	*StateDB
}

// AddBalance implements the legacy AddBalance without BalanceChangeReason
func (a *StateUpgradeAdapter) AddBalance(addr common.Address, amount *uint256.Int) {
	// Call the underlying AddBalance with BalanceChangeUnspecified
	a.StateDB.AddBalance(addr, amount, tracing.BalanceChangeUnspecified)
}

// SetNonce implements the legacy SetNonce without NonceChangeReason
func (a *StateUpgradeAdapter) SetNonce(addr common.Address, nonce uint64) {
	// Call the underlying SetNonce with NonceChangeUnspecified
	a.StateDB.SetNonce(addr, nonce, tracing.NonceChangeUnspecified)
}

// SetState implements the stateupgrade.StateDB interface
func (a *StateUpgradeAdapter) SetState(addr common.Address, key, value common.Hash) {
	// Call the underlying SetState which returns the old value but we ignore it
	_ = a.StateDB.SetState(addr, key, value)
}

// SetCode implements the stateupgrade.StateDB interface (no return value)
func (a *StateUpgradeAdapter) SetCode(addr common.Address, code []byte) {
	// Call the underlying SetCode method
	_ = a.StateDB.SetCode(addr, code)
}

// NewStateUpgradeAdapter creates a new adapter for state upgrade compatibility
func NewStateUpgradeAdapter(stateDB *StateDB) *StateUpgradeAdapter {
	return &StateUpgradeAdapter{StateDB: stateDB}
}
