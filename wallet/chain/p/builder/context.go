// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"github.com/luxfi/constants"
	"github.com/luxfi/ids"
	"github.com/luxfi/protocol/p/txs/fee"
	"github.com/luxfi/runtime"
	"github.com/luxfi/vm/components/gas"
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

func NewConsensusRuntime(networkID uint32, xAssetID ids.ID) (*runtime.Runtime, error) {
	return NewConsensusRuntimeWithChainID(networkID, constants.PlatformChainID, xAssetID)
}

func NewConsensusRuntimeWithChainID(networkID uint32, chainID ids.ID, xAssetID ids.ID) (*runtime.Runtime, error) {
	lookup := ids.NewAliaser()
	rt := &runtime.Runtime{
		NetworkID: networkID,
		ChainID:   chainID,
		XAssetID:  xAssetID,
	}
	return rt, lookup.Alias(chainID, Alias)
}
