// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package gossiper

import (
	"context"

	"github.com/luxfi/ids"
	"github.com/luxfi/consensus/engine/common"
)

type Gossiper interface {
	Run(common.AppSender)
	TriggerGossip(context.Context) error // may be triggered by run already
	HandleAppGossip(ctx context.Context, nodeID ids.NodeID, msg []byte) error
	Done() // wait after stop
}
