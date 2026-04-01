// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package profiles

import (
	"testing"
)

func TestListProfiles(t *testing.T) {
	names := ListProfiles()
	if len(names) != 4 {
		t.Errorf("expected 4 profiles, got %d: %v", len(names), names)
	}

	expected := []string{"fast", "standard", "turbo", "ultra"}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("profile %d: expected %q, got %q", i, name, names[i])
		}
	}
}

func TestGetProfile(t *testing.T) {
	tests := []struct {
		name        string
		sampleSize  int
		description string
	}{
		{"standard", 3, "Conservative mainnet settings for production-safe operation"},
		{"fast", 3, "Balanced testnet settings for development"},
		{"turbo", 3, "Ultra-fast 3-node local network with K=3, Alpha=2/3, Beta=2"},
		{"ultra", 1, "Maximum aggression for single-node dev mode with K=1 consensus"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := GetProfile(tt.name)
			if err != nil {
				t.Fatalf("GetProfile(%q): %v", tt.name, err)
			}
			if profile.Name != tt.name {
				t.Errorf("name: expected %q, got %q", tt.name, profile.Name)
			}
			if profile.Description != tt.description {
				t.Errorf("description: expected %q, got %q", tt.description, profile.Description)
			}
			if profile.Consensus.SampleSize != tt.sampleSize {
				t.Errorf("sample-size: expected %d, got %d", tt.sampleSize, profile.Consensus.SampleSize)
			}
		})
	}
}

func TestMustGetProfile(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustGetProfile panicked: %v", r)
		}
	}()
	p := MustGetProfile("turbo")
	if p == nil {
		t.Error("MustGetProfile returned nil")
	}
}

func TestGetProfileNotFound(t *testing.T) {
	_, err := GetProfile("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent profile")
	}
}

func TestGetProfileMap(t *testing.T) {
	m, err := GetProfileMap("turbo")
	if err != nil {
		t.Fatalf("GetProfileMap: %v", err)
	}
	if m["name"] != "turbo" {
		t.Errorf("expected name=turbo, got %v", m["name"])
	}
}

func TestValidateProfile(t *testing.T) {
	for _, name := range ListProfiles() {
		m, err := GetProfileMap(name)
		if err != nil {
			t.Fatalf("GetProfileMap(%q): %v", name, err)
		}
		if err := ValidateProfile(m); err != nil {
			t.Errorf("ValidateProfile(%q): %v", name, err)
		}
	}
}

func TestValidateProfileMissingFields(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{"missing name", map[string]interface{}{"description": "x"}},
		{"missing consensus", map[string]interface{}{"name": "x", "description": "x", "health": map[string]interface{}{}, "network": map[string]interface{}{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateProfile(tt.data); err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestDefaultProfileForNetwork(t *testing.T) {
	tests := []struct {
		network  string
		expected string
	}{
		{"mainnet", "turbo"},
		{"testnet", "turbo"},
		{"devnet", "turbo"},
		{"local", "turbo"},
		{"dev", "ultra"},
		{"unknown", "turbo"},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			got := DefaultProfileForNetwork(tt.network)
			if got != tt.expected {
				t.Errorf("DefaultProfileForNetwork(%q): expected %q, got %q", tt.network, tt.expected, got)
			}
		})
	}
}

func TestToNodeConfig(t *testing.T) {
	p := MustGetProfile("turbo")
	config := p.ToNodeConfig()

	// Check a few key values
	if config["snow-sample-size"] != 3 {
		t.Errorf("expected snow-sample-size=3, got %v", config["snow-sample-size"])
	}
	if config["snow-quorum-size"] != 2 {
		t.Errorf("expected snow-quorum-size=2, got %v", config["snow-quorum-size"])
	}
	if config["consensus-frontier-poll-frequency"] != "10ms" {
		t.Errorf("expected consensus-frontier-poll-frequency=10ms, got %v", config["consensus-frontier-poll-frequency"])
	}
	if config["network-ping-timeout"] != "1s" {
		t.Errorf("expected network-ping-timeout=1s, got %v", config["network-ping-timeout"])
	}
}
