// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"math/big"
)

// Config represents the SDK configuration
type Config struct {
	LogLevel string
	DataDir  string
	Network  *NetworkConfig
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	NetrunnerEndpoint string
	NodeEndpoint      string
	NetworkID         uint32
	APIEndpoint       string
	P2PPort           int
	HTTPPort          int
	StakingPort       int
	LogLevel          string
	DataDir           string
	DBType            string
	GenesisFile       string
	StakeAmount       uint64
}

// ChainID returns a big.Int representation of the NetworkID
func (nc *NetworkConfig) ChainID() *big.Int {
	return big.NewInt(int64(nc.NetworkID))
}

// Default returns a default configuration
func Default() *Config {
	return &Config{
		LogLevel: "info",
		DataDir:  "~/.luxd",
		Network:  DefaultNetworkConfig(),
	}
}

// DefaultNetworkConfig returns default network configuration
func DefaultNetworkConfig() *NetworkConfig {
	return &NetworkConfig{
		NetrunnerEndpoint: "localhost:8080",
		NodeEndpoint:      "http://localhost:9650",
		NetworkID:         12345, // Local network ID
		APIEndpoint:       "/ext/bc/C/rpc",
		P2PPort:           9651,
		HTTPPort:          9650,
		StakingPort:       9652,
		LogLevel:          "info",
		DataDir:           "~/.luxd",
		DBType:            "badgerdb",
		GenesisFile:       "",
		StakeAmount:       2000,
	}
}