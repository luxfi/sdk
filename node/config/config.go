// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"

	consensusconfig "github.com/luxfi/consensus/config"
	"github.com/luxfi/const"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/crypto/bls/signer/localsigner"
	"github.com/luxfi/crypto/hash"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/api/server"
	"github.com/luxfi/sdk/node/benchlist"
	"github.com/luxfi/sdk/node/chains"
	"github.com/luxfi/sdk/node/config/node"
	"github.com/luxfi/sdk/node/genesis/builder"
	"github.com/luxfi/sdk/node/nets"
	"github.com/luxfi/sdk/node/network"
	"github.com/luxfi/sdk/node/network/dialer"
	"github.com/luxfi/sdk/node/vms/components/gas"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
	"github.com/luxfi/sdk/node/vms/platformvm/validators/fee"
	"github.com/luxfi/sdk/node/vms/proposervm"
	"github.com/luxfi/p2p/throttling"
	"github.com/luxfi/sdk/utils/compression"
	"github.com/luxfi/sdk/utils/ips"
	"github.com/luxfi/sdk/utils/perms"
	"github.com/luxfi/sdk/utils/profiler"
	"github.com/luxfi/sdk/utils/storage"
	"github.com/luxfi/sdk/utils/timer"
	luxtls "github.com/luxfi/tls"
	"github.com/luxfi/trace"
	"github.com/luxfi/upgrade"
	"github.com/luxfi/version"
)

// TrackerTargeterConfig contains resource allocation configurations
type TrackerTargeterConfig struct {
	VdrAlloc           float64 `json:"vdrAlloc"`
	MaxNonVdrUsage     float64 `json:"maxNonVdrUsage"`
	MaxNonVdrNodeUsage float64 `json:"maxNonVdrNodeUsage"`
}

const (
	chainConfigFileName  = "config"
	chainUpgradeFileName = "upgrade"
	subnetConfigFileExt  = ".json"
)

var (
	// Deprecated key --> deprecation message (i.e. which key replaces it)
	// TODO: deprecate "BootstrapIDsKey" and "BootstrapIPsKey"
	deprecatedKeys = map[string]string{}

	errConflictingLPOpinion                   = errors.New("supporting and objecting to the same LP")
	errConflictingImplicitLPOpinion           = errors.New("objecting to enabled LP")
	errSybilProtectionDisabledStakerWeights   = errors.New("sybil protection disabled weights must be positive")
	errSybilProtectionDisabledOnPublicNetwork = errors.New("sybil protection disabled on public network")
	errInvalidUptimeRequirement               = errors.New("uptime requirement must be in the range [0, 1]")
	errMinValidatorStakeAboveMax              = errors.New("minimum validator stake can't be greater than maximum validator stake")
	errInvalidDelegationFee                   = errors.New("delegation fee must be in the range [0, 1,000,000]")
	errInvalidMinStakeDuration                = errors.New("min stake duration must be > 0")
	errMinStakeDurationAboveMax               = errors.New("max stake duration can't be less than min stake duration")
	errStakeMaxConsumptionTooLarge            = fmt.Errorf("max stake consumption must be less than or equal to %d", reward.PercentDenominator)
	errStakeMaxConsumptionBelowMin            = errors.New("stake max consumption can't be less than min stake consumption")
	errStakeMintingPeriodBelowMin             = errors.New("stake minting period can't be less than max stake duration")
	errCannotTrackPrimaryNetwork              = errors.New("cannot track primary network")
	errStakingKeyContentUnset                 = fmt.Errorf("%s key not set but %s set", StakingTLSKeyContentKey, StakingCertContentKey)
	errStakingCertContentUnset                = fmt.Errorf("%s key set but %s not set", StakingTLSKeyContentKey, StakingCertContentKey)
	errMissingStakingSigningKeyFile           = errors.New("missing staking signing key file")
	errPluginDirNotADirectory                 = errors.New("plugin dir is not a directory")
	errCannotReadDirectory                    = errors.New("cannot read directory")
	errUnmarshalling                          = errors.New("unmarshalling failed")
	errFileDoesNotExist                       = errors.New("file does not exist")
	errDevNetworkGenesisMismatch              = errors.New("genesis mismatch: dev-network.json exists but produces different genesis hash; delete datadir or use original network config")
)

// DevNetworkConfig captures immutable network state for dev mode persistence.
// Once written to dev-network.json, this ensures the same genesis is produced
// on every restart, making C-Chain/EVM state persistence work correctly.
type DevNetworkConfig struct {
	// Version for forward compatibility
	Version int `json:"version"`

	// Genesis start time - captured on first boot, reused on restart
	StartTime uint64 `json:"startTime"`

	// Node identity
	NodeID string `json:"nodeId"`

	// BLS credentials (hex-encoded)
	BLSPublicKey string `json:"blsPublicKey"`
	BLSPopProof  string `json:"blsPopProof"`

	// Computed genesis bytes and hash (for verification)
	GenesisBytes []byte `json:"genesisBytes"`
	GenesisHash  string `json:"genesisHash"` // Actual hash of GenesisBytes

	// X-Chain asset ID (LUX token ID)
	XAssetID string `json:"xAssetId"`

	// C-Chain genesis (stored separately for EVM immutability)
	CChainGenesis string `json:"cChainGenesis"`
}

const (
	devNetworkConfigVersion  = 1
	devNetworkConfigFilename = "dev-network.json"
)

// loadDevNetworkConfig attempts to load dev-network.json from the data directory.
// Returns nil if the file doesn't exist (first boot scenario).
func loadDevNetworkConfig(dataDir string) (*DevNetworkConfig, error) {
	path := filepath.Join(dataDir, devNetworkConfigFilename)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // First boot - no config yet
		}
		return nil, fmt.Errorf("failed to read dev network config: %w", err)
	}

	var cfg DevNetworkConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse dev network config: %w", err)
	}

	if cfg.Version != devNetworkConfigVersion {
		return nil, fmt.Errorf("unsupported dev network config version: %d (expected %d)", cfg.Version, devNetworkConfigVersion)
	}

	return &cfg, nil
}

// saveDevNetworkConfig atomically writes the dev network config to disk.
// Uses write-to-temp-then-rename pattern for crash safety.
func saveDevNetworkConfig(dataDir string, cfg *DevNetworkConfig) error {
	path := filepath.Join(dataDir, devNetworkConfigFilename)
	tempPath := path + ".tmp"

	cfg.Version = devNetworkConfigVersion

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dev network config: %w", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write temp dev network config: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath) // Clean up on failure
		return fmt.Errorf("failed to rename dev network config: %w", err)
	}

	return nil
}

func getConsensusConfig(v *viper.Viper) consensusconfig.Parameters {
	// Start with default parameters
	p := consensusconfig.DefaultParams()

	// Override with config values if set
	if v.IsSet(ConsensusSampleSizeKey) {
		p.K = v.GetInt(ConsensusSampleSizeKey)
	}
	if v.IsSet(ConsensusPreferenceQuorumSizeKey) {
		p.AlphaPreference = v.GetInt(ConsensusPreferenceQuorumSizeKey)
	}
	if v.IsSet(ConsensusConfidenceQuorumSizeKey) {
		p.AlphaConfidence = v.GetInt(ConsensusConfidenceQuorumSizeKey)
	}
	if v.IsSet(ConsensusCommitThresholdKey) {
		p.Beta = uint32(v.GetInt(ConsensusCommitThresholdKey))
	}
	if v.IsSet(ConsensusConcurrentRepollsKey) {
		p.ConcurrentRepolls = v.GetInt(ConsensusConcurrentRepollsKey)
	}
	if v.IsSet(ConsensusOptimalProcessingKey) {
		p.OptimalProcessing = v.GetInt(ConsensusOptimalProcessingKey)
	}
	if v.IsSet(ConsensusMaxProcessingKey) {
		p.MaxOutstandingItems = v.GetInt(ConsensusMaxProcessingKey)
	}
	if v.IsSet(ConsensusMaxTimeProcessingKey) {
		p.MaxItemProcessingTime = v.GetDuration(ConsensusMaxTimeProcessingKey)
	}
	if v.IsSet(ConsensusQuorumSizeKey) {
		p.AlphaPreference = v.GetInt(ConsensusQuorumSizeKey)
		p.AlphaConfidence = p.AlphaPreference
	}
	return p
}

func getLoggingConfig(v *viper.Viper) (log.Config, error) {
	loggingConfig := log.Config{}
	loggingConfig.Directory = getExpandedArg(v, LogsDirKey)
	var err error
	loggingConfig.LogLevel, err = log.ToLevel(v.GetString(LogLevelKey))
	if err != nil {
		return loggingConfig, err
	}
	logDisplayLevel := v.GetString(LogLevelKey)
	if v.IsSet(LogDisplayLevelKey) {
		logDisplayLevel = v.GetString(LogDisplayLevelKey)
	}
	loggingConfig.DisplayLevel, err = log.ToLevel(logDisplayLevel)
	if err != nil {
		return loggingConfig, err
	}
	loggingConfig.LogFormat, err = log.ToFormat(v.GetString(LogFormatKey), os.Stdout.Fd())
	loggingConfig.DisableWriterDisplaying = v.GetBool(LogDisableDisplayPluginLogsKey)
	loggingConfig.MaxSize = int(v.GetUint(LogRotaterMaxSizeKey))
	loggingConfig.MaxFiles = int(v.GetUint(LogRotaterMaxFilesKey))
	loggingConfig.MaxAge = int(v.GetUint(LogRotaterMaxAgeKey))
	loggingConfig.Compress = v.GetBool(LogRotaterCompressEnabledKey)

	return loggingConfig, err
}

func getHTTPConfig(v *viper.Viper) (node.HTTPConfig, error) {
	var (
		httpsKey  []byte
		httpsCert []byte
		err       error
	)
	switch {
	case v.IsSet(HTTPSKeyContentKey):
		rawContent := v.GetString(HTTPSKeyContentKey)
		httpsKey, err = base64.StdEncoding.DecodeString(rawContent)
		if err != nil {
			return node.HTTPConfig{}, fmt.Errorf("unable to decode base64 content: %w", err)
		}
	case v.IsSet(HTTPSKeyFileKey):
		httpsKeyFilepath := getExpandedArg(v, HTTPSKeyFileKey)
		httpsKey, err = os.ReadFile(filepath.Clean(httpsKeyFilepath))
		if err != nil {
			return node.HTTPConfig{}, err
		}
	}

	switch {
	case v.IsSet(HTTPSCertContentKey):
		rawContent := v.GetString(HTTPSCertContentKey)
		httpsCert, err = base64.StdEncoding.DecodeString(rawContent)
		if err != nil {
			return node.HTTPConfig{}, fmt.Errorf("unable to decode base64 content: %w", err)
		}
	case v.IsSet(HTTPSCertFileKey):
		httpsCertFilepath := getExpandedArg(v, HTTPSCertFileKey)
		httpsCert, err = os.ReadFile(filepath.Clean(httpsCertFilepath))
		if err != nil {
			return node.HTTPConfig{}, err
		}
	}

	return node.HTTPConfig{
		HTTPConfig: server.HTTPConfig{
			ReadTimeout:       v.GetDuration(HTTPReadTimeoutKey),
			ReadHeaderTimeout: v.GetDuration(HTTPReadHeaderTimeoutKey),
			WriteTimeout:      v.GetDuration(HTTPWriteTimeoutKey),
			IdleTimeout:       v.GetDuration(HTTPIdleTimeoutKey),
		},
		APIConfig: node.APIConfig{
			APIIndexerConfig: node.APIIndexerConfig{
				IndexAPIEnabled:      v.GetBool(IndexEnabledKey),
				IndexAllowIncomplete: v.GetBool(IndexAllowIncompleteKey),
			},
			AdminAPIEnabled:    v.GetBool(AdminAPIEnabledKey),
			InfoAPIEnabled:     v.GetBool(InfoAPIEnabledKey),
			KeystoreAPIEnabled: v.GetBool(KeystoreAPIEnabledKey),
			MetricsAPIEnabled:  v.GetBool(MetricsAPIEnabledKey),
			HealthAPIEnabled:   v.GetBool(HealthAPIEnabledKey),
		},
		HTTPHost:           v.GetString(HTTPHostKey),
		HTTPPort:           uint16(v.GetUint(HTTPPortKey)),
		HTTPSEnabled:       v.GetBool(HTTPSEnabledKey),
		HTTPSKey:           httpsKey,
		HTTPSCert:          httpsCert,
		HTTPAllowedOrigins: v.GetStringSlice(HTTPAllowedOrigins),
		HTTPAllowedHosts:   v.GetStringSlice(HTTPAllowedHostsKey),
		ShutdownTimeout:    v.GetDuration(HTTPShutdownTimeoutKey),
		ShutdownWait:       v.GetDuration(HTTPShutdownWaitKey),
	}, nil
}

func getRouterHealthConfig(v *viper.Viper, halflife time.Duration) (RouterHealthConfig, error) {
	config := RouterHealthConfig{
		MaxDropRate:            v.GetFloat64(RouterHealthMaxDropRateKey),
		MaxOutstandingRequests: int(v.GetUint(RouterHealthMaxOutstandingRequestsKey)),
		MaxOutstandingDuration: v.GetDuration(NetworkHealthMaxOutstandingDurationKey),
		MaxRunTimeRequests:     v.GetDuration(NetworkMaximumTimeoutKey),
		MaxDropRateHalflife:    halflife,
	}
	switch {
	case config.MaxDropRate < 0 || config.MaxDropRate > 1:
		return RouterHealthConfig{}, fmt.Errorf("%q must be in [0,1]", RouterHealthMaxDropRateKey)
	case config.MaxOutstandingDuration <= 0:
		return RouterHealthConfig{}, fmt.Errorf("%q must be positive", NetworkHealthMaxOutstandingDurationKey)
	case config.MaxRunTimeRequests <= 0:
		return RouterHealthConfig{}, fmt.Errorf("%q must be positive", NetworkMaximumTimeoutKey)
	}
	return config, nil
}

func getAdaptiveTimeoutConfig(v *viper.Viper) (timer.AdaptiveTimeoutConfig, error) {
	config := timer.AdaptiveTimeoutConfig{
		InitialTimeout:     v.GetDuration(NetworkInitialTimeoutKey),
		MinimumTimeout:     v.GetDuration(NetworkMinimumTimeoutKey),
		MaximumTimeout:     v.GetDuration(NetworkMaximumTimeoutKey),
		TimeoutHalflife:    v.GetDuration(NetworkTimeoutHalflifeKey),
		TimeoutCoefficient: v.GetFloat64(NetworkTimeoutCoefficientKey),
	}
	switch {
	case config.MinimumTimeout < 1:
		return timer.AdaptiveTimeoutConfig{}, fmt.Errorf("%q must be positive", NetworkMinimumTimeoutKey)
	case config.MinimumTimeout > config.MaximumTimeout:
		return timer.AdaptiveTimeoutConfig{}, fmt.Errorf("%q must be >= %q", NetworkMaximumTimeoutKey, NetworkMinimumTimeoutKey)
	case config.InitialTimeout < config.MinimumTimeout || config.InitialTimeout > config.MaximumTimeout:
		return timer.AdaptiveTimeoutConfig{}, fmt.Errorf("%q must be in [%q, %q]", NetworkInitialTimeoutKey, NetworkMinimumTimeoutKey, NetworkMaximumTimeoutKey)
	case config.TimeoutHalflife <= 0:
		return timer.AdaptiveTimeoutConfig{}, fmt.Errorf("%q must > 0", NetworkTimeoutHalflifeKey)
	case config.TimeoutCoefficient < 1:
		return timer.AdaptiveTimeoutConfig{}, fmt.Errorf("%q must be >= 1", NetworkTimeoutCoefficientKey)
	}

	return config, nil
}

func getNetworkConfig(
	v *viper.Viper,
	networkID uint32,
	sybilProtectionEnabled bool,
	halflife time.Duration,
) (network.Config, error) {
	// Set the max number of recent inbound connections upgraded to be
	// equal to the max number of inbound connections per second.
	maxInboundConnsPerSec := v.GetFloat64(NetworkInboundThrottlerMaxConnsPerSecKey)
	upgradeCooldown := v.GetDuration(NetworkInboundConnUpgradeThrottlerCooldownKey)
	upgradeCooldownInSeconds := upgradeCooldown.Seconds()
	maxRecentConnsUpgraded := int(math.Ceil(maxInboundConnsPerSec * upgradeCooldownInSeconds))

	compressionType, err := compression.TypeFromString(v.GetString(NetworkCompressionTypeKey))
	if err != nil {
		return network.Config{}, err
	}

	// Allow private IPs by default for local development and testing.
	// Use --network-allow-private-ips=false to prohibit for production deployments.
	allowPrivateIPs := true
	if v.IsSet(NetworkAllowPrivateIPsKey) {
		allowPrivateIPs = v.GetBool(NetworkAllowPrivateIPsKey)
	}

	var supportedLPs set.Set[uint32]
	for _, lp := range v.GetIntSlice(LPSupportKey) {
		if lp < 0 || lp > math.MaxInt32 {
			return network.Config{}, fmt.Errorf("invalid LP: %d", lp)
		}
		supportedLPs.Add(uint32(lp))
	}

	var objectedLPs set.Set[uint32]
	for _, lp := range v.GetIntSlice(LPObjectKey) {
		if lp < 0 || lp > math.MaxInt32 {
			return network.Config{}, fmt.Errorf("invalid LP: %d", lp)
		}
		objectedLPs.Add(uint32(lp))
	}
	if supportedLPs.Overlaps(objectedLPs) {
		return network.Config{}, errConflictingLPOpinion
	}
	if constants.ScheduledLPs.Overlaps(objectedLPs) {
		return network.Config{}, errConflictingImplicitLPOpinion
	}

	// Because this node version has scheduled these LPs, we should notify
	// peers that we support these upgrades.
	supportedLPs.Union(constants.ScheduledLPs)

	// To decrease unnecessary network traffic, peers will not be notified of
	// objection or support of activated LPs.
	supportedLPs.Difference(constants.ActivatedLPs)
	objectedLPs.Difference(constants.ActivatedLPs)

	config := network.Config{
		ThrottlerConfig: network.ThrottlerConfig{
			MaxInboundConnsPerSec: maxInboundConnsPerSec,
			InboundConnUpgradeThrottlerConfig: throttling.InboundConnUpgradeThrottlerConfig{
				UpgradeCooldown:        upgradeCooldown,
				MaxRecentConnsUpgraded: maxRecentConnsUpgraded,
			},

			InboundMsgThrottlerConfig: throttling.InboundMsgThrottlerConfig{
				MsgByteThrottlerConfig: throttling.MsgByteThrottlerConfig{
					AtLargeAllocSize:    v.GetUint64(InboundThrottlerAtLargeAllocSizeKey),
					VdrAllocSize:        v.GetUint64(InboundThrottlerVdrAllocSizeKey),
					NodeMaxAtLargeBytes: v.GetUint64(InboundThrottlerNodeMaxAtLargeBytesKey),
				},
				BandwidthThrottlerConfig: throttling.BandwidthThrottlerConfig{
					RefillRate:   v.GetUint64(InboundThrottlerBandwidthRefillRateKey),
					MaxBurstSize: v.GetUint64(InboundThrottlerBandwidthMaxBurstSizeKey),
				},
				MaxProcessingMsgsPerNode: v.GetUint64(InboundThrottlerMaxProcessingMsgsPerNodeKey),
				CPUThrottlerConfig: throttling.SystemThrottlerConfig{
					MaxRecheckDelay: v.GetDuration(InboundThrottlerCPUMaxRecheckDelayKey),
				},
				DiskThrottlerConfig: throttling.SystemThrottlerConfig{
					MaxRecheckDelay: v.GetDuration(InboundThrottlerDiskMaxRecheckDelayKey),
				},
			},

			OutboundMsgThrottlerConfig: throttling.MsgByteThrottlerConfig{
				AtLargeAllocSize:    v.GetUint64(OutboundThrottlerAtLargeAllocSizeKey),
				VdrAllocSize:        v.GetUint64(OutboundThrottlerVdrAllocSizeKey),
				NodeMaxAtLargeBytes: v.GetUint64(OutboundThrottlerNodeMaxAtLargeBytesKey),
			},
		},

		HealthConfig: network.HealthConfig{
			Enabled:                                 sybilProtectionEnabled,
			MaxTimeSinceMsgSent:                     v.GetDuration(NetworkHealthMaxTimeSinceMsgSentKey),
			MaxTimeSinceMsgReceived:                 v.GetDuration(NetworkHealthMaxTimeSinceMsgReceivedKey),
			MaxPortionSendQueueBytesFull:            v.GetFloat64(NetworkHealthMaxPortionSendQueueFillKey),
			MinConnectedPeers:                       v.GetUint(NetworkHealthMinPeersKey),
			MaxSendFailRate:                         v.GetFloat64(NetworkHealthMaxSendFailRateKey),
			SendFailRateHalflife:                    halflife,
			NoIngressValidatorConnectionGracePeriod: v.GetDuration(NetworkNoIngressValidatorConnectionsGracePeriodKey),
		},

		ProxyEnabled:           v.GetBool(NetworkTCPProxyEnabledKey),
		ProxyReadHeaderTimeout: v.GetDuration(NetworkTCPProxyReadTimeoutKey),

		DialerConfig: dialer.Config{
			ThrottleRps:       v.GetUint32(NetworkOutboundConnectionThrottlingRpsKey),
			ConnectionTimeout: v.GetDuration(NetworkOutboundConnectionTimeoutKey),
		},

		TLSKeyLogFile: v.GetString(NetworkTLSKeyLogFileKey),

		TimeoutConfig: network.TimeoutConfig{
			PingPongTimeout:      v.GetDuration(NetworkPingTimeoutKey),
			ReadHandshakeTimeout: v.GetDuration(NetworkReadHandshakeTimeoutKey),
		},

		PeerListGossipConfig: network.PeerListGossipConfig{
			PeerListNumValidatorIPs: v.GetUint32(NetworkPeerListNumValidatorIPsKey),
			PeerListPullGossipFreq:  v.GetDuration(NetworkPeerListPullGossipFreqKey),
			PeerListBloomResetFreq:  v.GetDuration(NetworkPeerListBloomResetFreqKey),
		},

		DelayConfig: network.DelayConfig{
			MaxReconnectDelay:     v.GetDuration(NetworkMaxReconnectDelayKey),
			InitialReconnectDelay: v.GetDuration(NetworkInitialReconnectDelayKey),
		},

		MaxClockDifference:           v.GetDuration(NetworkMaxClockDifferenceKey),
		CompressionType:              compressionType,
		PingFrequency:                v.GetDuration(NetworkPingFrequencyKey),
		AllowPrivateIPs:              allowPrivateIPs,
		UptimeMetricFreq:             v.GetDuration(UptimeMetricFreqKey),
		MaximumInboundMessageTimeout: v.GetDuration(NetworkMaximumInboundTimeoutKey),

		SupportedLPs: supportedLPs,
		ObjectedLPs:  objectedLPs,

		RequireValidatorToConnect: v.GetBool(NetworkRequireValidatorToConnectKey),
		PeerReadBufferSize:        int(v.GetUint(NetworkPeerReadBufferSizeKey)),
		PeerWriteBufferSize:       int(v.GetUint(NetworkPeerWriteBufferSizeKey)),
	}

	switch {
	case config.HealthConfig.MaxTimeSinceMsgSent < 0:
		return network.Config{}, fmt.Errorf("%s must be >= 0", NetworkHealthMaxTimeSinceMsgSentKey)
	case config.HealthConfig.MaxTimeSinceMsgReceived < 0:
		return network.Config{}, fmt.Errorf("%s must be >= 0", NetworkHealthMaxTimeSinceMsgReceivedKey)
	case config.HealthConfig.MaxSendFailRate < 0 || config.HealthConfig.MaxSendFailRate > 1:
		return network.Config{}, fmt.Errorf("%s must be in [0,1]", NetworkHealthMaxSendFailRateKey)
	case config.HealthConfig.MaxPortionSendQueueBytesFull < 0 || config.HealthConfig.MaxPortionSendQueueBytesFull > 1:
		return network.Config{}, fmt.Errorf("%s must be in [0,1]", NetworkHealthMaxPortionSendQueueFillKey)
	case config.DialerConfig.ConnectionTimeout < 0:
		return network.Config{}, fmt.Errorf("%q must be >= 0", NetworkOutboundConnectionTimeoutKey)
	case config.PeerListPullGossipFreq < 0:
		return network.Config{}, fmt.Errorf("%s must be >= 0", NetworkPeerListPullGossipFreqKey)
	case config.PeerListBloomResetFreq < 0:
		return network.Config{}, fmt.Errorf("%s must be >= 0", NetworkPeerListBloomResetFreqKey)
	case config.ThrottlerConfig.InboundMsgThrottlerConfig.CPUThrottlerConfig.MaxRecheckDelay < constants.MinInboundThrottlerMaxRecheckDelay:
		return network.Config{}, fmt.Errorf("%s must be >= %d", InboundThrottlerCPUMaxRecheckDelayKey, constants.MinInboundThrottlerMaxRecheckDelay)
	case config.ThrottlerConfig.InboundMsgThrottlerConfig.DiskThrottlerConfig.MaxRecheckDelay < constants.MinInboundThrottlerMaxRecheckDelay:
		return network.Config{}, fmt.Errorf("%s must be >= %d", InboundThrottlerDiskMaxRecheckDelayKey, constants.MinInboundThrottlerMaxRecheckDelay)
	case config.MaxReconnectDelay < 0:
		return network.Config{}, fmt.Errorf("%s must be >= 0", NetworkMaxReconnectDelayKey)
	case config.InitialReconnectDelay < 0:
		return network.Config{}, fmt.Errorf("%s must be >= 0", NetworkInitialReconnectDelayKey)
	case config.MaxReconnectDelay < config.InitialReconnectDelay:
		return network.Config{}, fmt.Errorf("%s must be >= %s", NetworkMaxReconnectDelayKey, NetworkInitialReconnectDelayKey)
	case config.PingPongTimeout < 0:
		return network.Config{}, fmt.Errorf("%s must be >= 0", NetworkPingTimeoutKey)
	case config.PingFrequency < 0:
		return network.Config{}, fmt.Errorf("%s must be >= 0", NetworkPingFrequencyKey)
	case config.PingPongTimeout <= config.PingFrequency:
		return network.Config{}, fmt.Errorf("%s must be > %s", NetworkPingTimeoutKey, NetworkPingFrequencyKey)
	case config.ReadHandshakeTimeout < 0:
		return network.Config{}, fmt.Errorf("%s must be >= 0", NetworkReadHandshakeTimeoutKey)
	case config.MaxClockDifference < 0:
		return network.Config{}, fmt.Errorf("%s must be >= 0", NetworkMaxClockDifferenceKey)
	}
	return config, nil
}

func getBenchlistConfig(v *viper.Viper, consensusParameters consensusconfig.Parameters) (benchlist.Config, error) {
	// AlphaConfidence is used here to ensure that benching can't cause a
	// liveness failure. If AlphaPreference were used, the benchlist may grow to
	// a point that committing would be extremely unlikely to happen.
	_ = consensusParameters.AlphaConfidence
	_ = consensusParameters.K
	// Validate configuration but return empty benchlist.Config as it's deprecated
	duration := v.GetDuration(BenchlistDurationKey)
	minFailingDuration := v.GetDuration(BenchlistMinFailingDurationKey)
	switch {
	case duration < 0:
		return benchlist.Config{}, fmt.Errorf("%q must be >= 0", BenchlistDurationKey)
	case minFailingDuration < 0:
		return benchlist.Config{}, fmt.Errorf("%q must be >= 0", BenchlistMinFailingDurationKey)
	}
	return benchlist.Config{}, nil
}

func getStateSyncConfig(v *viper.Viper) (node.StateSyncConfig, error) {
	var (
		config       = node.StateSyncConfig{}
		stateSyncIPs = strings.Split(v.GetString(StateSyncIPsKey), ",")
		stateSyncIDs = strings.Split(v.GetString(StateSyncIDsKey), ",")
	)

	for _, ip := range stateSyncIPs {
		if ip == "" {
			continue
		}
		addr, err := ips.ParseAddrPort(ip)
		if err != nil {
			return node.StateSyncConfig{}, fmt.Errorf("couldn't parse state sync ip %s: %w", ip, err)
		}
		config.StateSyncIPs = append(config.StateSyncIPs, addr)
	}

	for _, id := range stateSyncIDs {
		if id == "" {
			continue
		}
		nodeID, err := ids.NodeIDFromString(id)
		if err != nil {
			return node.StateSyncConfig{}, fmt.Errorf("couldn't parse state sync peer id %s: %w", id, err)
		}
		config.StateSyncIDs = append(config.StateSyncIDs, nodeID)
	}

	lenIPs := len(config.StateSyncIPs)
	lenIDs := len(config.StateSyncIDs)
	if lenIPs != lenIDs {
		return node.StateSyncConfig{}, fmt.Errorf("expected the number of stateSyncIPs (%d) to match the number of stateSyncIDs (%d)", lenIPs, lenIDs)
	}

	return config, nil
}

func getBootstrapConfig(v *viper.Viper, networkID uint32) (node.BootstrapConfig, error) {
	config := node.BootstrapConfig{
		BootstrapBeaconConnectionTimeout:        v.GetDuration(BootstrapBeaconConnectionTimeoutKey),
		BootstrapMaxTimeGetAncestors:            v.GetDuration(BootstrapMaxTimeGetAncestorsKey),
		BootstrapAncestorsMaxContainersSent:     int(v.GetUint(BootstrapAncestorsMaxContainersSentKey)),
		BootstrapAncestorsMaxContainersReceived: int(v.GetUint(BootstrapAncestorsMaxContainersReceivedKey)),
		SkipBootstrap:                           v.GetBool(SkipBootstrapKey),
		EnableAutomining:                        v.GetBool(EnableAutominingKey),
	}

	// length equality.
	ipsSet := v.IsSet(BootstrapIPsKey)
	idsSet := v.IsSet(BootstrapIDsKey)
	if ipsSet && !idsSet {
		return node.BootstrapConfig{}, fmt.Errorf("set %q but didn't set %q", BootstrapIPsKey, BootstrapIDsKey)
	}
	if !ipsSet && idsSet {
		return node.BootstrapConfig{}, fmt.Errorf("set %q but didn't set %q", BootstrapIDsKey, BootstrapIPsKey)
	}
	if !ipsSet && !idsSet {
		var err error
		config.Bootstrappers, err = builder.SampleBootstrappers(networkID, 5)
		if err != nil {
			return node.BootstrapConfig{}, fmt.Errorf("failed to sample bootstrappers: %w", err)
		}
		return config, nil
	}

	bootstrapIPs := strings.Split(v.GetString(BootstrapIPsKey), ",")
	config.Bootstrappers = make([]builder.Bootstrapper, 0, len(bootstrapIPs))
	for _, bootstrapIP := range bootstrapIPs {
		ip := strings.TrimSpace(bootstrapIP)
		if ip == "" {
			continue
		}
		addr, err := ips.ParseAddrPort(ip)
		if err != nil {
			return node.BootstrapConfig{}, fmt.Errorf("couldn't parse bootstrap ip %s: %w", ip, err)
		}
		config.Bootstrappers = append(config.Bootstrappers, builder.Bootstrapper{
			// ID is populated below
			IP: addr,
		})
	}

	bootstrapIDs := strings.Split(v.GetString(BootstrapIDsKey), ",")
	bootstrapNodeIDs := make([]ids.NodeID, 0, len(bootstrapIDs))
	for _, bootstrapID := range bootstrapIDs {
		id := strings.TrimSpace(bootstrapID)
		if id == "" {
			continue
		}
		nodeID, err := ids.NodeIDFromString(id)
		if err != nil {
			return node.BootstrapConfig{}, fmt.Errorf("couldn't parse bootstrap peer id %s: %w", id, err)
		}
		bootstrapNodeIDs = append(bootstrapNodeIDs, nodeID)
	}

	if len(config.Bootstrappers) != len(bootstrapNodeIDs) {
		return node.BootstrapConfig{}, fmt.Errorf("expected the number of bootstrapIPs (%d) to match the number of bootstrapIDs (%d)", len(config.Bootstrappers), len(bootstrapNodeIDs))
	}
	for i, nodeID := range bootstrapNodeIDs {
		config.Bootstrappers[i].ID = nodeID
	}

	return config, nil
}

func getIPConfig(v *viper.Viper) (node.IPConfig, error) {
	ipConfig := node.IPConfig{
		PublicIP:                  v.GetString(PublicIPKey),
		PublicIPResolutionService: v.GetString(PublicIPResolutionServiceKey),
		PublicIPResolutionFreq:    v.GetDuration(PublicIPResolutionFreqKey),
		ListenHost:                v.GetString(StakingHostKey),
		ListenPort:                uint16(v.GetUint(StakingPortKey)),
	}
	if ipConfig.PublicIPResolutionFreq <= 0 {
		return node.IPConfig{}, fmt.Errorf("%q must be > 0", PublicIPResolutionFreqKey)
	}
	if ipConfig.PublicIP != "" && ipConfig.PublicIPResolutionService != "" {
		return node.IPConfig{}, fmt.Errorf("only one of --%s and --%s can be given", PublicIPKey, PublicIPResolutionServiceKey)
	}
	return ipConfig, nil
}

func getProfilerConfig(v *viper.Viper) (profiler.Config, error) {
	config := profiler.Config{
		Dir:         getExpandedArg(v, ProfileDirKey),
		Enabled:     v.GetBool(ProfileContinuousEnabledKey),
		Freq:        v.GetDuration(ProfileContinuousFreqKey),
		MaxNumFiles: v.GetInt(ProfileContinuousMaxFilesKey),
	}
	if config.Freq < 0 {
		return profiler.Config{}, fmt.Errorf("%s must be >= 0", ProfileContinuousFreqKey)
	}
	return config, nil
}

func getStakingTLSCertFromFlag(v *viper.Viper) (tls.Certificate, error) {
	stakingKeyRawContent := v.GetString(StakingTLSKeyContentKey)
	stakingKeyContent, err := base64.StdEncoding.DecodeString(stakingKeyRawContent)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("unable to decode base64 content: %w", err)
	}

	stakingCertRawContent := v.GetString(StakingCertContentKey)
	stakingCertContent, err := base64.StdEncoding.DecodeString(stakingCertRawContent)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("unable to decode base64 content: %w", err)
	}

	cert, err := luxtls.LoadTLSCertFromBytes(stakingKeyContent, stakingCertContent)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed creating cert: %w", err)
	}

	return *cert, nil
}

func getStakingTLSCertFromFile(v *viper.Viper) (tls.Certificate, error) {
	// Parse the staking key/cert paths and expand environment variables
	stakingKeyPath := getExpandedArg(v, StakingTLSKeyPathKey)
	stakingCertPath := getExpandedArg(v, StakingCertPathKey)

	// If staking key/cert locations are specified but not found, error
	if v.IsSet(StakingTLSKeyPathKey) || v.IsSet(StakingCertPathKey) {
		if _, err := os.Stat(stakingKeyPath); os.IsNotExist(err) {
			return tls.Certificate{}, fmt.Errorf("couldn't find staking key at %s", stakingKeyPath)
		} else if _, err := os.Stat(stakingCertPath); os.IsNotExist(err) {
			return tls.Certificate{}, fmt.Errorf("couldn't find staking certificate at %s", stakingCertPath)
		}
	} else {
		// Create the staking key/cert if [stakingKeyPath] and [stakingCertPath] don't exist
		if err := luxtls.InitNodeStakingKeyPair(stakingKeyPath, stakingCertPath); err != nil {
			return tls.Certificate{}, fmt.Errorf("couldn't generate staking key/cert: %w", err)
		}
	}

	// Load and parse the staking key/cert
	cert, err := luxtls.LoadTLSCertFromFiles(stakingKeyPath, stakingCertPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("couldn't read staking certificate: %w", err)
	}
	return *cert, nil
}

func getStakingTLSCert(v *viper.Viper) (tls.Certificate, error) {
	if v.GetBool(StakingEphemeralCertEnabledKey) {
		// Use an ephemeral staking key/cert
		cert, err := luxtls.NewTLSCert()
		if err != nil {
			return tls.Certificate{}, fmt.Errorf("couldn't generate ephemeral staking key/cert: %w", err)
		}
		return *cert, nil
	}

	switch {
	case v.IsSet(StakingTLSKeyContentKey) && !v.IsSet(StakingCertContentKey):
		return tls.Certificate{}, errStakingCertContentUnset
	case !v.IsSet(StakingTLSKeyContentKey) && v.IsSet(StakingCertContentKey):
		return tls.Certificate{}, errStakingKeyContentUnset
	case v.IsSet(StakingTLSKeyContentKey) && v.IsSet(StakingCertContentKey):
		return getStakingTLSCertFromFlag(v)
	default:
		cert, err := getStakingTLSCertFromFile(v)
		if err != nil {
			return tls.Certificate{}, err
		}
		// Verify Leaf is populated
		if cert.Leaf == nil {
			return tls.Certificate{}, fmt.Errorf("cert.Leaf is nil after loading from file")
		}
		return cert, nil
	}
}

func getStakingSigner(v *viper.Viper) (bls.Signer, error) {
	if v.GetBool(StakingEphemeralSignerEnabledKey) {
		key, err := localsigner.New()
		if err != nil {
			return nil, fmt.Errorf("couldn't generate ephemeral signing key: %w", err)
		}
		return key, nil
	}

	if v.IsSet(StakingSignerKeyContentKey) {
		signerKeyRawContent := v.GetString(StakingSignerKeyContentKey)
		signerKeyContent, err := base64.StdEncoding.DecodeString(signerKeyRawContent)
		if err != nil {
			return nil, fmt.Errorf("unable to decode base64 content: %w", err)
		}
		key, err := localsigner.FromBytes(signerKeyContent)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse signing key: %w", err)
		}
		return key, nil
	}

	signingKeyPath := getExpandedArg(v, StakingSignerKeyPathKey)
	_, err := os.Stat(signingKeyPath)
	if !errors.Is(err, fs.ErrNotExist) {
		signingKeyBytes, err := os.ReadFile(signingKeyPath)
		if err != nil {
			return nil, err
		}
		key, err := localsigner.FromBytes(signingKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse signing key: %w", err)
		}
		return key, nil
	}

	if v.IsSet(StakingSignerKeyPathKey) {
		return nil, errMissingStakingSigningKeyFile
	}

	key, err := localsigner.New()
	if err != nil {
		return nil, fmt.Errorf("couldn't generate new signing key: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(signingKeyPath), perms.ReadWriteExecute); err != nil {
		return nil, fmt.Errorf("couldn't create path for signing key at %s: %w", signingKeyPath, err)
	}

	keyBytes := key.ToBytes()
	if err := os.WriteFile(signingKeyPath, keyBytes, perms.ReadWrite); err != nil {
		return nil, fmt.Errorf("couldn't write new signing key to %s: %w", signingKeyPath, err)
	}
	if err := os.Chmod(signingKeyPath, perms.ReadOnly); err != nil {
		return nil, fmt.Errorf("couldn't restrict permissions on new signing key at %s: %w", signingKeyPath, err)
	}
	return key, nil
}

func getStakingConfig(v *viper.Viper, networkID uint32) (node.StakingConfig, error) {
	config := node.StakingConfig{
		SybilProtectionEnabled:        v.GetBool(SybilProtectionEnabledKey),
		SybilProtectionDisabledWeight: v.GetUint64(SybilProtectionDisabledWeightKey),
		PartialSyncPrimaryNetwork:     v.GetBool(PartialSyncPrimaryNetworkKey),
		StakingKeyPath:                getExpandedArg(v, StakingTLSKeyPathKey),
		StakingCertPath:               getExpandedArg(v, StakingCertPathKey),
		StakingSignerPath:             getExpandedArg(v, StakingSignerKeyPathKey),
	}
	if !config.SybilProtectionEnabled && config.SybilProtectionDisabledWeight == 0 {
		return node.StakingConfig{}, errSybilProtectionDisabledStakerWeights
	}

	if !config.SybilProtectionEnabled && (networkID == constants.MainnetID || networkID == constants.TestnetID) && !v.GetBool(DevModeKey) {
		return node.StakingConfig{}, errSybilProtectionDisabledOnPublicNetwork
	}

	var err error
	config.StakingTLSCert, err = getStakingTLSCert(v)
	if err != nil {
		return node.StakingConfig{}, err
	}
	config.StakingSigningKey, err = getStakingSigner(v)
	if err != nil {
		return node.StakingConfig{}, err
	}
	if networkID != constants.MainnetID && networkID != constants.TestnetID {
		config.UptimeRequirement = v.GetFloat64(UptimeRequirementKey)
		config.MinValidatorStake = v.GetUint64(MinValidatorStakeKey)
		config.MaxValidatorStake = v.GetUint64(MaxValidatorStakeKey)
		config.MinDelegatorStake = v.GetUint64(MinDelegatorStakeKey)
		config.MinStakeDuration = v.GetDuration(MinStakeDurationKey)
		config.MaxStakeDuration = v.GetDuration(MaxStakeDurationKey)
		config.RewardConfig.MaxConsumptionRate = v.GetUint64(StakeMaxConsumptionRateKey)
		config.RewardConfig.MinConsumptionRate = v.GetUint64(StakeMinConsumptionRateKey)
		config.RewardConfig.MintingPeriod = v.GetDuration(StakeMintingPeriodKey)
		config.RewardConfig.SupplyCap = v.GetUint64(StakeSupplyCapKey)
		config.MinDelegationFee = v.GetUint32(MinDelegatorFeeKey)
		switch {
		case config.UptimeRequirement < 0 || config.UptimeRequirement > 1:
			return node.StakingConfig{}, errInvalidUptimeRequirement
		case config.MinValidatorStake > config.MaxValidatorStake:
			return node.StakingConfig{}, errMinValidatorStakeAboveMax
		case config.MinDelegationFee > 1_000_000:
			return node.StakingConfig{}, errInvalidDelegationFee
		case config.MinStakeDuration <= 0:
			return node.StakingConfig{}, errInvalidMinStakeDuration
		case config.MaxStakeDuration < config.MinStakeDuration:
			return node.StakingConfig{}, errMinStakeDurationAboveMax
		case config.RewardConfig.MaxConsumptionRate > reward.PercentDenominator:
			return node.StakingConfig{}, errStakeMaxConsumptionTooLarge
		case config.RewardConfig.MaxConsumptionRate < config.RewardConfig.MinConsumptionRate:
			return node.StakingConfig{}, errStakeMaxConsumptionBelowMin
		case config.RewardConfig.MintingPeriod < config.MaxStakeDuration:
			return node.StakingConfig{}, errStakeMintingPeriodBelowMin
		}
	} else {
		config.StakingConfig = builder.GetStakingConfig(networkID)
	}
	return config, nil
}

func getTxFeeConfig(v *viper.Viper, networkID uint32) builder.TxFeeConfig {
	if networkID != constants.MainnetID && networkID != constants.TestnetID {
		return builder.TxFeeConfig{
			CreateAssetTxFee: v.GetUint64(CreateAssetTxFeeKey),
			TxFee:            v.GetUint64(TxFeeKey),
			DynamicFeeConfig: gas.Config{
				Weights: gas.Dimensions{
					gas.Bandwidth: v.GetUint64(DynamicFeesBandwidthWeightKey),
					gas.DBRead:    v.GetUint64(DynamicFeesDBReadWeightKey),
					gas.DBWrite:   v.GetUint64(DynamicFeesDBWriteWeightKey),
					gas.Compute:   v.GetUint64(DynamicFeesComputeWeightKey),
				},
				MaxCapacity:              gas.Gas(v.GetUint64(DynamicFeesMaxGasCapacityKey)),
				MaxPerSecond:             gas.Gas(v.GetUint64(DynamicFeesMaxGasPerSecondKey)),
				TargetPerSecond:          gas.Gas(v.GetUint64(DynamicFeesTargetGasPerSecondKey)),
				MinPrice:                 gas.Price(v.GetUint64(DynamicFeesMinGasPriceKey)),
				ExcessConversionConstant: gas.Gas(v.GetUint64(DynamicFeesExcessConversionConstantKey)),
			},
			ValidatorFeeConfig: fee.Config{
				Capacity:                 gas.Gas(v.GetUint64(ValidatorFeesCapacityKey)),
				Target:                   gas.Gas(v.GetUint64(ValidatorFeesTargetKey)),
				MinPrice:                 gas.Price(v.GetUint64(ValidatorFeesMinPriceKey)),
				ExcessConversionConstant: gas.Gas(v.GetUint64(ValidatorFeesExcessConversionConstantKey)),
			},
		}
	}
	return builder.GetTxFeeConfig(networkID)
}

func getUpgradeConfig(v *viper.Viper, networkID uint32) (upgrade.Config, error) {
	if !v.IsSet(UpgradeFileKey) && !v.IsSet(UpgradeFileContentKey) {
		return upgrade.GetConfig(networkID), nil
	}

	switch networkID {
	case constants.MainnetID, constants.TestnetID, constants.CustomID:
		return upgrade.Config{}, fmt.Errorf("cannot configure upgrades for networkID: %s",
			constants.NetworkName(networkID),
		)
	}

	var (
		upgradeBytes []byte
		err          error
	)
	switch {
	case v.IsSet(UpgradeFileKey):
		upgradeFileName := getExpandedArg(v, UpgradeFileKey)
		upgradeBytes, err = os.ReadFile(upgradeFileName)
		if err != nil {
			return upgrade.Config{}, fmt.Errorf("unable to read upgrade file: %w", err)
		}
	case v.IsSet(UpgradeFileContentKey):
		upgradeContent := v.GetString(UpgradeFileContentKey)
		upgradeBytes, err = base64.StdEncoding.DecodeString(upgradeContent)
		if err != nil {
			return upgrade.Config{}, fmt.Errorf("unable to decode upgrade base64 content: %w", err)
		}
	}

	var upgradeConfig upgrade.Config
	if err := json.Unmarshal(upgradeBytes, &upgradeConfig); err != nil {
		return upgrade.Config{}, fmt.Errorf("unable to unmarshal upgrade bytes: %w", err)
	}
	return upgradeConfig, nil
}

func getGenesisData(v *viper.Viper, networkID uint32, stakingCfg *builder.StakingConfig, dataDir string) ([]byte, ids.ID, error) {
	// Get allow-custom-genesis flag (defaults to true for development)
	allowCustomGenesis := v.GetBool(AllowCustomGenesisKey)

	// Handle dev mode genesis - dynamically generate genesis with the node's own credentials
	if v.GetBool(DevModeKey) && !v.IsSet(GenesisFileKey) && !v.IsSet(GenesisFileContentKey) && !v.IsSet(GenesisDBKey) {
		return getOrCreateDevModeGenesis(stakingCfg, dataDir)
	}

	// Check if genesis-db is specified for database replay
	if v.IsSet(GenesisDBKey) {
		if v.IsSet(GenesisFileKey) || v.IsSet(GenesisFileContentKey) {
			return nil, ids.Empty, fmt.Errorf("cannot specify %s with %s or %s", GenesisDBKey, GenesisFileKey, GenesisFileContentKey)
		}
		genesisDBPath := getExpandedArg(v, GenesisDBKey)
		genesisDBType := v.GetString(GenesisDBTypeKey)

		// Auto-detect database type based on path if not specified
		if genesisDBType == "" {
			genesisDBType = "badgerdb" // Default is always badgerdb
		}

		return builder.FromDatabase(networkID, genesisDBPath, genesisDBType, stakingCfg)
	}

	// try first loading genesis content directly from flag/env-var
	if v.IsSet(GenesisFileContentKey) {
		genesisData := v.GetString(GenesisFileContentKey)
		return builder.FromFlag(networkID, genesisData, stakingCfg, allowCustomGenesis)
	}

	// if content is not specified go for the file
	if v.IsSet(GenesisFileKey) {
		genesisFileName := getExpandedArg(v, GenesisFileKey)
		return builder.FromFile(networkID, genesisFileName, stakingCfg, allowCustomGenesis)
	}

	// finally if file is not specified/readable go for the predefined config
	config := builder.GetConfig(networkID)
	return builder.FromConfig(config)
}

// getOrCreateDevModeGenesis handles dev mode genesis with persistence.
// On first boot: generates genesis, saves to dev-network.json, returns genesis.
// On restart: loads from dev-network.json to ensure genesis hash stability.
// Returns: (genesisBytes, xAssetID, error) - xAssetID is the LUX token asset ID
func getOrCreateDevModeGenesis(stakingCfg *builder.StakingConfig, dataDir string) ([]byte, ids.ID, error) {
	// Try to load existing dev network config
	devCfg, err := loadDevNetworkConfig(dataDir)
	if err != nil {
		return nil, ids.Empty, fmt.Errorf("failed to load dev network config: %w", err)
	}

	if devCfg != nil {
		// Existing config found - check if credentials match
		// In dev mode with ephemeral certs, credentials will change on restart.
		// We log a warning but continue with the stored genesis since dev mode
		// is single-node and doesn't require credential consistency.
		expectedNodeID := stakingCfg.NodeID
		expectedBLSPK := fmt.Sprintf("0x%x", stakingCfg.BLSPublicKey)
		expectedBLSPoP := fmt.Sprintf("0x%x", stakingCfg.BLSProofOfPossession)

		credentialsMismatch := false
		if devCfg.NodeID != expectedNodeID {
			log.Warn("dev-network.json nodeID mismatch (using stored genesis anyway for dev mode)",
				"stored", devCfg.NodeID,
				"current", expectedNodeID,
			)
			credentialsMismatch = true
		}
		if devCfg.BLSPublicKey != expectedBLSPK {
			log.Warn("dev-network.json BLS public key mismatch (using stored genesis anyway for dev mode)")
			credentialsMismatch = true
		}
		if devCfg.BLSPopProof != expectedBLSPoP {
			log.Warn("dev-network.json BLS PoP mismatch (using stored genesis anyway for dev mode)")
			credentialsMismatch = true
		}
		if credentialsMismatch {
			log.Warn("credentials changed since first boot - for persistent staking, use --staking-ephemeral-cert-enabled=false with persistent key files")
		}

		// Verify the stored genesis hash matches what we'll compute from the bytes
		storedHash, err := ids.FromString(devCfg.GenesisHash)
		if err != nil {
			return nil, ids.Empty, fmt.Errorf("invalid genesis hash in dev-network.json: %w", err)
		}

		// Compute actual hash from stored bytes to verify integrity
		computedHashBytes := hash.ComputeHash256(devCfg.GenesisBytes)
		computedHash, err := ids.ToID(computedHashBytes)
		if err != nil {
			return nil, ids.Empty, fmt.Errorf("failed to convert computed hash to ID: %w", err)
		}

		if storedHash != computedHash {
			return nil, ids.Empty, fmt.Errorf("genesis bytes corrupted: stored hash %s != computed hash %s", storedHash, computedHash)
		}

		// Parse stored X-Chain asset ID
		xAssetID, err := ids.FromString(devCfg.XAssetID)
		if err != nil {
			return nil, ids.Empty, fmt.Errorf("invalid xAssetId in dev-network.json: %w", err)
		}

		log.Info("loaded dev network config",
			"path", filepath.Join(dataDir, devNetworkConfigFilename),
			"genesisHash", devCfg.GenesisHash,
			"xAssetID", devCfg.XAssetID,
			"startTime", devCfg.StartTime,
		)

		return devCfg.GenesisBytes, xAssetID, nil
	}

	// First boot - generate new genesis with current timestamp
	startTime := uint64(time.Now().Unix())

	// buildDevModeGenesis returns (genesisBytes, xAssetID) - the LUX token asset ID
	genesisBytes, xAssetID, err := buildDevModeGenesis(stakingCfg, startTime)
	if err != nil {
		return nil, ids.Empty, err
	}

	// Compute actual genesis hash from bytes (this is what the node uses for DB validation)
	genesisHashBytes := hash.ComputeHash256(genesisBytes)
	genesisHash, err := ids.ToID(genesisHashBytes)
	if err != nil {
		return nil, ids.Empty, fmt.Errorf("failed to convert genesis hash to ID: %w", err)
	}

	// Save for future restarts
	newDevCfg := &DevNetworkConfig{
		Version:       devNetworkConfigVersion,
		StartTime:     startTime,
		NodeID:        stakingCfg.NodeID,
		BLSPublicKey:  fmt.Sprintf("0x%x", stakingCfg.BLSPublicKey),
		BLSPopProof:   fmt.Sprintf("0x%x", stakingCfg.BLSProofOfPossession),
		GenesisBytes:  genesisBytes,
		GenesisHash:   genesisHash.String(),
		XAssetID:      xAssetID.String(),
		CChainGenesis: devModeCChainGenesis,
	}

	if err := saveDevNetworkConfig(dataDir, newDevCfg); err != nil {
		// Log warning but don't fail - genesis is still valid
		log.Warn("failed to save dev network config (persistence may not work on restart)",
			"error", err,
		)
	} else {
		log.Info("created dev network config",
			"path", filepath.Join(dataDir, devNetworkConfigFilename),
			"genesisHash", genesisHash.String(),
			"xAssetID", xAssetID.String(),
			"startTime", startTime,
		)
	}

	return genesisBytes, xAssetID, nil
}

// buildDevModeGenesis creates a genesis configuration for single-node development mode.
// It uses the node's own credentials as the sole validator.
func buildDevModeGenesis(stakingCfg *builder.StakingConfig, startTime uint64) ([]byte, ids.ID, error) {
	// Parse node ID from staking config
	nodeID, err := ids.NodeIDFromString(stakingCfg.NodeID)
	if err != nil {
		return nil, ids.Empty, fmt.Errorf("failed to parse node ID for dev mode: %w", err)
	}

	// Lux Treasury address: 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714
	// This is derived from LUX_MNEMONIC and funded on C-Chain
	var rewardAddress ids.ShortID
	treasuryAddr := "9011E888251AB053B7bD1cdB598Db4f9DEd94714"
	treasuryBytes, err := ids.ShortFromString(treasuryAddr)
	if err == nil {
		rewardAddress = treasuryBytes
	} else {
		// Fall back to a deterministic address derived from node ID
		copy(rewardAddress[:], nodeID[:20])
	}

	// Create dev mode config with embedded C-Chain genesis
	devCfg := builder.DevModeConfig{
		NodeID:        nodeID,
		BLSPublicKey:  fmt.Sprintf("0x%x", stakingCfg.BLSPublicKey),
		BLSPopProof:   fmt.Sprintf("0x%x", stakingCfg.BLSProofOfPossession),
		RewardAddress: rewardAddress,
		CChainGenesis: devModeCChainGenesis,
		StartTime:     startTime, // Use provided start time for determinism
	}

	return builder.ForDevMode(devCfg, stakingCfg)
}

// devModeCChainGenesis is the default C-Chain genesis for dev mode.
// Chain ID 1337, funds Lux Treasury (0x9011) and Anvil/Hardhat accounts.
const devModeCChainGenesis = `{
  "config": {
    "chainId": 1337,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "muirGlacierBlock": 0,
    "berlinBlock": 0,
    "londonBlock": 0,
    "arrowGlacierBlock": 0,
    "grayGlacierBlock": 0,
    "mergeNetsplitBlock": 0,
    "shanghaiTime": 0,
    "cancunTime": 0,
    "terminalTotalDifficulty": 0,
    "subnetEVMTimestamp": 0,
    "durangoTimestamp": 0,
    "etnaTimestamp": 253399622400,
    "feeConfig": {
      "gasLimit": 30000000,
      "targetBlockRate": 1,
      "minBaseFee": 1000000000,
      "targetGas": 100000000,
      "baseFeeChangeDenominator": 48,
      "minBlockGasCost": 0,
      "maxBlockGasCost": 1000000,
      "blockGasCostStep": 200000
    },
    "warpConfig": {
      "blockTimestamp": 0,
      "quorumNumerator": 67,
      "requirePrimaryNetworkSigners": false
    },
    "dexConfig": {
      "blockTimestamp": 0,
      "maxPools": 10000,
      "enableFlashLoans": true,
      "enableHooks": true
    }
  },
  "alloc": {
    "9011E888251AB053B7bD1cdB598Db4f9DEd94714": {
      "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
    },
    "f39Fd6e51aad88F6F4ce6aB8827279cffFb92266": {
      "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
    },
    "70997970C51812dc3A010C7d01b50e0d17dc79C8": {
      "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
    },
    "3C44CdDdB6a900fa2b585dd299e03d12FA4293BC": {
      "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
    },
    "90F79bf6EB2c4f870365E785982E1f101E93b906": {
      "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
    }
  },
  "nonce": "0x0",
  "timestamp": "0x0",
  "extraData": "0x",
  "gasLimit": "0x1c9c380",
  "difficulty": "0x0",
  "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "coinbase": "0x0000000000000000000000000000000000000000",
  "number": "0x0",
  "gasUsed": "0x0",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000"
}`

func getTrackedChains(v *viper.Viper) (set.Set[ids.ID], error) {
	trackChainsStr := v.GetString(TrackChainsKey)

	// Special value "all" means track all chains dynamically
	// This is indicated by returning nil (handled specially by chain manager)
	if strings.ToLower(strings.TrimSpace(trackChainsStr)) == "all" {
		return nil, nil // nil set means "track all"
	}

	trackChainsStrs := strings.Split(trackChainsStr, ",")
	trackedChainIDs := set.NewSet[ids.ID](len(trackChainsStrs))

	for _, chain := range trackChainsStrs {
		if chain == "" {
			continue
		}

		// Parse chain ID
		chainID, err := ids.FromString(chain)

		if err != nil {
			return nil, fmt.Errorf("couldn't parse chainID %q: %w", chain, err)
		}
		if chainID == constants.PrimaryNetworkID {
			return nil, errCannotTrackPrimaryNetwork
		}
		trackedChainIDs.Add(chainID)
	}
	return trackedChainIDs, nil
}

func getDatabaseConfig(v *viper.Viper, networkID uint32) (node.DatabaseConfig, error) {
	var (
		configBytes []byte
		err         error
	)
	if v.IsSet(DBConfigContentKey) {
		dbConfigContent := v.GetString(DBConfigContentKey)
		configBytes, err = base64.StdEncoding.DecodeString(dbConfigContent)
		if err != nil {
			return node.DatabaseConfig{}, fmt.Errorf("unable to decode base64 content: %w", err)
		}
	} else if v.IsSet(DBConfigFileKey) {
		path := getExpandedArg(v, DBConfigFileKey)
		configBytes, err = os.ReadFile(path)
		if err != nil {
			return node.DatabaseConfig{}, err
		}
	}

	return node.DatabaseConfig{
		Name:     v.GetString(DBTypeKey),
		ReadOnly: v.GetBool(DBReadOnlyKey),
		Path: filepath.Join(
			getExpandedArg(v, DBPathKey),
			constants.NetworkName(networkID),
		),
		Config: configBytes,
	}, nil
}

func getAliases(v *viper.Viper, name string, contentKey string, fileKey string) (map[ids.ID][]string, error) {
	var fileBytes []byte
	if v.IsSet(contentKey) {
		var err error
		aliasFlagContent := v.GetString(contentKey)
		fileBytes, err = base64.StdEncoding.DecodeString(aliasFlagContent)
		if err != nil {
			return nil, fmt.Errorf("unable to decode base64 content for %s: %w", name, err)
		}
	} else {
		aliasFilePath := filepath.Clean(getExpandedArg(v, fileKey))
		exists, err := storage.FileExists(aliasFilePath)
		if err != nil {
			return nil, err
		}

		if !exists {
			if v.IsSet(fileKey) {
				return nil, fmt.Errorf("%w: %s", errFileDoesNotExist, aliasFilePath)
			}
			return nil, nil
		}

		fileBytes, err = os.ReadFile(aliasFilePath)
		if err != nil {
			return nil, err
		}
	}

	aliasMap := make(map[ids.ID][]string)
	if err := json.Unmarshal(fileBytes, &aliasMap); err != nil {
		return nil, fmt.Errorf("%w on %s: %w", errUnmarshalling, name, err)
	}
	return aliasMap, nil
}

func getVMAliases(v *viper.Viper) (map[ids.ID][]string, error) {
	return getAliases(v, "vm aliases", VMAliasesContentKey, VMAliasesFileKey)
}

func getChainAliases(v *viper.Viper) (map[ids.ID][]string, error) {
	return getAliases(v, "chain aliases", ChainAliasesContentKey, ChainAliasesFileKey)
}

// getPathFromDirKey reads flag value from viper instance and then checks the folder existence
func getPathFromDirKey(v *viper.Viper, configKey string) (string, error) {
	configDir := getExpandedArg(v, configKey)
	cleanPath := filepath.Clean(configDir)
	ok, err := storage.FolderExists(cleanPath)
	if err != nil {
		return "", err
	}
	if ok {
		return cleanPath, nil
	}
	if v.IsSet(configKey) {
		// user specified a config dir explicitly, but dir does not exist.
		return "", fmt.Errorf("%w: %s", errCannotReadDirectory, cleanPath)
	}
	return "", nil
}

func getChainConfigsFromFlag(v *viper.Viper) (map[string]chains.ChainConfig, error) {
	chainConfigContentB64 := v.GetString(ChainConfigContentKey)
	chainConfigContent, err := base64.StdEncoding.DecodeString(chainConfigContentB64)
	if err != nil {
		return nil, fmt.Errorf("unable to decode base64 content: %w", err)
	}

	chainConfigs := make(map[string]chains.ChainConfig)
	if err := json.Unmarshal(chainConfigContent, &chainConfigs); err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON: %w", err)
	}
	return chainConfigs, nil
}

func getChainConfigsFromDir(v *viper.Viper) (map[string]chains.ChainConfig, error) {
	chainConfigPath, err := getPathFromDirKey(v, ChainConfigDirKey)
	if err != nil {
		return nil, err
	}

	if len(chainConfigPath) == 0 {
		return make(map[string]chains.ChainConfig), nil
	}

	return readChainConfigPath(chainConfigPath)
}

// getChainConfigs reads & puts chainConfigs to node config
func getChainConfigs(v *viper.Viper) (map[string]chains.ChainConfig, error) {
	if v.IsSet(ChainConfigContentKey) {
		return getChainConfigsFromFlag(v)
	}
	return getChainConfigsFromDir(v)
}

// readChainConfigPath reads chain config files from static directories and returns map with contents,
// if successful.
func readChainConfigPath(chainConfigPath string) (map[string]chains.ChainConfig, error) {
	chainDirs, err := filepath.Glob(filepath.Join(chainConfigPath, "*"))
	if err != nil {
		return nil, err
	}
	chainConfigMap := make(map[string]chains.ChainConfig)
	for _, chainDir := range chainDirs {
		dirInfo, err := os.Stat(chainDir)
		if err != nil {
			return nil, err
		}

		if !dirInfo.IsDir() {
			continue
		}

		// chainconfigdir/chainId/config.*
		configData, err := storage.ReadFileWithName(chainDir, chainConfigFileName)
		if err != nil {
			return chainConfigMap, err
		}

		// chainconfigdir/chainId/upgrade.*
		upgradeData, err := storage.ReadFileWithName(chainDir, chainUpgradeFileName)
		if err != nil {
			return chainConfigMap, err
		}

		chainConfigMap[dirInfo.Name()] = chains.ChainConfig{
			Config:  configData,
			Upgrade: upgradeData,
		}
	}
	return chainConfigMap, nil
}

// getNetConfigs reads net configs from the correct place
// (flag or file) and returns a non-nil map.
func getNetConfigs(v *viper.Viper, netIDs []ids.ID) (map[ids.ID]nets.Config, error) {
	if v.IsSet(NetConfigContentKey) {
		return getNetConfigsFromFlags(v, netIDs)
	}
	return getNetConfigsFromDir(v, netIDs)
}

func getNetConfigsFromFlags(v *viper.Viper, netIDs []ids.ID) (map[ids.ID]nets.Config, error) {
	netConfigContentB64 := v.GetString(NetConfigContentKey)
	netConfigContent, err := base64.StdEncoding.DecodeString(netConfigContentB64)
	if err != nil {
		return nil, fmt.Errorf("unable to decode base64 content: %w", err)
	}

	// partially parse configs to be filled by defaults later
	subnetConfigs := make(map[ids.ID]json.RawMessage, len(netIDs))
	if err := json.Unmarshal(netConfigContent, &subnetConfigs); err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON: %w", err)
	}

	res := make(map[ids.ID]nets.Config)
	for _, subnetID := range netIDs {
		config := getDefaultNetConfig(v)

		if rawNetConfigBytes, ok := subnetConfigs[subnetID]; ok {
			if err := json.Unmarshal(rawNetConfigBytes, &config); err != nil {
				return nil, err
			}

			// Alpha override disabled - field not available in consensus.Parameters
			// if config.ConsensusParameters.Alpha != nil && *config.ConsensusParameters.Alpha > 0 {
			//     config.ConsensusParameters.AlphaPreference = *config.ConsensusParameters.Alpha
			//     config.ConsensusParameters.AlphaConfidence = config.ConsensusParameters.AlphaPreference
			// }

			if err := config.Valid(); err != nil {
				return nil, err
			}
		}

		res[subnetID] = config
	}
	return res, nil
}

// getNetConfigsFromDir reads NetConfigs to node config map
func getNetConfigsFromDir(v *viper.Viper, subnetIDs []ids.ID) (map[ids.ID]nets.Config, error) {
	subnetConfigPath, err := getPathFromDirKey(v, NetConfigDirKey)
	if err != nil {
		return nil, err
	}

	subnetConfigs := make(map[ids.ID]nets.Config)

	// reads subnet config files from a path and given subnetIDs and returns a map.
	for _, subnetID := range subnetIDs {
		// Ensure default configuration
		config := getDefaultNetConfig(v)
		subnetConfigs[subnetID] = config

		if len(subnetConfigPath) == 0 {
			// subnet config path does not exist but not explicitly specified, so ignore it
			continue
		}

		filePath := filepath.Join(subnetConfigPath, subnetID.String()+subnetConfigFileExt)
		fileInfo, err := os.Stat(filePath)
		switch {
		case errors.Is(err, os.ErrNotExist):
			// this subnet config does not exist, the default configuration will be used
			continue
		case err != nil:
			return nil, err
		case fileInfo.IsDir():
			return nil, fmt.Errorf("%q is a directory, expected a file", fileInfo.Name())
		}

		// netConfigDir/netID.json
		file, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		// Update the default config with the values from the file
		if err := json.Unmarshal(file, &config); err != nil {
			return nil, fmt.Errorf("%w: %w", errUnmarshalling, err)
		}

		// Only override if Alpha is explicitly set (not zero)
		// Alpha override disabled - field not available in consensus.Parameters
		// if config.ConsensusParameters.Alpha != nil && *config.ConsensusParameters.Alpha > 0 {
		//     config.ConsensusParameters.AlphaPreference = *config.ConsensusParameters.Alpha
		//     config.ConsensusParameters.AlphaConfidence = config.ConsensusParameters.AlphaPreference
		// }

		if err := config.Valid(); err != nil {
			return nil, err
		}

		subnetConfigs[subnetID] = config
	}

	return subnetConfigs, nil
}

func getDefaultNetConfig(v *viper.Viper) nets.Config {
	config := nets.Config{
		ConsensusParameters:         getConsensusConfig(v),
		ValidatorOnly:               false,
		ProposerMinBlockDelay:       v.GetDuration(ProposerVMMinBlockDelayKey),
		ProposerNumHistoricalBlocks: proposervm.DefaultNumHistoricalBlocks,
		POAEnabled:                  v.GetBool(DevModeKey) || v.GetBool(POAModeEnabledKey),
		POASingleNodeMode:           v.GetBool(DevModeKey) || v.GetBool(POASingleNodeModeKey),
		POAMinBlockTime:             v.GetDuration(POAMinBlockTimeKey),
	}

	// If dev mode or POA mode is enabled, adjust consensus parameters
	if config.POAEnabled {
		config.ConsensusParameters = nets.GetPOAConsensusParameters()
		if config.POAMinBlockTime == 0 {
			config.POAMinBlockTime = 1 * time.Second
		}
		config.ProposerMinBlockDelay = config.POAMinBlockTime
	}

	return config
}

func getCPUTargeterConfig(v *viper.Viper) (TrackerTargeterConfig, error) {
	vdrAlloc := v.GetFloat64(CPUVdrAllocKey)
	maxNonVdrUsage := v.GetFloat64(CPUMaxNonVdrUsageKey)
	maxNonVdrNodeUsage := v.GetFloat64(CPUMaxNonVdrNodeUsageKey)
	switch {
	case vdrAlloc < 0:
		return TrackerTargeterConfig{}, fmt.Errorf("%q (%f) < 0", CPUVdrAllocKey, vdrAlloc)
	case maxNonVdrUsage < 0:
		return TrackerTargeterConfig{}, fmt.Errorf("%q (%f) < 0", CPUMaxNonVdrUsageKey, maxNonVdrUsage)
	case maxNonVdrNodeUsage < 0:
		return TrackerTargeterConfig{}, fmt.Errorf("%q (%f) < 0", CPUMaxNonVdrNodeUsageKey, maxNonVdrNodeUsage)
	default:
		return TrackerTargeterConfig{
			VdrAlloc:           vdrAlloc,
			MaxNonVdrUsage:     maxNonVdrUsage,
			MaxNonVdrNodeUsage: maxNonVdrNodeUsage,
		}, nil
	}
}

func getDiskSpaceConfig(v *viper.Viper) (requiredAvailableDiskSpace uint64, warningThresholdAvailableDiskSpace uint64, err error) {
	requiredAvailableDiskSpace = v.GetUint64(SystemTrackerRequiredAvailableDiskSpaceKey)
	warningThresholdAvailableDiskSpace = v.GetUint64(SystemTrackerWarningThresholdAvailableDiskSpaceKey)
	switch {
	case warningThresholdAvailableDiskSpace < requiredAvailableDiskSpace:
		return 0, 0, fmt.Errorf("%q (%d) < %q (%d)", SystemTrackerWarningThresholdAvailableDiskSpaceKey, warningThresholdAvailableDiskSpace, SystemTrackerRequiredAvailableDiskSpaceKey, requiredAvailableDiskSpace)
	default:
		return requiredAvailableDiskSpace, warningThresholdAvailableDiskSpace, nil
	}
}

func getDiskTargeterConfig(v *viper.Viper) (TrackerTargeterConfig, error) {
	vdrAlloc := v.GetFloat64(DiskVdrAllocKey)
	maxNonVdrUsage := v.GetFloat64(DiskMaxNonVdrUsageKey)
	maxNonVdrNodeUsage := v.GetFloat64(DiskMaxNonVdrNodeUsageKey)
	switch {
	case vdrAlloc < 0:
		return TrackerTargeterConfig{}, fmt.Errorf("%q (%f) < 0", DiskVdrAllocKey, vdrAlloc)
	case maxNonVdrUsage < 0:
		return TrackerTargeterConfig{}, fmt.Errorf("%q (%f) < 0", DiskMaxNonVdrUsageKey, maxNonVdrUsage)
	case maxNonVdrNodeUsage < 0:
		return TrackerTargeterConfig{}, fmt.Errorf("%q (%f) < 0", DiskMaxNonVdrNodeUsageKey, maxNonVdrNodeUsage)
	default:
		return TrackerTargeterConfig{
			VdrAlloc:           vdrAlloc,
			MaxNonVdrUsage:     maxNonVdrUsage,
			MaxNonVdrNodeUsage: maxNonVdrNodeUsage,
		}, nil
	}
}

func getTraceConfig(v *viper.Viper) (trace.Config, error) {
	exporterTypeStr := v.GetString(TracingExporterTypeKey)
	exporterType, err := trace.ExporterTypeFromString(exporterTypeStr)
	if err != nil {
		return trace.Config{}, err
	}

	return trace.Config{
		ExporterConfig: trace.ExporterConfig{
			Type:     exporterType,
			Endpoint: v.GetString(TracingEndpointKey),
			Insecure: v.GetBool(TracingInsecureKey),
			Headers:  v.GetStringMapString(TracingHeadersKey),
		},
		TraceSampleRate: v.GetFloat64(TracingSampleRateKey),
		AppName:         constants.AppName,
		Version:         version.Current.String(),
	}, nil
}

// Returns the path to the directory that contains VM binaries.
func getPluginDir(v *viper.Viper) (string, error) {
	pluginDir := getExpandedArg(v, PluginDirKey)

	if v.IsSet(PluginDirKey) {
		// If the flag was given, assert it exists and is a directory
		info, err := os.Stat(pluginDir)
		if err != nil {
			return "", fmt.Errorf("plugin dir %q not found: %w", pluginDir, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("%w: %q", errPluginDirNotADirectory, pluginDir)
		}
	} else {
		// If the flag wasn't given, make sure the default location exists.
		if err := os.MkdirAll(pluginDir, perms.ReadWriteExecute); err != nil {
			return "", fmt.Errorf("failed to create plugin dir at %s: %w", pluginDir, err)
		}
	}

	return pluginDir, nil
}

func GetNodeConfig(v *viper.Viper) (node.Config, error) {
	var (
		nodeConfig node.Config
		err        error
	)

	// Handle --dev flag first
	if v.GetBool(DevModeKey) {
		// Development mode sets various flags for single-node operation
		v.Set(SybilProtectionEnabledKey, false)
		v.Set(SybilProtectionDisabledWeightKey, 100)
		v.Set(ConsensusSampleSizeKey, 1)
		v.Set(ConsensusQuorumSizeKey, 1)
		// v.Set(ConsensusVirtuousCommitThresholdKey, 1) // Removed - key no longer exists
		// v.Set(ConsensusRogueCommitThresholdKey, 1) // Removed - key no longer exists
		v.Set(POASingleNodeModeKey, true)
		v.Set(SkipBootstrapKey, true)
		v.Set(NetworkHealthMinPeersKey, 0)
		// Enable automining for anvil-like behavior (auto-produce blocks on transactions)
		v.Set(EnableAutominingKey, true)
		// Use custom network (ID 1337) by default for dev mode unless explicitly set
		// This gives a standard dev chain ID like Hardhat/Anvil (1337)
		if !v.IsSet(NetworkNameKey) {
			v.Set(NetworkNameKey, "custom") // maps to network ID 1337
		}
		// Use ephemeral staking credentials for dev mode unless explicitly set
		// This avoids needing to match genesis validators with local staking keys
		if !v.IsSet(StakingEphemeralCertEnabledKey) && !v.IsSet(StakingTLSKeyContentKey) && !v.IsSet(StakingTLSKeyPathKey) {
			v.Set(StakingEphemeralCertEnabledKey, true)
		}
		if !v.IsSet(StakingEphemeralSignerEnabledKey) && !v.IsSet(StakingSignerKeyContentKey) && !v.IsSet(StakingSignerKeyPathKey) {
			v.Set(StakingEphemeralSignerEnabledKey, true)
		}
		// Use port 8545 for dev mode (standard Ethereum RPC port, like Anvil/Hardhat)
		// This avoids conflicts with mainnet (9630), testnet (9640), devnet (9650) ports
		if !v.IsSet(HTTPPortKey) {
			v.Set(HTTPPortKey, 8545)
		}
	}

	nodeConfig.PluginDir, err = getPluginDir(v)
	if err != nil {
		return node.Config{}, err
	}

	// GPU configuration - must be initialized early before any GPU accelerators
	gpuConfig := GPUConfig{
		Enabled:     v.GetBool(GPUEnabledKey),
		Backend:     v.GetString(GPUBackendKey),
		DeviceIndex: v.GetInt(GPUDeviceKey),
		LogLevel:    v.GetString(GPULogLevelKey),
	}
	if err := SetGlobalGPUConfig(gpuConfig); err != nil {
		return node.Config{}, fmt.Errorf("invalid GPU configuration: %w", err)
	}

	nodeConfig.ConsensusShutdownTimeout = v.GetDuration(ConsensusShutdownTimeoutKey)
	if nodeConfig.ConsensusShutdownTimeout < 0 {
		return node.Config{}, fmt.Errorf("%q must be >= 0", ConsensusShutdownTimeoutKey)
	}

	// Gossiping
	nodeConfig.FrontierPollFrequency = v.GetDuration(ConsensusFrontierPollFrequencyKey)
	if nodeConfig.FrontierPollFrequency < 0 {
		return node.Config{}, fmt.Errorf("%s must be >= 0", ConsensusFrontierPollFrequencyKey)
	}

	// App handling
	nodeConfig.ConsensusAppConcurrency = int(v.GetUint(ConsensusAppConcurrencyKey))
	if nodeConfig.ConsensusAppConcurrency <= 0 {
		return node.Config{}, fmt.Errorf("%s must be > 0", ConsensusAppConcurrencyKey)
	}

	nodeConfig.UseCurrentHeight = v.GetBool(ProposerVMUseCurrentHeightKey)

	// Logging
	// 	nodeConfig.LoggingConfig, err = getLoggingConfig(v)
	// 	if err != nil {
	// 		return node.Config{}, err
	// 	}
	//
	// Network ID - use network name (shorthand flags removed)
	// if v.GetBool(MainnetKey) {
	// 	nodeConfig.NetworkID = constants.MainnetChainID
	// } else if v.GetBool(TestnetKey) {
	// 	nodeConfig.NetworkID = constants.TestnetChainID
	// } else if v.GetBool(CustomnetKey) {
	// 	nodeConfig.NetworkID = constants.CustomID
	// } else {
	networkName := v.GetString(NetworkNameKey)
	nodeConfig.NetworkID, err = constants.NetworkID(networkName)
	if err != nil {
		return node.Config{}, err
	}
	// }

	// Database
	nodeConfig.DatabaseConfig, err = getDatabaseConfig(v, nodeConfig.NetworkID)
	if err != nil {
		return node.Config{}, err
	}

	// IP configuration
	nodeConfig.IPConfig, err = getIPConfig(v)
	if err != nil {
		return node.Config{}, err
	}

	// Staking
	nodeConfig.StakingConfig, err = getStakingConfig(v, nodeConfig.NetworkID)
	if err != nil {
		return node.Config{}, err
	}

	// Chain tracking
	nodeConfig.TrackAllChains = v.GetBool(TrackAllChainsKey)
	if !nodeConfig.TrackAllChains {
		nodeConfig.TrackedChains, err = getTrackedChains(v)
		if err != nil {
			return node.Config{}, err
		}
		// If TrackedChains is nil, it means track-chains="all" was specified
		// This should enable TrackAllChains mode
		if nodeConfig.TrackedChains == nil {
			nodeConfig.TrackAllChains = true
		}
	}

	// HTTP APIs
	nodeConfig.HTTPConfig, err = getHTTPConfig(v)
	if err != nil {
		return node.Config{}, err
	}

	// Health
	nodeConfig.HealthCheckFreq = v.GetDuration(HealthCheckFreqKey)
	if nodeConfig.HealthCheckFreq < 0 {
		return node.Config{}, fmt.Errorf("%s must be positive", HealthCheckFreqKey)
	}
	// Halflife of continuous averager used in health checks
	healthCheckAveragerHalflife := v.GetDuration(HealthCheckAveragerHalflifeKey)
	if healthCheckAveragerHalflife <= 0 {
		return node.Config{}, fmt.Errorf("%s must be positive", HealthCheckAveragerHalflifeKey)
	}

	// Router
	// 	routerHealthCfg, err := getRouterHealthConfig(v, healthCheckAveragerHalflife)
	if err != nil {
		return node.Config{}, err
	}
	// 	// Convert RouterHealthConfig to node.HealthConfig
	// 	nodeConfig.RouterHealthConfig = node.HealthConfig{
	// 		MaxTimeSinceMsgReceived: routerHealthCfg.MaxOutstandingDuration,
	// 		MaxTimeSinceMsgSent:     routerHealthCfg.MaxOutstandingDuration,
	// 		MaxPortionSendQueueFull: routerHealthCfg.MaxDropRate,
	// 		MinConnectedPeers:       1,
	// 		ReadTimeout:             routerHealthCfg.MaxRunTimeRequests,
	// 		WriteTimeout:            routerHealthCfg.MaxRunTimeRequests,
	// 		MaxSendFailRate:         routerHealthCfg.MaxDropRate,
	// 	}

	// Metrics
	nodeConfig.MeterVMEnabled = v.GetBool(MeterVMsEnabledKey)

	// Adaptive Timeout Config
	nodeConfig.AdaptiveTimeoutConfig, err = getAdaptiveTimeoutConfig(v)
	if err != nil {
		return node.Config{}, err
	}

	// Upgrade config
	nodeConfig.UpgradeConfig, err = getUpgradeConfig(v, nodeConfig.NetworkID)
	if err != nil {
		return node.Config{}, err
	}

	// Network Config
	nodeConfig.NetworkConfig, err = getNetworkConfig(
		v,
		nodeConfig.NetworkID,
		nodeConfig.SybilProtectionEnabled,
		healthCheckAveragerHalflife,
	)
	if err != nil {
		return node.Config{}, err
	}

	// Net Configs
	subnetConfigs, err := getNetConfigs(v, nodeConfig.TrackedChains.List())
	if err != nil {
		return node.Config{}, fmt.Errorf("couldn't read net configs: %w", err)
	}

	primaryNetworkConfig := getDefaultNetConfig(v)
	if err := primaryNetworkConfig.Valid(); err != nil {
		return node.Config{}, fmt.Errorf("invalid consensus parameters: %w", err)
	}
	subnetConfigs[constants.PrimaryNetworkID] = primaryNetworkConfig

	nodeConfig.NetConfigs = subnetConfigs

	// Benchlist
	// Convert consensus.Parameters to PrismParameters for benchlist config
	// 	prismParams := PrismParameters{
	// 		K:               primaryNetworkConfig.ConsensusParameters.K,
	// 		AlphaPreference: primaryNetworkConfig.ConsensusParameters.AlphaPreference,
	// 		AlphaConfidence: primaryNetworkConfig.ConsensusParameters.AlphaConfidence,
	// 	}
	// getBenchlistConfig is called for validation only
	// 	_, err = getBenchlistConfig(v, prismParams)
	// 	if err != nil {
	// 		return node.Config{}, err
	// 	}
	// benchlist.Config from consensus package only has Deprecated field
	nodeConfig.BenchlistConfig = benchlist.Config{
		Deprecated: false,
	}

	// File Descriptor Limit
	nodeConfig.FdLimit = v.GetUint64(FdLimitKey)

	// Tx Fee
	nodeConfig.TxFeeConfig = getTxFeeConfig(v, nodeConfig.NetworkID)

	// Genesis Data
	genesisStakingCfg := nodeConfig.StakingConfig.StakingConfig

	// Add BLS key information for genesis replay
	if nodeConfig.StakingConfig.StakingSigningKey != nil {
		// Get NodeID from the certificate
		nodeID := ids.NodeIDFromCert(&ids.Certificate{
			Raw:       nodeConfig.StakingConfig.StakingTLSCert.Leaf.Raw,
			PublicKey: nodeConfig.StakingConfig.StakingTLSCert.Leaf.PublicKey,
		})
		genesisStakingCfg.NodeID = nodeID.String()

		// Get BLS public key and proof of possession
		pk := nodeConfig.StakingConfig.StakingSigningKey.PublicKey()
		genesisStakingCfg.BLSPublicKey = bls.PublicKeyToCompressedBytes(pk)

		// Generate proof of possession
		sig, err := nodeConfig.StakingConfig.StakingSigningKey.SignProofOfPossession(genesisStakingCfg.BLSPublicKey)
		if err != nil {
			return node.Config{}, fmt.Errorf("failed to generate BLS proof of possession: %w", err)
		}
		if sig != nil {
			genesisStakingCfg.BLSProofOfPossession = bls.SignatureToBytes(sig)
		}
	}

	// Get data directory for dev network config persistence
	dataDir := getExpandedArg(v, DataDirKey)

	nodeConfig.GenesisBytes, nodeConfig.LuxAssetID, err = getGenesisData(v, nodeConfig.NetworkID, &genesisStakingCfg, dataDir)
	if err != nil {
		return node.Config{}, fmt.Errorf("unable to load genesis file: %w", err)
	}

	// StateSync Configs
	nodeConfig.StateSyncConfig, err = getStateSyncConfig(v)
	if err != nil {
		return node.Config{}, err
	}

	// Bootstrap Configs
	nodeConfig.BootstrapConfig, err = getBootstrapConfig(v, nodeConfig.NetworkID)
	if err != nil {
		return node.Config{}, err
	}

	// Chain Configs
	nodeConfig.ChainConfigs, err = getChainConfigs(v)
	if err != nil {
		return node.Config{}, fmt.Errorf("couldn't read chain configs: %w", err)
	}

	// Profiler
	nodeConfig.ProfilerConfig, err = getProfilerConfig(v)
	if err != nil {
		return node.Config{}, err
	}

	// VM Aliases
	nodeConfig.VMAliases, err = getVMAliases(v)
	if err != nil {
		return node.Config{}, err
	}
	// Chain aliases
	nodeConfig.ChainAliases, err = getChainAliases(v)
	if err != nil {
		return node.Config{}, err
	}

	nodeConfig.SystemTrackerFrequency = v.GetDuration(SystemTrackerFrequencyKey)
	nodeConfig.SystemTrackerProcessingHalflife = v.GetDuration(SystemTrackerProcessingHalflifeKey)
	nodeConfig.SystemTrackerCPUHalflife = v.GetDuration(SystemTrackerCPUHalflifeKey)
	nodeConfig.SystemTrackerDiskHalflife = v.GetDuration(SystemTrackerDiskHalflifeKey)

	nodeConfig.RequiredAvailableDiskSpace, nodeConfig.WarningThresholdAvailableDiskSpace, err = getDiskSpaceConfig(v)
	if err != nil {
		return node.Config{}, err
	}

	// 	cpuTargeterCfg, err := getCPUTargeterConfig(v)
	if err != nil {
		return node.Config{}, err
	}
	// Convert TrackerTargeterConfig to node.TargeterConfig
	// 	nodeConfig.CPUTargeterConfig = node.TargeterConfig{
	// 		VdrAlloc:           cpuTargeterCfg.VdrAlloc,
	// 		MaxNonVdrUsage:     cpuTargeterCfg.MaxNonVdrUsage,
	// 		MaxNonVdrNodeUsage: cpuTargeterCfg.MaxNonVdrNodeUsage,
	// 	}

	// 	diskTargeterCfg, err := getDiskTargeterConfig(v)
	if err != nil {
		return node.Config{}, err
	}
	// Convert TrackerTargeterConfig to node.TargeterConfig
	// 	nodeConfig.DiskTargeterConfig = node.TargeterConfig{
	// 		VdrAlloc:           diskTargeterCfg.VdrAlloc,
	// 		MaxNonVdrUsage:     diskTargeterCfg.MaxNonVdrUsage,
	// 		MaxNonVdrNodeUsage: diskTargeterCfg.MaxNonVdrNodeUsage,
	// 	}

	nodeConfig.TraceConfig, err = getTraceConfig(v)
	if err != nil {
		return node.Config{}, err
	}

	nodeConfig.ChainDataDir = getExpandedArg(v, ChainDataDirKey)

	nodeConfig.ProcessContextFilePath = getExpandedArg(v, ProcessContextFileKey)

	nodeConfig.ProvidedFlags = providedFlags(v)

	// Initialize logger if not already set
	// 	if nodeConfig.Log == nil {
	// 		nodeConfig.Log = log.New()
	// 	}

	return nodeConfig, nil
}

func providedFlags(v *viper.Viper) map[string]interface{} {
	settings := v.AllSettings()
	customSettings := make(map[string]interface{}, len(settings))
	for key, val := range settings {
		if v.IsSet(key) {
			customSettings[key] = val
		}
	}
	return customSettings
}
