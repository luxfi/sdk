// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"context"
	"errors"

	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/utils/constants"
	"github.com/luxfi/node/utils/logging"
	"github.com/luxfi/node/vms/platformvm/warp"
   "github.com/luxfi/node/consensus"
	"github.com/luxfi/node/api/metrics"
   "github.com/luxfi/node/consensus/validators"
	"github.com/luxfi/node/upgrade"
	"github.com/luxfi/evm/localsigner"
	"github.com/luxfi/evm/interfaces"
)

var (
	testChainID  = ids.ID{5, 4, 3, 2, 1}
	testCChainID = ids.ID{1, 2, 3, 4, 5}
	testXChainID = ids.ID{2, 3, 4, 5, 6}
)

func TestConsensusContext() *consensus.Context {
	signer, err := localsigner.New()
	if err != nil {
		panic(err)
	}
	pk := signer.PublicKey()
	networkID := constants.UnitTestID
	chainID := testChainID

	ctx := &consensus.Context{
		NetworkID:       networkID,
		SubnetID:        ids.Empty,
		ChainID:         chainID,
		NodeID:          ids.GenerateTestNodeID(),
		XChainID:        testXChainID,
		CChainID:        testCChainID,
		NetworkUpgrades: upgrade.Config{},
		PublicKey:       pk,
		WarpSigner:      warp.NewSigner(signer.Inner(), networkID, chainID),
		Log:             logging.NoLog{},
		BCLookup:        ids.NewAliaser(),
		Metrics:         metrics.NewPrefixGatherer(),
		ChainDataDir:    "",
		ValidatorState:  NewTestValidatorState(),
	}

	aliaser := ctx.BCLookup.(ids.Aliaser)
	_ = aliaser.Alias(testCChainID, "C")
	_ = aliaser.Alias(testCChainID, testCChainID.String())
	_ = aliaser.Alias(testXChainID, "X")
	_ = aliaser.Alias(testXChainID, testXChainID.String())

	return ctx
}

// TestChainContext returns a test interfaces.ChainContext
func TestChainContext() *interfaces.ChainContext {
	return &interfaces.ChainContext{
		NetworkID: constants.UnitTestID,
		SubnetID:  interfaces.SubnetID{},
		ChainID:   interfaces.ChainID{5, 4, 3, 2, 1},
		NodeID:    interfaces.GenerateTestNodeID(),
	}
}

// TestValidatorState is a test implementation of validator state
type TestValidatorState struct {
	GetMinimumHeightF       func(context.Context) (uint64, error)
	GetCurrentHeightF       func(context.Context) (uint64, error)
	GetSubnetIDF            func(context.Context, ids.ID) (ids.ID, error)
	GetValidatorSetF        func(context.Context, uint64, ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error)
	GetCurrentValidatorSetF func(context.Context, ids.ID) (map[ids.ID]*validators.GetCurrentValidatorOutput, uint64, error)
}

func NewTestValidatorState() *TestValidatorState {
	return &TestValidatorState{
		GetMinimumHeightF: func(context.Context) (uint64, error) {
			return 0, nil
		},
		GetCurrentHeightF: func(context.Context) (uint64, error) {
			return 0, nil
		},
		GetSubnetIDF: func(_ context.Context, chainID ids.ID) (ids.ID, error) {
			subnetID, ok := map[ids.ID]ids.ID{
				constants.PlatformChainID: constants.PrimaryNetworkID,
				testXChainID:              constants.PrimaryNetworkID,
				testCChainID:              constants.PrimaryNetworkID,
			}[chainID]
			if !ok {
				return ids.Empty, errors.New("unknown chain")
			}
			return subnetID, nil
		},
		GetValidatorSetF: func(context.Context, uint64, ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
			return map[ids.NodeID]*validators.GetValidatorOutput{}, nil
		},
		GetCurrentValidatorSetF: func(context.Context, ids.ID) (map[ids.ID]*validators.GetCurrentValidatorOutput, uint64, error) {
			return map[ids.ID]*validators.GetCurrentValidatorOutput{}, 0, nil
		},
	}
}

func (tvs *TestValidatorState) GetMinimumHeight(ctx context.Context) (uint64, error) {
	if tvs.GetMinimumHeightF != nil {
		return tvs.GetMinimumHeightF(ctx)
	}
	return 0, nil
}

func (tvs *TestValidatorState) GetCurrentHeight(ctx context.Context) (uint64, error) {
	if tvs.GetCurrentHeightF != nil {
		return tvs.GetCurrentHeightF(ctx)
	}
	return 0, nil
}

func (tvs *TestValidatorState) GetSubnetID(ctx context.Context, chainID ids.ID) (ids.ID, error) {
	if tvs.GetSubnetIDF != nil {
		return tvs.GetSubnetIDF(ctx, chainID)
	}
	return ids.Empty, nil
}

func (tvs *TestValidatorState) GetValidatorSet(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	if tvs.GetValidatorSetF != nil {
		return tvs.GetValidatorSetF(ctx, height, subnetID)
	}
	return nil, nil
}

func (tvs *TestValidatorState) GetCurrentValidatorSet(ctx context.Context, subnetID ids.ID) (map[ids.ID]*validators.GetCurrentValidatorOutput, uint64, error) {
	if tvs.GetCurrentValidatorSetF != nil {
		return tvs.GetCurrentValidatorSetF(ctx, subnetID)
	}
	return nil, 0, nil
}
