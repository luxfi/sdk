// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package configspec

import (
	"testing"
)

func TestSpec(t *testing.T) {
	s, err := Spec()
	if err != nil {
		t.Fatalf("Spec() failed: %v", err)
	}
	if s == nil {
		t.Fatal("Spec() returned nil")
	}
	if s.Version == "" {
		t.Error("spec version is empty")
	}
	if len(s.Flags) == 0 {
		t.Error("spec has no flags")
	}
	t.Logf("Loaded spec version %s with %d flags", s.Version, len(s.Flags))
}

func TestMustSpec(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustSpec() panicked: %v", r)
		}
	}()
	s := MustSpec()
	if s == nil {
		t.Fatal("MustSpec() returned nil")
	}
}

func TestGetFlag(t *testing.T) {
	s := MustSpec()

	f := s.GetFlag("network-id")
	if f == nil {
		t.Fatal("GetFlag returned nil for network-id")
	}
	if f.Key != "network-id" {
		t.Errorf("got key %q, want network-id", f.Key)
	}

	f = s.GetFlag("nonexistent-flag-xyz")
	if f != nil {
		t.Error("GetFlag should return nil for non-existent flag")
	}
}

func TestFlagsByCategory(t *testing.T) {
	s := MustSpec()

	flags := s.FlagsByCategory(CategoryConsensus)
	if len(flags) == 0 {
		t.Error("CategoryConsensus has no flags")
	}

	for _, f := range flags {
		if f.Category != CategoryConsensus {
			t.Errorf("flag %q has category %q, expected consensus", f.Key, f.Category)
		}
	}
}

func TestAllKeys(t *testing.T) {
	keys := AllKeys()
	if len(keys) < 100 {
		t.Errorf("expected at least 100 keys, got %d", len(keys))
	}
}

func TestKnownKey(t *testing.T) {
	if !KnownKey("network-id") {
		t.Error("network-id should be known")
	}
	if KnownKey("unknown-flag-xyz") {
		t.Error("unknown-flag-xyz should not be known")
	}
}

func TestVersion(t *testing.T) {
	v := Version()
	if v == "" {
		t.Error("Version() returned empty string")
	}
}

func TestNodeVersion(t *testing.T) {
	nv := NodeVersion()
	if nv == "" {
		t.Error("NodeVersion() returned empty string")
	}
}
