// Copyright (C) 2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

// Package txs provides transaction types for validator management.
package txs

import (
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
)

// ConvertToL1Validator contains validator information for network-to-L1 conversion
// As above, so below - validators are unified across all network layers
type ConvertToL1Validator struct {
	NodeID ids.NodeID `serialize:"true" json:"nodeID"`
	Weight uint64     `serialize:"true" json:"weight"`
	Signer Signer     `serialize:"true" json:"signer"`
}

// Backward compatibility aliases
type ConvertNetToL1Validator = ConvertToL1Validator
type ConvertNetworkToL1Validator = ConvertToL1Validator

// Signer contains the BLS signature for a validator
type Signer struct {
	PublicKey [bls.PublicKeyLen]byte `serialize:"true" json:"publicKey"`
	Signature [bls.SignatureLen]byte `serialize:"true" json:"signature"`
}
