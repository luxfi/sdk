// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utxotest

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/const"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/vms/components/lux"
	"github.com/luxfi/sdk/wallet/primary/common"
)

func NewDeterministicChainUTXOs(t *testing.T, utxoSets map[ids.ID][]*lux.UTXO) *DeterministicChainUTXOs {
	globalUTXOs := common.NewUTXOs()
	for netID, utxos := range utxoSets {
		for _, utxo := range utxos {
			require.NoError(
				t, globalUTXOs.AddUTXO(context.Background(), netID, constants.PlatformChainID, utxo),
			)
		}
	}
	return &DeterministicChainUTXOs{
		ChainUTXOs: common.NewChainUTXOs(constants.PlatformChainID, globalUTXOs),
	}
}

type DeterministicChainUTXOs struct {
	common.ChainUTXOs
}

func (c *DeterministicChainUTXOs) UTXOs(ctx context.Context, sourceChainID ids.ID) ([]*lux.UTXO, error) {
	utxos, err := c.ChainUTXOs.UTXOs(ctx, sourceChainID)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(utxos, func(a, b *lux.UTXO) int {
		return a.Compare(&b.UTXOID)
	})
	return utxos, nil
}
