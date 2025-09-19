package extstate

import (
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/core/tracing"
	"github.com/holiman/uint256"
	"github.com/luxfi/evm/core/state"
	"github.com/luxfi/evm/precompile/contract"
)

// ExtState wraps a StateDB for extended state operations
type ExtState struct {
	stateDB *state.StateDB
}

// New creates a new ExtState instance
func New(stateDB *state.StateDB) *ExtState {
	return &ExtState{
		stateDB: stateDB,
	}
}

// PrecompileAdapter adapts ExtState for precompile usage
type PrecompileAdapter struct {
	extState *ExtState
}

// NewPrecompileAdapter creates a new PrecompileAdapter
func NewPrecompileAdapter(extState *ExtState) contract.StateDB {
	return &PrecompileAdapter{
		extState: extState,
	}
}

// GetState gets state value
func (p *PrecompileAdapter) GetState(addr common.Address, key common.Hash) common.Hash {
	if p.extState.stateDB != nil {
		return p.extState.stateDB.GetState(addr, key)
	}
	return common.Hash{}
}

// SetState sets state value
func (p *PrecompileAdapter) SetState(addr common.Address, key common.Hash, value common.Hash) {
	if p.extState.stateDB != nil {
		p.extState.stateDB.SetState(addr, key, value)
	}
}

// GetNonce gets account nonce
func (p *PrecompileAdapter) GetNonce(addr common.Address) uint64 {
	if p.extState.stateDB != nil {
		return p.extState.stateDB.GetNonce(addr)
	}
	return 0
}

// SetNonce sets account nonce
func (p *PrecompileAdapter) SetNonce(addr common.Address, nonce uint64) {
	if p.extState.stateDB != nil {
		p.extState.stateDB.SetNonce(addr, nonce, tracing.NonceChangeUnspecified)
	}
}

// GetBalance gets account balance
func (p *PrecompileAdapter) GetBalance(addr common.Address) *uint256.Int {
	if p.extState.stateDB != nil {
		return p.extState.stateDB.GetBalance(addr)
	}
	return new(uint256.Int)
}

// AddBalance adds balance to an account
func (p *PrecompileAdapter) AddBalance(addr common.Address, amount *uint256.Int) {
	if p.extState.stateDB != nil && amount != nil {
		p.extState.stateDB.AddBalance(addr, amount, tracing.BalanceChangeUnspecified)
	}
}

// CreateAccount creates a new account
func (p *PrecompileAdapter) CreateAccount(addr common.Address) {
	if p.extState.stateDB != nil {
		p.extState.stateDB.CreateAccount(addr)
	}
}

// Exist returns whether an account exists
func (p *PrecompileAdapter) Exist(addr common.Address) bool {
	if p.extState.stateDB != nil {
		return p.extState.stateDB.Exist(addr)
	}
	return false
}

// AddLog adds a log entry
func (p *PrecompileAdapter) AddLog(log *types.Log) {
	if p.extState.stateDB != nil {
		p.extState.stateDB.AddLog(log)
	}
}

// GetPredicateStorageSlots gets predicate storage slots
func (p *PrecompileAdapter) GetPredicateStorageSlots(address common.Address, index int) ([]byte, bool) {
	// Not implemented in base StateDB
	return nil, false
}

// GetTxHash gets the current transaction hash
func (p *PrecompileAdapter) GetTxHash() common.Hash {
	// Not implemented in base StateDB
	return common.Hash{}
}

// Snapshot returns the current revision number
func (p *PrecompileAdapter) Snapshot() int {
	if p.extState.stateDB != nil {
		return p.extState.stateDB.Snapshot()
	}
	return 0
}

// RevertToSnapshot reverts to a previous snapshot
func (p *PrecompileAdapter) RevertToSnapshot(snapshot int) {
	if p.extState.stateDB != nil {
		p.extState.stateDB.RevertToSnapshot(snapshot)
	}
}