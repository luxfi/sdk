// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package extstate

import (
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/tracing"
	"github.com/holiman/uint256"
)

// PrecompileStateDBAdapter adapts the new geth StateDB interface (with BalanceChangeReason)
// to the legacy precompile contract.StateDB interface that doesn't use BalanceChangeReason.
type PrecompileStateDBAdapter struct {
	*StateDB
}

// AddBalance implements the legacy AddBalance without BalanceChangeReason
func (a *PrecompileStateDBAdapter) AddBalance(addr common.Address, amount *uint256.Int) {
	// Use BalanceChangeUnspecified for precompile balance changes
	a.StateDB.AddBalance(addr, amount, tracing.BalanceChangeUnspecified)
}

// SetNonce implements the legacy SetNonce without NonceChangeReason
func (a *PrecompileStateDBAdapter) SetNonce(addr common.Address, nonce uint64) {
	// Use NonceChangeUnspecified for precompile nonce changes
	a.StateDB.SetNonce(addr, nonce, tracing.NonceChangeUnspecified)
}

// SetState implements the contract.StateDB interface (no return value)
func (a *PrecompileStateDBAdapter) SetState(addr common.Address, key, value common.Hash) {
	// Call the underlying SetState which returns the old value but we ignore it
	_ = a.StateDB.SetState(addr, key, value)
}

// NewPrecompileAdapter creates a new adapter for precompile compatibility
func NewPrecompileAdapter(stateDB *StateDB) *PrecompileStateDBAdapter {
	return &PrecompileStateDBAdapter{StateDB: stateDB}
}