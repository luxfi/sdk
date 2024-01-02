// Copyright (C) 2023-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package controller

import (
	"github.com/luxdefi/node/ids"
	"github.com/luxdefi/vmsdk/examples/tokenvm/storage"
)

type StateManager struct{}

func (*StateManager) IncomingWarpKey(sourceChainID ids.ID, msgID ids.ID) []byte {
	return storage.IncomingWarpKeyPrefix(sourceChainID, msgID)
}

func (*StateManager) OutgoingWarpKey(txID ids.ID) []byte {
	return storage.OutgoingWarpKeyPrefix(txID)
}
