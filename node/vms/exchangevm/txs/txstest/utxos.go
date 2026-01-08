// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txstest

import (
	"context"
	"fmt"

	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/chains/atomic"
	"github.com/luxfi/sdk/node/codec"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/exchangevm/state"
	"github.com/luxfi/sdk/node/wallet/chain/x/builder"
	"github.com/luxfi/sdk/node/wallet/chain/x/signer"
)

const maxPageSize uint64 = 1024

var (
	_ builder.Backend = (*walletUTXOsAdapter)(nil)
	_ signer.Backend  = (*walletUTXOsAdapter)(nil)
)

func newUTXOs(
	ctx context.Context,
	state state.State,
	sharedMemory atomic.SharedMemory,
	codec codec.Manager,
) *utxos {
	// Use empty chain ID - caller should set this if needed
	chainID := ids.Empty
	return &utxos{
		xchainID:     chainID,
		state:        state,
		sharedMemory: sharedMemory,
		codec:        codec,
	}
}

// SetChainID updates the chain ID for UTXO lookups
func (u *utxos) SetChainID(chainID ids.ID) {
	u.xchainID = chainID
}

type utxos struct {
	xchainID     ids.ID
	state        state.State
	sharedMemory atomic.SharedMemory
	codec        codec.Manager
}

func (u *utxos) UTXOs(addrs set.Set[ids.ShortID], sourceChainID ids.ID) ([]*lux.UTXO, error) {
	if sourceChainID == u.xchainID {
		return lux.GetAllUTXOs(u.state, addrs)
	}

	atomicUTXOs, _, _, err := lux.GetAtomicUTXOs(
		u.sharedMemory,
		u.codec,
		sourceChainID,
		addrs,
		ids.ShortEmpty,
		ids.Empty,
		int(maxPageSize),
	)
	return atomicUTXOs, err
}

func (u *utxos) GetUTXO(addrs set.Set[ids.ShortID], chainID, utxoID ids.ID) (*lux.UTXO, error) {
	if chainID == u.xchainID {
		return u.state.GetUTXO(utxoID)
	}

	atomicUTXOs, _, _, err := lux.GetAtomicUTXOs(
		u.sharedMemory,
		u.codec,
		chainID,
		addrs,
		ids.ShortEmpty,
		ids.Empty,
		int(maxPageSize),
	)
	if err != nil {
		return nil, fmt.Errorf("problem retrieving atomic UTXOs: %w", err)
	}
	for _, utxo := range atomicUTXOs {
		if utxo.InputID() == utxoID {
			return utxo, nil
		}
	}
	return nil, database.ErrNotFound
}

type walletUTXOsAdapter struct {
	utxos *utxos
	addrs set.Set[ids.ShortID]
}

func (w *walletUTXOsAdapter) UTXOs(_ context.Context, sourceChainID ids.ID) ([]*lux.UTXO, error) {
	return w.utxos.UTXOs(w.addrs, sourceChainID)
}

func (w *walletUTXOsAdapter) GetUTXO(_ context.Context, chainID, utxoID ids.ID) (*lux.UTXO, error) {
	return w.utxos.GetUTXO(w.addrs, chainID, utxoID)
}
