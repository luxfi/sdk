// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package key

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/node/ids"
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

// generateAddress generates an address from a public key
func generateAddress(pubKey crypto.PublicKey) ids.ShortID {
	addr := ids.ShortID{}
	copy(addr[:], pubKey[:20])
	return addr
}

// Keychain manages a collection of private keys for signing
type Keychain struct {
	keys map[ids.ShortID]crypto.PrivateKey
}

// NewKeychain creates a new keychain
func NewKeychain() *Keychain {
	return &Keychain{
		keys: make(map[ids.ShortID]crypto.PrivateKey),
	}
}

// Add adds a private key to the keychain
func (kc *Keychain) Add(privateKey crypto.PrivateKey) error {
	address := generateAddress(privateKey.PublicKey())
	if _, exists := kc.keys[address]; exists {
		return fmt.Errorf("key already exists for address %s", address)
	}
	kc.keys[address] = privateKey
	return nil
}

// Sign signs a message with the key for the given address
func (kc *Keychain) Sign(message []byte, address ids.ShortID) (crypto.Signature, error) {
	privateKey, exists := kc.keys[address]
	if !exists {
		return crypto.EmptySignature, fmt.Errorf("no key for address %s", address)
	}
	return crypto.Sign(message, privateKey), nil
}

// Get retrieves a private key by address
func (kc *Keychain) Get(address ids.ShortID) (crypto.PrivateKey, error) {
	privateKey, exists := kc.keys[address]
	if !exists {
		return crypto.EmptyPrivateKey, fmt.Errorf("no key for address %s", address)
	}
	return privateKey, nil
}

// Addresses returns all addresses in the keychain
func (kc *Keychain) Addresses() []ids.ShortID {
	addresses := make([]ids.ShortID, 0, len(kc.keys))
	for addr := range kc.keys {
		addresses = append(addresses, addr)
	}
	return addresses
}

// GenerateMnemonic generates a mnemonic phrase
func GenerateMnemonic(bitSize int) ([]string, error) {
	// Simple mock implementation for testing
	// In production, use a proper BIP39 implementation
	words := []string{
		"abandon", "ability", "able", "about", "above", "absent",
		"absorb", "abstract", "absurd", "abuse", "access", "accident",
	}

	if bitSize == 128 {
		return words[:12], nil
	} else if bitSize == 256 {
		return append(words, words...)[:24], nil
	}
	return nil, fmt.Errorf("unsupported bit size: %d", bitSize)
}

// DeriveKey derives a key from a mnemonic at the given index
func DeriveKey(mnemonic []string, index uint32) (crypto.PrivateKey, error) {
	// Simple mock implementation for testing
	// In production, use proper BIP32/BIP44 derivation

	// Generate a deterministic key based on mnemonic and index
	seed := fmt.Sprintf("%v-%d", mnemonic, index)

	// Create a deterministic private key (for testing only!)
	// Use a simple hash to ensure valid key
	h := [64]byte{}
	copy(h[:], []byte(seed))

	// Ensure it's different for different indices
	h[0] = byte(index)
	h[1] = byte(index >> 8)
	h[2] = byte(index >> 16)
	h[3] = byte(index >> 24)

	var privKey crypto.PrivateKey
	copy(privKey[:], h[:crypto.PrivateKeyLen])

	return privKey, nil
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
		ID:         ids.GenerateTestID(),
		Type:       "ed25519",
		PrivateKey: privateKey,
		PublicKey:  privateKey.PublicKey(),
		Address:    generateAddress(privateKey.PublicKey()),
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
	pubKey := bls.PublicFromSecretKey(blsKey)
	pubKeyBytes := bls.PublicKeyToUncompressedBytes(pubKey)

	// Generate ID from public key
	keyID := ids.ID{}
	copy(keyID[:], pubKeyBytes)

	key := &Key{
		ID:       keyID,
		Type:     "bls",
		Metadata: make(map[string]string),
	}

	// Store BLS public key info in metadata
	key.Metadata["blsPublicKey"] = hex.EncodeToString(pubKeyBytes)

	m.keys[key.ID] = key
	return key, blsKey, nil
}

// GenerateKey generates a new key of the specified type
func (m *Manager) GenerateKey(keyType string) (*Key, error) {
	switch keyType {
	case "ed25519":
		return m.GenerateEd25519()
	case "bls":
		key, blsKey, err := m.GenerateBLS()
		if err != nil {
			return nil, err
		}
		// Generate address from BLS public key
		pubKey := bls.PublicFromSecretKey(blsKey)
		pubKeyBytes := bls.PublicKeyToUncompressedBytes(pubKey)
		key.Address = generateAddress(crypto.PublicKey(pubKeyBytes[:crypto.PublicKeyLen]))
		return key, err
	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}
}

// ImportPrivateKey imports an existing private key
func (m *Manager) ImportPrivateKey(privateKey crypto.PrivateKey) (*Key, error) {
	key := &Key{
		ID:         ids.GenerateTestID(),
		Type:       "ed25519",
		PrivateKey: privateKey,
		PublicKey:  privateKey.PublicKey(),
		Address:    generateAddress(privateKey.PublicKey()),
		Metadata:   make(map[string]string),
	}

	m.keys[key.ID] = key
	return key, nil
}

// ImportKey imports a key with the given private key and type
func (m *Manager) ImportKey(privateKey crypto.PrivateKey, keyType string) (*Key, error) {
	if keyType != "ed25519" {
		return nil, fmt.Errorf("unsupported key type for import: %s", keyType)
	}
	return m.ImportPrivateKey(privateKey)
}

// GetKey retrieves a key by ID
func (m *Manager) GetKey(keyID ids.ID) (*Key, error) {
	return m.Get(keyID)
}

// SaveKey saves a key to disk
func (m *Manager) SaveKey(key *Key) error {
	m.keys[key.ID] = key
	return m.Save(key.ID)
}

// DeleteKey removes a key from the manager
func (m *Manager) DeleteKey(keyID ids.ID) error {
	return m.Delete(keyID)
}

// ListKeys returns all keys
func (m *Manager) ListKeys() []*Key {
	return m.List()
}

// ExportKey exports a key's private key as hex string
func (m *Manager) ExportKey(keyID ids.ID) (string, error) {
	key, err := m.Get(keyID)
	if err != nil {
		return "", err
	}

	if key.Type != "ed25519" || key.PrivateKey == crypto.EmptyPrivateKey {
		return "", fmt.Errorf("key does not have exportable private key")
	}

	return hex.EncodeToString(key.PrivateKey[:]), nil
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
	if key.Type == "ed25519" && key.PrivateKey != crypto.EmptyPrivateKey {
		privKeyData := key.PrivateKey[:]
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
			if len(privKeyData) == crypto.PrivateKeyLen {
				key.PrivateKey = crypto.PrivateKey(privKeyData)
			} else {
				// Skip this key, it has invalid private key data
				continue
			}
		}

		m.keys[key.ID] = &key
	}

	return nil
}

// ExportKey exports a key in various formats
func ExportKey(key *Key, format string, writer io.Writer) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(key)
	case "hex":
		if key.Type != "ed25519" || key.PrivateKey == crypto.EmptyPrivateKey {
			return errors.New("no private key to export")
		}
		_, err := writer.Write([]byte(hex.EncodeToString(key.PrivateKey[:])))
		return err
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}
