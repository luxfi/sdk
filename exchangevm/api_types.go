// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package exchangevm

import (
	"github.com/luxfi/codec/jsonrpc"
	"github.com/luxfi/consensus/core/choices"
	"github.com/luxfi/formatting"
)

// GetTxFeeReply is the response from a GetTxFee call
type GetTxFeeReply struct {
	TxFee            json.Uint64 `json:"txFee"`
	CreateAssetTxFee json.Uint64 `json:"createAssetTxFee"`
}

// GetTxStatusReply is the response from a GetTxStatus call
type GetTxStatusReply struct {
	Status choices.Status `json:"status"`
}

// BuildGenesisArgs are arguments for BuildGenesis
type BuildGenesisArgs struct {
	Encoding    formatting.Encoding `json:"encoding"`
	GenesisData map[string]AssetDefinition
}

// AssetDefinition defines an asset in genesis
type AssetDefinition struct {
	Name         string                   `json:"name"`
	Symbol       string                   `json:"symbol"`
	Denomination byte                     `json:"denomination"`
	InitialState map[string][]interface{} `json:"initialState"`
}

// BuildGenesisReply is the reply from BuildGenesis
type BuildGenesisReply struct {
	Bytes    string              `json:"bytes"`
	Encoding formatting.Encoding `json:"encoding"`
}

// GetAssetDescriptionArgs are arguments for GetAssetDescription
type GetAssetDescriptionArgs struct {
	AssetID string `json:"assetID"`
}

// GetAssetDescriptionReply is the reply from GetAssetDescription
type GetAssetDescriptionReply struct {
	AssetID      string `json:"assetID"`
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	Denomination byte   `json:"denomination"`
}

// GetBalanceArgs are arguments for GetBalance
type GetBalanceArgs struct {
	Address        string `json:"address"`
	AssetID        string `json:"assetID"`
	IncludePartial bool   `json:"includePartial"`
}

// GetBalanceReply is the reply from GetBalance
type GetBalanceReply struct {
	Balance json.Uint64 `json:"balance"`
	UTXOIDs []string    `json:"utxoIDs"`
}

// Balance represents an asset balance
type Balance struct {
	Asset   string      `json:"asset"`
	Balance json.Uint64 `json:"balance"`
}

// GetAllBalancesArgs are arguments for GetAllBalances
type GetAllBalancesArgs struct {
	JSONAddress    interface{} `json:"address"`
	IncludePartial bool        `json:"includePartial"`
}

// GetAllBalancesReply is the reply from GetAllBalances
type GetAllBalancesReply struct {
	Balances []Balance `json:"balances"`
}
