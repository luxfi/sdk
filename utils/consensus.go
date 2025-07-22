// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"context"
	"errors"

	"github.com/luxfi/node/api/metrics"
	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/consensus"
	"github.com/luxfi/node/consensus/validators"
	"github.com/luxfi/node/consensus/validators/validatorstest"
	"github.com/luxfi/node/upgrade/upgradetest"
	"github.com/luxfi/node/utils/constants"
	"github.com/luxfi/node/utils/crypto/bls/signer/localsigner"
	"github.com/luxfi/node/utils/logging"
	"github.com/luxfi/node/vms/platformvm/warp"
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
		NetworkUpgrades: upgradetest.GetConfig(upgradetest.Latest),
		PublicKey:       pk,
		WarpSigner:      warp.NewSigner(signer, networkID, chainID),
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
	}
}
