// Package interfaces provides common interfaces to break import cycles
package interfaces

import (
	"math/big"
	"time"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
)

// MockableTimer is an interface for a timer that can be mocked
type MockableTimer interface {
	Time() time.Time
}

// ChainHeaderReader defines methods needed to access the local blockchain during header verification.
type ChainHeaderReader interface {
	// Config retrieves the blockchain's chain configuration.
	Config() *ChainConfig

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
}

// ChainReader defines a small collection of methods needed to access the local
// blockchain during header and/or uncle verification.
type ChainReader interface {
	ChainHeaderReader

	// GetBlock retrieves a block from the database by hash and number.
	GetBlock(hash common.Hash, number uint64) *types.Block
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
		uncles []*types.Header, withdrawals []*types.Withdrawal) (*types.Block, error)

	// FinalizeAndAssemble runs any post-transaction state modifications and assembles the final block.
	FinalizeAndAssemble(chain ChainHeaderReader, header *types.Header, state StateDB, txs []*types.Transaction,
		uncles []*types.Header, receipts []*types.Receipt, withdrawals []*types.Withdrawal) (*types.Block, error)

	// Seal generates a new sealing request for the given input block and pushes it to the sealer.
	Seal(chain ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error

	// SealHash returns the hash of a block prior to it being sealed.
	SealHash(header *types.Header) common.Hash

	// CalcDifficulty is the difficulty adjustment algorithm.
	CalcDifficulty(chain ChainHeaderReader, time uint64, parent *types.Header) *big.Int

	// Close terminates any background threads maintained by the consensus engine.
	Close() error
}

// Minimal ChainConfig interface to avoid importing params
type ChainConfig interface {
	GetChainID() *big.Int
	GetEIP150Block() *big.Int
	GetEIP150Hash() common.Hash
	GetEIP155Block() *big.Int
	GetEIP158Block() *big.Int
	GetByzantiumBlock() *big.Int
	GetConstantinopleBlock() *big.Int
	GetPetersburgBlock() *big.Int
	GetIstanbulBlock() *big.Int
	GetMuirGlacierBlock() *big.Int
	GetBerlinBlock() *big.Int
	GetLondonBlock() *big.Int
}