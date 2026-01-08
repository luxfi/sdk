// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"testing"

	"github.com/cloudflare/circl/kem/mlkem/mlkem768"
	"github.com/luxfi/ids"
	"github.com/stretchr/testify/require"
)

// TestNewTeleportMessage tests creating a new teleport message
func TestNewTeleportMessage(t *testing.T) {
	require := require.New(t)

	sourceChain := ids.GenerateTestID()
	destChain := ids.GenerateTestID()
	payload := []byte("test transfer payload")
	nonce := uint64(12345)

	msg := NewTeleportMessage(TeleportTransfer, sourceChain, destChain, nonce, payload)

	require.Equal(TeleportVersion, msg.Version)
	require.Equal(TeleportTransfer, msg.MessageType)
	require.Equal(sourceChain, msg.SourceChainID)
	require.Equal(destChain, msg.DestChainID)
	require.Equal(nonce, msg.Nonce)
	require.Equal(payload, msg.Payload)
	require.False(msg.Encrypted)
}

// TestTeleportMessageValidate tests message validation
func TestTeleportMessageValidate(t *testing.T) {
	require := require.New(t)

	tests := []struct {
		name        string
		msg         *TeleportMessage
		expectError error
	}{
		{
			name: "valid message",
			msg: &TeleportMessage{
				Version:       TeleportVersion,
				MessageType:   TeleportTransfer,
				SourceChainID: ids.GenerateTestID(),
				DestChainID:   ids.GenerateTestID(),
				Nonce:         1,
				Payload:       []byte("payload"),
			},
			expectError: nil,
		},
		{
			name: "invalid version",
			msg: &TeleportMessage{
				Version:       99, // Wrong version
				MessageType:   TeleportTransfer,
				SourceChainID: ids.GenerateTestID(),
				DestChainID:   ids.GenerateTestID(),
				Nonce:         1,
				Payload:       []byte("payload"),
			},
			expectError: ErrInvalidTeleportVersion,
		},
		{
			name: "invalid message type",
			msg: &TeleportMessage{
				Version:       TeleportVersion,
				MessageType:   99, // Invalid type
				SourceChainID: ids.GenerateTestID(),
				DestChainID:   ids.GenerateTestID(),
				Nonce:         1,
				Payload:       []byte("payload"),
			},
			expectError: ErrInvalidTeleportType,
		},
		{
			name: "empty payload",
			msg: &TeleportMessage{
				Version:       TeleportVersion,
				MessageType:   TeleportTransfer,
				SourceChainID: ids.GenerateTestID(),
				DestChainID:   ids.GenerateTestID(),
				Nonce:         1,
				Payload:       []byte{}, // Empty
			},
			expectError: ErrMissingPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if tt.expectError != nil {
				require.ErrorIs(err, tt.expectError)
			} else {
				require.NoError(err)
			}
		})
	}
}

// TestTeleportMessageToWarpMessage tests conversion to Warp message
func TestTeleportMessageToWarpMessage(t *testing.T) {
	require := require.New(t)

	sourceChain := ids.GenerateTestID()
	destChain := ids.GenerateTestID()
	payload := []byte("test payload for warp")
	networkID := uint32(96369)

	teleport := NewTeleportMessage(TeleportLock, sourceChain, destChain, 100, payload)

	warpMsg, err := teleport.ToWarpMessage(networkID)
	require.NoError(err)
	require.NotNil(warpMsg)
	require.Equal(networkID, warpMsg.NetworkID)
	require.Equal(sourceChain, warpMsg.SourceChainID)
	require.NotEmpty(warpMsg.Payload)
}

// TestNewPrivateTeleportMessage tests encrypted message creation
func TestNewPrivateTeleportMessage(t *testing.T) {
	require := require.New(t)

	sourceChain := ids.GenerateTestID()
	destChain := ids.GenerateTestID()
	payload := []byte("confidential cross-chain data")
	nonce := uint64(42)

	// Generate real ML-KEM-768 key pair
	scheme := mlkem768.Scheme()
	pubKey, _, err := scheme.GenerateKeyPair()
	require.NoError(err)
	recipientPubKey, err := pubKey.MarshalBinary()
	require.NoError(err)
	recipientKeyID := []byte("recipient-key-123")

	msg, err := NewPrivateTeleportMessage(sourceChain, destChain, nonce, payload, recipientPubKey, recipientKeyID)
	require.NoError(err)
	require.NotNil(msg)

	require.Equal(TeleportVersion, msg.Version)
	require.Equal(TeleportPrivate, msg.MessageType)
	require.True(msg.Encrypted)
	require.NotEmpty(msg.Payload)
	require.NotEqual(payload, msg.Payload) // Should be encrypted
}

// TestPrivateTeleportMessageDecrypt tests decryption of private messages
func TestPrivateTeleportMessageDecrypt(t *testing.T) {
	require := require.New(t)

	sourceChain := ids.GenerateTestID()
	destChain := ids.GenerateTestID()
	originalPayload := []byte("secret message for cross-chain transfer")
	nonce := uint64(999)

	// Generate real ML-KEM-768 key pair
	scheme := mlkem768.Scheme()
	pubKey, privKey, err := scheme.GenerateKeyPair()
	require.NoError(err)
	recipientPubKey, err := pubKey.MarshalBinary()
	require.NoError(err)
	recipientPrivKey, err := privKey.MarshalBinary()
	require.NoError(err)
	recipientKeyID := []byte("test-key")

	// Create encrypted message
	msg, err := NewPrivateTeleportMessage(sourceChain, destChain, nonce, originalPayload, recipientPubKey, recipientKeyID)
	require.NoError(err)
	require.True(msg.Encrypted)

	// Decrypt
	decrypted, err := msg.DecryptPayload(recipientPrivKey)
	require.NoError(err)
	require.Equal(originalPayload, decrypted)
}

// TestTeleportMessageString tests string representation
func TestTeleportMessageString(t *testing.T) {
	require := require.New(t)

	msg := NewTeleportMessage(
		TeleportSwap,
		ids.GenerateTestID(),
		ids.GenerateTestID(),
		123,
		[]byte("swap data"),
	)

	str := msg.String()
	require.Contains(str, "Teleport")
	require.Contains(str, "Swap")
	require.Contains(str, "nonce=123")
}

// TestTeleportMessageCodecRoundTrip tests serialization
func TestTeleportMessageCodecRoundTrip(t *testing.T) {
	require := require.New(t)

	original := &TeleportMessage{
		Version:       TeleportVersion,
		MessageType:   TeleportGovernance,
		SourceChainID: ids.GenerateTestID(),
		DestChainID:   ids.GenerateTestID(),
		Nonce:         12345,
		Payload:       []byte("governance vote payload"),
		Encrypted:     false,
	}

	// Encode
	encoded, err := Codec.Marshal(CodecVersion, original)
	require.NoError(err)

	// Decode
	decoded := &TeleportMessage{}
	_, err = Codec.Unmarshal(encoded, decoded)
	require.NoError(err)

	// Verify
	require.Equal(original.Version, decoded.Version)
	require.Equal(original.MessageType, decoded.MessageType)
	require.Equal(original.SourceChainID, decoded.SourceChainID)
	require.Equal(original.DestChainID, decoded.DestChainID)
	require.Equal(original.Nonce, decoded.Nonce)
	require.Equal(original.Payload, decoded.Payload)
	require.Equal(original.Encrypted, decoded.Encrypted)
}

// TestTeleportTransferPayload tests transfer payload handling
func TestTeleportTransferPayload(t *testing.T) {
	require := require.New(t)

	assetID := ids.GenerateTestID()
	amount := uint64(1000000)
	sender := []byte("0x1234567890abcdef")
	recipient := []byte("0xfedcba0987654321")
	fee := uint64(100)
	memo := []byte("test transfer")

	payload := NewTransferPayload(assetID, amount, sender, recipient, fee, memo)

	require.Equal(assetID, payload.AssetID)
	require.Equal(amount, payload.Amount)
	require.Equal(sender, payload.Sender)
	require.Equal(recipient, payload.Recipient)
	require.Equal(fee, payload.Fee)
	require.Equal(memo, payload.Memo)

	// Test serialization
	encoded, err := payload.Bytes()
	require.NoError(err)

	// Test parsing
	parsed, err := ParseTransferPayload(encoded)
	require.NoError(err)
	require.Equal(payload.AssetID, parsed.AssetID)
	require.Equal(payload.Amount, parsed.Amount)
	require.Equal(payload.Sender, parsed.Sender)
	require.Equal(payload.Recipient, parsed.Recipient)
	require.Equal(payload.Fee, parsed.Fee)
	require.Equal(payload.Memo, parsed.Memo)
}

// TestTeleportAttestPayload tests attestation payload handling
func TestTeleportAttestPayload(t *testing.T) {
	require := require.New(t)

	payload := &TeleportAttestPayload{
		AttestationType: 1,
		Timestamp:       1234567890,
		Data:            []byte("price: 100.50 USD"),
		AttesterID:      ids.GenerateTestNodeID(),
	}

	// Encode
	encoded, err := Codec.Marshal(CodecVersion, payload)
	require.NoError(err)

	// Decode
	decoded := &TeleportAttestPayload{}
	_, err = Codec.Unmarshal(encoded, decoded)
	require.NoError(err)

	// Verify
	require.Equal(payload.AttestationType, decoded.AttestationType)
	require.Equal(payload.Timestamp, decoded.Timestamp)
	require.Equal(payload.Data, decoded.Data)
	require.Equal(payload.AttesterID, decoded.AttesterID)
}

// TestSignatureType tests signature type utilities
func TestSignatureType(t *testing.T) {
	require := require.New(t)

	// Test recommended type
	recommended := RecommendedSignatureType()
	require.Equal(SigTypeRingtail, recommended)

	// Test quantum safety
	require.False(SigTypeBLS.IsQuantumSafe())
	require.True(SigTypeRingtail.IsQuantumSafe())
	require.True(SigTypeHybrid.IsQuantumSafe())

	// Test string representation
	require.Equal("BLS", SigTypeBLS.String())
	require.Equal("Ringtail", SigTypeRingtail.String())
	require.Equal("Hybrid", SigTypeHybrid.String())
}

// TestTeleportTypes tests all teleport types
func TestTeleportTypes(t *testing.T) {
	require := require.New(t)

	// Verify constants are sequential
	require.Equal(TeleportType(0), TeleportTransfer)
	require.Equal(TeleportType(1), TeleportSwap)
	require.Equal(TeleportType(2), TeleportLock)
	require.Equal(TeleportType(3), TeleportUnlock)
	require.Equal(TeleportType(4), TeleportAttest)
	require.Equal(TeleportType(5), TeleportGovernance)
	require.Equal(TeleportType(6), TeleportPrivate)
}

// TestTeleportMessageAllTypes tests creating messages of all types
func TestTeleportMessageAllTypes(t *testing.T) {
	require := require.New(t)

	types := []TeleportType{
		TeleportTransfer,
		TeleportSwap,
		TeleportLock,
		TeleportUnlock,
		TeleportAttest,
		TeleportGovernance,
	}

	for _, tt := range types {
		msg := NewTeleportMessage(
			tt,
			ids.GenerateTestID(),
			ids.GenerateTestID(),
			uint64(tt), // Use type as nonce
			[]byte("payload"),
		)

		err := msg.Validate()
		require.NoError(err, "type %d should be valid", tt)
	}
}

// TestTeleportVersion tests version constant
func TestTeleportVersion(t *testing.T) {
	require := require.New(t)
	require.Equal(uint8(1), TeleportVersion)
}
