// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package profiles provides embedded network tuning profiles for the Lux SDK.
// These profiles configure consensus, health, and network parameters for different
// deployment scenarios (mainnet, testnet, devnet, dev mode).
//
// Available profiles:
//   - standard: Conservative mainnet settings for production-safe operation
//   - fast:     Balanced testnet settings for development
//   - turbo:    Aggressive 3-node local network with 2/3 quorum
//   - ultra:    Maximum aggression for single-node dev mode with K=1 consensus
package profiles

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed *.json
var profileFS embed.FS

// Profile represents a network tuning profile.
type Profile struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Consensus   ConsensusConfig `json:"consensus"`
	Health      HealthConfig    `json:"health"`
	Network     NetworkConfig   `json:"network"`
}

// ConsensusConfig holds consensus tuning parameters.
type ConsensusConfig struct {
	SampleSize            int    `json:"sample-size"`
	PreferenceQuorumSize  int    `json:"preference-quorum-size"`
	ConfidenceQuorumSize  int    `json:"confidence-quorum-size"`
	CommitThreshold       int    `json:"commit-threshold"`
	ConcurrentRepolls     int    `json:"concurrent-repolls"`
	OptimalProcessing     int    `json:"optimal-processing"`
	MaxProcessing         int    `json:"max-processing"`
	FrontierPollFreq      string `json:"frontier-poll-frequency"`
	ProposerMinBlockDelay string `json:"proposer-min-block-delay,omitempty"`
}

// HealthConfig holds health check tuning parameters.
type HealthConfig struct {
	CheckFrequency   string `json:"check-frequency"`
	AveragerHalflife string `json:"averager-halflife"`
}

// NetworkConfig holds network tuning parameters.
type NetworkConfig struct {
	MaxReconnectDelay     string `json:"max-reconnect-delay"`
	InitialReconnectDelay string `json:"initial-reconnect-delay"`
	InitialTimeout        string `json:"initial-timeout"`
	MinimumTimeout        string `json:"minimum-timeout"`
	MaximumTimeout        string `json:"maximum-timeout"`
	TimeoutHalflife       string `json:"timeout-halflife"`
	ReadHandshakeTimeout  string `json:"read-handshake-timeout"`
	PingTimeout           string `json:"ping-timeout"`
	PingFrequency         string `json:"ping-frequency"`
}

// Consensus derives the sampling-consensus parameters (the lux/consensus engine's
// alpha-of-K snowball/snowman knobs: K=sample, alpha=quorum, beta=commit) for a
// validator set of size n. Consensus is Byzantine-fault-tolerant by construction —
// there is no non-BFT variant — so the result always satisfies the engine invariant
// 2*alpha - K >= floor((K-1)/3)+1; "BFT" is implied, not a flavor.
//
// This is the ENGINE/agreement layer, NOT post-quantum finality. Quasar (lux/quasar,
// the per-round QuasarCert + pqLayers in the CR's spec.consensus) seals each agreed
// event in a PQ weighted certificate ON TOP of this — a separate concern; this struct
// carries no PQ config.
//
// It is the single source of truth for sampling safety: a deploy derives K/alpha from
// the LIVE validator count, never hardcodes them (the live K=5/alpha=3 drift was
// sub-BFT — 2*3-5 = 1 < floor((5-1)/3)+1 = 2 — which the engine correctly refuses).
//
//   - K (sample size)  = min(n, 20)        — Avalanche caps the poll sample at 20.
//   - alpha (quorum)   = ceil(0.75*K)      — Avalanche's 15/20 = 75% ratio (K>=4).
//                                            For K<=3, 75% would force unanimity, so
//                                            the Byzantine floor is used to preserve
//                                            liveness (one tolerable fault).
//
// Examples: n=5 -> K=5,alpha=4 (80%); n=20+ -> K=20,alpha=15 (exactly Avalanche);
// n=3 -> K=3,alpha=2 (67%, BFT-minimal).
func Consensus(n int) ConsensusConfig {
	k := n
	if k < 1 {
		k = 1
	}
	if k > 20 {
		k = 20 // Avalanche samples at most 20 validators per poll
	}
	var alpha int
	if k >= 4 {
		alpha = (3*k + 3) / 4 // ceil(0.75*k): Avalanche's 75% ratio, BFT-safe + 1+ fault-tolerant
	} else {
		alpha = bftAlphaFloor(k) // tiny sets: BFT minimum keeps the chain live
	}
	return ConsensusConfig{
		SampleSize:           k,
		PreferenceQuorumSize: alpha,
		ConfidenceQuorumSize: alpha,
		CommitThreshold:      20, // Avalanche mainnet finalization depth (BetaRogue)
		ConcurrentRepolls:    4,  // Avalanche default
		OptimalProcessing:    10, // Avalanche default
		MaxProcessing:        256,
		FrontierPollFreq:     "100ms",
	}
}

// bftAlphaFloor returns the smallest alpha satisfying the engine's Byzantine-safety
// invariant 2*alpha - K >= floor((K-1)/3)+1 (i.e. alpha >= ceil((K + floor((K-1)/3)+1) / 2)).
func bftAlphaFloor(k int) int {
	need := (k-1)/3 + 1
	return (k + need + 1) / 2
}

// IsByzantineSafe reports whether a sample/quorum pair satisfies the consensus
// engine's Byzantine-safety invariant. Use it to validate any consensus config.
func IsByzantineSafe(sampleSize, quorumSize int) bool {
	if sampleSize < 1 || quorumSize < 1 || quorumSize > sampleSize {
		return false
	}
	return 2*quorumSize-sampleSize >= (sampleSize-1)/3+1
}

// GetProfile loads a profile by name.
// Returns error if profile not found or invalid.
func GetProfile(name string) (*Profile, error) {
	name = strings.ToLower(name)
	filename := name + ".json"

	data, err := profileFS.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("profile %q not found: %w", name, err)
	}

	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("invalid profile %q: %w", name, err)
	}

	return &profile, nil
}

// MustGetProfile loads a profile by name or panics on error.
func MustGetProfile(name string) *Profile {
	p, err := GetProfile(name)
	if err != nil {
		panic(err)
	}
	return p
}

// GetProfileMap loads a profile and returns it as map[string]interface{}.
// Useful for direct JSON manipulation.
func GetProfileMap(name string) (map[string]interface{}, error) {
	name = strings.ToLower(name)
	filename := name + ".json"

	data, err := profileFS.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("profile %q not found: %w", name, err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid profile %q: %w", name, err)
	}

	return m, nil
}

// ListProfiles returns names of all available profiles, sorted alphabetically.
func ListProfiles() []string {
	entries, err := profileFS.ReadDir(".")
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".json") {
			names = append(names, strings.TrimSuffix(name, ".json"))
		}
	}
	sort.Strings(names)
	return names
}

// ValidateProfile checks that a profile has required fields.
func ValidateProfile(data map[string]interface{}) error {
	required := []string{"name", "description", "consensus", "health", "network"}
	for _, field := range required {
		if _, ok := data[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate consensus section
	consensus, ok := data["consensus"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("consensus must be an object")
	}
	consensusRequired := []string{
		"sample-size", "preference-quorum-size", "confidence-quorum-size",
		"commit-threshold", "concurrent-repolls", "optimal-processing",
		"max-processing", "frontier-poll-frequency",
	}
	for _, field := range consensusRequired {
		if _, ok := consensus[field]; !ok {
			return fmt.Errorf("consensus missing required field: %s", field)
		}
	}

	// Validate health section
	health, ok := data["health"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("health must be an object")
	}
	healthRequired := []string{"check-frequency", "averager-halflife"}
	for _, field := range healthRequired {
		if _, ok := health[field]; !ok {
			return fmt.Errorf("health missing required field: %s", field)
		}
	}

	// Validate network section
	network, ok := data["network"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("network must be an object")
	}
	networkRequired := []string{
		"max-reconnect-delay", "initial-reconnect-delay", "initial-timeout",
		"minimum-timeout", "maximum-timeout", "timeout-halflife",
		"read-handshake-timeout", "ping-timeout", "ping-frequency",
	}
	for _, field := range networkRequired {
		if _, ok := network[field]; !ok {
			return fmt.Errorf("network missing required field: %s", field)
		}
	}

	return nil
}

// DefaultProfileForNetwork returns the default profile name for a network type.
// For local development, all networks use turbo for ultra-fast consensus.
func DefaultProfileForNetwork(networkName string) string {
	switch networkName {
	case "mainnet":
		return "turbo" // Ultra-fast for local 3-node mainnet
	case "testnet":
		return "turbo" // Ultra-fast for local 3-node testnet
	case "devnet", "local":
		return "turbo"
	case "dev":
		return "ultra"
	default:
		return "turbo" // Default to turbo for local development
	}
}

// ToNodeConfig converts a profile to luxd node configuration flags.
// Returns a map of flag keys to values suitable for JSON config.
func (p *Profile) ToNodeConfig() map[string]interface{} {
	cfg := map[string]interface{}{
		// Consensus
		"snow-sample-size":                  p.Consensus.SampleSize,
		"snow-quorum-size":                  p.Consensus.PreferenceQuorumSize,
		"snow-preference-quorum-size":       p.Consensus.PreferenceQuorumSize,
		"snow-confidence-quorum-size":       p.Consensus.ConfidenceQuorumSize,
		"snow-commit-threshold":             p.Consensus.CommitThreshold,
		"snow-concurrent-repolls":           p.Consensus.ConcurrentRepolls,
		"snow-optimal-processing":           p.Consensus.OptimalProcessing,
		"snow-max-processing":               p.Consensus.MaxProcessing,
		"consensus-frontier-poll-frequency": p.Consensus.FrontierPollFreq,
		// Health
		"health-check-frequency":         p.Health.CheckFrequency,
		"health-check-averager-halflife": p.Health.AveragerHalflife,
		// Network
		"network-max-reconnect-delay":     p.Network.MaxReconnectDelay,
		"network-initial-reconnect-delay": p.Network.InitialReconnectDelay,
		"network-initial-timeout":         p.Network.InitialTimeout,
		"network-minimum-timeout":         p.Network.MinimumTimeout,
		"network-maximum-timeout":         p.Network.MaximumTimeout,
		"network-timeout-halflife":        p.Network.TimeoutHalflife,
		"network-read-handshake-timeout":  p.Network.ReadHandshakeTimeout,
		"network-ping-timeout":            p.Network.PingTimeout,
		"network-ping-frequency":          p.Network.PingFrequency,
	}

	// Add proposer delay if specified
	if p.Consensus.ProposerMinBlockDelay != "" {
		cfg["proposervm-min-block-delay"] = p.Consensus.ProposerMinBlockDelay
	}

	return cfg
}
