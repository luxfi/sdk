// Copyright (C) 2023-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"context"

	"github.com/luxdefi/node/snow/engine/common"
	"github.com/luxdefi/node/utils/logging"
	"github.com/luxdefi/vmsdk/chain"
)

type VM interface {
	StopChan() chan struct{}
	EngineChan() chan<- common.Message
	PreferredBlock(context.Context) (*chain.StatelessBlock, error)
	Logger() logging.Logger
	Mempool() chain.Mempool
}
