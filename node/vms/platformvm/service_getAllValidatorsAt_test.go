// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build skip

package platformvm

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/const"
	"github.com/luxfi/sdk/node/vms/platformvm/genesis/genesistest"

	pchainapi "github.com/luxfi/sdk/node/vms/platformvm/api"
)

// TestGetAllValidatorsAt tests the GetAllValidatorsAt RPC endpoint
func TestGetAllValidatorsAt(t *testing.T) {
	require := require.New(t)
	service, _ := defaultService(t)

	genesis := genesistest.New(t, genesistest.Config{})

	args := GetAllValidatorsAtArgs{}
	response := GetAllValidatorsAtReply{}

	service.vm.ctx.Lock.Lock()
	lastAccepted := service.vm.manager.LastAccepted()
	lastAcceptedBlk, err := service.vm.manager.GetBlock(lastAccepted)
	require.NoError(err)
	service.vm.ctx.Lock.Unlock()

	// Test at genesis height
	args.Height = pchainapi.Height(lastAcceptedBlk.Height())
	require.NoError(service.GetAllValidatorsAt(&http.Request{}, &args, &response))

	// Should have at least the primary network
	require.Contains(response.ValidatorSets, constants.PrimaryNetworkID)
	require.Len(response.ValidatorSets[constants.PrimaryNetworkID], len(genesis.Validators))

	// Verify ValidatorSets is not nil and is a proper map
	require.NotNil(response.ValidatorSets)
	require.IsType(response.ValidatorSets, map[ids.ID]map[ids.NodeID]*validators.GetValidatorOutput{})

	// Test with proposed height
	args.Height = pchainapi.Height(pchainapi.ProposedHeight)
	require.NoError(service.GetAllValidatorsAt(context.WithValue(context.Background(), struct{}{}, "test"), &args, &response))
	require.Contains(response.ValidatorSets, constants.PrimaryNetworkID)

	// Verify each validator set has proper structure
	for netID, validatorSet := range response.ValidatorSets {
		require.NotNil(validatorSet, "validator set for net %s should not be nil", netID)
		for nodeID, validator := range validatorSet {
			require.Equal(nodeID, validator.NodeID, "nodeID mismatch in validator set")
			require.NotZero(validator.Weight, "validator weight should not be zero")
		}
	}
}
