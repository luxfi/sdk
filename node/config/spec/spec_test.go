// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package spec

import (
	"encoding/json"
	"testing"
)

func TestSpec(t *testing.T) {
	s := Spec()
	if s == nil {
		t.Fatal("Spec() returned nil")
	}
	if s.Version == "" {
		t.Error("spec version is empty")
	}
	if len(s.Flags) == 0 {
		t.Error("spec has no flags")
	}
	if len(s.Categories) == 0 {
		t.Error("spec has no categories")
	}
}

func TestSpecHasExpectedFlags(t *testing.T) {
	s := Spec()

	// Check some key flags exist
	expectedFlags := []string{
		"network-id",
		"log-level",
		"http-port",
		"db-type",
		"consensus-sample-size",
		"staking-port",
		"api-admin-enabled",
	}

	for _, key := range expectedFlags {
		if f := s.GetFlag(key); f == nil {
			t.Errorf("expected flag %q not found", key)
		}
	}
}

func TestGetFlag(t *testing.T) {
	s := Spec()

	// Test existing flag
	f := s.GetFlag("network-id")
	if f == nil {
		t.Fatal("GetFlag returned nil for network-id")
	}
	if f.Key != "network-id" {
		t.Errorf("got key %q, want network-id", f.Key)
	}
	if f.Type != TypeString {
		t.Errorf("got type %q, want %q", f.Type, TypeString)
	}
	if f.Category != CategoryNetwork {
		t.Errorf("got category %q, want %q", f.Category, CategoryNetwork)
	}

	// Test non-existent flag
	f = s.GetFlag("nonexistent-flag-xyz")
	if f != nil {
		t.Error("GetFlag should return nil for non-existent flag")
	}
}

func TestFlagsByCategory(t *testing.T) {
	s := Spec()

	categories := []Category{
		CategoryConsensus,
		CategoryNetwork,
		CategoryHTTP,
		CategoryLogging,
		CategoryStaking,
	}

	for _, cat := range categories {
		flags := s.FlagsByCategory(cat)
		if len(flags) == 0 {
			t.Errorf("category %q has no flags", cat)
		}
		for _, f := range flags {
			if f.Category != cat {
				t.Errorf("flag %q has category %q, expected %q", f.Key, f.Category, cat)
			}
		}
	}
}

func TestCategoryDescriptions(t *testing.T) {
	s := Spec()

	// All categories used in flags should have descriptions
	usedCategories := make(map[Category]bool)
	for _, f := range s.Flags {
		usedCategories[f.Category] = true
	}

	for cat := range usedCategories {
		if desc, ok := s.Categories[cat]; !ok {
			t.Errorf("category %q has no description", cat)
		} else if desc == "" {
			t.Errorf("category %q has empty description", cat)
		}
	}
}

func TestSpecJSON(t *testing.T) {
	s := Spec()

	data, err := s.JSON()
	if err != nil {
		t.Fatalf("JSON() failed: %v", err)
	}

	// Verify it's valid JSON by unmarshaling
	var parsed ConfigSpec
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if parsed.Version != s.Version {
		t.Errorf("version mismatch: got %q, want %q", parsed.Version, s.Version)
	}
	if len(parsed.Flags) != len(s.Flags) {
		t.Errorf("flags count mismatch: got %d, want %d", len(parsed.Flags), len(s.Flags))
	}
}

func TestFlagTypes(t *testing.T) {
	s := Spec()

	validTypes := map[FlagType]bool{
		TypeBool:           true,
		TypeInt:            true,
		TypeUint:           true,
		TypeUint64:         true,
		TypeFloat64:        true,
		TypeDuration:       true,
		TypeString:         true,
		TypeStringSlice:    true,
		TypeIntSlice:       true,
		TypeStringToString: true,
	}

	for _, f := range s.Flags {
		if !validTypes[f.Type] {
			t.Errorf("flag %q has invalid type %q", f.Key, f.Type)
		}
	}
}

func TestFlagCount(t *testing.T) {
	s := Spec()

	// We expect a substantial number of flags
	minExpected := 100
	if len(s.Flags) < minExpected {
		t.Errorf("expected at least %d flags, got %d", minExpected, len(s.Flags))
	}

	t.Logf("Total flags: %d", len(s.Flags))

	// Log counts by category
	counts := make(map[Category]int)
	for _, f := range s.Flags {
		counts[f.Category]++
	}
	for cat, count := range counts {
		t.Logf("  %s: %d flags", cat, count)
	}
}

func TestNoEmptyDescriptions(t *testing.T) {
	s := Spec()

	for _, f := range s.Flags {
		if f.Description == "" {
			t.Errorf("flag %q has empty description", f.Key)
		}
	}
}

func TestNoDuplicateKeys(t *testing.T) {
	s := Spec()

	seen := make(map[string]bool)
	for _, f := range s.Flags {
		if seen[f.Key] {
			t.Errorf("duplicate flag key: %q", f.Key)
		}
		seen[f.Key] = true
	}
}
