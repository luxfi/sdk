// (c) 2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// warptest exposes common functionality for testing the warp package.
package warptest

import (
	"time"

	"github.com/luxfi/node/ids"
	"github.com/luxfi/evm/plugin/evm/validators/interfaces"
	stateinterfaces "github.com/luxfi/evm/plugin/evm/validators/state/interfaces"
)

var _ interfaces.ValidatorReader = &NoOpValidatorReader{}

type NoOpValidatorReader struct{}

func (NoOpValidatorReader) GetValidatorAndUptime(ids.ID) (stateinterfaces.Validator, time.Duration, time.Time, error) {
	return stateinterfaces.Validator{}, 0, time.Time{}, nil
}
