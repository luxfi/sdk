// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txstest

import (
	"time"

	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/vms/components/gas"
	"github.com/luxfi/sdk/node/vms/platformvm/config"
	"github.com/luxfi/sdk/node/vms/platformvm/txs/fee"
	"github.com/luxfi/sdk/node/wallet/chain/p/builder"
)

func newContext(
	ctx *consensusctx.Context,
	networkID uint32,
	luxAssetID ids.ID,
	cfg *config.Config,
	internalCfg *config.Internal,
	timestamp time.Time,
) *builder.Context {
	builderContext := &builder.Context{
		NetworkID: networkID,
		ChainID:   ctx.ChainID,
		XAssetID:  luxAssetID,
	}

	// For test purposes, populate the fee configuration
	// If dynamic fees are configured, use those; otherwise use static fees
	if internalCfg != nil && internalCfg.DynamicFeeConfig.Weights != (gas.Dimensions{}) {
		// Use dynamic fee configuration
		builderContext.ComplexityWeights = internalCfg.DynamicFeeConfig.Weights
		builderContext.GasPrice = internalCfg.DynamicFeeConfig.MinPrice
	}
	
	// Always populate static fees as fallback or for non-dynamic transactions
	if cfg != nil {
		builderContext.StaticFeeConfig = fee.StaticConfig{
			TxFee:                 cfg.TxFee,
			CreateAssetTxFee:      cfg.CreateAssetTxFee,
			CreateSubnetTxFee:     cfg.CreateNetTxFee,
			CreateBlockchainTxFee: cfg.CreateBlockchainTxFee,
		}
	}

	return builderContext
}
