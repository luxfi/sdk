// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"github.com/luxfi/runtime"
	"github.com/luxfi/ids"
)

const Alias = "X"

type Context struct {
	NetworkID        uint32
	BlockchainID     ids.ID
	XAssetID         ids.ID
	BaseTxFee        uint64
	CreateAssetTxFee uint64
}

func NewConsensusRuntime(
	networkID uint32,
	blockchainID ids.ID,
	xAssetID ids.ID,
) (*runtime.Runtime, error) {
	lookup := ids.NewAliaser()
	rt := &runtime.Runtime{
		NetworkID: networkID,
		ChainID:   blockchainID,
		XChainID:  blockchainID,
		XAssetID:  xAssetID,
	}
	return rt, lookup.Alias(blockchainID, Alias)
}
