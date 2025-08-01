// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//nolint:unused
package eth

import (
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/evm/core"
)

const blocksToKeep = 604_800 // Approx. 2 weeks worth of blocks assuming 2s block time

type chainWithFinalBlock struct {
	*core.BlockChain
}

// CurrentFinalBlock returns the current block below which blobs should not
// be maintained anymore for reorg purposes.
func (c *chainWithFinalBlock) CurrentFinalBlock() *types.Header {
	lastAccepted := c.LastAcceptedBlock().Header().Number.Uint64()
	if lastAccepted <= blocksToKeep {
		return nil
	}

	return c.GetHeaderByNumber(lastAccepted - blocksToKeep)
}
