// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package exchangevm

import "github.com/luxfi/sdk/utils/json"

// GetTxFeeReply is the response from a GetTxFee call
type GetTxFeeReply struct {
	TxFee            json.Uint64 `json:"txFee"`
	CreateAssetTxFee json.Uint64 `json:"createAssetTxFee"`
}