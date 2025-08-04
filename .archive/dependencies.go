// Copyright (C) 2023-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"context"

	"github.com/luxfi/consensus/engine/common"
	"github.com/luxfi/log"
	"github.com/luxfi/sdk/chain"
)

type VM interface {
	StopChan() chan struct{}
	EngineChan() chan<- common.Message
	PreferredBlock(context.Context) (*chain.StatelessBlock, error)
	Logger() logging.Logger
	Mempool() chain.Mempool
}
