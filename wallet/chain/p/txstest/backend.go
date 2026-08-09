// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txstest

import (
	"context"

	"github.com/luxfi/constants"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/proto/p/fx"
	"github.com/luxfi/proto/p/state"
	"github.com/luxfi/sdk/wallet/chain/p/builder"
	"github.com/luxfi/sdk/wallet/chain/p/signer"
	lux "github.com/luxfi/utxo"
)

var (
	_ builder.Backend = (*Backend)(nil)
	_ signer.Backend  = (*Backend)(nil)
)

func newBackend(
	addrs set.Set[ids.ShortID],
	state state.State,
) *Backend {
	return &Backend{
		addrs: addrs,
		state: state,
	}
}

type Backend struct {
	addrs set.Set[ids.ShortID]
	state state.State
}

func (b *Backend) UTXOs(_ context.Context, sourceChainID ids.ID) ([]*lux.UTXO, error) {
	// For test purposes, only return platform chain UTXOs
	if sourceChainID == constants.PlatformChainID {
		return lux.GetAllUTXOs(b.state, b.addrs)
	}
	// Return empty for cross-chain UTXOs in tests
	return nil, nil
}

func (b *Backend) GetUTXO(_ context.Context, chainID, utxoID ids.ID) (*lux.UTXO, error) {
	if chainID == constants.PlatformChainID {
		return b.state.GetUTXO(utxoID)
	}
	// Return nil for cross-chain UTXOs in tests
	return nil, nil
}

func (b *Backend) GetOwner(_ context.Context, ownerID ids.ID) (fx.Owner, error) {
	// For test purposes, treat ownerID as chain ID
	return b.state.GetNetOwner(ownerID)
}

func (b *Backend) GetNetOwner(_ context.Context, netID ids.ID) (fx.Owner, error) {
	return b.state.GetNetOwner(netID)
}
