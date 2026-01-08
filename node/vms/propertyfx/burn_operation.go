// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package propertyfx

import (
	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/sdk/node/vms/components/verify"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

type BurnOperation struct {
	secp256k1fx.Input `serialize:"true"`
}

func (*BurnOperation) InitCtx(*consensusctx.Context) {}

// InitializeContext implements the fxs.FxOperation interface
func (*BurnOperation) InitializeContext(*consensusctx.Context) error {
	return nil
}

func (*BurnOperation) Outs() []verify.State {
	return nil
}
