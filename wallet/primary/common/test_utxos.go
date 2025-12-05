// Copyright (C) 2019-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

//go:build test

package common

import (
	"context"
	"slices"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/node/utils/constants"
	"github.com/luxfi/node/vms/components/lux"
)

func NewDeterministicChainUTXOs(require *require.Assertions, utxoSets map[ids.ID][]*lux.UTXO) *DeterministicChainUTXOs {
	globalUTXOs := NewUTXOs()
	for netID, utxos := range utxoSets {
		for _, utxo := range utxos {
			require.NoError(
				globalUTXOs.AddUTXO(context.Background(), netID, constants.PlatformChainID, utxo),
			)
		}
	}
	return &DeterministicChainUTXOs{
		ChainUTXOs: NewChainUTXOs(constants.PlatformChainID, globalUTXOs),
	}
}

type DeterministicChainUTXOs struct {
	ChainUTXOs
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
