// (c) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package interfaces

import (
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/core/vm"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/params"
	"github.com/luxfi/geth/rpc"
	"github.com/luxfi/evm/commontype"
	"github.com/luxfi/node/ids"
)

// Manager handles validator management
type Manager interface {
	Connected(nodeID ids.NodeID)
	Disconnect(nodeID ids.NodeID) error
}

// ChainConfigInterface represents the chain configuration
type ChainConfigInterface interface {
	// Chain ID and basic config
	ChainID() *big.Int
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
	IsCancun(time uint64) bool
	IsPrague(time uint64) bool
	IsVerkle(time uint64) bool
	
	// Fork config  
	ForkBlock() *big.Int
	
	// EVM configuration
	Rules(num *big.Int, timestamp uint64) params.Rules
}

// ChainContextInterface supports retrieving headers and consensus parameters from the
// current blockchain to be used during transaction processing.
type ChainContextInterface interface {
	// Engine retrieves the chain's consensus engine.
	Engine() Engine

	// GetHeader returns the header corresponding to the hash/number argument pair.
	GetHeader(common.Hash, uint64) *types.Header
}

// ChainReaderInterface defines a small collection of methods needed to access the local
// blockchain.
type ChainReaderInterface interface {
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

	// GetBlock retrieves a block from the database by hash and number.
	GetBlock(hash common.Hash, number uint64) *types.Block

	// GetBlockByNumber retrieves a block from the database by number.
	GetBlockByNumber(number uint64) *types.Block

	// GetTd retrieves the total difficulty from the database by hash and number.
	GetTd(hash common.Hash, number uint64) *big.Int
}

// ChainHeaderReaderInterface defines a small collection of methods needed to access the local
// blockchain during header verification.
type ChainHeaderReaderInterface interface {
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
	
	// GetCoinbaseAt retrieves the coinbase at the given parent header
	GetCoinbaseAt(parent *types.Header) (common.Address, bool, error)
	
	// GetFeeConfigAt retrieves the fee config at the given parent header
	GetFeeConfigAt(parent *types.Header) (commontype.FeeConfig, *big.Int, error)
}

// EngineInterface is an algorithm agnostic consensus engine.
type EngineInterface interface {
	// Author retrieves the Ethereum address of the account that minted the given
	// block, which may be different from the header's coinbase if a consensus
	// engine is based on signatures.
	Author(header *types.Header) (common.Address, error)

	// VerifyHeader checks whether a header conforms to the consensus rules of a
	// given engine.
	VerifyHeader(chain ChainHeaderReader, header *types.Header) error

	// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
	// concurrently. The method returns a quit channel to abort the operations and
	// a results channel to retrieve the async verifications (the order is that of
	// the input slice).
	VerifyHeaders(chain ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error)

	// VerifyUncles verifies that the given block's uncles conform to the consensus
	// rules of a given engine.
	VerifyUncles(chain ChainReader, block *types.Block) error

	// Prepare initializes the consensus fields of a block header according to the
	// rules of a particular engine. The changes are executed inline.
	Prepare(chain ChainHeaderReader, header *types.Header) error

	// Finalize runs any post-transaction state modifications (e.g. block rewards
	// or process withdrawals) but does not assemble the block.
	//
	// Note: The state database might be updated to reflect any consensus rules
	// that happen at finalization (e.g. block rewards).
	Finalize(chain ChainHeaderReader, header *types.Header, state vm.StateDB, body *types.Body) error

	// FinalizeAndAssemble runs any post-transaction state modifications (e.g. block
	// rewards or process withdrawals) and assembles the final block.
	//
	// Note: The block header and state database might be updated to reflect any
	// consensus rules that happen at finalization (e.g. block rewards).
	FinalizeAndAssemble(chain ChainHeaderReader, header *types.Header, state vm.StateDB, body *types.Body,
		receipts []*types.Receipt) (*types.Block, error)

	// Seal generates a new sealing request for the given input block and pushes
	// the result into the given channel.
	//
	// Note, the method returns immediately and will send the result async. More
	// than one result may also be returned depending on the consensus algorithm.
	Seal(chain ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error

	// SealHash returns the hash of a block prior to it being sealed.
	SealHash(header *types.Header) common.Hash

	// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
	// that a new block should have.
	CalcDifficulty(chain ChainHeaderReader, time uint64, parent *types.Header) *big.Int

	// APIs returns the RPC APIs this consensus engine provides.
	APIs(chain ChainHeaderReader) []rpc.API

	// Close terminates any background threads maintained by the consensus engine.
	Close() error
}

// StateProcessor is the interface for processing blocks and managing state transitions
type StateProcessor interface {
	Process(block *types.Block, statedb vm.StateDB, cfg vm.Config) (*types.Receipt, error)
}

// Backend interface for Ethereum-like operations
type Backend interface {
	BlockChain() *BlockChain
	TxPool() *TxPool
}

// BlockChain represents the canonical chain
type BlockChain interface {
	Config() ChainConfig
	CurrentBlock() *types.Header
	GetBlock(hash common.Hash, number uint64) *types.Block
	GetBlockByHash(hash common.Hash) *types.Block
	GetBlockByNumber(number uint64) *types.Block
	GetHeader(hash common.Hash, number uint64) *types.Header
	GetHeaderByHash(hash common.Hash) *types.Header
	GetHeaderByNumber(number uint64) *types.Header
	GetTd(hash common.Hash, number uint64) *big.Int
	StateAt(root common.Hash) (vm.StateDB, error)
	SubscribeChainEvent(ch chan<- ChainEvent) Subscription
	SubscribeChainHeadEvent(ch chan<- ChainHeadEvent) Subscription
	SubscribeLogsEvent(ch chan<- []*types.Log) Subscription
}

// TxPool represents the transaction pool
type TxPool interface {
	Get(hash common.Hash) *types.Transaction
	Add(txs []*types.Transaction) []error
	Pending() map[common.Address][]*types.Transaction
	SubscribeTransactions(ch chan<- NewTxsEvent) Subscription
}

// SubscriptionInterface represents an event subscription
type SubscriptionInterface interface {
	Unsubscribe()
}

// Event types
type ChainEvent struct {
	Block *types.Block
	Hash  common.Hash
	Logs  []*types.Log
}

type ChainHeadEvent struct {
	Block *types.Block
}

type NewTxsEvent struct {
	Txs []*types.Transaction
}

// DatabaseInterface
type DatabaseInterface interface {
	ethdb.Database
}

// StateDatabase wraps access to tries and contract code
type StateDatabase interface {
	// OpenTrie opens the main account trie
	OpenTrie(root common.Hash) (Trie, error)

	// OpenStorageTrie opens the storage trie of an account
	OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash, trie Trie) (Trie, error)

	// CopyTrie returns an independent copy of the given trie
	CopyTrie(Trie) Trie

	// ContractCode retrieves a particular contract's code
	ContractCode(addr common.Address, codeHash common.Hash) ([]byte, error)

	// ContractCodeSize retrieves a particular contracts code's size
	ContractCodeSize(addr common.Address, codeHash common.Hash) (int, error)

	// DiskDB returns the underlying key-value disk database
	DiskDB() ethdb.KeyValueStore

	// TrieDB returns the underlying trie database
	TrieDB() *TrieDB
}

// Trie is the interface for Merkle Patricia tries
type Trie interface {
	// GetKey returns the sha3 preimage of a hashed key
	GetKey([]byte) []byte

	// GetAccount abstracts an account read from the trie
	GetAccount(address common.Address) (*types.StateAccount, error)

	// GetStorage returns the value for key stored in the trie
	GetStorage(addr common.Address, key []byte) ([]byte, error)

	// UpdateAccount abstracts an account write to the trie
	UpdateAccount(address common.Address, account *types.StateAccount) error

	// UpdateStorage associates key with value in the trie
	UpdateStorage(addr common.Address, key, value []byte) error

	// DeleteAccount abstracts an account deletion from the trie
	DeleteAccount(address common.Address) error

	// DeleteStorage removes any existing value for key from the trie
	DeleteStorage(addr common.Address, key []byte) error

	// GetAccount returns the account associated with the address
	TryGetAccount(address common.Address) (*types.StateAccount, error)

	// Hash returns the root hash of the trie
	Hash() common.Hash

	// Commit collects all dirty nodes in the trie and replaces them with the
	// corresponding node hash
	Commit(collectLeaf bool) (common.Hash, *TrieNodeSet, error)

	// NodeIterator returns an iterator that returns nodes of the trie
	NodeIterator(startKey []byte) NodeIterator

	// Close releases associated resources
	Close() error
}

// TrieDB is the interface for trie databases
type TrieDB interface {
	// Reference adds a new reference from a parent node to a child node
	Reference(child common.Hash, parent common.Hash)

	// Dereference removes an existing reference from a parent node to a child node
	Dereference(child common.Hash, parent common.Hash)

	// Cap iteratively flushes old but still referenced trie nodes until the total
	// memory usage goes below the given threshold
	Cap(limit common.StorageSize) error

	// Commit iterates over all the children of a particular node, writes them out
	// to disk
	Commit(node common.Hash, report bool) error

	// Close flushes the dangling preimages to disk and closes the trie database
	Close() error
}

// NodeIterator is an iterator to traverse the trie
type NodeIterator interface {
	Next(bool) bool
	Error() error
	Hash() common.Hash
	Parent() common.Hash
	Path() []byte
	NodeBlob() []byte
	LeafKey() []byte
	LeafBlob() []byte
	LeafProof() [][]byte
	AddResolver(NodeResolver)
}

// NodeResolver is used to resolve trie nodes from a NodeIterator
type NodeResolver func(owner common.Hash, path []byte, hash common.Hash) []byte

// TrieNodeSet represents a set of trie nodes
type TrieNodeSet struct {
	// Implementation details
}
