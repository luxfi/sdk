// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package indexer

import (
	"time"

	apitypes "github.com/luxfi/api/types"
	"github.com/luxfi/formatting"
	"github.com/luxfi/ids"
)

type FormattedContainer struct {
	ID        ids.ID              `json:"id"`
	Bytes     string              `json:"bytes"`
	Timestamp time.Time           `json:"timestamp"`
	Encoding  formatting.Encoding `json:"encoding"`
	Index     apitypes.Uint64     `json:"index"`
}

type GetLastAcceptedArgs struct {
	Encoding formatting.Encoding `json:"encoding"`
}

type GetContainerByIndexArgs struct {
	Index    apitypes.Uint64     `json:"index"`
	Encoding formatting.Encoding `json:"encoding"`
}

type GetContainerRangeArgs struct {
	StartIndex apitypes.Uint64     `json:"startIndex"`
	NumToFetch apitypes.Uint64     `json:"numToFetch"`
	Encoding   formatting.Encoding `json:"encoding"`
}

type GetContainerRangeResponse struct {
	Containers []FormattedContainer `json:"containers"`
}

type GetIndexArgs struct {
	ID ids.ID `json:"id"`
}

type GetIndexResponse struct {
	Index apitypes.Uint64 `json:"index"`
}

type IsAcceptedArgs struct {
	ID ids.ID `json:"id"`
}

type IsAcceptedResponse struct {
	IsAccepted bool `json:"isAccepted"`
}

type GetContainerByIDArgs struct {
	ID       ids.ID              `json:"id"`
	Encoding formatting.Encoding `json:"encoding"`
}
