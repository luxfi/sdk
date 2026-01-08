// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"context"

	validators "github.com/luxfi/consensus/validator"
	"github.com/luxfi/sdk/node/vms/platformvm/block"
	txexecutor "github.com/luxfi/sdk/node/vms/platformvm/txs/executor"
)

// VerifyWarpMessages verifies all warp messages in the block. If any of the
// warp messages are invalid, an error is returned.
func VerifyWarpMessages(
	ctx context.Context,
	networkID uint32,
	validatorState validators.State,
	pChainHeight uint64,
	b block.Block,
) error {
	for _, tx := range b.Txs() {
		err := txexecutor.VerifyWarpMessages(
			ctx,
			networkID,
			validatorState,
			pChainHeight,
			tx.Unsigned,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
