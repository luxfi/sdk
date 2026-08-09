// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package primary

import (
	"fmt"

	"github.com/luxfi/proto/p/stakeable"
	lux "github.com/luxfi/utxo"
	"github.com/luxfi/utxo/secp256k1fx"
	"github.com/luxfi/utxo/wire"
)

// zapUTXO reads a ZAP-native UTXO envelope into a UTXO.
//
// The node serves UTXOs as `utxo.UTXO.WireBytes()` — the envelope
// [TypeKind:1][ShapeKind:1][ZAP message] whose reader is wire.WrapUTXO.
// Reads go straight through the zero-copy wire accessors; nothing is
// re-encoded and no codec.Manager is involved on this path.
//
// Dispatch is on the inner (TypeKind, ShapeKind) discriminator, which the
// wire accessors verify — a TransferInput buffer cannot be read as a
// TransferOutput and yield garbage-but-deterministic fields.
func zapUTXO(b []byte) (*lux.UTXO, error) {
	u, err := wire.WrapUTXO(b)
	if err != nil {
		return nil, fmt.Errorf("wrap utxo: %w", err)
	}

	out, err := zapOutput(u.OutputBytes())
	if err != nil {
		return nil, err
	}

	return &lux.UTXO{
		UTXOID: lux.UTXOID{TxID: u.TxID(), OutputIndex: u.OutputIndex()},
		Asset:  lux.Asset{ID: u.AssetID()},
		Out:    out,
	}, nil
}

// zapOutput reads one output envelope, recursing once through a locked
// output to its inner transfer output. Stakeable locks nest exactly one
// level deep — stakeable.LockOut itself rejects nesting — so the recursion
// terminates by construction.
func zapOutput(b []byte) (lux.TransferableOut, error) {
	_, shape, err := wire.PeekDiscriminator(b)
	if err != nil {
		return nil, fmt.Errorf("output discriminator: %w", err)
	}

	switch shape {
	case wire.ShapeKindLockedOutput:
		lo, err := wire.WrapLockedOutput(b)
		if err != nil {
			return nil, fmt.Errorf("wrap locked output: %w", err)
		}
		inner, err := zapOutput(lo.TransferOutBytes())
		if err != nil {
			return nil, err
		}
		return &stakeable.LockOut{Locktime: lo.Locktime(), TransferableOut: inner}, nil

	case wire.ShapeKindTransferOutput:
		to, err := wire.WrapTransferOutput(b)
		if err != nil {
			return nil, fmt.Errorf("wrap transfer output: %w", err)
		}
		return &secp256k1fx.TransferOutput{
			Amt: to.Amount(),
			OutputOwners: secp256k1fx.OutputOwners{
				Locktime:  to.Locktime(),
				Threshold: to.Threshold(),
				Addrs:     to.AddressList().All(),
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported output shape 0x%02x", byte(shape))
	}
}
