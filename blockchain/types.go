// Copyright (C) 2020-2025, Lux Partners Limited All rights reserved.
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

// GenesisValidator defines a validator in the genesis
// Uses simplified fields for genesis configuration
type GenesisValidator struct {
	NodeID string `json:"nodeId"`
	Weight uint64 `json:"weight"`
}

// Validator is an alias for backward compatibility
type Validator = GenesisValidator

// Net represents a blockchain network with validator management capabilities
// As above, so below - unified across all network layers (primary, L1, L2, L3)
type Net struct {
	NetID               interface{} // ids.ID
	BlockchainID        interface{} // ids.ID
	OwnerAddress        *common.Address
	RPC                 string
	BootstrapValidators []interface{} // []Validator
}

// Subnet is an alias for backward compatibility
type Subnet = Net

// InitializeProofOfAuthority initializes a PoA validator manager
func (s *Net) InitializeProofOfAuthority(
	log interface{}, // logging.Logger
	network interface{}, // models.Network
	privateKey string,
	aggregatorLogger interface{}, // logging.Logger
	validatorManagerAddress string,
	v2_0_0 bool,
	signatureAggregatorEndpoint string,
) error {
	// TODO: Implement PoA initialization
	return nil
}

// InitializeProofOfStake initializes a PoS validator manager
func (s *Net) InitializeProofOfStake(
	log interface{}, // logging.Logger
	network interface{}, // models.Network
	privateKey string,
	aggregatorLogger interface{}, // logging.Logger
	posParams interface{}, // PoSParams
	managerAddress string,
	signatureAggregatorEndpoint string,
) error {
	// TODO: Implement PoS initialization
	return nil
}
