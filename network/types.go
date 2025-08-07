// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package network

import (
	"fmt"

	"github.com/luxfi/sdk/constants"
)

// NetworkKind represents the kind of network
type NetworkKind int

const (
	Undefined NetworkKind = iota
	Mainnet
	Testnet
	Local
	Devnet
)

// LegacyNetwork represents a network configuration with endpoint
// This is for compatibility with existing code
type LegacyNetwork struct {
	Kind     NetworkKind
	ID       uint32
	Endpoint string
	Name     string
}

var (
	// UndefinedNetwork is an undefined network
	UndefinedNetwork = LegacyNetwork{
		Kind: Undefined,
		ID:   0,
		Name: "undefined",
	}

	// MainnetNetwork is the mainnet network
	MainnetNetwork = LegacyNetwork{
		Kind:     Mainnet,
		ID:       constants.MainnetID,
		Endpoint: "https://api.lux.network",
		Name:     constants.MainnetName,
	}

	// TestnetNetwork is the testnet network
	TestnetNetwork = LegacyNetwork{
		Kind:     Testnet,
		ID:       constants.TestnetID,
		Endpoint: "https://api-test.lux.network",
		Name:     constants.TestnetName,
	}

	// LocalNetwork is the local network
	LocalNetwork = LegacyNetwork{
		Kind:     Local,
		ID:       constants.LocalID,
		Endpoint: "http://127.0.0.1:9630",
		Name:     constants.LocalName,
	}
)

// NetworkFromNetworkID returns a network from network ID
func NetworkFromNetworkID(networkID uint32) LegacyNetwork {
	switch networkID {
	case constants.MainnetID:
		return MainnetNetwork
	case constants.TestnetID:
		return TestnetNetwork
	case constants.LocalID:
		return LocalNetwork
	default:
		return LegacyNetwork{
			Kind:     Devnet,
			ID:       networkID,
			Endpoint: "",
			Name:     fmt.Sprintf("network-%d", networkID),
		}
	}
}

// GetNetworkByName returns a network by name
func GetNetworkByName(name string) (LegacyNetwork, error) {
	switch name {
	case constants.MainnetName:
		return MainnetNetwork, nil
	case constants.TestnetName:
		return TestnetNetwork, nil
	case constants.LocalName:
		return LocalNetwork, nil
	default:
		return UndefinedNetwork, fmt.Errorf("unknown network: %s", name)
	}
}

// String returns string representation of NetworkKind
func (k NetworkKind) String() string {
	switch k {
	case Mainnet:
		return "mainnet"
	case Testnet:
		return "testnet"
	case Local:
		return "local"
	case Devnet:
		return "devnet"
	default:
		return "undefined"
	}
}
