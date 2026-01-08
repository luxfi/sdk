// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package api

import (
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/utils/json"
	"github.com/luxfi/sdk/node/vms/types"
)

// APIL1Validator is the representation of a L1 validator sent over APIs.
type APIL1Validator struct {
	NodeID    ids.NodeID  `json:"nodeID"`
	Weight    json.Uint64 `json:"weight"`
	StartTime json.Uint64 `json:"startTime"`
	BaseL1Validator
}

// BaseL1Validator is the representation of a base L1 validator without the common parts with a staker.
type BaseL1Validator struct {
	ValidationID *ids.ID `json:"validationID,omitempty"`
	// PublicKey is the compressed BLS public key of the validator
	PublicKey             *types.JSONByteSlice `json:"publicKey,omitempty"`
	RemainingBalanceOwner *Owner               `json:"remainingBalanceOwner,omitempty"`
	DeactivationOwner     *Owner               `json:"deactivationOwner,omitempty"`
	MinNonce              *json.Uint64         `json:"minNonce,omitempty"`
	// Balance is the remaining amount of LUX this L1 validator has for paying
	// the continuous fee, according to the last accepted state. If the
	// validator is inactive, the balance will be 0.
	Balance *json.Uint64 `json:"balance,omitempty"`
}
