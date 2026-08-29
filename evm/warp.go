// Copyright (C) 2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"fmt"

	evmWarp "github.com/luxfi/evm/precompile/contracts/warp"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/warp"
)

// GetWarpMessagesFromLogs gets all unsigned warp messages contained in [logs].
func GetWarpMessagesFromLogs(
	logs []*types.Log,
) []*warp.Message {
	messages := []*warp.Message{}
	for _, txLog := range logs {
		msg, err := evmWarp.UnpackSendWarpEventDataToMessage(txLog.Data)
		if err == nil {
			messages = append(messages, msg)
		}
	}
	return messages
}

// ExtractWarpMessageFromLogs gets first unsigned warp message contained in [logs].
func ExtractWarpMessageFromLogs(
	logs []*types.Log,
) (*warp.Message, error) {
	messages := GetWarpMessagesFromLogs(logs)
	if len(messages) == 0 {
		return nil, fmt.Errorf("no warp message is present in evm logs")
	}
	return messages[0], nil
}

// ExtractWarpMessageFromReceipt gets first unsigned warp message contained in [receipt].
func ExtractWarpMessageFromReceipt(
	receipt *types.Receipt,
) (*warp.Message, error) {
	if receipt == nil {
		return nil, fmt.Errorf("empty receipt was given")
	}
	return ExtractWarpMessageFromLogs(receipt.Logs)
}
