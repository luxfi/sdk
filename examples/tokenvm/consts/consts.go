// Copyright (C) 2023-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package consts

import (
	"github.com/luxdefi/node/ids"
	"github.com/luxdefi/node/vms/platformvm/warp"
	"github.com/luxdefi/vmsdk/chain"
	"github.com/luxdefi/vmsdk/codec"
	"github.com/luxdefi/vmsdk/consts"
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
