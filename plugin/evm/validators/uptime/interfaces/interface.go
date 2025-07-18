// Copyright (C) 2019-2024, Hanzo Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package interfaces

import (
	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/snow/uptime"
	validatorsstateinterfaces "github.com/luxfi/evm/plugin/evm/validators/state/interfaces"
)

type PausableManager interface {
	uptime.Manager
	validatorsstateinterfaces.StateCallbackListener
	IsPaused(nodeID ids.NodeID) bool
}
