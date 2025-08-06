// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package blockchain

import (
	"math/big"

	"github.com/luxfi/geth/common"
)

// GenesisAccount defines an account in genesis
type GenesisAccount struct {
	Balance *big.Int                    `json:"balance"`
	Code    []byte                      `json:"code,omitempty"`
	Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
}

// Validator defines a validator in the genesis
type Validator struct {
	NodeID string `json:"nodeId"`
	Weight uint64 `json:"weight"`
}
