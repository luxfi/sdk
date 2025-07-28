package interfaces

import (
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/holiman/uint256"
)

// StateDB is an EVM database for full state querying.
type StateDB interface {
	CreateAccount(common.Address)

	SubBalance(common.Address, *uint256.Int, ...string) error
	AddBalance(common.Address, *uint256.Int, ...string) error
	GetBalance(common.Address) *uint256.Int

	GetNonce(common.Address) uint64
	SetNonce(common.Address, uint64)

	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte
	SetCode(common.Address, []byte)
	GetCodeSize(common.Address) int

	AddRefund(uint64)
	SubRefund(uint64)
	GetRefund() uint64

	GetCommittedState(common.Address, common.Hash) common.Hash
	GetState(common.Address, common.Hash) common.Hash
	SetState(common.Address, common.Hash, common.Hash)

	GetStorageRoot(addr common.Address) common.Hash

	Suicide(common.Address) bool
	HasSuicided(common.Address) bool

	// Exist reports whether the given account exists in state.
	Exist(common.Address) bool
	// Empty returns whether the given account is empty.
	Empty(common.Address) bool

	PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList)
	AddressInAccessList(addr common.Address) bool
	SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool)
	AddAddressToAccessList(addr common.Address)
	AddSlotToAccessList(addr common.Address, slot common.Hash)

	RevertToSnapshot(int)
	Snapshot() int

	AddLog(*types.Log)
	AddPreimage(common.Hash, []byte)

	ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error

	// Lux specific
	GetLogData() [][]byte
	GetPredicateStorageSlots(address common.Address, index int) ([]byte, bool)
	SetPredicateStorageSlots(address common.Address, predicates [][]byte)
}

// PrecompileManager is the interface for managing precompiles
type PrecompileManager interface {
	GetConfig(height uint64) *PrecompileConfig
}

// PrecompileConfig represents the configuration for precompiles at a given height
type PrecompileConfig struct {
	// Add fields as needed
}