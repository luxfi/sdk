// Copyright (C) 2019-2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package interfaces

import (
	"github.com/luxdefi/node/ids"
	"github.com/luxdefi/node/snow/uptime"
	validatorsstateinterfaces "github.com/luxdefi/evm/plugin/evm/validators/state/interfaces"
)

type PausableManager interface {
	uptime.Manager
	validatorsstateinterfaces.StateCallbackListener
	IsPaused(nodeID ids.NodeID) bool
}
