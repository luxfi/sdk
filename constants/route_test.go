// Copyright (C) 2022, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package constants

import (
	"strings"
	"testing"
)

// TestChainIsTheSegmentTheNodeServes pins the single segment every chain call
// hangs off. The node moved it from "bc" to "chain" and DELETED "bc", so a
// composer still spelling the old one 404s on every chain call there is.
// Reading luxconstants.ChainAliasPrefix is what carries this SDK across that
// move; this test is what proves it arrived.
func TestChainIsTheSegmentTheNodeServes(t *testing.T) {
	const uri = "http://localhost:9650"

	for _, alias := range []string{"P", "X", "C"} {
		got := Chain(uri, alias)

		if want := uri + "/v1/chain/" + alias; got != want {
			t.Fatalf("Chain(%q) = %q, want %q", alias, got, want)
		}
		if !strings.Contains(got, "/v1/chain/") {
			t.Fatalf("Chain(%q) = %q: missing the /v1/chain/ prefix the node serves", alias, got)
		}
		if strings.Contains(got, "/v1/bc") {
			t.Fatalf("Chain(%q) = %q: still carries /v1/bc, which the node deleted", alias, got)
		}
	}
}

// TestChainRPCIsTheAddressCallersDial covers the composition every RPC caller
// actually builds -- the chain address plus /rpc -- because that, and not the
// bare chain address, is what a 404 shows up on.
func TestChainRPCIsTheAddressCallersDial(t *testing.T) {
	got := Chain("http://localhost:9650", "C") + "/rpc"

	if want := "http://localhost:9650/v1/chain/C/rpc"; got != want {
		t.Fatalf("C-Chain RPC address = %q, want %q", got, want)
	}
	if strings.Contains(got, "/v1/bc") {
		t.Fatalf("C-Chain RPC address = %q: still carries the deleted /v1/bc", got)
	}
}
