// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package verify

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	consensuscontext "github.com/luxfi/consensus/context"
)

// testValidatorState is a test implementation of ValidatorState
type testValidatorState struct {
	height     uint64
	validators map[ids.ID]map[ids.NodeID]uint64
	subnets    map[ids.ID]ids.ID // chainID -> subnetID
	err        error
}

func (s *testValidatorState) GetCurrentHeight() (uint64, error) {
	return s.height, s.err
}

func (s *testValidatorState) GetMinimumHeight(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (s *testValidatorState) GetValidatorSet(height uint64, netID ids.ID) (map[ids.NodeID]uint64, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.validators[netID], nil
}

func (s *testValidatorState) GetNetID(ctx context.Context, chainID ids.ID) (ids.ID, error) {
	if s.err != nil {
		return ids.Empty, s.err
	}
	if subnet, ok := s.subnets[chainID]; ok {
		return subnet, nil
	}
	return ids.Empty, errMissing
}

func (s *testValidatorState) GetChainID(blockID ids.ID) (ids.ID, error) {
	return ids.Empty, nil
}

func (s *testValidatorState) GetCurrentValidators(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*consensuscontext.GetValidatorOutput, error) {
	return nil, nil
}

var errMissing = errors.New("missing")

func TestSameNet(t *testing.T) {
	netID0 := ids.GenerateTestID()
	netID1 := ids.GenerateTestID()
	chainID0 := ids.GenerateTestID()
	chainID1 := ids.GenerateTestID()

	tests := []struct {
		name     string
		chainCtx *ChainContext
		chainID  ids.ID
		result   error
	}{
		{
			name: "same chain",
			chainCtx: &ChainContext{
				ChainID:        chainID0,
				NetID:          netID0,
				ValidatorState: &testValidatorState{},
			},
			chainID: chainID0,
			result:  ErrSameChainID,
		},
		{
			name: "unknown chain",
			chainCtx: &ChainContext{
				ChainID: chainID0,
				NetID:   netID0,
				ValidatorState: &testValidatorState{
					subnets: map[ids.ID]ids.ID{},
					err:     errMissing,
				},
			},
			chainID: chainID1,
			result:  errMissing,
		},
		{
			name: "wrong subnet",
			chainCtx: &ChainContext{
				ChainID: chainID0,
				NetID:   netID0,
				ValidatorState: &testValidatorState{
					subnets: map[ids.ID]ids.ID{
						chainID1: netID1,
					},
				},
			},
			chainID: chainID1,
			result:  ErrMismatchedNetIDs,
		},
		{
			name: "same subnet",
			chainCtx: &ChainContext{
				ChainID: chainID0,
				NetID:   netID0,
				ValidatorState: &testValidatorState{
					subnets: map[ids.ID]ids.ID{
						chainID1: netID0,
					},
				},
			},
			chainID: chainID1,
			result:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SameNet(context.Background(), test.chainCtx, test.chainID)
			require.ErrorIs(t, result, test.result)
		})
	}
}
