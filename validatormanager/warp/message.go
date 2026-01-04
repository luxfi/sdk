// Copyright (C) 2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

// Package warp provides warp message types for validator management.
package warp

import (
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
	warpPayload "github.com/luxfi/warp/payload"
)

// PChainOwner represents an owner on the P-Chain
type PChainOwner struct {
	Threshold uint32        `serialize:"true" json:"threshold"`
	Addresses []ids.ShortID `serialize:"true" json:"addresses"`
}

// L1ConversionValidatorData contains validator information for network-to-L1 conversion
// As above, so below - validators are unified across all network layers
type L1ConversionValidatorData struct {
	NodeID       []byte                 `serialize:"true" json:"nodeID"`
	BLSPublicKey [bls.PublicKeyLen]byte `serialize:"true" json:"blsPublicKey"`
	Weight       uint64                 `serialize:"true" json:"weight"`
}

// NetToL1ConversionValidatorData is a backward compatibility alias for L1ConversionValidatorData.
type NetToL1ConversionValidatorData = L1ConversionValidatorData
type SubnetToL1ConversionValidatorData = L1ConversionValidatorData

// L1ConversionData contains the full network-to-L1 conversion payload
type L1ConversionData struct {
	NetID          ids.ID                      `serialize:"true" json:"netID"`
	SubnetID       ids.ID                      `serialize:"true" json:"subnetID"` // Deprecated: use NetID
	ManagerChainID ids.ID                      `serialize:"true" json:"managerChainID"`
	ManagerAddress []byte                      `serialize:"true" json:"managerAddress"`
	Validators     []L1ConversionValidatorData `serialize:"true" json:"validators"`
}

// NetToL1ConversionData is a backward compatibility alias for L1ConversionData.
type NetToL1ConversionData = L1ConversionData
type SubnetToL1ConversionData = L1ConversionData

// L1ConversionID calculates the ID for a network-to-L1 conversion
func L1ConversionID(data L1ConversionData) (ids.ID, error) {
	// TODO: Implement proper hashing of the conversion data
	return ids.GenerateTestID(), nil
}

// NetToL1ConversionID is a backward compatibility alias for L1ConversionID.
func NetToL1ConversionID(data NetToL1ConversionData) (ids.ID, error) {
	return L1ConversionID(data)
}

func SubnetToL1ConversionID(data SubnetToL1ConversionData) (ids.ID, error) {
	return L1ConversionID(data)
}

// NewL1Conversion creates a new network-to-L1 conversion message
func NewL1Conversion(conversionID ids.ID) (*warpPayload.AddressedCall, error) {
	// TODO: Implement proper message creation
	return &warpPayload.AddressedCall{}, nil
}

// NewNetToL1Conversion is a backward compatibility alias for NewL1Conversion.
func NewNetToL1Conversion(conversionID ids.ID) (*warpPayload.AddressedCall, error) {
	return NewL1Conversion(conversionID)
}

func NewSubnetToL1Conversion(conversionID ids.ID) (*warpPayload.AddressedCall, error) {
	return NewL1Conversion(conversionID)
}

// L1ValidatorRegistration represents an L1 validator registration
type L1ValidatorRegistration struct {
	validationID     ids.ID      // private field
	NodeID           ids.NodeID  `serialize:"true" json:"nodeID"`
	BLSPublicKey     []byte      `serialize:"true" json:"blsPublicKey"`
	Weight           uint64      `serialize:"true" json:"weight"`
	Expiry           uint64      `serialize:"true" json:"expiry"`
	RemainingBalance uint64      `serialize:"true" json:"remainingBalance"`
	DisableOwner     PChainOwner `serialize:"true" json:"disableOwner"`
}

// ValidationID returns the validation ID of the registration
func (r *L1ValidatorRegistration) ValidationID() ids.ID {
	return r.validationID
}

// L1ValidatorWeight represents an L1 validator weight update
type L1ValidatorWeight struct {
	ValidationID ids.ID `serialize:"true" json:"validationID"`
	Nonce        uint64 `serialize:"true" json:"nonce"`
	Weight       uint64 `serialize:"true" json:"weight"`
}

// ParseL1ValidatorWeight parses L1 validator weight from payload
func ParseL1ValidatorWeight(payload []byte) (*L1ValidatorWeight, error) {
	// TODO: Implement proper parsing
	return &L1ValidatorWeight{}, nil
}

// ParseRegisterL1Validator parses L1 validator registration from payload
func ParseRegisterL1Validator(payload []byte) (*L1ValidatorRegistration, error) {
	// TODO: Implement proper parsing
	return &L1ValidatorRegistration{}, nil
}

// L1ValidatorRegistrationPayload wraps AddressedCall with ValidationID
type L1ValidatorRegistrationPayload struct {
	*warpPayload.AddressedCall
	validationID ids.ID
}

// ValidationID returns the validation ID
func (p *L1ValidatorRegistrationPayload) ValidationID() ids.ID {
	return p.validationID
}

// NewRegisterL1Validator creates a new L1 validator registration payload
func NewRegisterL1Validator(subnetID ids.ID, nodeID ids.NodeID, blsPublicKey []byte, expiry uint64, balanceOwners PChainOwner, disableOwners PChainOwner, weight uint64) (*L1ValidatorRegistrationPayload, error) {
	// TODO: Implement proper message creation
	// Generate a validation ID based on the parameters
	validationID := ids.GenerateTestID()

	addressedCall := &warpPayload.AddressedCall{}
	return &L1ValidatorRegistrationPayload{
		AddressedCall: addressedCall,
		validationID:  validationID,
	}, nil
}

// NewL1ValidatorRegistration creates a new L1 validator registration message
func NewL1ValidatorRegistration(validationID ids.ID, valid bool) (*warpPayload.AddressedCall, error) {
	// TODO: Implement proper message creation
	return &warpPayload.AddressedCall{}, nil
}

// ParseAddressedCall parses an addressed call from payload
func ParseAddressedCall(payload []byte) (*warpPayload.AddressedCall, error) {
	// TODO: Implement proper parsing
	return &warpPayload.AddressedCall{}, nil
}

// ConvertStandaloneToNodeWarpMessage converts a standalone warp message to a node warp message
func ConvertStandaloneToNodeWarpMessage(standaloneMessage interface{}) (interface{}, error) {
	// TODO: Implement proper conversion
	return standaloneMessage, nil
}

// NewL1ValidatorWeight creates a new L1 validator weight message
func NewL1ValidatorWeight(validationID ids.ID, nonce uint64, weight uint64) (*warpPayload.AddressedCall, error) {
	// TODO: Implement proper message creation
	return &warpPayload.AddressedCall{}, nil
}
