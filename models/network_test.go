// Copyright (C) 2022, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package models

import (
	"strings"
	"testing"
)

// TestEndpointsAreTheRoutesTheNodeServes pins the endpoint helpers to the
// node's live chain prefix. These five spelled /v1/bc out longhand instead of
// asking the composer, so they kept 404ing after the node deleted that prefix
// -- the drift a literal invites and a composer cannot have.
func TestEndpointsAreTheRoutesTheNodeServes(t *testing.T) {
	const base = "http://localhost:9650"

	for name, got := range map[string]string{
		"GetRPCEndpoint":      GetRPCEndpoint(base, "C"),
		"GetWSEndpoint":       GetWSEndpoint(base, "C"),
		"CChainEndpoint":      Local.CChainEndpoint(),
		"BlockchainEndpoint":  Local.BlockchainEndpoint("2oYMBNV4eNHyqk2fjjV5nVQLDbtmNJzq5s3qs3Lo6ftnC6FByM"),
		"BlockchainEndpointC": Local.BlockchainEndpoint(""),
	} {
		if !strings.Contains(got, "/v1/chain/") {
			t.Errorf("%s = %q: missing the /v1/chain/ prefix the node serves", name, got)
		}
		if strings.Contains(got, "/v1/bc") {
			t.Errorf("%s = %q: still carries /v1/bc, which the node deleted", name, got)
		}
	}
}

// TestWSEndpointKeepsItsScheme guards the one helper whose suffix is not /rpc:
// composing the chain address must not quietly turn a socket into an RPC call.
func TestWSEndpointKeepsItsScheme(t *testing.T) {
	got := GetWSEndpoint("http://localhost:9650", "C")

	if want := "http://localhost:9650/v1/chain/C/ws"; got != want {
		t.Fatalf("GetWSEndpoint = %q, want %q", got, want)
	}
}
