// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package utils

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/luxfi/ids"
)

// GetDefaultBlockchainAirdropKeyName returns the default key name for blockchain airdrops
func GetDefaultBlockchainAirdropKeyName(blockchainName string) string {
	return fmt.Sprintf("%s-airdrop", blockchainName)
}

// Account represents an account in the genesis allocation
type Account struct {
	Balance *big.Int          `json:"balance"`
	Code    []byte            `json:"code,omitempty"`
	Storage map[string]string `json:"storage,omitempty"`
	Nonce   uint64            `json:"nonce,omitempty"`
}

// SubnetEvmGenesis represents a subnet EVM genesis configuration
type SubnetEvmGenesis struct {
	Config     map[string]interface{} `json:"config"`
	Alloc      map[string]Account     `json:"alloc"`
	Timestamp  uint64                 `json:"timestamp,omitempty"`
	GasLimit   uint64                 `json:"gasLimit"`
	Difficulty string                 `json:"difficulty,omitempty"`
	MixHash    string                 `json:"mixHash,omitempty"`
	Coinbase   string                 `json:"coinbase,omitempty"`
	Number     string                 `json:"number,omitempty"`
	GasUsed    string                 `json:"gasUsed,omitempty"`
	ParentHash string                 `json:"parentHash,omitempty"`
}

// ByteSliceToSubnetEvmGenesis converts a byte slice to a SubnetEVM genesis
func ByteSliceToSubnetEvmGenesis(bytes []byte) (*SubnetEvmGenesis, error) {
	var genesis SubnetEvmGenesis
	if err := json.Unmarshal(bytes, &genesis); err != nil {
		return nil, err
	}
	return &genesis, nil
}

// ByteSliceIsSubnetEvmGenesis checks if a byte slice is a SubnetEVM genesis
func ByteSliceIsSubnetEvmGenesis(bytes []byte) bool {
	var genesis SubnetEvmGenesis
	err := json.Unmarshal(bytes, &genesis)
	return err == nil && genesis.Config != nil
}

// GetBlockchainTx retrieves a blockchain transaction from the network
func GetBlockchainTx(endpoint string, blockchainID ids.ID) (interface{}, error) {
	// This is a stub implementation
	// In a real implementation, this would make an API call to retrieve the transaction
	// Returns interface{} to avoid importing node's internal packages
	return nil, fmt.Errorf("GetBlockchainTx not yet implemented")
}

// GetKeyNames returns a list of key names
func GetKeyNames(keyDir string, includeEwoq bool) ([]string, error) {
	// This is a stub implementation
	// In a real implementation, this would list all key files in the directory
	// TODO: Implement directory listing and key name extraction
	keys := []string{}
	if includeEwoq {
		keys = append(keys, "ewoq")
	}
	return keys, nil
}

// GetBlockchainIDFromAlias gets a blockchain ID from its alias on the network
func GetBlockchainIDFromAlias(endpoint string, alias string) (ids.ID, error) {
	// This is a stub implementation
	// In a real implementation, this would make an API call to resolve the alias
	// For now, return a special case for C-Chain
	if alias == "C" {
		// Return a dummy C-Chain ID
		return ids.FromString("2q9e4r6Mu3U68nU1fYjgbR6JvwrRx36CohpAX5UQxse55eZ9Tc")
	}
	return ids.Empty, fmt.Errorf("GetBlockchainIDFromAlias not yet implemented for alias: %s", alias)
}

// GetChainID extracts the chain ID from genesis data
func GetChainID(genesisData []byte) (*big.Int, error) {
	genesis, err := ByteSliceToSubnetEvmGenesis(genesisData)
	if err != nil {
		return nil, err
	}
	if genesis.Config == nil {
		return nil, fmt.Errorf("no config in genesis")
	}
	if chainID, ok := genesis.Config["chainId"]; ok {
		switch v := chainID.(type) {
		case float64:
			return big.NewInt(int64(v)), nil
		case int64:
			return big.NewInt(v), nil
		case string:
			id, ok := new(big.Int).SetString(v, 10)
			if !ok {
				return nil, fmt.Errorf("invalid chain ID string: %s", v)
			}
			return id, nil
		case *big.Int:
			return v, nil
		}
	}
	return nil, fmt.Errorf("chain ID not found in genesis")
}
