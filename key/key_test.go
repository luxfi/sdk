// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/crypto"
)

func TestManager_GenerateKey(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	tests := []struct {
		name    string
		keyType string
		wantErr bool
	}{
		{
			name:    "generate ed25519 key",
			keyType: "ed25519",
			wantErr: false,
		},
		{
			name:    "generate bls key",
			keyType: "bls",
			wantErr: false,
		},
		{
			name:    "invalid key type",
			keyType: "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := manager.GenerateKey(tt.keyType)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, key)
			require.Equal(t, tt.keyType, key.Type)
			require.NotEqual(t, ids.Empty, key.ID)
			require.NotNil(t, key.PrivateKey)
			require.NotNil(t, key.PublicKey)
			require.NotEqual(t, ids.ShortEmpty, key.Address)
		})
	}
}

func TestManager_SaveAndLoadKey(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// Generate a key
	key, err := manager.GenerateKey("ed25519")
	require.NoError(t, err)

	// Save the key
	err = manager.SaveKey(key)
	require.NoError(t, err)

	// Verify key file exists
	keyFile := filepath.Join(tmpDir, key.ID.String()+".key")
	_, err = os.Stat(keyFile)
	require.NoError(t, err)

	// Create new manager to test loading
	manager2, err := NewManager(tmpDir)
	require.NoError(t, err)

	// Get the loaded key
	loadedKey, err := manager2.GetKey(key.ID)
	require.NoError(t, err)
	require.Equal(t, key.ID, loadedKey.ID)
	require.Equal(t, key.Type, loadedKey.Type)
	require.Equal(t, key.Address, loadedKey.Address)
}

func TestManager_ImportKey(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// Generate a private key
	priv, err := crypto.GeneratePrivateKey()
	require.NoError(t, err)

	// Import the key
	key, err := manager.ImportKey(priv, "ed25519")
	require.NoError(t, err)
	require.NotNil(t, key)
	require.Equal(t, priv, key.PrivateKey)
}

func TestManager_ExportKey(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// Generate a key
	key, err := manager.GenerateKey("ed25519")
	require.NoError(t, err)

	// Export the key
	exported, err := manager.ExportKey(key.ID)
	require.NoError(t, err)
	require.NotEmpty(t, exported)
}

func TestManager_DeleteKey(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// Generate a key
	key, err := manager.GenerateKey("ed25519")
	require.NoError(t, err)

	// Delete the key
	err = manager.DeleteKey(key.ID)
	require.NoError(t, err)

	// Verify key is deleted
	_, err = manager.GetKey(key.ID)
	require.Error(t, err)
}

func TestManager_ListKeys(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// Generate multiple keys
	key1, err := manager.GenerateKey("ed25519")
	require.NoError(t, err)

	key2, err := manager.GenerateKey("bls")
	require.NoError(t, err)

	// List keys
	keys := manager.ListKeys()
	require.Len(t, keys, 2)

	// Verify keys are in the list
	keyIDs := make(map[ids.ID]bool)
	for _, k := range keys {
		keyIDs[k.ID] = true
	}
	require.True(t, keyIDs[key1.ID])
	require.True(t, keyIDs[key2.ID])
}

func TestGenerateMnemonic(t *testing.T) {
	// Test 12-word mnemonic
	mnemonic12, err := GenerateMnemonic(128)
	require.NoError(t, err)
	require.Len(t, mnemonic12, 12)

	// Test 24-word mnemonic
	mnemonic24, err := GenerateMnemonic(256)
	require.NoError(t, err)
	require.Len(t, mnemonic24, 24)
}

func TestDeriveKey(t *testing.T) {
	mnemonic, err := GenerateMnemonic(128)
	require.NoError(t, err)

	// Derive keys at different indices
	key1, err := DeriveKey(mnemonic, 0)
	require.NoError(t, err)

	key2, err := DeriveKey(mnemonic, 1)
	require.NoError(t, err)

	// Keys should be different
	require.NotEqual(t, key1, key2)

	// Same index should produce same key
	key1Again, err := DeriveKey(mnemonic, 0)
	require.NoError(t, err)
	require.Equal(t, key1, key1Again)
}

func TestKeychain_Sign(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// Generate a key
	key, err := manager.GenerateKey("ed25519")
	require.NoError(t, err)

	// Create keychain
	keychain := NewKeychain()
	keychain.Add(key.PrivateKey)

	// Sign a message
	message := []byte("test message")
	sig, err := keychain.Sign(message, key.Address)
	require.NoError(t, err)
	require.NotNil(t, sig)

	// Verify signature
	valid := crypto.Verify(message, key.PublicKey, sig)
	require.True(t, valid)
}

func TestKeychain_Addresses(t *testing.T) {
	keychain := NewKeychain()

	// Add multiple keys
	for i := 0; i < 3; i++ {
		priv, err := crypto.GeneratePrivateKey()
		require.NoError(t, err)
		keychain.Add(priv)
	}

	// Get addresses
	addresses := keychain.Addresses()
	require.Len(t, addresses, 3)

	// All addresses should be unique
	seen := make(map[ids.ShortID]bool)
	for _, addr := range addresses {
		require.False(t, seen[addr])
		seen[addr] = true
	}
}
