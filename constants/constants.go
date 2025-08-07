// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package constants

import (
	"errors"
	"os"
	"time"

	"github.com/luxfi/ids"
)

// Errors
var (
	ErrUnknownNetwork = errors.New("unknown network")
)

// Network IDs
const (
	MainnetID uint32 = 1
	TestnetID uint32 = 5
	LocalID   uint32 = 12345

	// Network names
	MainnetName = "mainnet"
	TestnetName = "testnet"
	LocalName   = "local"
)

// Network HRP (Human Readable Part) for addresses
const (
	MainnetHRP  = "lux"
	TestnetHRP  = "test"
	LocalHRP    = "local"
	FallbackHRP = "custom"
)

// GetHRP returns the HRP for a network ID
func GetHRP(networkID uint32) string {
	switch networkID {
	case MainnetID:
		return MainnetHRP
	case TestnetID:
		return TestnetHRP
	case LocalID:
		return LocalHRP
	default:
		return FallbackHRP
	}
}

// Chain IDs
var (
	PlatformChainID = ids.ID{'p', 'l', 'a', 't', 'f', 'o', 'r', 'm'}
	XChainID        = ids.ID{'x', 'c', 'h', 'a', 'i', 'n'}
	CChainID        = ids.ID{'c', 'c', 'h', 'a', 'i', 'n'}
)

// Chain aliases
const (
	PChainAlias = "P"
	XChainAlias = "X"
	CChainAlias = "C"
)

// Asset IDs
var (
	LuxAssetID = ids.ID{
		0x21, 0xe6, 0x73, 0x17, 0xcb, 0xc4, 0xbe, 0x2a,
		0xeb, 0x00, 0x67, 0x7a, 0xd6, 0x46, 0x27, 0x78,
		0xa8, 0xf5, 0x22, 0x74, 0xb9, 0xd6, 0x05, 0xdf,
		0x25, 0x91, 0xb2, 0x30, 0x27, 0xa8, 0x7d, 0xff,
	}
)

// Denominations
const (
	Wei  uint64 = 1
	GWei uint64 = 1e9
	LUX  uint64 = 1e18

	// Legacy compatibility
	NanoLux  = Wei
	MicroLux = GWei
)

// Staking parameters
const (
	// Minimum stake amounts
	MinValidatorStake = 2_000 * GWei // 2,000 LUX worth of GWei
	MinDelegatorStake = 25 * GWei    // 25 LUX worth of GWei

	// Maximum stake amounts
	MaxValidatorStake = 3_000_000 * GWei // 3M LUX worth of GWei
	MaxDelegatorStake = 3_000_000 * GWei // 3M LUX worth of GWei

	// Weight factors
	MaxValidatorWeightFactor = 5

	// Time parameters
	MinStakeDuration = 2 * 7 * 24 * time.Hour // 2 weeks
	MaxStakeDuration = 365 * 24 * time.Hour   // 1 year

	// Reward parameters
	MaxDelegationFee = 100 // 100% (in basis points / 100)
)

// Supply and economics
const (
	// Total supply
	TotalSupply = 720_000_000 * GWei // 720M LUX worth of GWei

	// Initial supply distribution
	InitialSupply = 360_000_000 * GWei // 360M LUX worth of GWei

	// Reward config
	RewardPercentDenominator = 1_000_000
	InflationRate            = 0.05 // 5% annual
)

// Transaction fees
const (
	// Base fees
	TxFee             = 1_000_000 * GWei  // 0.001 LUX
	CreateAssetTxFee  = 10_000_000 * GWei // 0.01 LUX
	CreateChainTxFee  = 1 * LUX
	CreateSubnetTxFee = 1 * LUX

	// Gas parameters
	GasPrice    = 25 * GWei
	MaxGasPrice = 1000 * GWei
	MinGasPrice = 1 * GWei
)

// Block parameters
const (
	// Block size limits
	MaxBlockSize    = 2 * 1024 * 1024 // 2 MiB
	MaxBlockGas     = 15_000_000
	TargetBlockRate = 2 * time.Second

	// Genesis block
	GenesisHeight    = 0
	GenesisTimestamp = 1640995200 // Jan 1, 2022 00:00:00 UTC
)

// VM parameters
const (
	// VM types
	EVMID      = "evm"
	WasmVMID   = "wasm"
	CustomVMID = "custom"
	TokenVMID  = "tokenvm"

	// VM versions
	EVMVersion    = "v0.13.0"
	WasmVMVersion = "v0.1.0"
)

// Consensus parameters
const (
	// Snow consensus
	SnowmanK                 = 20
	SnowmanAlphaPreference   = 15
	SnowmanAlphaConfidence   = 15
	SnowmanBeta              = 20
	SnowmanConcurrentRepolls = 4
	SnowmanOptimalProcessing = 10
	SnowmanMaxProcessing     = 1000
	SnowmanMaxTimeProcessing = 2 * time.Minute

	// For different network types
	TestnetSnowmanK               = 11
	TestnetSnowmanAlphaPreference = 7
	LocalSnowmanK                 = 5
	LocalSnowmanAlphaPreference   = 3
)

// Network timeouts
const (
	// Request timeouts
	RequestTimeout         = 30 * time.Second
	RequestRetryTimeout    = 1 * time.Second
	APIRequestTimeout      = 30 * time.Second
	APIRequestLargeTimeout = 2 * time.Minute

	// Gossip parameters
	GossipFrequency = 10 * time.Second
	GossipBatchSize = 30
	GossipPollSize  = 10

	// Health check
	HealthCheckFrequency   = 30 * time.Second
	MaxOutstandingRequests = 1024

	// Network limits
	MaxMessageSize     = 2 * 1024 * 1024 // 2 MiB
	MaxClockDifference = 10 * time.Second
)

// API endpoints
const (
	// RPC endpoints
	PublicAPIEndpoint   = "/ext/bc"
	AdminAPIEndpoint    = "/ext/admin"
	HealthAPIEndpoint   = "/ext/health"
	InfoAPIEndpoint     = "/ext/info"
	KeystoreAPIEndpoint = "/ext/keystore"
	MetricsAPIEndpoint  = "/ext/metrics"

	// Chain endpoints
	PChainEndpoint = "/ext/P"
	XChainEndpoint = "/ext/X"
	CChainEndpoint = "/ext/C"
)

// Database paths
const (
	// Default data directory
	DefaultDataDir = "~/.luxd"

	// Database names
	ChainDataDir = "chainData"
	StateDir     = "state"
	LogDir       = "logs"
	KeystoreDir  = "keystore"

	// Database prefixes
	ChainDBPrefix = "chain"
	StateDBPrefix = "state"
)

// Bootstrapping
const (
	// Bootstrap retry parameters
	BootstrapRetryAttempts = 50
	BootstrapRetryDelay    = 1 * time.Second

	// Bootstrap timeouts
	BootstrapTimeout  = 1 * time.Hour
	MinBootstrapPeers = 1
)

// Validator set parameters
const (
	// Validator limits
	MaxValidators        = 10_000
	MaxPendingValidators = 4_096

	// Subnet limits
	MaxSubnetValidators     = 100
	MinSubnetValidatorStake = 1 * GWei // 1 LUX worth of GWei
)

// Cross-chain (Warp) messaging
const (
	// Warp message size limits
	MaxWarpMessageSize    = 256 * 1024 // 256 KiB
	MaxWarpMessagePayload = 200 * 1024 // 200 KiB

	// Warp signature parameters
	WarpQuorumNumerator   = 67
	WarpQuorumDenominator = 100
)

// Platform limits
const (
	// Transaction limits
	MaxTxSize   = 64 * 1024 // 64 KiB
	MaxMemoSize = 256

	// UTXO limits
	MaxUTXOsToFetch = 1024

	// Import/Export limits
	MaxImportSize = 1024
)

// File permissions
const (
	UserOnlyWriteReadExecPerms = os.FileMode(0700)
)

// GetNetworkID returns the network ID from name
func GetNetworkID(name string) (uint32, error) {
	switch name {
	case MainnetName:
		return MainnetID, nil
	case TestnetName:
		return TestnetID, nil
	case LocalName:
		return LocalID, nil
	default:
		return 0, ErrUnknownNetwork
	}
}

// GetNetworkName returns the network name from ID
func GetNetworkName(networkID uint32) string {
	switch networkID {
	case MainnetID:
		return MainnetName
	case TestnetID:
		return TestnetName
	case LocalID:
		return LocalName
	default:
		return "unknown"
	}
}

// IsMainnet returns true if the network ID is mainnet
func IsMainnet(networkID uint32) bool {
	return networkID == MainnetID
}

// IsTestnet returns true if the network ID is testnet
func IsTestnet(networkID uint32) bool {
	return networkID == TestnetID
}

// IsLocal returns true if the network ID is local
func IsLocal(networkID uint32) bool {
	return networkID == LocalID
}
