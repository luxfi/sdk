// Copyright (C) 2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

package validatormanager

import (
	"errors"

	"github.com/luxfi/constants"
	"github.com/luxfi/evm/plugin/evm/client"
	"github.com/luxfi/ids"
	"github.com/luxfi/net"
)

// GetL1ValidatorUptimeSeconds returns the uptime of the L1 validator
func GetL1ValidatorUptimeSeconds(rpcURL string, nodeID ids.NodeID) (uint64, error) {
	ctx, cancel := apiRequestContext()
	defer cancel()
	networkEndpoint, blockchainID, err := netutil.SplitLuxgoRPCURI(rpcURL)
	if err != nil {
		return 0, err
	}
	evmCli := client.NewClient(networkEndpoint, blockchainID)
	validators, err := evmCli.GetCurrentValidators(ctx, []ids.NodeID{nodeID})
	if err != nil {
		return 0, err
	}
	if len(validators) > 0 {
		deductibleSeconds := uint64(constants.ValidatorUptimeDeductible.Seconds())
		if validators[0].UptimeSeconds > deductibleSeconds {
			return validators[0].UptimeSeconds - deductibleSeconds, nil
		}
		return 0, nil
	}

	return 0, errors.New("nodeID not found in validator set: " + nodeID.String())
}
