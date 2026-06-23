// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package profiles

import "testing"

// TestConsensus_MatchesAvalanche pins the Avalanche-grade derivation: K=min(20,N),
// alpha=ceil(0.75K) at the 15/20 ratio, BFT-safe at every size.
func TestConsensus_MatchesAvalanche(t *testing.T) {
	cases := []struct {
		n, wantK, wantAlpha int
	}{
		{1, 1, 1},   // single node
		{3, 3, 2},   // tiny: BFT-minimal (67%), keeps one tolerable fault
		{4, 4, 3},   // 75%
		{5, 5, 4},   // current Lux nets: 80% >= Avalanche's 75%
		{8, 8, 6},   // 75%
		{20, 20, 15}, // exactly Avalanche
		{50, 20, 15}, // capped at K=20, Avalanche
	}
	for _, c := range cases {
		got := Consensus(c.n)
		if got.SampleSize != c.wantK || got.PreferenceQuorumSize != c.wantAlpha || got.ConfidenceQuorumSize != c.wantAlpha {
			t.Errorf("Consensus(%d) = K=%d alpha=%d, want K=%d alpha=%d",
				c.n, got.SampleSize, got.PreferenceQuorumSize, c.wantK, c.wantAlpha)
		}
		if !IsByzantineSafe(got.SampleSize, got.PreferenceQuorumSize) {
			t.Errorf("Consensus(%d) = K=%d alpha=%d is NOT Byzantine-safe", c.n, got.SampleSize, got.PreferenceQuorumSize)
		}
	}
}

// TestIsByzantineSafe pins the invariant the engine enforces, including the exact
// live drift that bricked the rollout.
func TestIsByzantineSafe(t *testing.T) {
	if IsByzantineSafe(5, 3) {
		t.Error("K=5/alpha=3 must be rejected (2*3-5 = 1 < 2) — this is the live drift the new node refuses")
	}
	for _, c := range []struct {
		k, a int
		want bool
	}{
		{5, 4, true}, {5, 5, true}, {3, 2, true}, {1, 1, true},
		{20, 15, true}, {20, 13, false}, {4, 3, true}, {4, 2, false},
	} {
		if got := IsByzantineSafe(c.k, c.a); got != c.want {
			t.Errorf("IsByzantineSafe(%d,%d) = %v, want %v", c.k, c.a, got, c.want)
		}
	}
}

// TestAllProfilesByzantineSafe ensures no shipped profile is sub-BFT.
func TestAllProfilesByzantineSafe(t *testing.T) {
	for _, name := range ListProfiles() {
		p, err := GetProfile(name)
		if err != nil {
			t.Fatalf("GetProfile(%q): %v", name, err)
		}
		if !IsByzantineSafe(p.Consensus.SampleSize, p.Consensus.PreferenceQuorumSize) {
			t.Errorf("profile %q is NOT Byzantine-safe: K=%d alpha=%d",
				name, p.Consensus.SampleSize, p.Consensus.PreferenceQuorumSize)
		}
	}
}
