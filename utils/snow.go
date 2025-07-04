// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"github.com/luxdefi/node/api/metrics"
	"github.com/luxdefi/node/ids"
	"github.com/luxdefi/node/snow"
	"github.com/luxdefi/node/utils/crypto/bls"
	"github.com/luxdefi/node/utils/logging"
)

func TestSnowContext() *snow.Context {
	sk, err := localsigner.New()
	if err != nil {
		panic(err)
	}
	pk := sk.PublicKey()
	networkID := constants.UnitTestID
	chainID := testChainID

	ctx := &snow.Context{
		NetworkID:       networkID,
		SubnetID:        ids.Empty,
		ChainID:         chainID,
		NodeID:          ids.GenerateTestNodeID(),
		XChainID:        testXChainID,
		CChainID:        testCChainID,
		NetworkUpgrades: upgradetest.GetConfig(upgradetest.Latest),
		PublicKey:       pk,
		WarpSigner:      warp.NewSigner(sk, networkID, chainID),
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

func NewTestValidatorState() *validatorstest.State {
	return &validatorstest.State{
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
