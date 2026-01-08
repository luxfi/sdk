// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/luxfi/genesis/pkg/genesis"
	"github.com/luxfi/sdk/node/genesis/builder"
	"github.com/luxfi/sdk/node/trace"
	"github.com/luxfi/sdk/utils/compression"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/utils/dynamicip"
	"github.com/luxfi/sdk/utils/ulimit"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/sdk/node/vms/components/gas"
	"github.com/luxfi/sdk/node/vms/proposervm"

	consensusconfig "github.com/luxfi/consensus/config"
)

const (
	DefaultHTTPPort    = 9630
	DefaultStakingPort = 9631

	LuxNodeDataDirVar    = "LUXD_DATA_DIR"
	defaultUnexpandedDataDir = "$" + LuxNodeDataDirVar

	DefaultProcessContextFilename = "process.json"
)

var (
	// [defaultUnexpandedDataDir] will be expanded when reading the flags
	defaultDataDir              = filepath.Join("$HOME", ".lux")
	defaultDBDir                = filepath.Join(defaultUnexpandedDataDir, "db")
	defaultLogDir               = filepath.Join(defaultUnexpandedDataDir, "logs")
	defaultProfileDir           = filepath.Join(defaultUnexpandedDataDir, "profiles")
	defaultStakingPath          = filepath.Join(defaultUnexpandedDataDir, "staking")
	defaultStakingTLSKeyPath    = filepath.Join(defaultStakingPath, "staker.key")
	defaultStakingCertPath      = filepath.Join(defaultStakingPath, "staker.crt")
	defaultStakingSignerKeyPath = filepath.Join(defaultStakingPath, "signer.key")
	defaultConfigDir            = filepath.Join(defaultUnexpandedDataDir, "configs")
	defaultChainConfigDir       = filepath.Join(defaultConfigDir, "chains")
	defaultVMConfigDir          = filepath.Join(defaultConfigDir, "vms")
	defaultVMAliasFilePath      = filepath.Join(defaultVMConfigDir, "aliases.json")
	defaultChainAliasFilePath   = filepath.Join(defaultChainConfigDir, "aliases.json")
	defaultNetConfigDir      = filepath.Join(defaultConfigDir, "subnets")
	defaultPluginDir            = filepath.Join(defaultUnexpandedDataDir, "plugins", "current")
	defaultChainDataDir         = filepath.Join(defaultUnexpandedDataDir, "chainData")
	defaultProcessContextPath   = filepath.Join(defaultUnexpandedDataDir, DefaultProcessContextFilename)
)

func deprecateFlags(fs *pflag.FlagSet) error {
	for key, message := range deprecatedKeys {
		if err := fs.MarkDeprecated(key, message); err != nil {
			return err
		}
	}
	return nil
}

func addProcessFlags(fs *pflag.FlagSet) {
	// If true, print the version and quit.
	fs.Bool(VersionKey, false, "If true, print version and quit")
	fs.Bool(VersionJSONKey, false, "If true, print version in JSON format and quit")
}

func addNodeFlags(fs *pflag.FlagSet) {
	// Development mode
	fs.Bool(DevModeKey, false, "Enables development mode with single-node consensus, no sybil protection, and other dev-friendly settings")

	// Home directory
	fs.String(DataDirKey, defaultDataDir, "Sets the base data directory where default sub-directories will be placed unless otherwise specified.")
	// System
	fs.Uint64(FdLimitKey, ulimit.DefaultFDLimit, "Attempts to raise the process file descriptor limit to at least this value and error if the value is above the system max")

	// Plugin directory
	fs.String(PluginDirKey, defaultPluginDir, "Path to the plugin directory")

	// Config File
	fs.String(ConfigFileKey, "", fmt.Sprintf("Specifies a config file. Ignored if %s is specified", ConfigContentKey))
	fs.String(ConfigContentKey, "", "Specifies base64 encoded config content")
	fs.String(ConfigContentTypeKey, "json", "Specifies the format of the base64 encoded config content. Available values: 'json', 'yaml', 'toml'")

	// Genesis
	fs.String(GenesisFileKey, "", fmt.Sprintf("Specifies a genesis config file path. Ignored when running standard networks or if %s is specified",
		GenesisFileContentKey))
	fs.String(GenesisFileContentKey, "", "Specifies base64 encoded genesis content")
	fs.String(GenesisDBKey, "", "Path to existing genesis database for replay. Cannot be used with genesis-file or genesis-file-content")
	fs.String(GenesisDBTypeKey, "badgerdb", "Database type to use for genesis database. Must be one of {pebbledb, badgerdb}")
	fs.Uint64(GenesisBlockLimitKey, 0, "Limit number of blocks to replay during genesis (0 = all blocks)")
	fs.Bool(AllowCustomGenesisKey, true, "Allow custom genesis for mainnet/testnet (default: true for development, set false for production)")

	// Upgrade
	fs.String(UpgradeFileKey, "", fmt.Sprintf("Specifies an upgrade config file path. Ignored when running standard networks or if %s is specified",
		UpgradeFileContentKey))
	fs.String(UpgradeFileContentKey, "", "Specifies base64 encoded upgrade content")

	// Network ID
	fs.String(NetworkNameKey, constants.MainnetName, "Network ID this node will connect to")
// 	fs.Bool(MainnetKey, false, "Connect to Lux mainnet (network ID 96369)")
// 	fs.Bool(TestnetKey, false, "Connect to Lux testnet (network ID 96368)")
// 	fs.Bool(LocalnetKey, false, "Connect to local network (network ID 1337)")

	// LP flagging
	fs.IntSlice(LPSupportKey, nil, "LPs to support adoption")
	fs.IntSlice(LPObjectKey, nil, "LPs to object adoption")

	// LUX fees:
	// Validator fees:
	fs.Uint64(ValidatorFeesCapacityKey, uint64(builder.LocalValidatorFeeConfig.Capacity), "Maximum number of validators")
	fs.Uint64(ValidatorFeesTargetKey, uint64(builder.LocalValidatorFeeConfig.Target), "Target number of validators")
	fs.Uint64(ValidatorFeesMinPriceKey, uint64(builder.LocalValidatorFeeConfig.MinPrice), "Minimum validator price in nLUX per second")
	fs.Uint64(ValidatorFeesExcessConversionConstantKey, uint64(builder.LocalValidatorFeeConfig.ExcessConversionConstant), "Constant to convert validator excess price")
	// Dynamic fees:
	fs.Uint64(DynamicFeesBandwidthWeightKey, builder.LocalDynamicFeeConfig.Weights[gas.Bandwidth], "Complexity multiplier used to convert Bandwidth into Gas")
	fs.Uint64(DynamicFeesDBReadWeightKey, builder.LocalDynamicFeeConfig.Weights[gas.DBRead], "Complexity multiplier used to convert DB Reads into Gas")
	fs.Uint64(DynamicFeesDBWriteWeightKey, builder.LocalDynamicFeeConfig.Weights[gas.DBWrite], "Complexity multiplier used to convert DB Writes into Gas")
	fs.Uint64(DynamicFeesComputeWeightKey, builder.LocalDynamicFeeConfig.Weights[gas.Compute], "Complexity multiplier used to convert Compute into Gas")
	fs.Uint64(DynamicFeesMaxGasCapacityKey, uint64(builder.LocalDynamicFeeConfig.MaxCapacity), "Maximum amount of Gas the chain is allowed to store for future use")
	fs.Uint64(DynamicFeesMaxGasPerSecondKey, uint64(builder.LocalDynamicFeeConfig.MaxPerSecond), "Rate at which Gas is stored for future use")
	fs.Uint64(DynamicFeesTargetGasPerSecondKey, uint64(builder.LocalDynamicFeeConfig.TargetPerSecond), "Target rate of Gas usage")
	fs.Uint64(DynamicFeesMinGasPriceKey, uint64(builder.LocalDynamicFeeConfig.MinPrice), "Minimum Gas price")
	fs.Uint64(DynamicFeesExcessConversionConstantKey, uint64(builder.LocalDynamicFeeConfig.ExcessConversionConstant), "Constant to convert excess Gas to the Gas price")
	// Static fees:
	fs.Uint64(TxFeeKey, genesis.LocalParams.TxFee, "Transaction fee, in nLUX")
	fs.Uint64(CreateAssetTxFeeKey, genesis.LocalParams.CreateAssetTxFee, "Transaction fee, in nLUX, for transactions that create new assets")
	// Database
	fs.String(DBTypeKey, "badgerdb", "Default database type to use for all chains. Must be one of {badgerdb, pebbledb, memdb}")
	fs.Bool(DBReadOnlyKey, false, "If true, database writes are to memory and never persisted. May still initialize database directory/files on disk if they don't exist")
	fs.String(DBPathKey, defaultDBDir, "Path to database directory")
	fs.String(DBConfigFileKey, "", fmt.Sprintf("Path to database config file. Ignored if %s is specified", DBConfigContentKey))
	fs.String(DBConfigContentKey, "", "Specifies base64 encoded database config content")

	// Per-chain database configuration
	fs.String(PChainDBTypeKey, "", "Database type for P-Chain. If not specified, uses default db-type")
	fs.String(XChainDBTypeKey, "", "Database type for X-Chain. If not specified, uses default db-type")
	fs.String(CChainDBTypeKey, "", "Database type for C-Chain. If not specified, uses default db-type")

	// Logging
	fs.String(LogsDirKey, defaultLogDir, "Logging directory for Lux")
	fs.String(LogLevelKey, "info", "The log level. Should be one of {verbo, debug, trace, info, warn, error, fatal, off}")
	fs.String(LogDisplayLevelKey, "", "The log display level. If left blank, will inherit the value of log-level. Otherwise, should be one of {verbo, debug, trace, info, warn, error, fatal, off}")
// 	fs.String(LogFormatKey, "auto", "Format to use for log output. Should be one of {auto, json, terminal}")
	fs.Uint(LogRotaterMaxSizeKey, 8, "The maximum file size in megabytes of the log file before it gets rotated.")
	fs.Uint(LogRotaterMaxFilesKey, 7, "The maximum number of old log files to retain. 0 means retain all old log files.")
	fs.Uint(LogRotaterMaxAgeKey, 0, "The maximum number of days to retain old log files based on the timestamp encoded in their filename. 0 means retain all old log files.")
	fs.Bool(LogRotaterCompressEnabledKey, false, "Enables the compression of rotated log files through gzip.")
	fs.Bool(LogDisableDisplayPluginLogsKey, false, "Disables displaying plugin logs in stdout.")

	// Peer List Gossip
	fs.Uint(NetworkPeerListNumValidatorIPsKey, constants.DefaultNetworkPeerListNumValidatorIPs, "Number of validator IPs to gossip to other nodes")
	fs.Duration(NetworkPeerListPullGossipFreqKey, constants.DefaultNetworkPeerListPullGossipFreq, "Frequency to request peers from other nodes")
	fs.Duration(NetworkPeerListBloomResetFreqKey, constants.DefaultNetworkPeerListBloomResetFreq, "Frequency to recalculate the bloom filter used to request new peers from other nodes")

	// Public IP Resolution
	fs.String(PublicIPKey, "", "Public IP of this node for P2P communication")
	fs.Duration(PublicIPResolutionFreqKey, 5*time.Minute, "Frequency at which this node resolves/updates its public IP and renew NAT mappings, if applicable")
	fs.String(PublicIPResolutionServiceKey, "", fmt.Sprintf("Only acceptable values are %q, %q or %q. When provided, the node will use that service to periodically resolve/update its public IP", dynamicip.OpenDNSName, dynamicip.IFConfigCoName, dynamicip.IFConfigMeName))

	// Inbound Connection Throttling
	fs.Duration(NetworkInboundConnUpgradeThrottlerCooldownKey, constants.DefaultInboundConnUpgradeThrottlerCooldown, "Upgrade an inbound connection from a given IP at most once per this duration. If 0, don't rate-limit inbound connection upgrades")
	fs.Float64(NetworkInboundThrottlerMaxConnsPerSecKey, constants.DefaultInboundThrottlerMaxConnsPerSec, "Max number of inbound connections to accept (from all peers) per second")
	// Outbound Connection Throttling
	fs.Uint(NetworkOutboundConnectionThrottlingRpsKey, constants.DefaultOutboundConnectionThrottlingRps, "Make at most this number of outgoing peer connection attempts per second")
	fs.Duration(NetworkOutboundConnectionTimeoutKey, constants.DefaultOutboundConnectionTimeout, "Timeout when dialing a peer")
	// Timeouts
	fs.Duration(NetworkInitialTimeoutKey, constants.DefaultNetworkInitialTimeout, "Initial timeout value of the adaptive timeout manager")
	fs.Duration(NetworkMinimumTimeoutKey, constants.DefaultNetworkMinimumTimeout, "Minimum timeout value of the adaptive timeout manager")
	fs.Duration(NetworkMaximumTimeoutKey, constants.DefaultNetworkMaximumTimeout, "Maximum timeout value of the adaptive timeout manager")
	fs.Duration(NetworkMaximumInboundTimeoutKey, constants.DefaultNetworkMaximumInboundTimeout, "Maximum timeout value of an inbound message. Defines duration within which an incoming message must be fulfilled. Incoming messages containing deadline higher than this value will be overridden with this value.")
	fs.Duration(NetworkTimeoutHalflifeKey, constants.DefaultNetworkTimeoutHalflife, "Halflife of average network response time. Higher value --> network timeout is less volatile. Can't be 0")
	fs.Float64(NetworkTimeoutCoefficientKey, constants.DefaultNetworkTimeoutCoefficient, "Multiplied by average network response time to get the network timeout. Must be >= 1")
	fs.Duration(NetworkReadHandshakeTimeoutKey, constants.DefaultNetworkReadHandshakeTimeout, "Timeout value for reading handshake messages")
	fs.Duration(NetworkPingTimeoutKey, constants.DefaultPingPongTimeout, "Timeout value for Ping-Pong with a peer")
	fs.Duration(NetworkPingFrequencyKey, constants.DefaultPingFrequency, "Frequency of pinging other peers")
	fs.Duration(NetworkNoIngressValidatorConnectionsGracePeriodKey, constants.DefaultNoIngressValidatorConnectionGracePeriod, "Time after which nodes are expected to be connected to us if we are a primary network validator, otherwise a health check fails")
	fs.String(NetworkCompressionTypeKey, constants.DefaultNetworkCompressionType.String(), fmt.Sprintf("Compression type for outbound messages. Must be one of [%s, %s]", compression.TypeZstd, compression.TypeNone))

	fs.Duration(NetworkMaxClockDifferenceKey, constants.DefaultNetworkMaxClockDifference, "Max allowed clock difference value between this node and peers")
	// Allow private IPs by default for local development and testing.
	// Set to false to prohibit for production deployments.
	fs.Bool(NetworkAllowPrivateIPsKey, true, "Allows the node to initiate outbound connection attempts to peers with private IPs. Default is true. Set to false to prohibit for security-sensitive production deployments")
	fs.Bool(NetworkRequireValidatorToConnectKey, constants.DefaultNetworkRequireValidatorToConnect, "If true, this node will only maintain a connection with another node if this node is a validator, the other node is a validator, or the other node is a beacon")
	fs.Uint(NetworkPeerReadBufferSizeKey, constants.DefaultNetworkPeerReadBufferSize, "Size, in bytes, of the buffer that we read peer messages into (there is one buffer per peer)")
	fs.Uint(NetworkPeerWriteBufferSizeKey, constants.DefaultNetworkPeerWriteBufferSize, "Size, in bytes, of the buffer that we write peer messages into (there is one buffer per peer)")

	fs.Bool(NetworkTCPProxyEnabledKey, constants.DefaultNetworkTCPProxyEnabled, "Require all P2P connections to be initiated with a TCP proxy header")
	// The PROXY protocol specification recommends setting this value to be at
	// least 3 seconds to cover a TCP retransmit.
	// Ref: https://www.haproxy.org/download/2.3/doc/proxy-protocol.txt
	// Specifying a timeout of 0 will actually result in a timeout of 200ms, but
	// a timeout of 0 should generally not be provided.
	fs.Duration(NetworkTCPProxyReadTimeoutKey, constants.DefaultNetworkTCPProxyReadTimeout, "Maximum duration to wait for a TCP proxy header")

	fs.String(NetworkTLSKeyLogFileKey, "", "TLS key log file path. Should only be specified for debugging")

	// Benchlist
	fs.Int(BenchlistFailThresholdKey, constants.DefaultBenchlistFailThreshold, "Number of consecutive failed queries before benchlisting a node")
	fs.Duration(BenchlistDurationKey, constants.DefaultBenchlistDuration, "Max amount of time a peer is benchlisted after surpassing the threshold")
	fs.Duration(BenchlistMinFailingDurationKey, constants.DefaultBenchlistMinFailingDuration, "Minimum amount of time messages to a peer must be failing before the peer is benched")

	// Router
	fs.Uint(ConsensusAppConcurrencyKey, constants.DefaultConsensusAppConcurrency, "Maximum number of goroutines to use when handling App messages on a chain")
	fs.Duration(ConsensusShutdownTimeoutKey, constants.DefaultConsensusShutdownTimeout, "Timeout before killing an unresponsive chain")
	fs.Duration(ConsensusFrontierPollFrequencyKey, constants.DefaultFrontierPollFrequency, "Frequency of polling for new consensus frontiers")

	// Inbound Throttling
	fs.Uint64(InboundThrottlerAtLargeAllocSizeKey, constants.DefaultInboundThrottlerAtLargeAllocSize, "Size, in bytes, of at-large byte allocation in inbound message throttler")
	fs.Uint64(InboundThrottlerVdrAllocSizeKey, constants.DefaultInboundThrottlerVdrAllocSize, "Size, in bytes, of validator byte allocation in inbound message throttler")
	fs.Uint64(InboundThrottlerNodeMaxAtLargeBytesKey, constants.DefaultInboundThrottlerNodeMaxAtLargeBytes, "Max number of bytes a node can take from the inbound message throttler's at-large allocation. Must be at least the max message size")
	fs.Uint64(InboundThrottlerMaxProcessingMsgsPerNodeKey, constants.DefaultInboundThrottlerMaxProcessingMsgsPerNode, "Max number of messages currently processing from a given node")
	fs.Uint64(InboundThrottlerBandwidthRefillRateKey, constants.DefaultInboundThrottlerBandwidthRefillRate, "Max average inbound bandwidth usage of a peer, in bytes per second. See BandwidthThrottler")
	fs.Uint64(InboundThrottlerBandwidthMaxBurstSizeKey, constants.DefaultInboundThrottlerBandwidthMaxBurstSize, "Max inbound bandwidth a node can use at once. Must be at least the max message size. See BandwidthThrottler")
	fs.Duration(InboundThrottlerCPUMaxRecheckDelayKey, constants.DefaultInboundThrottlerCPUMaxRecheckDelay, "In the CPU-based network throttler, check at least this often whether the node's CPU usage has fallen to an acceptable level")
	fs.Duration(InboundThrottlerDiskMaxRecheckDelayKey, constants.DefaultInboundThrottlerDiskMaxRecheckDelay, "In the disk-based network throttler, check at least this often whether the node's disk usage has fallen to an acceptable level")

	// Outbound Throttling
	fs.Uint64(OutboundThrottlerAtLargeAllocSizeKey, constants.DefaultOutboundThrottlerAtLargeAllocSize, "Size, in bytes, of at-large byte allocation in outbound message throttler")
	fs.Uint64(OutboundThrottlerVdrAllocSizeKey, constants.DefaultOutboundThrottlerVdrAllocSize, "Size, in bytes, of validator byte allocation in outbound message throttler")
	fs.Uint64(OutboundThrottlerNodeMaxAtLargeBytesKey, constants.DefaultOutboundThrottlerNodeMaxAtLargeBytes, "Max number of bytes a node can take from the outbound message throttler's at-large allocation. Must be at least the max message size")

	// HTTP APIs
	fs.String(HTTPHostKey, "127.0.0.1", "Address of the HTTP server. If the address is empty or a literal unspecified IP address, the server will bind on all available unicast and anycast IP addresses of the local system")
	fs.Uint(HTTPPortKey, DefaultHTTPPort, "Port of the HTTP server. If the port is 0 a port number is automatically chosen")
	fs.Bool(HTTPSEnabledKey, false, "Upgrade the HTTP server to HTTPs")
	fs.String(HTTPSKeyFileKey, "", fmt.Sprintf("TLS private key file for the HTTPs server. Ignored if %s is specified", HTTPSKeyContentKey))
	fs.String(HTTPSKeyContentKey, "", "Specifies base64 encoded TLS private key for the HTTPs server")
	fs.String(HTTPSCertFileKey, "", fmt.Sprintf("TLS certificate file for the HTTPs server. Ignored if %s is specified", HTTPSCertContentKey))
	fs.String(HTTPSCertContentKey, "", "Specifies base64 encoded TLS certificate for the HTTPs server")
	fs.String(HTTPAllowedOrigins, "*", "Origins to allow on the HTTP port. Defaults to * which allows all origins. Example: https://*.lux.network https://*.lux-test.network")
	fs.StringSlice(HTTPAllowedHostsKey, []string{"localhost"}, "List of acceptable host names in API requests. Provide the wildcard ('*') to accept requests from all hosts. API requests where the Host field is empty or an IP address will always be accepted. An API call whose HTTP Host field isn't acceptable will receive a 403 error code")
	fs.Duration(HTTPShutdownWaitKey, 0, "Duration to wait after receiving SIGTERM or SIGINT before initiating shutdown. The /health endpoint will return unhealthy during this duration")
	fs.Duration(HTTPShutdownTimeoutKey, 10*time.Second, "Maximum duration to wait for existing connections to complete during node shutdown")
	fs.Duration(HTTPReadTimeoutKey, 3600*time.Second, "Maximum duration for reading the entire request, including the body. A zero or negative value means there will be no timeout")
	fs.Duration(HTTPReadHeaderTimeoutKey, 30*time.Second, fmt.Sprintf("Maximum duration to read request headers. The connection's read deadline is reset after reading the headers. If %s is zero, the value of %s is used. If both are zero, there is no timeout.", HTTPReadHeaderTimeoutKey, HTTPReadTimeoutKey))
	fs.Duration(HTTPWriteTimeoutKey, 3600*time.Second, "Maximum duration before timing out writes of the response. It is reset whenever a new request's header is read. A zero or negative value means there will be no timeout.")
	fs.Duration(HTTPIdleTimeoutKey, 120*time.Second, fmt.Sprintf("Maximum duration to wait for the next request when keep-alives are enabled. If %s is zero, the value of %s is used. If both are zero, there is no timeout.", HTTPIdleTimeoutKey, HTTPReadTimeoutKey))

	// Enable/Disable APIs
	fs.Bool(AdminAPIEnabledKey, false, "If true, this node exposes the Admin API")
	fs.Bool(InfoAPIEnabledKey, true, "If true, this node exposes the Info API")
	fs.Bool(KeystoreAPIEnabledKey, false, "If true, this node exposes the Keystore API")
	fs.Bool(MetricsAPIEnabledKey, true, "If true, this node exposes the Metrics API")
	fs.Bool(HealthAPIEnabledKey, true, "If true, this node exposes the Health API")

	// Health Checks
	fs.Duration(HealthCheckFreqKey, 30*time.Second, "Time between health checks")
	fs.Duration(HealthCheckAveragerHalflifeKey, constants.DefaultHealthCheckAveragerHalflife, "Halflife of averager when calculating a running average in a health check")
	// Network Layer Health
	fs.Duration(NetworkHealthMaxTimeSinceMsgSentKey, constants.DefaultNetworkHealthMaxTimeSinceMsgSent, "Network layer returns unhealthy if haven't sent a message for at least this much time")
	fs.Duration(NetworkHealthMaxTimeSinceMsgReceivedKey, constants.DefaultNetworkHealthMaxTimeSinceMsgReceived, "Network layer returns unhealthy if haven't received a message for at least this much time")
	fs.Float64(NetworkHealthMaxPortionSendQueueFillKey, constants.DefaultNetworkHealthMaxPortionSendQueueFill, "Network layer returns unhealthy if more than this portion of the pending send queue is full")
	fs.Uint(NetworkHealthMinPeersKey, constants.DefaultNetworkHealthMinPeers, "Network layer returns unhealthy if connected to less than this many peers")
	fs.Float64(NetworkHealthMaxSendFailRateKey, constants.DefaultNetworkHealthMaxSendFailRate, "Network layer reports unhealthy if more than this portion of attempted message sends fail")
	// Router Health
	fs.Float64(RouterHealthMaxDropRateKey, 1, "Node reports unhealthy if the router drops more than this portion of messages")
	fs.Uint(RouterHealthMaxOutstandingRequestsKey, 1024, "Node reports unhealthy if there are more than this many outstanding consensus requests (Get, PullQuery, etc.) over all chains")
	fs.Duration(NetworkHealthMaxOutstandingDurationKey, 5*time.Minute, "Node reports unhealthy if there has been a request outstanding for this duration")

	// Staking
	fs.String(StakingHostKey, "", "Address of the consensus server. If the address is empty or a literal unspecified IP address, the server will bind on all available unicast and anycast IP addresses of the local system") // Bind to all interfaces by default.
	fs.Uint(StakingPortKey, DefaultStakingPort, "Port of the consensus server. If the port is 0 a port number is automatically chosen")
	fs.Bool(StakingEphemeralCertEnabledKey, false, "If true, the node uses an ephemeral staking TLS key and certificate, and has an ephemeral node ID")
	fs.String(StakingTLSKeyPathKey, defaultStakingTLSKeyPath, fmt.Sprintf("Path to the TLS private key for staking. Ignored if %s is specified", StakingTLSKeyContentKey))
	fs.String(StakingTLSKeyContentKey, "", "Specifies base64 encoded TLS private key for staking")
	fs.String(StakingCertPathKey, defaultStakingCertPath, fmt.Sprintf("Path to the TLS certificate for staking. Ignored if %s is specified", StakingCertContentKey))
	fs.String(StakingCertContentKey, "", "Specifies base64 encoded TLS certificate for staking")
	fs.Bool(StakingEphemeralSignerEnabledKey, false, "If true, the node uses an ephemeral staking signer key")
	fs.String(StakingSignerKeyPathKey, defaultStakingSignerKeyPath, fmt.Sprintf("Path to the signer private key for staking. Ignored if %s is specified", StakingSignerKeyContentKey))
	fs.String(StakingSignerKeyContentKey, "", "Specifies base64 encoded signer private key for staking")
	fs.Bool(SybilProtectionEnabledKey, true, "Enables sybil protection. If enabled, Network TLS is required")
	fs.Uint64(SybilProtectionDisabledWeightKey, 100, "Weight to provide to each peer when sybil protection is disabled")
	fs.Bool(PartialSyncPrimaryNetworkKey, false, "Only sync the P-chain on the Primary Network. If the node is a Primary Network validator, it will report unhealthy")
	// Uptime Requirement
	fs.Float64(UptimeRequirementKey, genesis.LocalParams.UptimeRequirement, "Fraction of time a validator must be online to receive rewards")
	// Minimum Stake required to validate the Primary Network
	fs.Uint64(MinValidatorStakeKey, genesis.LocalParams.MinValidatorStake, "Minimum stake, in nLUX, required to validate the primary network")
	// Maximum Stake that can be staked and delegated to a validator on the Primary Network
	fs.Uint64(MaxValidatorStakeKey, genesis.LocalParams.MaxValidatorStake, "Maximum stake, in nLUX, that can be placed on a validator on the primary network")
	// Minimum Stake that can be delegated on the Primary Network
	fs.Uint64(MinDelegatorStakeKey, genesis.LocalParams.MinDelegatorStake, "Minimum stake, in nLUX, that can be delegated on the primary network")
	fs.Uint64(MinDelegatorFeeKey, uint64(genesis.LocalParams.MinDelegationFee), "Minimum delegation fee, in the range [0, 1000000], that can be charged for delegation on the primary network")
	// Minimum Stake Duration
	fs.Duration(MinStakeDurationKey, time.Duration(genesis.LocalParams.MinStakeDuration)*time.Second, "Minimum staking duration")
	// Maximum Stake Duration
	fs.Duration(MaxStakeDurationKey, time.Duration(genesis.LocalParams.MaxStakeDuration)*time.Second, "Maximum staking duration")
	// Stake Reward Configs
	fs.Uint64(StakeMaxConsumptionRateKey, genesis.LocalParams.RewardConfig.MaxConsumptionRate, "Maximum consumption rate of the remaining tokens to mint in the staking function")
	fs.Uint64(StakeMinConsumptionRateKey, genesis.LocalParams.RewardConfig.MinConsumptionRate, "Minimum consumption rate of the remaining tokens to mint in the staking function")
	fs.Duration(StakeMintingPeriodKey, time.Duration(genesis.LocalParams.RewardConfig.MintingPeriod)*time.Second, "Consumption period of the staking function")
	fs.Uint64(StakeSupplyCapKey, genesis.LocalParams.RewardConfig.SupplyCap, "Supply cap of the staking function")
	// Chain tracking
	fs.String(TrackChainsKey, "", "Comma-separated list of chain IDs to track. A node tracking chains will sync those chains and track validator uptimes. Use --track-all-chains for development networks")
	fs.Bool(TrackAllChainsKey, false, "If true, track all chains automatically. Useful for development and testing networks where you want to sync all deployed chains without specifying each one")

	// State syncing
	fs.String(StateSyncIPsKey, "", "Comma separated list of state sync peer ips to connect to. Example: 127.0.0.1:9630,127.0.0.1:9631")
	fs.String(StateSyncIDsKey, "", "Comma separated list of state sync peer ids to connect to. Example: NodeID-JR4dVmy6ffUGAKCBDkyCbeZbyHQBeDsET,NodeID-8CrVPQZ4VSqgL8zTdvL14G8HqAfrBr4z")

	// Bootstrapping
	fs.String(BootstrapIPsKey, "", "Comma separated list of bootstrap peer ips to connect to. Example: 127.0.0.1:9630,127.0.0.1:9631")
	fs.String(BootstrapIDsKey, "", "Comma separated list of bootstrap peer ids to connect to. Example: NodeID-JR4dVmy6ffUGAKCBDkyCbeZbyHQBeDsET,NodeID-8CrVPQZ4VSqgL8zTdvL14G8HqAfrBr4z")
	fs.Bool(SkipBootstrapKey, false, "If true, skip the bootstrapping phase and start processing immediately")
	fs.Bool(EnableAutominingKey, false, "If true, enable automining in POA mode")
	fs.Duration(BootstrapBeaconConnectionTimeoutKey, time.Minute, "Timeout before emitting a warn log when connecting to bootstrapping beacons")
	fs.Duration(BootstrapMaxTimeGetAncestorsKey, 50*time.Millisecond, "Max Time to spend fetching a container and its ancestors when responding to a GetAncestors")
	fs.Uint(BootstrapAncestorsMaxContainersSentKey, 2000, "Max number of containers in an Ancestors message sent by this node")
	fs.Uint(BootstrapAncestorsMaxContainersReceivedKey, 2000, "This node reads at most this many containers from an incoming Ancestors message")

	// Consensus - use defaults from consensus config package
	defaultParams := consensusconfig.DefaultParams()
	fs.Int(ConsensusSampleSizeKey, defaultParams.K, "Number of nodes to query for each network poll")
	fs.Int(ConsensusQuorumSizeKey, defaultParams.AlphaConfidence, "Threshold of nodes required to update this node's preference and increase its confidence in a network poll")
	fs.Int(ConsensusPreferenceQuorumSizeKey, defaultParams.AlphaPreference, fmt.Sprintf("Threshold of nodes required to update this node's preference in a network poll. Ignored if %s is provided", ConsensusQuorumSizeKey))
	fs.Int(ConsensusConfidenceQuorumSizeKey, defaultParams.AlphaConfidence, fmt.Sprintf("Threshold of nodes required to increase this node's confidence in a network poll. Ignored if %s is provided", ConsensusQuorumSizeKey))

	fs.Int(ConsensusCommitThresholdKey, int(defaultParams.Beta), "Beta value to use for consensus")

	fs.Int(ConsensusConcurrentRepollsKey, defaultParams.ConcurrentRepolls, "Minimum number of concurrent polls for finalizing consensus")
	fs.Int(ConsensusOptimalProcessingKey, defaultParams.OptimalProcessing, "Optimal number of processing containers in consensus")
	fs.Int(ConsensusMaxProcessingKey, defaultParams.MaxOutstandingItems, "Maximum number of processing items to be considered healthy")
	fs.Duration(ConsensusMaxTimeProcessingKey, defaultParams.MaxItemProcessingTime, "Maximum amount of time an item should be processing and still be healthy")

	// ProposerVM
	fs.Bool(ProposerVMUseCurrentHeightKey, false, "Have the ProposerVM always report the last accepted P-chain block height")
	fs.Duration(ProposerVMMinBlockDelayKey, proposervm.DefaultMinBlockDelay, "Minimum delay to enforce when building a chain++ block for the primary network chains and the default minimum delay for subnets")

	// Metrics
	fs.Bool(MeterVMsEnabledKey, true, "Enable Meter VMs to track VM performance with more granularity")
	fs.Duration(UptimeMetricFreqKey, 30*time.Second, "Frequency of renewing this node's average uptime metric")

	// Indexer
	fs.Bool(IndexEnabledKey, false, "If true, index all accepted containers and transactions and expose them via an API")
	fs.Bool(IndexAllowIncompleteKey, false, "If true, allow running the node in such a way that could cause an index to miss transactions. Ignored if index is disabled")

	// Config Directories
	fs.String(ChainConfigDirKey, defaultChainConfigDir, fmt.Sprintf("Chain specific configurations parent directory. Ignored if %s is specified", ChainConfigContentKey))
	fs.String(ChainConfigContentKey, "", "Specifies base64 encoded chains configurations")
	fs.String(NetConfigDirKey, defaultNetConfigDir, fmt.Sprintf("Net specific configurations parent directory. Ignored if %s is specified", NetConfigContentKey))
	fs.String(NetConfigContentKey, "", "Specifies base64 encoded subnets configurations")

	// Chain Data Directory
	fs.String(ChainDataDirKey, defaultChainDataDir, "Chain specific data directory")
	fs.String(ImportChainDataKey, "", "Path to import blockchain data from another chain into C-Chain")

	// Profiles
	fs.String(ProfileDirKey, defaultProfileDir, "Path to the profile directory")
	fs.Bool(ProfileContinuousEnabledKey, false, "Whether the app should continuously produce performance profiles")
	fs.Duration(ProfileContinuousFreqKey, 15*time.Minute, "How frequently to rotate performance profiles")
	fs.Int(ProfileContinuousMaxFilesKey, 5, "Maximum number of historical profiles to keep")

	// Aliasing
	fs.String(VMAliasesFileKey, defaultVMAliasFilePath, fmt.Sprintf("Specifies a JSON file that maps vmIDs with custom aliases. Ignored if %s is specified", VMAliasesContentKey))
	fs.String(VMAliasesContentKey, "", "Specifies base64 encoded maps vmIDs with custom aliases")
	fs.String(ChainAliasesFileKey, defaultChainAliasFilePath, fmt.Sprintf("Specifies a JSON file that maps blockchainIDs with custom aliases. Ignored if %s is specified", ChainConfigContentKey))
	fs.String(ChainAliasesContentKey, "", "Specifies base64 encoded map from blockchainID to custom aliases")

	// Delays
	fs.Duration(NetworkInitialReconnectDelayKey, constants.DefaultNetworkInitialReconnectDelay, "Initial delay duration must be waited before attempting to reconnect a peer")
	fs.Duration(NetworkMaxReconnectDelayKey, constants.DefaultNetworkMaxReconnectDelay, "Maximum delay duration must be waited before attempting to reconnect a peer")

	// System resource trackers
	fs.Duration(SystemTrackerFrequencyKey, 500*time.Millisecond, "Frequency to check the real system usage of tracked processes. More frequent checks --> usage metrics are more accurate, but more expensive to track")
	fs.Duration(SystemTrackerProcessingHalflifeKey, 15*time.Second, "Halflife to use for the processing requests tracker. Larger halflife --> usage metrics change more slowly")
	fs.Duration(SystemTrackerCPUHalflifeKey, 15*time.Second, "Halflife to use for the cpu tracker. Larger halflife --> cpu usage metrics change more slowly")
	fs.Duration(SystemTrackerDiskHalflifeKey, time.Minute, "Halflife to use for the disk tracker. Larger halflife --> disk usage metrics change more slowly")
	fs.Uint64(SystemTrackerRequiredAvailableDiskSpaceKey, units.GiB/2, "Minimum number of available bytes on disk, under which the node will shutdown.")
	fs.Uint64(SystemTrackerWarningThresholdAvailableDiskSpaceKey, units.GiB, fmt.Sprintf("Warning threshold for the number of available bytes on disk, under which the node will be considered unhealthy.  Must be >= [%s]", SystemTrackerRequiredAvailableDiskSpaceKey))

	// CPU management
	fs.Float64(CPUVdrAllocKey, float64(runtime.NumCPU()), "Maximum number of CPUs to allocate for use by validators. Value should be in range [0, total core count]")
	fs.Float64(CPUMaxNonVdrUsageKey, .8*float64(runtime.NumCPU()), "Number of CPUs that if fully utilized, will rate limit all non-validators. Value should be in range [0, total core count]")
	fs.Float64(CPUMaxNonVdrNodeUsageKey, float64(runtime.NumCPU())/8, "Maximum number of CPUs that a non-validator can utilize. Value should be in range [0, total core count]")

	// Disk management
	fs.Float64(DiskVdrAllocKey, 1000*units.GiB, "Maximum number of disk reads/writes per second to allocate for use by validators. Must be > 0")
	fs.Float64(DiskMaxNonVdrUsageKey, 1000*units.GiB, "Number of disk reads/writes per second that, if fully utilized, will rate limit all non-validators. Must be >= 0")
	fs.Float64(DiskMaxNonVdrNodeUsageKey, 1000*units.GiB, "Maximum number of disk reads/writes per second that a non-validator can utilize. Must be >= 0")

	// Opentelemetry tracing
	fs.String(TracingExporterTypeKey, trace.Disabled.String(), fmt.Sprintf("Type of exporter to use for tracing. Options are [%s, %s, %s]", trace.Disabled, trace.GRPC, trace.HTTP))
	fs.String(TracingEndpointKey, "", "The endpoint to send trace data to. If unspecified, the default endpoint will be used; depending on the exporter type")
	fs.Bool(TracingInsecureKey, true, "If true, don't use TLS when sending trace data")
	fs.Float64(TracingSampleRateKey, 0.1, "The fraction of traces to sample. If >= 1, always sample. If <= 0, never sample")
	fs.StringToString(TracingHeadersKey, map[string]string{}, "The headers to provide the trace indexer")

	fs.String(ProcessContextFileKey, defaultProcessContextPath, "The path to write process context to (including PID, API URI, and staking address).")

	// POA Mode
	fs.Bool(POAModeEnabledKey, false, "Enable Proof of Authority mode for subnets")
	fs.Bool(POASingleNodeModeKey, false, "Enable single node POA mode (no consensus required)")
	fs.Duration(POAMinBlockTimeKey, 1*time.Second, "Minimum time between blocks in POA mode")
	fs.StringSlice(POAAuthorizedNodesKey, nil, "List of authorized nodes for POA mode")

	// GPU Acceleration
	fs.Bool(GPUEnabledKey, true, "Enable GPU acceleration for cryptographic operations (auto-detects Metal on macOS, CUDA on Linux)")
	fs.String(GPUBackendKey, "auto", "GPU backend to use. Options: auto (auto-detect), metal (macOS), cuda (Linux), cpu (fallback)")
	fs.Int(GPUDeviceKey, 0, "GPU device index to use when multiple GPUs are available")
	fs.String(GPULogLevelKey, "warn", "GPU subsystem log level. Options: debug, info, warn, error")

	// Force flags
	fs.Bool(ForceIgnoreChecksumKey, false, "Force ignore checksum validation errors (use with caution)")
}

// BuildFlagSet returns a complete set of flags for node
func BuildFlagSet() *pflag.FlagSet {
	fs := pflag.NewFlagSet(constants.AppName, pflag.ContinueOnError)
	addProcessFlags(fs)
	addNodeFlags(fs)
	return fs
}

// getExpandedArg gets the string in viper corresponding to [key] and expands
// any variables using the OS env. If the [LuxNodeDataDirVar] var is used,
// we expand the value of the variable with the string in viper corresponding to
// [DataDirKey].
func getExpandedArg(v *viper.Viper, key string) string {
	return os.Expand(
		v.GetString(key),
		func(strVar string) string {
			if strVar == LuxNodeDataDirVar {
				return os.ExpandEnv(v.GetString(DataDirKey))
			}
			return os.Getenv(strVar)
		},
	)
}
