// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/utils/constants"
)

const Alias = "X"

type Context struct {
	NetworkID        uint32
	BlockchainID     ids.ID
	XAssetID       ids.ID
	BaseTxFee        uint64
	CreateAssetTxFee uint64
}

func NewConsensusContext(
	networkID uint32,
	blockchainID ids.ID,
	luxAssetID ids.ID,
) (*consensusctx.Context, error) {
	lookup := ids.NewAliaser()
	ctx := &consensusctx.Context{
		NetworkID:   networkID,
		NetID:    constants.PrimaryNetworkID,
		ChainID:     blockchainID,
		XChainID:    blockchainID,
		XAssetID: luxAssetID,
	}
	return ctx, lookup.Alias(blockchainID, Alias)
}
