// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package interfaces

import (
	"github.com/luxfi/consensus/uptime"
	validatorsstateinterfaces "github.com/luxfi/evm/plugin/evm/validators/state/interfaces"
	"github.com/luxfi/ids"
)

type PausableManager interface {
	uptime.Manager
	validatorsstateinterfaces.StateCallbackListener
	IsPaused(nodeID ids.NodeID) bool
}
