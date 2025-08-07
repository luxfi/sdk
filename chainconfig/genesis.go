// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chainconfig

import (
	"encoding/json"
	"math/big"

	"github.com/luxfi/evm/core"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/common/math"
)

// GenesisBuilder helps construct EVM genesis configurations
type GenesisBuilder struct {
	genesis *core.Genesis
}

// NewGenesisBuilder creates a new genesis builder with defaults
func NewGenesisBuilder() *GenesisBuilder {
	return &GenesisBuilder{
		genesis: &core.Genesis{
			Config:     DefaultChainConfig(),
			Difficulty: big.NewInt(0),
			GasLimit:   8_000_000,
			Alloc:      make(core.GenesisAlloc),
		},
	}
}

// WithChainConfig sets the chain configuration
func (b *GenesisBuilder) WithChainConfig(config *params.ChainConfig) *GenesisBuilder {
	b.genesis.Config = config
	return b
}

// WithGasLimit sets the gas limit
func (b *GenesisBuilder) WithGasLimit(gasLimit uint64) *GenesisBuilder {
	b.genesis.GasLimit = gasLimit
	return b
}

// WithTimestamp sets the genesis timestamp
func (b *GenesisBuilder) WithTimestamp(timestamp uint64) *GenesisBuilder {
	b.genesis.Timestamp = timestamp
	return b
}

// WithExtraData sets the extra data
func (b *GenesisBuilder) WithExtraData(extraData []byte) *GenesisBuilder {
	b.genesis.ExtraData = extraData
	return b
}

// WithAllocation adds an account allocation
func (b *GenesisBuilder) WithAllocation(address common.Address, balance *big.Int) *GenesisBuilder {
	if b.genesis.Alloc == nil {
		b.genesis.Alloc = make(core.GenesisAlloc)
	}
	b.genesis.Alloc[address] = core.GenesisAccount{
		Balance: balance,
	}
	return b
}

// WithAllocations adds multiple account allocations
func (b *GenesisBuilder) WithAllocations(allocations map[common.Address]*big.Int) *GenesisBuilder {
	if b.genesis.Alloc == nil {
		b.genesis.Alloc = make(core.GenesisAlloc)
	}
	for address, balance := range allocations {
		b.genesis.Alloc[address] = core.GenesisAccount{
			Balance: balance,
		}
	}
	return b
}

// WithContract adds a contract with code and storage
func (b *GenesisBuilder) WithContract(address common.Address, balance *big.Int, code []byte, storage map[common.Hash]common.Hash) *GenesisBuilder {
	if b.genesis.Alloc == nil {
		b.genesis.Alloc = make(core.GenesisAlloc)
	}
	b.genesis.Alloc[address] = core.GenesisAccount{
		Balance: balance,
		Code:    code,
		Storage: storage,
	}
	return b
}

// Build returns the constructed genesis
func (b *GenesisBuilder) Build() *core.Genesis {
	return b.genesis
}

// ToJSON converts the genesis to JSON bytes
func (b *GenesisBuilder) ToJSON() ([]byte, error) {
	return json.MarshalIndent(b.genesis, "", "  ")
}

// DefaultGenesis creates a default genesis configuration
func DefaultGenesis() *core.Genesis {
	return NewGenesisBuilder().Build()
}

// CreateAirdropGenesis creates a genesis with airdrop allocations
func CreateAirdropGenesis(
	chainConfig *params.ChainConfig,
	airdropAddresses []common.Address,
	airdropAmount *big.Int,
) *core.Genesis {
	builder := NewGenesisBuilder().WithChainConfig(chainConfig)
	
	for _, address := range airdropAddresses {
		builder.WithAllocation(address, airdropAmount)
	}
	
	return builder.Build()
}

// CreateDevGenesis creates a genesis suitable for development
func CreateDevGenesis(chainID *big.Int) *core.Genesis {
	// Pre-funded development account
	devAddress := common.HexToAddress("0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC")
	devBalance := math.MustParseBig256("100000000000000000000000000") // 100M tokens
	
	return NewGenesisBuilder().
		WithChainConfig(LocalChainConfig(chainID)).
		WithGasLimit(15_000_000).
		WithAllocation(devAddress, devBalance).
		Build()
}

// ParseGenesis parses genesis from JSON bytes
func ParseGenesis(data []byte) (*core.Genesis, error) {
	genesis := &core.Genesis{}
	if err := json.Unmarshal(data, genesis); err != nil {
		return nil, err
	}
	return genesis, nil
}

// ValidateGenesis validates a genesis configuration
func ValidateGenesis(genesis *core.Genesis) error {
	// Basic validation
	if genesis.Config == nil {
		return ErrInvalidChainConfig
	}
	if genesis.Config.ChainID == nil {
		return ErrMissingChainID
	}
	if genesis.GasLimit == 0 {
		return ErrInvalidGasLimit
	}
	return nil
}