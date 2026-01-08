// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package admin

import (
	"errors"
	"net/http"
	"testing"

	"github.com/luxfi/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/luxfi/database/memdb"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/chains"
	"github.com/luxfi/sdk/utils/formatting"
	"github.com/luxfi/sdk/node/vms/registry/registrymock"
	"github.com/luxfi/sdk/node/vms/vmsmock"
)

var (
	errTest = errors.New("test error")
)

type loadVMsTest struct {
	admin          *Admin
	mockVMManager  *vmsmock.Manager
	mockVMRegistry *registrymock.VMRegistry
}

func initLoadVMsTest(t *testing.T) *loadVMsTest {
	ctrl := gomock.NewController(t)

	mockVMRegistry := registrymock.NewVMRegistry(ctrl)
	mockVMManager := vmsmock.NewManager(ctrl)

	return &loadVMsTest{
		admin: &Admin{Config: Config{
			Log:          log.NewNoOpLogger(),
			VMRegistry:   mockVMRegistry,
			VMManager:    mockVMManager,
			ChainManager: chains.TestManager,
		}},
		mockVMManager:  mockVMManager,
		mockVMRegistry: mockVMRegistry,
	}
}

// Tests behavior for LoadVMs if everything succeeds.
func TestLoadVMsSuccess(t *testing.T) {
	require := require.New(t)

	resources := initLoadVMsTest(t)

	id1 := ids.GenerateTestID()
	id2 := ids.GenerateTestID()

	newVMs := []ids.ID{id1, id2}
	failedVMs := map[ids.ID]error{
		ids.GenerateTestID(): errTest,
	}
	// every vm is at least aliased to itself.
	alias1 := []string{id1.String(), "vm1-alias-1", "vm1-alias-2"}
	alias2 := []string{id2.String(), "vm2-alias-1", "vm2-alias-2"}
	// we expect that we dedup the redundant alias of vmId.
	expectedVMRegistry := map[ids.ID][]string{
		id1: alias1[1:],
		id2: alias2[1:],
	}

	resources.mockVMRegistry.EXPECT().Reload(gomock.Any()).Times(1).Return(newVMs, failedVMs, nil)
	resources.mockVMManager.EXPECT().Aliases(id1).Times(1).Return(alias1, nil)
	resources.mockVMManager.EXPECT().Aliases(id2).Times(1).Return(alias2, nil)

	// execute test
	reply := LoadVMsReply{}
	require.NoError(resources.admin.LoadVMs(&http.Request{}, nil, &reply))
	require.Equal(expectedVMRegistry, reply.NewVMs)
}

// Tests behavior for LoadVMs if we fail to reload vms.
func TestLoadVMsReloadFails(t *testing.T) {
	require := require.New(t)

	resources := initLoadVMsTest(t)

	// Reload fails
	resources.mockVMRegistry.EXPECT().Reload(gomock.Any()).Times(1).Return(nil, nil, errTest)

	reply := LoadVMsReply{}
	err := resources.admin.LoadVMs(&http.Request{}, nil, &reply)
	require.ErrorIs(err, errTest)
}

// Tests behavior for LoadVMs if we fail to fetch our aliases
func TestLoadVMsGetAliasesFails(t *testing.T) {
	require := require.New(t)

	resources := initLoadVMsTest(t)

	id1 := ids.GenerateTestID()
	id2 := ids.GenerateTestID()
	newVMs := []ids.ID{id1, id2}
	failedVMs := map[ids.ID]error{
		ids.GenerateTestID(): errTest,
	}
	// every vm is at least aliased to itself.
	alias1 := []string{id1.String(), "vm1-alias-1", "vm1-alias-2"}

	resources.mockVMRegistry.EXPECT().Reload(gomock.Any()).Times(1).Return(newVMs, failedVMs, nil)
	resources.mockVMManager.EXPECT().Aliases(id1).Times(1).Return(alias1, nil)
	resources.mockVMManager.EXPECT().Aliases(id2).Times(1).Return(nil, errTest)

	reply := LoadVMsReply{}
	err := resources.admin.LoadVMs(&http.Request{}, nil, &reply)
	require.ErrorIs(err, errTest)
}

// Tests behavior for ListVMs if everything succeeds.
func TestListVMsSuccess(t *testing.T) {
	require := require.New(t)

	resources := initLoadVMsTest(t)

	id1 := ids.GenerateTestID()
	id2 := ids.GenerateTestID()

	vmIDs := []ids.ID{id1, id2}
	// every vm is at least aliased to itself.
	alias1 := []string{id1.String(), "vm1-alias-1", "vm1-alias-2"}
	alias2 := []string{id2.String(), "vm2-alias-1"}

	resources.mockVMManager.EXPECT().ListFactories().Times(1).Return(vmIDs, nil)
	resources.mockVMManager.EXPECT().Aliases(id1).Times(1).Return(alias1, nil)
	resources.mockVMManager.EXPECT().Aliases(id2).Times(1).Return(alias2, nil)

	reply := ListVMsReply{}
	require.NoError(resources.admin.ListVMs(nil, nil, &reply))

	require.Len(reply.VMs, 2)
	require.Equal(id1.String(), reply.VMs[id1.String()].ID)
	require.Equal([]string{"vm1-alias-1", "vm1-alias-2"}, reply.VMs[id1.String()].Aliases)
	require.Equal(id2.String(), reply.VMs[id2.String()].ID)
	require.Equal([]string{"vm2-alias-1"}, reply.VMs[id2.String()].Aliases)
}

// Tests behavior for ListVMs if we fail to list factories.
func TestListVMsListFactoriesFails(t *testing.T) {
	require := require.New(t)

	resources := initLoadVMsTest(t)

	resources.mockVMManager.EXPECT().ListFactories().Times(1).Return(nil, errTest)

	reply := ListVMsReply{}
	err := resources.admin.ListVMs(nil, nil, &reply)
	require.ErrorIs(err, errTest)
}

// Tests behavior for ListVMs if we fail to get aliases.
func TestListVMsGetAliasesFails(t *testing.T) {
	require := require.New(t)

	resources := initLoadVMsTest(t)

	id1 := ids.GenerateTestID()
	vmIDs := []ids.ID{id1}

	resources.mockVMManager.EXPECT().ListFactories().Times(1).Return(vmIDs, nil)
	resources.mockVMManager.EXPECT().Aliases(id1).Times(1).Return(nil, errTest)

	reply := ListVMsReply{}
	err := resources.admin.ListVMs(nil, nil, &reply)
	require.ErrorIs(err, errTest)
}

func TestServiceDBGet(t *testing.T) {
	a := &Admin{Config: Config{
		Log: log.NewNoOpLogger(),
		DB:  memdb.New(),
	}}

	helloBytes := []byte("hello")
	helloHex, err := formatting.Encode(formatting.HexNC, helloBytes)
	require.NoError(t, err)

	worldBytes := []byte("world")
	worldHex, err := formatting.Encode(formatting.HexNC, worldBytes)
	require.NoError(t, err)

	require.NoError(t, a.DB.Put(helloBytes, worldBytes))

	tests := []struct {
		name          string
		key           string
		expectedValue string
		expectedError bool
	}{
		{
			name:          "key exists",
			key:           helloHex,
			expectedValue: worldHex,
			expectedError: false,
		},
		{
			name:          "key doesn't exist",
			key:           "",
			expectedValue: "",
			expectedError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)

			reply := &DBGetReply{}
			err := a.DbGet(
				nil,
				&DBGetArgs{
					Key: test.key,
				},
				reply,
			)
			if test.expectedError {
				require.Error(err)
			} else {
				require.NoError(err)
				require.Equal(test.expectedValue, reply.Value)
			}
		})
	}
}

// mockChainTracker implements ChainTracker for testing
type mockChainTracker struct {
	trackedChains map[ids.ID]struct{}
	trackError    error
}

func newMockChainTracker() *mockChainTracker {
	return &mockChainTracker{
		trackedChains: make(map[ids.ID]struct{}),
	}
}

func (m *mockChainTracker) TrackChain(chainID ids.ID) error {
	if m.trackError != nil {
		return m.trackError
	}
	m.trackedChains[chainID] = struct{}{}
	return nil
}

func (m *mockChainTracker) TrackedChains() set.Set[ids.ID] {
	result := set.NewSet[ids.ID](len(m.trackedChains))
	for id := range m.trackedChains {
		result.Add(id)
	}
	return result
}

func TestSetTrackedChainsSuccess(t *testing.T) {
	require := require.New(t)

	tracker := newMockChainTracker()
	a := &Admin{Config: Config{
		Log:     log.NewNoOpLogger(),
		Network: tracker,
	}}

	chain1 := ids.GenerateTestID()
	chain2 := ids.GenerateTestID()

	args := &SetTrackedChainsArgs{
		Chains: []string{chain1.String(), chain2.String()},
	}
	reply := &SetTrackedChainsReply{}

	require.NoError(a.SetTrackedChains(nil, args, reply))
	require.Len(reply.TrackedChains, 2)

	// Verify chains are tracked
	tracked := tracker.TrackedChains()
	require.True(tracked.Contains(chain1))
	require.True(tracked.Contains(chain2))
}

func TestSetTrackedChainsInvalidChainID(t *testing.T) {
	require := require.New(t)

	tracker := newMockChainTracker()
	a := &Admin{Config: Config{
		Log:     log.NewNoOpLogger(),
		Network: tracker,
	}}

	args := &SetTrackedChainsArgs{
		Chains: []string{"invalid-chain-id"},
	}
	reply := &SetTrackedChainsReply{}

	err := a.SetTrackedChains(nil, args, reply)
	require.Error(err)
	require.Contains(err.Error(), "invalid chain ID")
}

func TestSetTrackedChainsNoNetwork(t *testing.T) {
	require := require.New(t)

	a := &Admin{Config: Config{
		Log:     log.NewNoOpLogger(),
		Network: nil,
	}}

	args := &SetTrackedChainsArgs{
		Chains: []string{ids.GenerateTestID().String()},
	}
	reply := &SetTrackedChainsReply{}

	err := a.SetTrackedChains(nil, args, reply)
	require.Error(err)
	require.Contains(err.Error(), "network not available")
}

func TestGetTrackedChainsSuccess(t *testing.T) {
	require := require.New(t)

	tracker := newMockChainTracker()
	chain1 := ids.GenerateTestID()
	chain2 := ids.GenerateTestID()
	tracker.trackedChains[chain1] = struct{}{}
	tracker.trackedChains[chain2] = struct{}{}

	a := &Admin{Config: Config{
		Log:     log.NewNoOpLogger(),
		Network: tracker,
	}}

	reply := &GetTrackedChainsReply{}
	require.NoError(a.GetTrackedChains(nil, nil, reply))
	require.Len(reply.TrackedChains, 2)
}

func TestGetTrackedChainsNoNetwork(t *testing.T) {
	require := require.New(t)

	a := &Admin{Config: Config{
		Log:     log.NewNoOpLogger(),
		Network: nil,
	}}

	reply := &GetTrackedChainsReply{}
	err := a.GetTrackedChains(nil, nil, reply)
	require.Error(err)
	require.Contains(err.Error(), "network not available")
}
