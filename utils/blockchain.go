// Copyright (C) 2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/luxfi/ids"
	"github.com/luxfi/rpc"
)

// GetDefaultBlockchainAirdropKeyName returns the default key name for blockchain airdrops
func GetDefaultBlockchainAirdropKeyName(blockchainName string) string {
	return fmt.Sprintf("%s-airdrop", blockchainName)
}

// Account represents an account in the genesis allocation
type Account struct {
	Balance interface{}       `json:"balance"`        // Can be string (hex) or number
	Code    interface{}       `json:"code,omitempty"` // Can be string (hex) or []byte
	Storage map[string]string `json:"storage,omitempty"`
	Nonce   interface{}       `json:"nonce,omitempty"` // Can be string (hex) or uint64
}

// GetBalance returns the balance as a big.Int
func (a *Account) GetBalance() *big.Int {
	if a.Balance == nil {
		return big.NewInt(0)
	}
	switch v := a.Balance.(type) {
	case string:
		// Handle hex string
		balance := new(big.Int)
		if strings.HasPrefix(v, "0x") {
			balance.SetString(v[2:], 16)
		} else {
			balance.SetString(v, 10)
		}
		return balance
	case float64:
		return big.NewInt(int64(v))
	case int64:
		return big.NewInt(v)
	default:
		return big.NewInt(0)
	}
}

// GetCode returns the code as a byte slice
func (a *Account) GetCode() []byte {
	if a.Code == nil {
		return nil
	}
	switch v := a.Code.(type) {
	case string:
		// Handle hex string
		if strings.HasPrefix(v, "0x") {
			code, _ := hex.DecodeString(v[2:])
			return code
		}
		return []byte(v)
	case []byte:
		return v
	default:
		return nil
	}
}

// UnmarshalJSON custom unmarshaler for Account to handle hex string balances
func (a *Account) UnmarshalJSON(data []byte) error {
	type Alias Account
	aux := &struct {
		Balance interface{} `json:"balance"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Parse balance (hex string or numeric)
	if aux.Balance != nil {
		switch v := aux.Balance.(type) {
		case string:
			// Remove quotes if present
			balance := new(big.Int)
			if strings.HasPrefix(v, "0x") {
				if _, ok := balance.SetString(v[2:], 16); !ok {
					return fmt.Errorf("invalid hex balance: %s", v)
				}
			} else {
				if _, ok := balance.SetString(v, 10); !ok {
					return fmt.Errorf("invalid balance: %s", v)
				}
			}
			a.Balance = balance
		case float64:
			a.Balance = big.NewInt(int64(v))
		}
	}

	return nil
}

// EVMGenesis represents a EVM genesis configuration
type EVMGenesis struct {
	Config     map[string]interface{} `json:"config"`
	Alloc      map[string]Account     `json:"alloc"`
	Timestamp  interface{}            `json:"timestamp,omitempty"` // Can be string (hex) or uint64
	GasLimit   interface{}            `json:"gasLimit"`            // Can be string (hex) or uint64
	Difficulty string                 `json:"difficulty,omitempty"`
	MixHash    string                 `json:"mixHash,omitempty"`
	Coinbase   string                 `json:"coinbase,omitempty"`
	Number     string                 `json:"number,omitempty"`
	GasUsed    string                 `json:"gasUsed,omitempty"`
	ParentHash string                 `json:"parentHash,omitempty"`
	Nonce      string                 `json:"nonce,omitempty"`
	ExtraData  string                 `json:"extraData,omitempty"`
}

// UnmarshalJSON custom unmarshaler to handle hex string or numeric values
func (g *EVMGenesis) UnmarshalJSON(data []byte) error {
	type Alias EVMGenesis
	aux := &struct {
		Timestamp interface{} `json:"timestamp,omitempty"`
		GasLimit  interface{} `json:"gasLimit"`
		*Alias
	}{
		Alias: (*Alias)(g),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Parse timestamp (numeric or hex string)
	if aux.Timestamp != nil {
		switch v := aux.Timestamp.(type) {
		case float64:
			g.Timestamp = uint64(v)
		case string:
			if strings.HasPrefix(v, "0x") {
				val, err := strconv.ParseUint(v[2:], 16, 64)
				if err != nil {
					return err
				}
				g.Timestamp = val
			} else {
				val, err := strconv.ParseUint(v, 10, 64)
				if err != nil {
					return err
				}
				g.Timestamp = val
			}
		}
	}

	// Parse gasLimit (numeric or hex string)
	if aux.GasLimit != nil {
		switch v := aux.GasLimit.(type) {
		case float64:
			g.GasLimit = uint64(v)
		case string:
			if strings.HasPrefix(v, "0x") {
				val, err := strconv.ParseUint(v[2:], 16, 64)
				if err != nil {
					return err
				}
				g.GasLimit = val
			} else {
				val, err := strconv.ParseUint(v, 10, 64)
				if err != nil {
					return err
				}
				g.GasLimit = val
			}
		}
	}

	return nil
}

// ByteSliceToEVMGenesis converts a byte slice to a EVM genesis
func ByteSliceToEVMGenesis(bytes []byte) (*EVMGenesis, error) {
	var genesis EVMGenesis
	if err := json.Unmarshal(bytes, &genesis); err != nil {
		return nil, err
	}
	return &genesis, nil
}

// ByteSliceIsEVMGenesis checks if a byte slice is a EVM genesis
func ByteSliceIsEVMGenesis(bytes []byte) bool {
	var genesis EVMGenesis
	err := json.Unmarshal(bytes, &genesis)
	return err == nil && genesis.Config != nil
}

// GetKeyNames returns a list of key names from the given directory.
func GetKeyNames(keyDir string, includeEwoq bool) ([]string, error) {
	entries, err := os.ReadDir(keyDir)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".pk" {
			continue
		}
		keys = append(keys, strings.TrimSuffix(name, ".pk"))
	}
	if includeEwoq {
		keys = append(keys, "ewoq")
	}
	return keys, nil
}

// GetBlockchainIDFromAlias gets a blockchain ID from its alias on the network.
func GetBlockchainIDFromAlias(endpoint string, alias string) (ids.ID, error) {
	type getBlockchainIDArgs struct {
		Alias string `json:"alias"`
	}
	type getBlockchainIDReply struct {
		BlockchainID ids.ID `json:"blockchainID"`
	}
	requester := rpc.NewEndpointRequester(endpoint + "/ext/info")
	ctx, cancel := GetAPIContext()
	defer cancel()
	reply := &getBlockchainIDReply{}
	if err := requester.SendRequest(ctx, "info.getBlockchainID", &getBlockchainIDArgs{
		Alias: alias,
	}, reply); err != nil {
		return ids.Empty, err
	}
	return reply.BlockchainID, nil
}

// GetChainID extracts the chain ID from genesis data
func GetChainID(genesisData []byte) (*big.Int, error) {
	genesis, err := ByteSliceToEVMGenesis(genesisData)
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
