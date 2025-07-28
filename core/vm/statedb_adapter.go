// (c) 2019-2024, Lux Industries, Inc.
// All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	ethtypes "github.com/luxfi/geth/core/types"
	"github.com/luxfi/evm/precompile/contract"
	"github.com/luxfi/geth/common"
	"github.com/holiman/uint256"
)

// stateDBAdapter adapts core/vm StateDB to precompile/contract StateDB interface
type stateDBAdapter struct {
	StateDB
}

// NewStateDBAdapter creates a new StateDB adapter
func NewStateDBAdapter(db StateDB) contract.StateDB {
	return &stateDBAdapter{StateDB: db}
}

// AddLog implements contract.StateDB
func (s *stateDBAdapter) AddLog(log *ethtypes.Log) {
	// Convert from types.Log to the AddLog parameters expected by StateDB
	s.StateDB.AddLog(log.Address, log.Topics, log.Data, log.BlockNumber)
}

// GetLogData implements contract.StateDB
func (s *stateDBAdapter) GetLogData() (topics [][]common.Hash, data [][]byte) {
	return s.StateDB.GetLogData()
}

// GetBalance implements contract.StateDB
func (s *stateDBAdapter) GetBalance(addr common.Address) *uint256.Int {
	return s.StateDB.GetBalance(addr)
}

// AddBalance implements contract.StateDB
func (s *stateDBAdapter) AddBalance(addr common.Address, amount *uint256.Int) {
	s.StateDB.AddBalance(addr, amount)
}

// GetState implements contract.StateDB
func (s *stateDBAdapter) GetState(addr common.Address, key common.Hash) common.Hash {
	return s.StateDB.GetState(addr, key)
}

// SetState implements contract.StateDB
func (s *stateDBAdapter) SetState(addr common.Address, key, value common.Hash) {
	s.StateDB.SetState(addr, key, value)
}

// GetNonce implements contract.StateDB
func (s *stateDBAdapter) GetNonce(addr common.Address) uint64 {
	return s.StateDB.GetNonce(addr)
}

// SetNonce implements contract.StateDB
func (s *stateDBAdapter) SetNonce(addr common.Address, nonce uint64) {
	s.StateDB.SetNonce(addr, nonce)
}

// CreateAccount implements contract.StateDB
func (s *stateDBAdapter) CreateAccount(addr common.Address) {
	s.StateDB.CreateAccount(addr)
}

// Exist implements contract.StateDB
func (s *stateDBAdapter) Exist(addr common.Address) bool {
	return s.StateDB.Exist(addr)
}

// GetPredicateStorageSlots implements contract.StateDB
func (s *stateDBAdapter) GetPredicateStorageSlots(address common.Address, index int) ([]byte, bool) {
	return s.StateDB.GetPredicateStorageSlots(address, index)
}

// SetPredicateStorageSlots implements contract.StateDB
func (s *stateDBAdapter) SetPredicateStorageSlots(address common.Address, predicates [][]byte) {
	s.StateDB.SetPredicateStorageSlots(address, predicates)
}

// GetTxHash implements contract.StateDB
func (s *stateDBAdapter) GetTxHash() common.Hash {
	return s.StateDB.GetTxHash()
}

// Snapshot implements contract.StateDB
func (s *stateDBAdapter) Snapshot() int {
	return s.StateDB.Snapshot()
}

// RevertToSnapshot implements contract.StateDB
func (s *stateDBAdapter) RevertToSnapshot(snapshot int) {
	s.StateDB.RevertToSnapshot(snapshot)
}