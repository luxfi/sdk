// (c) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"github.com/luxfi/node/ids"
	luxValidators "github.com/luxfi/node/consensus/validators"
	warpValidators "github.com/luxfi/evm/warp/validators"
)

// newValidatorStateWrapper wraps the node's ValidatorState to match the warp validators.State interface
func newValidatorStateWrapper(state luxValidators.State) *warpValidators.State {
	// Use warp's NewState to handle the Primary Network special cases
	// The empty ids are for the primary network ID and subnet ID which will be filled
	// based on the context when used
	return warpValidators.NewState(state, ids.Empty, ids.Empty, false)
}