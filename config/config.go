// Copyright (C) 2024, Lux Partners Limited. All rights reserved.
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