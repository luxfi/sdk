// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/consensus/validator/uptime"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/sdk/utils/timer/mockable"
	"github.com/luxfi/sdk/node/vms/platformvm/config"
	"github.com/luxfi/sdk/node/vms/platformvm/fx"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
	"github.com/luxfi/sdk/node/vms/platformvm/utxo"
)

type Backend struct {
	Config       *config.Internal
	Ctx          *consensusctx.Context
	Clk          *mockable.Clock
	Fx           fx.Fx
	FlowChecker  utxo.Verifier
	Uptimes      uptime.Calculator
	Rewards      reward.Calculator
	Bootstrapped *utils.Atomic[bool]
	Log          log.Logger
}

// SharedMemory provides cross-chain atomic operations
type SharedMemory interface {
	Get(peerChainID ids.ID, keys [][]byte) ([][]byte, error)
	Apply(requests map[ids.ID]interface{}, batch ...interface{}) error
}
