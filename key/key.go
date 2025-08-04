// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/crypto"
)

// Key represents a cryptographic key with metadata
type Key struct {
	ID         ids.ID            `json:"id"`
	Type       string            `json:"type"` // "ed25519", "bls", "secp256k1"
	PrivateKey crypto.PrivateKey `json:"-"`
	PublicKey  crypto.PublicKey  `json:"publicKey"`
	Address    ids.ShortID       `json:"address"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// Manager handles key generation, storage, and retrieval
type Manager struct {
	keyDir string
	keys   map[ids.ID]*Key
}

// NewManager creates a new key manager
func NewManager(keyDir string) (*Manager, error) {
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	m := &Manager{
		keyDir: keyDir,
		keys:   make(map[ids.ID]*Key),
	}

	// Load existing keys
	if err := m.loadKeys(); err != nil {
		return nil, fmt.Errorf("failed to load keys: %w", err)
	}

	return m, nil
}

// GenerateEd25519 generates a new Ed25519 key
func (m *Manager) GenerateEd25519() (*Key, error) {
	privateKey, err := crypto.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	key := &Key{
		ID:         ids.GenerateID(),
		Type:       "ed25519",
		PrivateKey: privateKey,
		PublicKey:  privateKey.PublicKey(),
		Address:    privateKey.PublicKey().Address(),
		Metadata:   make(map[string]string),
	}

	m.keys[key.ID] = key
	return key, nil
}

// GenerateBLS generates a new BLS key
func (m *Manager) GenerateBLS() (*Key, *bls.SecretKey, error) {
	blsKey, err := bls.NewSecretKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate BLS key: %w", err)
	}

	// Create a wrapper key
	key := &Key{
		ID:       ids.GenerateID(),
		Type:     "bls",
		Metadata: make(map[string]string),
	}

	// Store BLS public key info in metadata
	pubKey := bls.PublicKeyFromSecretKey(blsKey)
	key.Metadata["blsPublicKey"] = hex.EncodeToString(bls.PublicKeyToBytes(pubKey))

	m.keys[key.ID] = key
	return key, blsKey, nil
}

// ImportPrivateKey imports an existing private key
func (m *Manager) ImportPrivateKey(privateKey crypto.PrivateKey) (*Key, error) {
	key := &Key{
		ID:         ids.GenerateID(),
		Type:       "ed25519",
		PrivateKey: privateKey,
		PublicKey:  privateKey.PublicKey(),
		Address:    privateKey.PublicKey().Address(),
		Metadata:   make(map[string]string),
	}

	m.keys[key.ID] = key
	return key, nil
}

// Get retrieves a key by ID
func (m *Manager) Get(keyID ids.ID) (*Key, error) {
	key, exists := m.keys[keyID]
	if !exists {
		return nil, errors.New("key not found")
	}
	return key, nil
}

// GetByAddress retrieves a key by address
func (m *Manager) GetByAddress(address ids.ShortID) (*Key, error) {
	for _, key := range m.keys {
		if key.Address == address {
			return key, nil
		}
	}
	return nil, errors.New("key not found for address")
}

// List returns all keys
func (m *Manager) List() []*Key {
	keys := make([]*Key, 0, len(m.keys))
	for _, key := range m.keys {
		keys = append(keys, key)
	}
	return keys
}

// Delete removes a key
func (m *Manager) Delete(keyID ids.ID) error {
	if _, exists := m.keys[keyID]; !exists {
		return errors.New("key not found")
	}

	delete(m.keys, keyID)
	
	// Delete from disk
	keyFile := filepath.Join(m.keyDir, fmt.Sprintf("%s.key", keyID))
	if err := os.Remove(keyFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete key file: %w", err)
	}

	return nil
}

// Save persists a key to disk
func (m *Manager) Save(keyID ids.ID) error {
	key, exists := m.keys[keyID]
	if !exists {
		return errors.New("key not found")
	}

	keyFile := filepath.Join(m.keyDir, fmt.Sprintf("%s.key", keyID))
	
	// Serialize key (excluding private key for security)
	data, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal key: %w", err)
	}

	if err := os.WriteFile(keyFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	// Save private key separately (encrypted in production)
	privKeyFile := filepath.Join(m.keyDir, fmt.Sprintf("%s.priv", keyID))
	if key.PrivateKey != nil {
		privKeyData := key.PrivateKey.Bytes()
		if err := os.WriteFile(privKeyFile, privKeyData, 0600); err != nil {
			return fmt.Errorf("failed to write private key file: %w", err)
		}
	}

	return nil
}

// SaveAll persists all keys to disk
func (m *Manager) SaveAll() error {
	for keyID := range m.keys {
		if err := m.Save(keyID); err != nil {
			return err
		}
	}
	return nil
}

// loadKeys loads all keys from disk
func (m *Manager) loadKeys() error {
	entries, err := os.ReadDir(m.keyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No keys yet
		}
		return err
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".key" {
			continue
		}

		keyFile := filepath.Join(m.keyDir, entry.Name())
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return fmt.Errorf("failed to read key file %s: %w", entry.Name(), err)
		}

		var key Key
		if err := json.Unmarshal(data, &key); err != nil {
			return fmt.Errorf("failed to unmarshal key %s: %w", entry.Name(), err)
		}

		// Try to load private key
		privKeyFile := filepath.Join(m.keyDir, fmt.Sprintf("%s.priv", key.ID))
		if privKeyData, err := os.ReadFile(privKeyFile); err == nil && key.Type == "ed25519" {
			// Reconstruct private key (simplified - in production this would be encrypted)
			key.PrivateKey, err = crypto.LoadPrivateKey(privKeyData)
			if err != nil {
				return fmt.Errorf("failed to load private key for %s: %w", key.ID, err)
			}
		}

		m.keys[key.ID] = &key
	}

	return nil
}

// GenerateMnemonic generates a BIP39 mnemonic phrase
func GenerateMnemonic() (string, error) {
	// Generate 256 bits of entropy
	entropy := make([]byte, 32)
	if _, err := rand.Read(entropy); err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}

	// In production, use a proper BIP39 implementation
	// For now, return a hex representation
	return hex.EncodeToString(entropy), nil
}

// DeriveKey derives a key from a mnemonic phrase
func DeriveKey(mnemonic string, index uint32) (*Key, error) {
	// In production, implement proper BIP39/BIP32 derivation
	// For now, use the mnemonic as seed
	seed, err := hex.DecodeString(mnemonic)
	if err != nil {
		return nil, fmt.Errorf("invalid mnemonic: %w", err)
	}

	// Use seed to generate deterministic key
	// This is a simplified version - real implementation would use proper HD derivation
	privateKey, err := crypto.LoadPrivateKey(seed[:32])
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	return &Key{
		ID:         ids.GenerateID(),
		Type:       "ed25519",
		PrivateKey: privateKey,
		PublicKey:  privateKey.PublicKey(),
		Address:    privateKey.PublicKey().Address(),
		Metadata: map[string]string{
			"derivationPath": fmt.Sprintf("m/44'/9000'/0'/0/%d", index),
		},
	}, nil
}

// ExportKey exports a key in various formats
func ExportKey(key *Key, format string, writer io.Writer) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(key)
	case "hex":
		if key.PrivateKey == nil {
			return errors.New("no private key to export")
		}
		_, err := writer.Write([]byte(hex.EncodeToString(key.PrivateKey.Bytes())))
		return err
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}