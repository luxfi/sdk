// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"context"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/luxfi/ids"
	"github.com/luxfi/consensus/validator"
	"github.com/luxfi/consensus/validator/validatorsmock"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/crypto/bls/signer/localsigner"
	"github.com/luxfi/math/set"
	"github.com/luxfi/metric"
	"github.com/luxfi/upgrade"
	"github.com/luxfi/const"
)

var (
	subnetID = ids.GenerateTestID()
)

// testValidatorStateAdapter wraps validators.State to implement ValidatorState
// converting GetValidatorOutput to ValidatorData
type testValidatorStateAdapter struct {
	validators.State
}

func (t *testValidatorStateAdapter) GetValidatorSet(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*ValidatorData, error) {
	validatorSet, err := t.State.GetValidatorSet(ctx, height, subnetID)
	if err != nil {
		return nil, err
	}

	result := make(map[ids.NodeID]*ValidatorData, len(validatorSet))
	for nodeID, validator := range validatorSet {
		result[nodeID] = &ValidatorData{
			NodeID:    validator.NodeID,
			PublicKey: validator.PublicKey,
			Weight:    validator.Weight,
		}
	}
	return result, nil
}

func TestGetCanonicalValidatorSet(t *testing.T) {
	type test struct {
		name           string
		stateF         func(*gomock.Controller) validators.State
		expectedVdrs   []*Validator
		expectedWeight uint64
		expectedErr    error
	}

	tests := []test{
		{
			name: "can't get validator set",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, subnetID).Return(nil, errTest)
				return state
			},
			expectedErr: errTest,
		},
		{
			name: "all validators have public keys; no duplicate pub keys",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, subnetID).Return(
					map[ids.NodeID]*validators.GetValidatorOutput{
						testVdrs[0].nodeID: {
							NodeID:    testVdrs[0].nodeID,
							PublicKey: bls.PublicKeyToUncompressedBytes(testVdrs[0].vdr.PublicKey),
							Weight:    testVdrs[0].vdr.Weight,
						},
						testVdrs[1].nodeID: {
							NodeID:    testVdrs[1].nodeID,
							PublicKey: bls.PublicKeyToUncompressedBytes(testVdrs[1].vdr.PublicKey),
							Weight:    testVdrs[1].vdr.Weight,
						},
					},
					nil,
				)
				return state
			},
			expectedVdrs:   []*Validator{testVdrs[0].vdr, testVdrs[1].vdr},
			expectedWeight: 6,
			expectedErr:    nil,
		},
		{
			name: "all validators have public keys; duplicate pub keys",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, subnetID).Return(
					map[ids.NodeID]*validators.GetValidatorOutput{
						testVdrs[0].nodeID: {
							NodeID:    testVdrs[0].nodeID,
							PublicKey: bls.PublicKeyToUncompressedBytes(testVdrs[0].vdr.PublicKey),
							Weight:    testVdrs[0].vdr.Weight,
						},
						testVdrs[1].nodeID: {
							NodeID:    testVdrs[1].nodeID,
							PublicKey: bls.PublicKeyToUncompressedBytes(testVdrs[1].vdr.PublicKey),
							Weight:    testVdrs[1].vdr.Weight,
						},
						testVdrs[2].nodeID: {
							NodeID:    testVdrs[2].nodeID,
							PublicKey: bls.PublicKeyToUncompressedBytes(testVdrs[0].vdr.PublicKey),
							Weight:    testVdrs[0].vdr.Weight,
						},
					},
					nil,
				)
				return state
			},
			expectedVdrs: []*Validator{
				{
					PublicKey:      testVdrs[0].vdr.PublicKey,
					PublicKeyBytes: testVdrs[0].vdr.PublicKeyBytes,
					Weight:         testVdrs[0].vdr.Weight * 2,
					NodeIDs: []ids.NodeID{
						testVdrs[0].nodeID,
						testVdrs[2].nodeID,
					},
				},
				testVdrs[1].vdr,
			},
			expectedWeight: 9,
			expectedErr:    nil,
		},
		{
			name: "validator without public key; no duplicate pub keys",
			stateF: func(ctrl *gomock.Controller) validators.State {
				state := validatorsmock.NewState(ctrl)
				state.EXPECT().GetValidatorSet(gomock.Any(), pChainHeight, subnetID).Return(
					map[ids.NodeID]*validators.GetValidatorOutput{
						testVdrs[0].nodeID: {
							NodeID:    testVdrs[0].nodeID,
							PublicKey: nil,
							Weight:    testVdrs[0].vdr.Weight,
						},
						testVdrs[1].nodeID: {
							NodeID:    testVdrs[1].nodeID,
							PublicKey: bls.PublicKeyToUncompressedBytes(testVdrs[1].vdr.PublicKey),
							Weight:    testVdrs[1].vdr.Weight,
						},
					},
					nil,
				)
				return state
			},
			expectedVdrs:   []*Validator{testVdrs[1].vdr},
			expectedWeight: 6,
			expectedErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)

			state := tt.stateF(ctrl)
			// Wrap validators.State to implement ValidatorState
			wrappedState := &testValidatorStateAdapter{
				State: state,
			}

			validators, err := GetCanonicalValidatorSetFromSubsubnetID(t.Context(), wrappedState, pChainHeight, subnetID)
			require.ErrorIs(err, tt.expectedErr)
			if err != nil {
				return
			}
			require.Equal(tt.expectedWeight, validators.TotalWeight)

			// These are pointers so have to test equality like this
			require.Len(validators.Validators, len(tt.expectedVdrs))
			for i, expectedVdr := range tt.expectedVdrs {
				gotVdr := validators.Validators[i]
				expectedPKBytes := bls.PublicKeyToUncompressedBytes(expectedVdr.PublicKey)
				gotPKBytes := bls.PublicKeyToUncompressedBytes(gotVdr.PublicKey)
				require.Equal(expectedPKBytes, gotPKBytes)
				require.Equal(expectedVdr.PublicKeyBytes, gotVdr.PublicKeyBytes)
				require.Equal(expectedVdr.Weight, gotVdr.Weight)
				require.ElementsMatch(expectedVdr.NodeIDs, gotVdr.NodeIDs)
			}
		})
	}
}

func TestFilterValidators(t *testing.T) {
	sk0, err := localsigner.New()
	require.NoError(t, err)
	pk0 := sk0.PublicKey()
	vdr0 := &Validator{
		PublicKey:      pk0,
		PublicKeyBytes: bls.PublicKeyToUncompressedBytes(pk0),
		Weight:         1,
	}

	sk1, err := localsigner.New()
	require.NoError(t, err)
	pk1 := sk1.PublicKey()
	vdr1 := &Validator{
		PublicKey:      pk1,
		PublicKeyBytes: bls.PublicKeyToUncompressedBytes(pk1),
		Weight:         2,
	}

	type test struct {
		name         string
		indices      set.Bits
		vdrs         []*Validator
		expectedVdrs []*Validator
		expectedErr  error
	}

	tests := []test{
		{
			name:         "empty",
			indices:      set.NewBits(),
			vdrs:         []*Validator{},
			expectedVdrs: []*Validator{},
			expectedErr:  nil,
		},
		{
			name:        "unknown validator",
			indices:     set.NewBits(2),
			vdrs:        []*Validator{vdr0, vdr1},
			expectedErr: ErrUnknownValidator,
		},
		{
			name:    "two filtered out",
			indices: set.NewBits(),
			vdrs: []*Validator{
				vdr0,
				vdr1,
			},
			expectedVdrs: []*Validator{},
			expectedErr:  nil,
		},
		{
			name:    "one filtered out",
			indices: set.NewBits(1),
			vdrs: []*Validator{
				vdr0,
				vdr1,
			},
			expectedVdrs: []*Validator{
				vdr1,
			},
			expectedErr: nil,
		},
		{
			name:    "none filtered out",
			indices: set.NewBits(0, 1),
			vdrs: []*Validator{
				vdr0,
				vdr1,
			},
			expectedVdrs: []*Validator{
				vdr0,
				vdr1,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			vdrs, err := FilterValidators(tt.indices, tt.vdrs)
			require.ErrorIs(err, tt.expectedErr)
			if tt.expectedErr != nil {
				return
			}
			require.Equal(tt.expectedVdrs, vdrs)
		})
	}
}

func TestSumWeight(t *testing.T) {
	vdr0 := &Validator{
		Weight: 1,
	}
	vdr1 := &Validator{
		Weight: 2,
	}
	vdr2 := &Validator{
		Weight: math.MaxUint64,
	}

	type test struct {
		name        string
		vdrs        []*Validator
		expectedSum uint64
		expectedErr error
	}

	tests := []test{
		{
			name:        "empty",
			vdrs:        []*Validator{},
			expectedSum: 0,
		},
		{
			name:        "one",
			vdrs:        []*Validator{vdr0},
			expectedSum: 1,
		},
		{
			name:        "two",
			vdrs:        []*Validator{vdr0, vdr1},
			expectedSum: 3,
		},
		{
			name:        "overflow",
			vdrs:        []*Validator{vdr0, vdr2},
			expectedErr: ErrWeightOverflow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			sum, err := SumWeight(tt.vdrs)
			require.ErrorIs(err, tt.expectedErr)
			if tt.expectedErr != nil {
				return
			}
			require.Equal(tt.expectedSum, sum)
		})
	}
}

func BenchmarkGetCanonicalValidatorSet(b *testing.B) {
	pChainHeight := uint64(1)
	subnetID := ids.GenerateTestID()
	numNodes := 10_000
	getValidatorOutputs := make([]*validators.GetValidatorOutput, 0, numNodes)
	for i := 0; i < numNodes; i++ {
		nodeID := ids.GenerateTestNodeID()
		blsPrivateKey, err := localsigner.New()
		require.NoError(b, err)
		blsPublicKey := blsPrivateKey.PublicKey()
		getValidatorOutputs = append(getValidatorOutputs, &validators.GetValidatorOutput{
			NodeID:    nodeID,
			PublicKey: bls.PublicKeyToUncompressedBytes(blsPublicKey),
			Weight:    20,
		})
	}

	for _, size := range []int{0, 1, 10, 100, 1_000, 10_000} {
		getValidatorsOutput := make(map[ids.NodeID]*validators.GetValidatorOutput)
		for i := 0; i < size; i++ {
			validator := getValidatorOutputs[i]
			getValidatorsOutput[validator.NodeID] = validator
		}
		// Create a simple validator state for benchmarking
		wrappedState := newMockValidatorState(
			func() map[ids.NodeID]*ValidatorData {
				result := make(map[ids.NodeID]*ValidatorData, len(getValidatorsOutput))
				for nodeID, vdr := range getValidatorsOutput {
					result[nodeID] = &ValidatorData{
						NodeID:    vdr.NodeID,
						PublicKey: vdr.PublicKey,
						Weight:    vdr.Weight,
					}
				}
				return result
			}(),
			nil,
		)

		b.Run(strconv.Itoa(size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := GetCanonicalValidatorSetFromSubsubnetID(b.Context(), wrappedState, pChainHeight, subnetID)
				require.NoError(b, err)
			}
		})
	}
}

// mockValidatorState is a test mock that tracks call counts
type mockValidatorState struct {
	callCount int
	data      map[ids.NodeID]*ValidatorData
	err       error
}

func (m *mockValidatorState) GetValidatorSet(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*ValidatorData, error) {
	m.callCount++
	return m.data, m.err
}

func newMockValidatorState(data map[ids.NodeID]*ValidatorData, err error) *mockValidatorState {
	return &mockValidatorState{data: data, err: err}
}

func TestCachedValidatorState(t *testing.T) {
	ctx := context.Background()
	height := uint64(100)
	subnet1 := ids.GenerateTestID()
	subnet2 := ids.GenerateTestID()

	// Create test validator data
	nodeID1 := ids.GenerateTestNodeID()
	nodeID2 := ids.GenerateTestNodeID()
	testData := map[ids.NodeID]*ValidatorData{
		nodeID1: {
			NodeID:    nodeID1,
			PublicKey: bls.PublicKeyToUncompressedBytes(testVdrs[0].vdr.PublicKey),
			Weight:    100,
		},
		nodeID2: {
			NodeID:    nodeID2,
			PublicKey: bls.PublicKeyToUncompressedBytes(testVdrs[1].vdr.PublicKey),
			Weight:    200,
		},
	}

	type test struct {
		name              string
		state             *mockValidatorState
		upgradeConfig     *upgrade.Config
		networkID         uint32
		expectedCallCount int
		operations        func(*testing.T, *CachedValidatorState)
	}

	tests := []test{
		{
			name:              "pre-Granite no caching",
			state:             newMockValidatorState(testData, nil),
			upgradeConfig:     &upgrade.Config{GraniteTime: time.Now().Add(1 * time.Hour)},
			networkID:         constants.MainnetID,
			expectedCallCount: 2, // Should call underlying state twice (no caching)
			operations: func(t *testing.T, cached *CachedValidatorState) {
				vdrs1, err := cached.GetValidatorSet(ctx, height, subnet1)
				require.NoError(t, err)
				require.Equal(t, testData, vdrs1)
				
				vdrs2, err := cached.GetValidatorSet(ctx, height, subnet1)
				require.NoError(t, err)
				require.Equal(t, testData, vdrs2)
			},
		},
		{
			name:              "post-Granite with caching",
			state:             newMockValidatorState(testData, nil),
			upgradeConfig:     &upgrade.Config{GraniteTime: time.Now().Add(-1 * time.Hour)},
			networkID:         constants.MainnetID,
			expectedCallCount: 1, // Should call underlying state once, then use cache
			operations: func(t *testing.T, cached *CachedValidatorState) {
				vdrs1, err := cached.GetValidatorSet(ctx, height, subnet1)
				require.NoError(t, err)
				require.Equal(t, testData, vdrs1)
				
				vdrs2, err := cached.GetValidatorSet(ctx, height, subnet1)
				require.NoError(t, err)
				require.Equal(t, testData, vdrs2)
			},
		},
		{
			name:              "different heights cached separately",
			state:             newMockValidatorState(testData, nil),
			upgradeConfig:     &upgrade.Config{GraniteTime: time.Now().Add(-1 * time.Hour)},
			networkID:         constants.MainnetID,
			expectedCallCount: 2, // Two different heights = two calls
			operations: func(t *testing.T, cached *CachedValidatorState) {
				vdrs1, err := cached.GetValidatorSet(ctx, height, subnet1)
				require.NoError(t, err)
				require.Equal(t, testData, vdrs1)
				
				vdrs2, err := cached.GetValidatorSet(ctx, height+1, subnet1)
				require.NoError(t, err)
				require.Equal(t, testData, vdrs2)
			},
		},
		{
			name:              "different subnets cached separately",
			state:             newMockValidatorState(testData, nil),
			upgradeConfig:     &upgrade.Config{GraniteTime: time.Now().Add(-1 * time.Hour)},
			networkID:         constants.MainnetID,
			expectedCallCount: 2, // Two different subnets = two calls
			operations: func(t *testing.T, cached *CachedValidatorState) {
				vdrs1, err := cached.GetValidatorSet(ctx, height, subnet1)
				require.NoError(t, err)
				require.Equal(t, testData, vdrs1)
				
				vdrs2, err := cached.GetValidatorSet(ctx, height, subnet2)
				require.NoError(t, err)
				require.Equal(t, testData, vdrs2)
			},
		},
		{
			name:              "error propagates without caching",
			state:             newMockValidatorState(nil, errTest),
			upgradeConfig:     &upgrade.Config{GraniteTime: time.Now().Add(-1 * time.Hour)},
			networkID:         constants.MainnetID,
			expectedCallCount: 1,
			operations: func(t *testing.T, cached *CachedValidatorState) {
				_, err := cached.GetValidatorSet(ctx, height, subnet1)
				require.ErrorIs(t, err, errTest)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			registerer := metric.NewRegistry()

			cached, err := NewCachedValidatorState(tt.state, tt.upgradeConfig, tt.networkID, registerer)
			require.NoError(err)
			require.NotNil(cached)

			// Run test operations
			tt.operations(t, cached)

			// Verify call count
			require.Equal(tt.expectedCallCount, tt.state.callCount, "unexpected number of calls to underlying state")
		})
	}
}
