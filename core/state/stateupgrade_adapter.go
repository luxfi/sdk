// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/tracing"
	"github.com/holiman/uint256"
)

// StateUpgradeAdapter adapts the new geth StateDB interface (with BalanceChangeReason)
// to the legacy stateupgrade.StateDB interface that doesn't use BalanceChangeReason.
type StateUpgradeAdapter struct {
	*StateDB
}

// AddBalance implements the legacy AddBalance without BalanceChangeReason
func (a *StateUpgradeAdapter) AddBalance(addr common.Address, amount *uint256.Int) {
	// Use BalanceChangeUnspecified for state upgrade balance changes
	a.StateDB.AddBalance(addr, amount, tracing.BalanceChangeUnspecified)
}

// SetNonce implements the legacy SetNonce without NonceChangeReason
func (a *StateUpgradeAdapter) SetNonce(addr common.Address, nonce uint64) {
	// Use NonceChangeUnspecified for state upgrade nonce changes
	a.StateDB.SetNonce(addr, nonce, tracing.NonceChangeUnspecified)
}

// SetState implements the stateupgrade.StateDB interface
func (a *StateUpgradeAdapter) SetState(addr common.Address, key, value common.Hash) {
	// Call the underlying SetState which returns the old value but we ignore it
	_ = a.StateDB.SetState(addr, key, value)
}

// SetCode implements the stateupgrade.StateDB interface (no return value)
func (a *StateUpgradeAdapter) SetCode(addr common.Address, code []byte) {
	// Call the underlying SetCode which returns the previous code but we ignore it
	_ = a.StateDB.SetCode(addr, code)
}

// NewStateUpgradeAdapter creates a new adapter for state upgrade compatibility
func NewStateUpgradeAdapter(stateDB *StateDB) *StateUpgradeAdapter {
	return &StateUpgradeAdapter{StateDB: stateDB}
}