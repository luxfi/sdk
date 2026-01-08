// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"errors"
	"fmt"

	"github.com/luxfi/ids"
)

// TeleportVersion is the current version of the Teleport protocol
const TeleportVersion uint8 = 1

// TeleportType represents the type of cross-chain operation
type TeleportType uint8

const (
	// TeleportTransfer is a standard asset transfer between chains
	TeleportTransfer TeleportType = iota
	// TeleportSwap is an atomic swap between chains
	TeleportSwap
	// TeleportLock locks assets on the source chain
	TeleportLock
	// TeleportUnlock unlocks assets on the destination chain
	TeleportUnlock
	// TeleportAttest is an attestation message (oracle data, price feeds)
	TeleportAttest
	// TeleportGovernance is a cross-chain governance message
	TeleportGovernance
	// TeleportPrivate is an encrypted private transfer
	TeleportPrivate
)

var (
	ErrInvalidTeleportVersion = errors.New("invalid teleport version")
	ErrInvalidTeleportType    = errors.New("invalid teleport type")
	ErrMissingPayload         = errors.New("teleport message missing payload")
	ErrDecryptRequired        = errors.New("payload is encrypted but no decryption key provided")
)

// TeleportMessage wraps a Warp message for cross-chain bridging operations.
// This is the high-level abstraction for Teleport cross-chain messaging.
//
// Warp 1.5 supports three signature types:
// - BitSetSignature: Classical BLS (legacy)
// - RingtailSignature: Quantum-safe (recommended)
// - HybridBLSRTSignature: BLS+RT hybrid (deprecated)
type TeleportMessage struct {
	// Version is the Teleport protocol version
	Version uint8 `serialize:"true"`

	// MessageType indicates the type of cross-chain operation
	MessageType TeleportType `serialize:"true"`

	// SourceChainID identifies the source blockchain
	SourceChainID ids.ID `serialize:"true"`

	// DestChainID identifies the destination blockchain
	DestChainID ids.ID `serialize:"true"`

	// Nonce prevents replay attacks
	Nonce uint64 `serialize:"true"`

	// Payload contains the application-specific message data
	// For TeleportPrivate, this should be an EncryptedWarpPayload
	Payload []byte `serialize:"true"`

	// Encrypted indicates whether the payload is encrypted
	Encrypted bool `serialize:"true"`
}

// NewTeleportMessage creates a new Teleport message for cross-chain communication
func NewTeleportMessage(
	messageType TeleportType,
	sourceChainID ids.ID,
	destChainID ids.ID,
	nonce uint64,
	payload []byte,
) *TeleportMessage {
	return &TeleportMessage{
		Version:       TeleportVersion,
		MessageType:   messageType,
		SourceChainID: sourceChainID,
		DestChainID:   destChainID,
		Nonce:         nonce,
		Payload:       payload,
		Encrypted:     false,
	}
}

// NewPrivateTeleportMessage creates an encrypted Teleport message
func NewPrivateTeleportMessage(
	sourceChainID ids.ID,
	destChainID ids.ID,
	nonce uint64,
	payload []byte,
	recipientPubKey []byte,
	recipientKeyID []byte,
) (*TeleportMessage, error) {
	// Encrypt the payload using ML-KEM + AES-256-GCM
	encrypted, err := EncryptWarpPayload(payload, recipientPubKey, recipientKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt teleport payload: %w", err)
	}

	// Serialize the encrypted payload
	encryptedBytes, err := Codec.Marshal(CodecVersion, encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize encrypted payload: %w", err)
	}

	return &TeleportMessage{
		Version:       TeleportVersion,
		MessageType:   TeleportPrivate,
		SourceChainID: sourceChainID,
		DestChainID:   destChainID,
		Nonce:         nonce,
		Payload:       encryptedBytes,
		Encrypted:     true,
	}, nil
}

// ToWarpMessage converts a TeleportMessage to an UnsignedMessage for signing
func (t *TeleportMessage) ToWarpMessage(networkID uint32) (*UnsignedMessage, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}

	// Serialize the TeleportMessage as the Warp payload
	teleportPayload, err := Codec.Marshal(CodecVersion, t)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize teleport message: %w", err)
	}

	return NewUnsignedMessage(networkID, t.SourceChainID, teleportPayload)
}

// Validate checks if the TeleportMessage is well-formed
func (t *TeleportMessage) Validate() error {
	if t.Version != TeleportVersion {
		return fmt.Errorf("%w: got %d, expected %d", ErrInvalidTeleportVersion, t.Version, TeleportVersion)
	}

	if t.MessageType > TeleportPrivate {
		return fmt.Errorf("%w: %d", ErrInvalidTeleportType, t.MessageType)
	}

	if len(t.Payload) == 0 {
		return ErrMissingPayload
	}

	return nil
}

// DecryptPayload decrypts an encrypted TeleportMessage payload
func (t *TeleportMessage) DecryptPayload(recipientPrivKey []byte) ([]byte, error) {
	if !t.Encrypted {
		return t.Payload, nil
	}

	// Deserialize the encrypted payload
	encrypted := &EncryptedWarpPayload{}
	_, err := Codec.Unmarshal(t.Payload, encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize encrypted payload: %w", err)
	}

	// Decrypt using ML-KEM
	plaintext, err := encrypted.Decrypt(recipientPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt teleport payload: %w", err)
	}

	return plaintext, nil
}

// String returns a human-readable representation of the TeleportMessage
func (t *TeleportMessage) String() string {
	typeStr := "Unknown"
	switch t.MessageType {
	case TeleportTransfer:
		typeStr = "Transfer"
	case TeleportSwap:
		typeStr = "Swap"
	case TeleportLock:
		typeStr = "Lock"
	case TeleportUnlock:
		typeStr = "Unlock"
	case TeleportAttest:
		typeStr = "Attest"
	case TeleportGovernance:
		typeStr = "Governance"
	case TeleportPrivate:
		typeStr = "Private"
	}

	encryptedStr := ""
	if t.Encrypted {
		encryptedStr = " [encrypted]"
	}

	return fmt.Sprintf("Teleport{v%d %s%s: %s -> %s, nonce=%d, payload=%d bytes}",
		t.Version, typeStr, encryptedStr,
		t.SourceChainID.String()[:8], t.DestChainID.String()[:8],
		t.Nonce, len(t.Payload))
}

// TeleportTransferPayload represents a cross-chain asset transfer
type TeleportTransferPayload struct {
	// AssetID is the asset being transferred
	AssetID ids.ID `serialize:"true"`

	// Amount is the quantity being transferred
	Amount uint64 `serialize:"true"`

	// Sender is the source address (chain-specific encoding)
	Sender []byte `serialize:"true"`

	// Recipient is the destination address (chain-specific encoding)
	Recipient []byte `serialize:"true"`

	// Fee paid for the bridge operation
	Fee uint64 `serialize:"true"`

	// Memo is optional metadata
	Memo []byte `serialize:"true"`
}

// NewTransferPayload creates a new transfer payload for cross-chain asset transfers
func NewTransferPayload(
	assetID ids.ID,
	amount uint64,
	sender []byte,
	recipient []byte,
	fee uint64,
	memo []byte,
) *TeleportTransferPayload {
	return &TeleportTransferPayload{
		AssetID:   assetID,
		Amount:    amount,
		Sender:    sender,
		Recipient: recipient,
		Fee:       fee,
		Memo:      memo,
	}
}

// Bytes serializes the transfer payload
func (p *TeleportTransferPayload) Bytes() ([]byte, error) {
	return Codec.Marshal(CodecVersion, p)
}

// ParseTransferPayload deserializes a transfer payload
func ParseTransferPayload(data []byte) (*TeleportTransferPayload, error) {
	payload := &TeleportTransferPayload{}
	_, err := Codec.Unmarshal(data, payload)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

// TeleportAttestPayload represents an attestation (oracle data)
type TeleportAttestPayload struct {
	// AttestationType identifies what is being attested
	AttestationType uint8 `serialize:"true"`

	// Timestamp of the attestation
	Timestamp uint64 `serialize:"true"`

	// Data is the attestation payload (e.g., price feed, compute result)
	Data []byte `serialize:"true"`

	// AttesterID identifies who created the attestation
	AttesterID ids.NodeID `serialize:"true"`
}

// Warp 1.5 Signature Selection
// ============================

// SignatureType indicates which signature algorithm to use
type SignatureType uint8

const (
	// SigTypeBLS uses classical BLS signatures (Warp 1.0 compatibility)
	SigTypeBLS SignatureType = iota
	// SigTypeRingtail uses quantum-safe Ringtail signatures (recommended)
	SigTypeRingtail
	// SigTypeHybrid uses BLS+Ringtail hybrid (deprecated)
	SigTypeHybrid
)

// RecommendedSignatureType returns the recommended signature type for Warp 1.5
// This is Ringtail (quantum-safe) by default
func RecommendedSignatureType() SignatureType {
	return SigTypeRingtail
}

// IsQuantumSafe returns whether the signature type provides post-quantum security
func (s SignatureType) IsQuantumSafe() bool {
	return s == SigTypeRingtail || s == SigTypeHybrid
}

// String returns the name of the signature type
func (s SignatureType) String() string {
	switch s {
	case SigTypeBLS:
		return "BLS"
	case SigTypeRingtail:
		return "Ringtail"
	case SigTypeHybrid:
		return "Hybrid"
	default:
		return "Unknown"
	}
}
