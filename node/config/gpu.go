// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"fmt"
	"runtime"
	"sync"
)

// GPUConfig holds GPU acceleration configuration.
type GPUConfig struct {
	// Enabled controls whether GPU acceleration is used
	Enabled bool

	// Backend specifies which GPU backend to use: "auto", "metal", "cuda", "cpu"
	Backend string

	// DeviceIndex specifies which GPU device to use when multiple are available
	DeviceIndex int

	// LogLevel sets the GPU subsystem log level: "debug", "info", "warn", "error"
	LogLevel string
}

// DefaultGPUConfig returns the default GPU configuration.
func DefaultGPUConfig() GPUConfig {
	return GPUConfig{
		Enabled:     true,
		Backend:     "auto",
		DeviceIndex: 0,
		LogLevel:    "warn",
	}
}

// Validate checks that the GPU configuration is valid.
func (c GPUConfig) Validate() error {
	switch c.Backend {
	case "auto", "metal", "cuda", "cpu":
		// Valid backends
	default:
		return fmt.Errorf("invalid GPU backend %q: must be auto, metal, cuda, or cpu", c.Backend)
	}

	// Validate backend is supported on current platform
	if c.Backend == "metal" && runtime.GOOS != "darwin" {
		return fmt.Errorf("metal backend is only supported on macOS")
	}
	if c.Backend == "cuda" && runtime.GOOS == "darwin" {
		return fmt.Errorf("cuda backend is not supported on macOS")
	}

	if c.DeviceIndex < 0 {
		return fmt.Errorf("GPU device index must be non-negative")
	}

	switch c.LogLevel {
	case "debug", "info", "warn", "error":
		// Valid log levels
	default:
		return fmt.Errorf("invalid GPU log level %q: must be debug, info, warn, or error", c.LogLevel)
	}

	return nil
}

// ResolveBackend returns the actual backend to use based on configuration.
// If Backend is "auto", it detects the best available backend.
func (c GPUConfig) ResolveBackend() string {
	if !c.Enabled {
		return "cpu"
	}

	if c.Backend != "auto" {
		return c.Backend
	}

	// Auto-detect based on platform
	switch runtime.GOOS {
	case "darwin":
		return "metal"
	case "linux":
		return "cuda"
	default:
		return "cpu"
	}
}

// Global GPU configuration (set during node initialization)
var (
	globalGPUConfig     GPUConfig
	globalGPUConfigOnce sync.Once
	globalGPUConfigSet  bool
)

// SetGlobalGPUConfig sets the global GPU configuration.
// This should be called once during node initialization before any GPU accelerators are created.
func SetGlobalGPUConfig(cfg GPUConfig) error {
	var setErr error
	globalGPUConfigOnce.Do(func() {
		if err := cfg.Validate(); err != nil {
			setErr = err
			return
		}
		globalGPUConfig = cfg
		globalGPUConfigSet = true
	})
	return setErr
}

// GetGlobalGPUConfig returns the global GPU configuration.
// If not set, returns the default configuration.
func GetGlobalGPUConfig() GPUConfig {
	if !globalGPUConfigSet {
		return DefaultGPUConfig()
	}
	return globalGPUConfig
}

// IsGPUEnabled returns whether GPU acceleration is enabled globally.
func IsGPUEnabled() bool {
	return GetGlobalGPUConfig().Enabled
}

// GPUBackend returns the configured GPU backend.
func GPUBackend() string {
	return GetGlobalGPUConfig().ResolveBackend()
}
