// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"context"
	"fmt"
	"time"

	"github.com/luxfi/database"
	"github.com/luxfi/const"
)

func (vm *VM) HealthCheck(context.Context) (interface{}, error) {
	localPrimaryValidator, err := vm.state.GetCurrentValidator(
		constants.PrimaryNetworkID,
		vm.nodeID,
	)
	switch err {
	case nil:
		vm.metrics.SetTimeUntilUnstake(time.Until(localPrimaryValidator.EndTime))
	case database.ErrNotFound:
		vm.metrics.SetTimeUntilUnstake(0)
	default:
		return nil, fmt.Errorf("couldn't get current local validator: %w", err)
	}

	for chainID := range vm.TrackedChains {
		localChainValidator, err := vm.state.GetCurrentValidator(
			chainID,
			vm.nodeID,
		)
		switch err {
		case nil:
			vm.metrics.SetTimeUntilNetUnstake(chainID, time.Until(localChainValidator.EndTime))
		case database.ErrNotFound:
			vm.metrics.SetTimeUntilNetUnstake(chainID, 0)
		default:
			return nil, fmt.Errorf("couldn't get current chain validator of %q: %w", chainID, err)
		}
	}
	return nil, nil
}
