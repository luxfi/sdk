// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package spec provides a machine-readable specification of all luxd configuration flags.
// This is the single source of truth for configuration - all consumers (CLI, SDK, netrunner)
// should derive their flag knowledge from this spec.
package spec

import (
	"encoding/json"
	"time"
)

// FlagType represents the type of a configuration flag.
type FlagType string

const (
	TypeBool           FlagType = "bool"
	TypeInt            FlagType = "int"
	TypeUint           FlagType = "uint"
	TypeUint64         FlagType = "uint64"
	TypeFloat64        FlagType = "float64"
	TypeDuration       FlagType = "duration"
	TypeString         FlagType = "string"
	TypeStringSlice    FlagType = "string-slice"
	TypeIntSlice       FlagType = "int-slice"
	TypeStringToString FlagType = "string-to-string"
)

// Category groups related configuration flags.
type Category string

const (
	CategoryProcess    Category = "process"
	CategoryNode       Category = "node"
	CategoryDatabase   Category = "database"
	CategoryNetwork    Category = "network"
	CategoryConsensus  Category = "consensus"
	CategoryStaking    Category = "staking"
	CategoryHTTP       Category = "http"
	CategoryAPI        Category = "api"
	CategoryHealth     Category = "health"
	CategoryLogging    Category = "logging"
	CategoryThrottler  Category = "throttler"
	CategorySystem     Category = "system"
	CategoryBootstrap  Category = "bootstrap"
	CategoryChain      Category = "chain"
	CategoryProfile    Category = "profile"
	CategoryMetrics    Category = "metrics"
	CategoryGenesis    Category = "genesis"
	CategoryFees       Category = "fees"
	CategoryIndex      Category = "index"
	CategoryTracing    Category = "tracing"
	CategoryPOA        Category = "poa"
	CategoryDev        Category = "dev"
)

// FlagSpec describes a single configuration flag.
type FlagSpec struct {
	// Key is the flag name (e.g., "network-id", "log-level")
	Key string `json:"key"`

	// Type is the flag's data type
	Type FlagType `json:"type"`

	// Default is the default value (nil if no default)
	Default interface{} `json:"default,omitempty"`

	// Description is the human-readable documentation
	Description string `json:"description"`

	// Category groups related flags for organization
	Category Category `json:"category"`

	// Deprecated indicates if this flag is deprecated
	Deprecated bool `json:"deprecated,omitempty"`

	// DeprecatedMessage explains what to use instead
	DeprecatedMessage string `json:"deprecated_message,omitempty"`

	// ReplacedBy is the key of the replacement flag (if deprecated)
	ReplacedBy string `json:"replaced_by,omitempty"`

	// Required indicates if this flag must be explicitly set
	Required bool `json:"required,omitempty"`

	// Sensitive indicates if the value should be masked in logs
	Sensitive bool `json:"sensitive,omitempty"`

	// Constraints contains validation rules
	Constraints *Constraints `json:"constraints,omitempty"`

	// Since indicates when this flag was introduced (version string)
	Since string `json:"since,omitempty"`
}

// Constraints defines validation rules for a flag.
type Constraints struct {
	// Min is the minimum value (for numeric types)
	Min interface{} `json:"min,omitempty"`

	// Max is the maximum value (for numeric types)
	Max interface{} `json:"max,omitempty"`

	// Enum lists allowed values (for string types)
	Enum []string `json:"enum,omitempty"`

	// Pattern is a regex pattern (for string types)
	Pattern string `json:"pattern,omitempty"`

	// RequiredWith lists flags that must also be set
	RequiredWith []string `json:"required_with,omitempty"`

	// ConflictsWith lists flags that cannot be set together
	ConflictsWith []string `json:"conflicts_with,omitempty"`
}

// ConfigSpec is the complete specification of all luxd configuration flags.
type ConfigSpec struct {
	// Version is the spec version (semver)
	Version string `json:"version"`

	// NodeVersion is the luxd version this spec was generated from
	NodeVersion string `json:"node_version"`

	// GeneratedAt is when this spec was generated (RFC3339)
	GeneratedAt string `json:"generated_at"`

	// Flags contains all flag specifications
	Flags []FlagSpec `json:"flags"`

	// Categories lists all categories with descriptions
	Categories map[Category]string `json:"categories"`
}

// Spec returns the complete configuration specification.
// This is the single source of truth for all luxd flags.
func Spec() *ConfigSpec {
	return &ConfigSpec{
		Version:     "1.0.0",
		NodeVersion: "1.22.18",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Flags:       allFlags(),
		Categories:  categoryDescriptions(),
	}
}

// categoryDescriptions returns descriptions for all categories.
func categoryDescriptions() map[Category]string {
	return map[Category]string{
		CategoryProcess:   "Process-level flags (version, help)",
		CategoryNode:      "Node configuration (data directory, plugins)",
		CategoryDatabase:  "Database configuration (type, path, per-chain)",
		CategoryNetwork:   "P2P networking (timeouts, throttling, peers)",
		CategoryConsensus: "Consensus parameters (sample size, quorum, thresholds)",
		CategoryStaking:   "Staking configuration (keys, sybil protection)",
		CategoryHTTP:      "HTTP server configuration (host, port, TLS)",
		CategoryAPI:       "API enablement (admin, info, keystore, metrics)",
		CategoryHealth:    "Health check configuration (frequency, thresholds)",
		CategoryLogging:   "Logging configuration (level, format, rotation)",
		CategoryThrottler: "Rate limiting and resource throttling",
		CategorySystem:    "System resource tracking (CPU, disk, memory)",
		CategoryBootstrap: "Bootstrap and state sync configuration",
		CategoryChain:     "Chain-specific configuration directories",
		CategoryProfile:   "Performance profiling configuration",
		CategoryMetrics:   "Metrics collection and reporting",
		CategoryGenesis:   "Genesis file and upgrade configuration",
		CategoryFees:      "Transaction fee configuration",
		CategoryIndex:     "Indexer configuration",
		CategoryTracing:   "OpenTelemetry tracing configuration",
		CategoryPOA:       "Proof of Authority mode configuration",
		CategoryDev:       "Development and testing flags",
	}
}

// GetFlag returns the spec for a specific flag, or nil if not found.
func (s *ConfigSpec) GetFlag(key string) *FlagSpec {
	for i := range s.Flags {
		if s.Flags[i].Key == key {
			return &s.Flags[i]
		}
	}
	return nil
}

// FlagsByCategory returns all flags in a specific category.
func (s *ConfigSpec) FlagsByCategory(cat Category) []FlagSpec {
	var result []FlagSpec
	for _, f := range s.Flags {
		if f.Category == cat {
			result = append(result, f)
		}
	}
	return result
}

// DeprecatedFlags returns all deprecated flags.
func (s *ConfigSpec) DeprecatedFlags() []FlagSpec {
	var result []FlagSpec
	for _, f := range s.Flags {
		if f.Deprecated {
			result = append(result, f)
		}
	}
	return result
}

// JSON returns the spec as formatted JSON.
func (s *ConfigSpec) JSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// ValidateValue checks if a value is valid for a flag.
func (f *FlagSpec) ValidateValue(value interface{}) error {
	// Type checking and constraint validation would go here
	// For now, return nil (valid)
	return nil
}
