// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package models

import (
	"context"
	"fmt"
	"time"

	lux_constants "github.com/luxfi/node/utils/constants"
	"github.com/luxfi/sdk/constants"
)

type Network int64

const (
	Undefined Network = iota
	Mainnet
	Testnet
	Local
	Devnet
)

// Aliases for compatibility
const UndefinedNetwork = Undefined

func (s Network) String() string {
	switch s {
	case Mainnet:
		return "Mainnet"
	case Testnet:
		return "Testnet"
	case Local:
		return "Local Network"
	case Devnet:
		return "Devnet"
	}
	return "Unknown Network"
}

func (s Network) NetworkID() (uint32, error) {
	switch s {
	case Mainnet:
		return lux_constants.MainnetID, nil
	case Testnet:
		return lux_constants.TestnetID, nil
	case Local:
		return constants.LocalNetworkID, nil
	}
	return 0, fmt.Errorf("unsupported network")
}

func (s Network) NetworkIDFlagValue() string {
	id, err := s.NetworkID()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d", id)
}

func (s Network) ID() uint32 {
	id, _ := s.NetworkID()
	return id
}

func (s Network) Kind() Network {
	return s
}

// Name returns the network name as a string
func (s Network) Name() string {
	return s.String()
}

func (s Network) HandlePublicNetworkSimulation() bool {
	return s == Local
}

func NetworkFromString(s string) Network {
	switch s {
	case Mainnet.String():
		return Mainnet
	case Testnet.String():
		return Testnet
	case Local.String():
		return Local
	}
	return Undefined
}

func NetworkFromNetworkID(networkID uint32) Network {
	switch networkID {
	case lux_constants.MainnetID:
		return Mainnet
	case lux_constants.TestnetID:
		return Testnet
	case constants.LocalNetworkID:
		return Local
	}
	return Undefined
}

// ConvertClusterToNetwork converts a local cluster network to appropriate network type
func ConvertClusterToNetwork(network Network) Network {
	// If it's already a specific network type, return it
	if network == Testnet || network == Mainnet {
		return network
	}
	// Local networks remain local
	return Local
}

// NewLocalNetwork creates a new Local network instance
func NewLocalNetwork() Network {
	return Local
}

// NewTestnetNetwork creates a new Testnet network instance
func NewTestnetNetwork() Network {
	return Testnet
}

// NewMainnetNetwork creates a new Mainnet network instance
func NewMainnetNetwork() Network {
	return Mainnet
}

// NewDevnetNetwork creates a new Devnet network instance
func NewDevnetNetwork() Network {
	return Devnet
}

// NewNetworkFromCluster creates a network instance associated with a cluster
func NewNetworkFromCluster(network Network, clusterName string) Network {
	// For now, the cluster name doesn't change the network type
	// but it could be used to associate the network with a specific cluster
	// This function exists primarily for API consistency and future extensibility
	return network
}

// BootstrappingContext returns a context for bootstrapping operations
func (s Network) BootstrappingContext() (context.Context, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	return ctx, cancel
}

// Endpoint returns the RPC endpoint for the network
func (s Network) Endpoint() string {
	switch s {
	case Mainnet:
		return constants.MainnetAPIEndpoint
	case Testnet:
		return constants.TestnetAPIEndpoint
	case Local:
		return constants.LocalAPIEndpoint
	case Devnet:
		return constants.DevnetAPIEndpoint
	default:
		return constants.LocalAPIEndpoint
	}
}

// ClusterName returns the cluster name for the network
func (s Network) ClusterName() string {
	switch s {
	case Local:
		return "local"
	default:
		return ""
	}
}

// WSEndpoints returns the WebSocket endpoints for the network
func (s Network) WSEndpoints() []string {
	switch s {
	case Mainnet:
		return []string{constants.MainnetWSEndpoint}
	case Testnet:
		return []string{constants.TestnetWSEndpoint}
	case Local:
		return []string{constants.LocalWSEndpoint}
	case Devnet:
		return []string{constants.DevnetWSEndpoint}
	default:
		return []string{constants.LocalWSEndpoint}
	}
}

// CChainEndpoint returns the C-Chain endpoint for the network
func (s Network) CChainEndpoint() string {
	baseEndpoint := s.Endpoint()
	return fmt.Sprintf("%s/ext/bc/C/rpc", baseEndpoint)
}

// CChainWSEndpoint returns the C-Chain WebSocket endpoint for the network
func (s Network) CChainWSEndpoint() string {
	wsEndpoints := s.WSEndpoints()
	if len(wsEndpoints) > 0 {
		return wsEndpoints[0]
	}
	return ""
}

// Equals checks if two networks are equal
func (s Network) Equals(other Network) bool {
	return s == other
}

// SDKNetwork returns the network as an SDK network type
// This is for compatibility with the SDK package
func (s Network) SDKNetwork() interface{} {
	// Convert to SDK network type
	// This returns an interface{} to avoid circular dependencies
	// The caller should type assert to the appropriate SDK network type
	return s
}

// GetNetworkFromSidecarNetworkName returns a network based on sidecar network name
func GetNetworkFromSidecarNetworkName(name string) Network {
	switch name {
	case "mainnet", "Mainnet":
		return Mainnet
	case "testnet", "Testnet":
		return Testnet
	case "local", "Local":
		return Local
	case "devnet", "Devnet":
		return Devnet
	default:
		return Undefined
	}
}

// GetRPCEndpoint returns the RPC endpoint for a given base endpoint and blockchain ID
func GetRPCEndpoint(baseEndpoint string, blockchainID string) string {
	return fmt.Sprintf("%s/ext/bc/%s/rpc", baseEndpoint, blockchainID)
}

// GetWSEndpoint returns the WebSocket endpoint for a given base endpoint and blockchain ID
func GetWSEndpoint(baseEndpoint string, blockchainID string) string {
	return fmt.Sprintf("%s/ext/bc/%s/ws", baseEndpoint, blockchainID)
}

// GenesisParams represents network genesis parameters
type GenesisParams struct {
	MinDelegationFee  uint32
	MinValidatorStake uint64
	MinDelegatorStake uint64
	MinStakeDuration  time.Duration
	MaxStakeDuration  time.Duration
}

// GenesisParams returns the genesis parameters for the network
func (s Network) GenesisParams() *GenesisParams {
	// Default values based on Avalanche mainnet
	switch s {
	case Mainnet, Testnet:
		return &GenesisParams{
			MinDelegationFee:  20000,                  // 2%
			MinValidatorStake: 2000 * 1_000_000_000,   // 2000 LUX
			MinDelegatorStake: 25 * 1_000_000_000,     // 25 LUX
			MinStakeDuration:  2 * 7 * 24 * time.Hour, // 2 weeks
			MaxStakeDuration:  365 * 24 * time.Hour,   // 1 year
		}
	case Local, Devnet:
		return &GenesisParams{
			MinDelegationFee:  10000,                // 1%
			MinValidatorStake: 1 * 1_000_000_000,    // 1 LUX
			MinDelegatorStake: 1 * 1_000_000_000,    // 1 LUX
			MinStakeDuration:  24 * time.Hour,       // 1 day
			MaxStakeDuration:  365 * 24 * time.Hour, // 1 year
		}
	default:
		return &GenesisParams{
			MinDelegationFee:  10000,                // 1%
			MinValidatorStake: 1 * 1_000_000_000,    // 1 LUX
			MinDelegatorStake: 1 * 1_000_000_000,    // 1 LUX
			MinStakeDuration:  24 * time.Hour,       // 1 day
			MaxStakeDuration:  365 * 24 * time.Hour, // 1 year
		}
	}
}

// BlockchainEndpoint returns the blockchain RPC endpoint for a given blockchain ID
func (s Network) BlockchainEndpoint(blockchainID string) string {
	baseEndpoint := s.Endpoint()
	if blockchainID == "" {
		return fmt.Sprintf("%s/ext/bc/C/rpc", baseEndpoint)
	}
	return fmt.Sprintf("%s/ext/bc/%s/rpc", baseEndpoint, blockchainID)
}
