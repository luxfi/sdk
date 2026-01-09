// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package configspec provides the embedded luxd configuration specification.
//
// Deprecated: Use github.com/luxfi/config/spec instead. This package will be
// removed in a future release. The canonical location for configspec is now
// github.com/luxfi/config/spec.
//
// This is a snapshot of the node's configuration spec for use by SDK consumers
// without requiring a dependency on the node package.
//
// The spec.json file is generated from the node using:
//
//	go run github.com/luxfi/node/cmd/config dump-spec --format=json > spec.json
//
// To regenerate, run:
//
//	go generate ./...
package configspec

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:generate sh -c "cd ../../node && go run ./cmd/config dump-spec --format=json > ../sdk/configspec/spec.json"

//go:embed spec.json
var specJSON []byte

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
	CategoryProcess   Category = "process"
	CategoryNode      Category = "node"
	CategoryDatabase  Category = "database"
	CategoryNetwork   Category = "network"
	CategoryConsensus Category = "consensus"
	CategoryStaking   Category = "staking"
	CategoryHTTP      Category = "http"
	CategoryAPI       Category = "api"
	CategoryHealth    Category = "health"
	CategoryLogging   Category = "logging"
	CategoryThrottler Category = "throttler"
	CategorySystem    Category = "system"
	CategoryBootstrap Category = "bootstrap"
	CategoryChain     Category = "chain"
	CategoryProfile   Category = "profile"
	CategoryMetrics   Category = "metrics"
	CategoryGenesis   Category = "genesis"
	CategoryFees      Category = "fees"
	CategoryIndex     Category = "index"
	CategoryTracing   Category = "tracing"
	CategoryPOA       Category = "poa"
	CategoryDev       Category = "dev"
)

// Constraints defines validation rules for a flag.
type Constraints struct {
	Min           interface{} `json:"min,omitempty"`
	Max           interface{} `json:"max,omitempty"`
	Enum          []string    `json:"enum,omitempty"`
	Pattern       string      `json:"pattern,omitempty"`
	RequiredWith  []string    `json:"required_with,omitempty"`
	ConflictsWith []string    `json:"conflicts_with,omitempty"`
}

// FlagSpec describes a single configuration flag.
type FlagSpec struct {
	Key               string       `json:"key"`
	Type              FlagType     `json:"type"`
	Default           interface{}  `json:"default,omitempty"`
	Description       string       `json:"description"`
	Category          Category     `json:"category"`
	Deprecated        bool         `json:"deprecated,omitempty"`
	DeprecatedMessage string       `json:"deprecated_message,omitempty"`
	ReplacedBy        string       `json:"replaced_by,omitempty"`
	Required          bool         `json:"required,omitempty"`
	Sensitive         bool         `json:"sensitive,omitempty"`
	Constraints       *Constraints `json:"constraints,omitempty"`
	Since             string       `json:"since,omitempty"`
}

// ConfigSpec is the complete specification of all luxd configuration flags.
type ConfigSpec struct {
	Version     string              `json:"version"`
	NodeVersion string              `json:"node_version"`
	GeneratedAt string              `json:"generated_at"`
	Flags       []FlagSpec          `json:"flags"`
	Categories  map[Category]string `json:"categories"`
}

var (
	cachedSpec *ConfigSpec
	specOnce   sync.Once
	specErr    error
)

// Spec returns the embedded configuration specification.
// The spec is parsed once and cached for subsequent calls.
func Spec() (*ConfigSpec, error) {
	specOnce.Do(func() {
		cachedSpec = &ConfigSpec{}
		specErr = json.Unmarshal(specJSON, cachedSpec)
	})
	if specErr != nil {
		return nil, specErr
	}
	return cachedSpec, nil
}

// MustSpec returns the embedded configuration specification or panics on error.
func MustSpec() *ConfigSpec {
	s, err := Spec()
	if err != nil {
		panic(err)
	}
	return s
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

// AllKeys returns all flag keys.
func (s *ConfigSpec) AllKeys() []string {
	keys := make([]string, len(s.Flags))
	for i, f := range s.Flags {
		keys[i] = f.Key
	}
	return keys
}

// KnownKey returns true if the key is a known configuration flag.
func (s *ConfigSpec) KnownKey(key string) bool {
	return s.GetFlag(key) != nil
}

// Version returns the spec version.
func Version() string {
	return MustSpec().Version
}

// NodeVersion returns the node version this spec was generated from.
func NodeVersion() string {
	return MustSpec().NodeVersion
}

// AllKeys returns all known configuration keys.
func AllKeys() []string {
	return MustSpec().AllKeys()
}

// KnownKey checks if a key is a valid configuration flag.
func KnownKey(key string) bool {
	return MustSpec().KnownKey(key)
}
