// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// warptest exposes common functionality for testing the warp package.
package warptest

import (
	"time"

	"github.com/luxfi/evm/plugin/evm/validators/interfaces"
	stateinterfaces "github.com/luxfi/evm/plugin/evm/validators/state/interfaces"
	"github.com/luxfi/ids"
)

var _ interfaces.ValidatorReader = (*NoOpValidatorReader)(nil)

type NoOpValidatorReader struct{}

func (NoOpValidatorReader) GetValidatorAndUptime(ids.ID) (stateinterfaces.Validator, time.Duration, time.Time, error) {
	return stateinterfaces.Validator{}, 0, time.Time{}, nil
}
