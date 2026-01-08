// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"github.com/luxfi/const"
	"github.com/luxfi/ids"
)

// ChainValidator validates a chain on the Lux network.
type ChainValidator struct {
	Validator `serialize:"true"`

	// ID of the chain this validator is validating
	Chain ids.ID `serialize:"true" json:"chainID"`
}

// ChainID is the ID of the chain this validator is validating
func (v *ChainValidator) ChainID() ids.ID {
	return v.Chain
}

// Verify this validator is valid
func (v *ChainValidator) Verify() error {
	switch v.Chain {
	case constants.PrimaryNetworkID:
		return errBadChainID
	default:
		return v.Validator.Verify()
	}
}
