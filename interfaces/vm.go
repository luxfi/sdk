package interfaces

import (
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
)

// EVM is the interface for the Ethereum Virtual Machine
type EVM interface {
	// Context returns the EVM context
	Context() VMContext
	
	// Call executes the contract with the given input
	Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error)
	
	// CallCode executes the contract with the caller's context
	CallCode(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error)
	
	// DelegateCall executes the contract with the caller's context and storage
	DelegateCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error)
	
	// StaticCall executes the contract with read-only access
	StaticCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error)
	
	// Create creates a new contract
	Create(caller ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error)
	
	// Create2 creates a new contract with deterministic address
	Create2(caller ContractRef, code []byte, gas uint64, endowment *big.Int, salt *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error)
}

// VMContext provides the context for EVM execution
type VMContext interface {
	// CanTransfer checks if the account has enough balance
	CanTransfer(db StateDB, addr common.Address, amount *big.Int) bool
	
	// Transfer transfers amount from one account to another
	Transfer(db StateDB, sender, recipient common.Address, amount *big.Int) error
	
	// GetHash returns the hash of a block by number
	GetHash(n uint64) common.Hash
	
	// Message information
	Origin() common.Address
	GasPrice() *big.Int
	BlockNumber() *big.Int
	Time() uint64
	Difficulty() *big.Int
	GasLimit() uint64
	Coinbase() common.Address
}

// ContractRef is the interface for a contract caller
type ContractRef interface {
	Address() common.Address
}

// TxContext provides context about the transaction
type TxContext struct {
	Origin   common.Address
	GasPrice *big.Int
}

// BlockContext provides context about the block
type BlockContext struct {
	CanTransfer func(StateDB, common.Address, *big.Int) bool
	Transfer    func(StateDB, common.Address, common.Address, *big.Int) error
	GetHash     func(uint64) common.Hash
	Coinbase    common.Address
	GasLimit    uint64
	BlockNumber *big.Int
	Time        uint64
	Difficulty  *big.Int
	BaseFee     *big.Int
}

// Message represents a transaction message
type Message interface {
	From() common.Address
	To() *common.Address
	GasPrice() *big.Int
	GasFeeCap() *big.Int
	GasTipCap() *big.Int
	Gas() uint64
	Value() *big.Int
	Nonce() uint64
	IsFake() bool
	Data() []byte
	AccessList() types.AccessList
}