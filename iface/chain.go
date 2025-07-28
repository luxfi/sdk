// (c) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package iface provides neutral interfaces to break import cycles.
// This package should have minimal dependencies and serve as the
// contract between different layers of the system.
package iface

import (
	"fmt"
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/evm/core/types"
	"github.com/holiman/uint256"
)

// ChainHeaderReader defines methods needed to access the local blockchain during header verification.
type ChainHeaderReader interface {
	// Config retrieves the blockchain's chain configuration.
	Config() ChainConfig
	
	// CurrentHeader retrieves the current header from the local chain.
	CurrentHeader() *types.Header
	
	// GetHeader retrieves a block header from the database by hash and number.
	GetHeader(hash common.Hash, number uint64) *types.Header
	
	// GetHeaderByNumber retrieves a block header from the database by number.
	GetHeaderByNumber(number uint64) *types.Header
	
	// GetHeaderByHash retrieves a block header from the database by its hash.
	GetHeaderByHash(hash common.Hash) *types.Header
	
	// GetTd retrieves the total difficulty from the database by hash and number.
	GetTd(hash common.Hash, number uint64) *big.Int
	
	// GetCoinbaseAt returns the configured coinbase address at the given timestamp
	GetCoinbaseAt(timestamp uint64) common.Address
	
	// GetFeeConfigAt returns the fee configuration at the given timestamp
	GetFeeConfigAt(timestamp uint64) (FeeConfig, error)
}

// ChainReader defines a small collection of methods needed to access the local
// blockchain during header and/or uncle verification.
type ChainReader interface {
	ChainHeaderReader
	
	// GetBlock retrieves a block from the database by hash and number.
	GetBlock(hash common.Hash, number uint64) *types.Block
}

// ChainConfig represents the chain configuration
type ChainConfig interface {
	// Chain identification
	GetChainID() *big.Int
	
	// Fork activation checks
	IsHomestead(num *big.Int) bool
	IsEIP150(num *big.Int) bool
	IsEIP155(num *big.Int) bool
	IsEIP158(num *big.Int) bool
	IsByzantium(num *big.Int) bool
	IsConstantinople(num *big.Int) bool
	IsPetersburg(num *big.Int) bool
	IsIstanbul(num *big.Int) bool
	IsBerlin(num *big.Int) bool
	IsLondon(num *big.Int) bool
	IsShanghai(num *big.Int, time uint64) bool
	IsCancun(time uint64) bool
	
	// Lux-specific methods  
	// For v2.0.0, all upgrades are active at genesis
	IsGenesis(time uint64) bool
	AllowedFeeRecipients() bool
	GenesisRules(blockNum *big.Int, timestamp uint64) GenesisRules
	
	// AsGeth returns the underlying geth ChainConfig for compatibility
	AsGeth() interface{}
}

// GenesisRules defines the Genesis-specific fork rules
type GenesisRules interface {
	IsGenesis() bool
	
	// Precompile access
	PredicatersExist() bool
	PredicaterExists(addr common.Address) bool
	GetActivePrecompiles() map[common.Address]interface{}
	GetPredicaters() map[common.Address]interface{}
	GetAccepterPrecompiles() map[common.Address]interface{}
}

// FeeConfig represents the fee configuration
type FeeConfig interface {
	// Basic getters for fee configuration
	GetGasLimit() *big.Int
	GetTargetBlockRate() uint64
	GetMinBaseFee() *big.Int
	GetTargetGas() *big.Int
	GetBaseFeeChangeDenominator() *big.Int
	GetMinBlockGasCost() *big.Int
	GetMaxBlockGasCost() *big.Int
	GetBlockGasCostStep() *big.Int
}

// PrecompileConfig represents a precompile configuration
type PrecompileConfig interface {
	Address() common.Address
	IsDisabled() bool
	Timestamp() *uint64
}

// PrecompileRegistry manages precompile modules
type PrecompileRegistry interface {
	// GetPrecompileModule returns a precompile module by key
	GetPrecompileModule(key string) (PrecompileModule, bool)
	
	// GetPrecompileModuleByAddress returns a precompile module by address
	GetPrecompileModuleByAddress(address common.Address) (PrecompileModule, bool)
	
	// RegisteredModules returns all registered modules
	RegisteredModules() []PrecompileModule
}

// PrecompileModule represents a precompile module
type PrecompileModule interface {
	// Address returns the address of the precompile
	Address() common.Address
	
	// Contract returns the precompile contract
	Contract() interface{}
	
	// Configurator returns the configurator for this precompile
	Configurator() interface{}
	
	// DefaultConfig returns the default config for this precompile
	DefaultConfig() interface{}
	
	// MakeConfig creates a new config instance
	MakeConfig() interface{}
	
	// ConfigKey returns the configuration key for this module
	ConfigKey() string
}

// ChainContext provides consensus context  
type ChainContext struct {
	NetworkID uint32
	SubnetID  SubnetID
	ChainID   ChainID
	NodeID    NodeID

	// Node version
	AppVersion uint32

	// Chain configuration
	ChainDataDir string
	
	// Validator state
	ValidatorState ValidatorState
}

// NodeID is a 32-byte identifier for nodes
type NodeID [32]byte

// String returns the string representation of a NodeID
func (id NodeID) String() string {
	return fmt.Sprintf("%x", id[:])
}

// SubnetID is a 32-byte subnet identifier  
type SubnetID [32]byte

// String returns the string representation of a SubnetID
func (id SubnetID) String() string {
	return fmt.Sprintf("%x", id[:])
}

// ChainID is a 32-byte chain identifier
type ChainID [32]byte

// String returns the string representation of a ChainID
func (id ChainID) String() string {
	return fmt.Sprintf("%x", id[:])
}

// StateDB is an EVM database for full state querying
type StateDB interface {
	GetBalance(common.Address) *uint256.Int
	GetNonce(common.Address) uint64
	GetCode(common.Address) []byte
	GetState(common.Address, common.Hash) common.Hash
	Exist(common.Address) bool
	Empty(common.Address) bool
}

// Bits provides a bit set interface
type Bits interface {
	// Add adds a bit to the set
	Add(i int)
	
	// Contains checks if a bit is in the set
	Contains(i int) bool
	
	// Remove removes a bit from the set
	Remove(i int)
	
	// Clear clears all bits
	Clear()
	
	// Len returns the number of bits set
	Len() int
	
	// Bytes returns the byte representation
	Bytes() []byte
}

// Engine is an algorithm agnostic consensus engine.
type Engine interface {
	// Author retrieves the Ethereum address of the account that minted the given block.
	Author(header *types.Header) (common.Address, error)

	// VerifyHeader checks whether a header conforms to the consensus rules of a given engine.
	VerifyHeader(chain ChainHeaderReader, header *types.Header, seal bool) error

	// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers concurrently.
	VerifyHeaders(chain ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error)

	// VerifyUncles verifies that the given block's uncles conform to the consensus rules.
	VerifyUncles(chain ChainReader, block *types.Block) error

	// Prepare initializes the consensus fields of a block header according to the rules.
	Prepare(chain ChainHeaderReader, header *types.Header) error

	// Finalize runs any post-transaction state modifications and assembles the final block.
	Finalize(chain ChainHeaderReader, header *types.Header, state StateDB, txs []*types.Transaction,
		uncles []*types.Header) (*types.Block, error)

	// FinalizeAndAssemble runs any post-transaction state modifications and assembles the final block.
	FinalizeAndAssemble(chain ChainHeaderReader, header *types.Header, state StateDB, txs []*types.Transaction,
		uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error)

	// Seal generates a new sealing request for the given input block and pushes it to the sealer.
	Seal(chain ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error

	// SealHash returns the hash of a block prior to it being sealed.
	SealHash(header *types.Header) common.Hash

	// CalcDifficulty is the difficulty adjustment algorithm.
	CalcDifficulty(chain ChainHeaderReader, time uint64, parent *types.Header) *big.Int

	// Close terminates any background threads maintained by the consensus engine.
	Close() error
}