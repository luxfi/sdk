// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package models

// Validator represents a validator on any network layer (primary, L1, L2, L3)
// As above, so below - validators are unified across all network layers
type Validator struct {
	NodeID string `json:"NodeID"`

	Weight uint64 `json:"Weight"`

	Balance uint64 `json:"Balance"`

	BLSPublicKey string `json:"BLSPublicKey"`

	BLSProofOfPossession string `json:"BLSProofOfPossession"`

	ChangeOwnerAddr string `json:"ChangeOwnerAddr"`

	ValidationID string `json:"ValidationID"`
}

// Backward compatibility aliases
type NetValidator = Validator
type SubnetValidator = Validator
