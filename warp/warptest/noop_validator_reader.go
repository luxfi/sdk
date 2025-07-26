// (c) 2024, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// warptest exposes common functionality for testing the warp package.
package warptest

import (
	"time"

	"github.com/luxfi/node/ids"
	validatorinterfaces "github.com/luxfi/evm/plugin/evm/validators/interfaces"
	stateinterfaces "github.com/luxfi/evm/plugin/evm/validators/state/interfaces"
)

var _ validatorinterfaces.ValidatorReader = &NoOpValidatorReader{}

type NoOpValidatorReader struct{}

func (NoOpValidatorReader) GetValidatorAndUptime(ids.ID) (stateinterfaces.Validator, time.Duration, time.Time, error) {
	return stateinterfaces.Validator{}, 0, time.Time{}, nil
}
