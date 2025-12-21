// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/ids"
	"github.com/luxfi/constants"
	"github.com/luxfi/node/vms/components/gas"
	"github.com/luxfi/node/vms/platformvm/txs/fee"
)

const Alias = "P"

type Context struct {
	NetworkID         uint32
	ChainID           ids.ID // Optional: if set, overrides default PlatformChainID
	XAssetID          ids.ID
	ComplexityWeights gas.Dimensions
	GasPrice          gas.Price
	StaticFeeConfig   fee.StaticConfig
}

func NewConsensusContext(networkID uint32, luxAssetID ids.ID) (*consensusctx.Context, error) {
	return NewConsensusContextWithChainID(networkID, constants.PlatformChainID, luxAssetID)
}

func NewConsensusContextWithChainID(networkID uint32, chainID ids.ID, luxAssetID ids.ID) (*consensusctx.Context, error) {
	lookup := ids.NewAliaser()
	ctx := &consensusctx.Context{
		NetworkID:   networkID,
		NetID:    constants.PrimaryNetworkID,
		ChainID:     chainID,
		XAssetID: luxAssetID,
	}
	return ctx, lookup.Alias(chainID, Alias)
}
