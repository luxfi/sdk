// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"context"
	"errors"

	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/constants"
)

var (
	testChainID  = interfaces.ID{5, 4, 3, 2, 1}
	testCChainID = interfaces.ID{1, 2, 3, 4, 5}
	testXChainID = interfaces.ID{2, 3, 4, 5, 6}
)

func TestConsensusContext() *interfaces.ChainContext {
	signer, err := localsigner.New()
	if err != nil {
		panic(err)
	}
	pk := signer.PublicKey()
	networkID := constants.UnitTestID
	chainID := testChainID

	ctx := &interfaces.ChainContext{
		NetworkID:       networkID,
		SubnetID:        interfaces.EmptyID,
		ChainID:         chainID,
		NodeID:          interfaces.GenerateTestNodeID(),
		XChainID:        testXChainID,
		CChainID:        testCChainID,
		NetworkUpgrades: interfaces.GetConfig(interfaces.Latest),
		PublicKey:       pk,
		WarpSigner:      interfaces.NewSigner(signer, networkID, chainID),
		Log:             logging.NoLog{},
		BCLookup:        ids.NewAliaser(),
		Metrics:         interfaces.NewPrefixGatherer(),
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

// TestValidatorState is a test implementation of validator state
type TestValidatorState struct {
	GetCurrentHeightF func(context.Context) (uint64, error)
	GetSubnetIDF      func(context.Context, interfaces.ID) (interfaces.ID, error)
	GetValidatorSetF  func(context.Context, uint64, interfaces.ID) (map[interfaces.NodeID]*interfaces.GetValidatorOutput, error)
}

func NewTestValidatorState() *TestValidatorState {
	return &TestValidatorState{
		GetCurrentHeightF: func(context.Context) (uint64, error) {
			return 0, nil
		},
		GetSubnetIDF: func(_ context.Context, chainID interfaces.ID) (interfaces.ID, error) {
			subnetID, ok := map[interfaces.ID]interfaces.ID{
				constants.PlatformChainID: constants.PrimaryNetworkID,
				testXChainID:              constants.PrimaryNetworkID,
				testCChainID:              constants.PrimaryNetworkID,
			}[chainID]
			if !ok {
				return interfaces.EmptyID, errors.New("unknown chain")
			}
			return subnetID, nil
		},
		GetValidatorSetF: func(context.Context, uint64, interfaces.ID) (map[interfaces.NodeID]*interfaces.GetValidatorOutput, error) {
			return map[interfaces.NodeID]*interfaces.GetValidatorOutput{}, nil
		},
	}
}
