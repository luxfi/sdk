// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Genesis represents an EVM genesis block
type Genesis struct {
	Config    *ChainConfig                      `json:"config"`
	Alloc     map[common.Address]GenesisAccount `json:"alloc"`
	Timestamp uint64                            `json:"timestamp"`
	GasLimit  uint64                            `json:"gasLimit"`
}

// ChainConfig represents the chain configuration
type ChainConfig struct {
	ChainID *big.Int `json:"chainId"`
}

// GenesisAccount represents an account in the genesis block
type GenesisAccount struct {
	Balance *big.Int                    `json:"balance"`
	Code    []byte                      `json:"code,omitempty"`
	Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
}
