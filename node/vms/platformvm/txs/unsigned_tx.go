// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

// ContextInitializable defines the interface for initializing context
type ContextInitializable interface {
	InitCtx(ctx *consensusctx.Context)
}

// UnsignedTx is an unsigned transaction
type UnsignedTx interface {
	// TODO: Remove this initialization pattern from both the platformvm and the
	// avm.
	ContextInitializable
	secp256k1fx.UnsignedTx
	SetBytes(unsignedBytes []byte)

	// InputIDs returns the set of inputs this transaction consumes
	InputIDs() set.Set[ids.ID]

	Outputs() []*lux.TransferableOutput

	// Attempts to verify this transaction without any provided state.
	SyntacticVerify(ctx *consensusctx.Context) error

	// Visit calls [visitor] with this transaction's concrete type
	Visit(visitor Visitor) error
}
