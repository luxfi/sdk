// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validator

import (
	"errors"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/platformvm"
)

// GetRemainingValidationTime returns the time remaining for [nodeID] on [chainID].
func GetRemainingValidationTime(networkEndpoint string, nodeID ids.NodeID, chainID ids.ID, startTime time.Time) (time.Duration, error) {
	ctx, cancel := apiRequestContext()
	defer cancel()
	platformCli := platformvm.NewClient(networkEndpoint)
	vs, err := platformCli.GetCurrentValidators(ctx, chainID, nil)
	if err != nil {
		return 0, err
	}
	for _, v := range vs {
		if v.NodeID == nodeID {
			return time.Unix(int64(v.EndTime), 0).Sub(startTime), nil
		}
	}
	return 0, errors.New("nodeID not found in validator set: " + nodeID.String())
}
