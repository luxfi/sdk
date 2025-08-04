// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package consts

import (
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/chain"
	"github.com/luxfi/sdk/codec"
	"github.com/luxfi/sdk/consts"
	"github.com/luxfi/vms/platformvm/warp"
)

const (
	HRP    = "token"
	Name   = "tokenvm"
	Symbol = "TKN"
)

var ID ids.ID

func init() {
	b := make([]byte, consts.IDLen)
	copy(b, []byte(Name))
	vmID, err := ids.ToID(b)
	if err != nil {
		panic(err)
	}
	ID = vmID
}

// Instantiate registry here so it can be imported by any package. We set these
// values in [controller/registry].
var (
	ActionRegistry *codec.TypeParser[chain.Action, *warp.Message, bool]
	AuthRegistry   *codec.TypeParser[chain.Auth, *warp.Message, bool]
)
