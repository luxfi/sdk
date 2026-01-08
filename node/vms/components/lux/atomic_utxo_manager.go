// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package lux

import (
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
)

// AtomicUTXOManager defines the interface for managing atomic UTXOs
type AtomicUTXOManager interface {
	// GetAtomicUTXOs returns the UTXOs controlled by [addrs] from the given [chainID]
	GetAtomicUTXOs(
		chainID ids.ID,
		addrs set.Set[ids.ShortID],
		startAddr ids.ShortID,
		startUTXOID ids.ID,
		limit int,
	) ([]*UTXO, ids.ShortID, ids.ID, error)
}
