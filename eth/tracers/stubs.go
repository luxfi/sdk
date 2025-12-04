// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package tracers

import (
	"github.com/luxfi/evm/rpc"
)

// StateReleaseFunc is a function that releases the state after tracing.
// This is a stub implementation - the full implementation is in api.go.disabled
type StateReleaseFunc = func()

// Backend is the interface required by tracers.
// This is a stub - the full interface is in api.go.disabled
type Backend interface{}

// APIs returns the RPC descriptors the tracers package offers.
// This is a stub implementation - the full implementation is in api.go.disabled
func APIs(backend Backend) []rpc.API {
	return []rpc.API{}
}
