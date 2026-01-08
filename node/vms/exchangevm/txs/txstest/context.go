// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package txstest

import (
	"context"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/vms/exchangevm/config"
	"github.com/luxfi/sdk/node/wallet/chain/x/builder"
)

func newContext(
	ctx context.Context,
	cfg *config.Config,
	feeAssetID ids.ID,
) *builder.Context {
	// Use default values - these should be set by caller if needed
	networkID := uint32(1) // Default to mainnet
	chainID := ids.Empty   // Caller should set this

	return &builder.Context{
		NetworkID:        networkID,
		BlockchainID:     chainID,
		XAssetID:       feeAssetID,
		BaseTxFee:        cfg.TxFee,
		CreateAssetTxFee: cfg.CreateAssetTxFee,
	}
}
