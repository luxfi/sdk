// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package keyutil provides utilities for loading private keys that integrate
// with the Lux CLI key management system (~/.lux/keys/).
package keyutil

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/go-bip39"
)

const (
	// LuxKeysDir is the CLI key storage directory
	LuxKeysDir = ".lux/keys"
	// MnemonicFile is the mnemonic filename used by CLI
	MnemonicFile = "mnemonic.txt"
	// ECPrivateKeyFile is the EC private key filename used by CLI
	ECPrivateKeyFile = "ec/private.key"
)

// MustLoadKey loads a secp256k1 private key from (in order of priority):
// 1. LUX_MNEMONIC environment variable (BIP39 mnemonic phrase)
// 2. LUX_PRIVATE_KEY environment variable (hex-encoded)
// 3. LUX_KEY_NAME environment variable (key name from ~/.lux/keys/<name>/)
// 4. Key name provided as first command line argument
// 5. ~/.lux/keys/default/ if it exists
//
// Panics with a helpful message if no key is provided.
func MustLoadKey() *secp256k1.PrivateKey {
	key, err := LoadKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading private key: %s\n", err)
		fmt.Fprintf(os.Stderr, "\nUsage (in order of priority):\n")
		fmt.Fprintf(os.Stderr, "  1. Set LUX_MNEMONIC env var (BIP39 mnemonic)\n")
		fmt.Fprintf(os.Stderr, "  2. Set LUX_PRIVATE_KEY env var (hex-encoded)\n")
		fmt.Fprintf(os.Stderr, "  3. Set LUX_KEY_NAME env var (key from ~/.lux/keys/)\n")
		fmt.Fprintf(os.Stderr, "  4. Pass key name as argument: %s <key-name>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nAvailable keys in ~/.lux/keys/:\n")
		listAvailableKeys()
		fmt.Fprintf(os.Stderr, "\nCreate keys with: lux key create <name>\n")
		os.Exit(1)
	}
	return key
}

// LoadKey attempts to load a private key using the priority order above.
func LoadKey() (*secp256k1.PrivateKey, error) {
	// 1. Try LUX_MNEMONIC environment variable
	if mnemonic := os.Getenv("LUX_MNEMONIC"); mnemonic != "" {
		return keyFromMnemonic(mnemonic)
	}

	// 2. Try LUX_PRIVATE_KEY environment variable
	if keyStr := os.Getenv("LUX_PRIVATE_KEY"); keyStr != "" {
		return parseHexKey(keyStr)
	}

	// 3. Try LUX_KEY_NAME environment variable
	if keyName := os.Getenv("LUX_KEY_NAME"); keyName != "" {
		return LoadKeyByName(keyName)
	}

	// 4. Try key name from command line arguments
	if len(os.Args) > 1 {
		keyName := os.Args[1]
		// Check if it's a key name (exists in ~/.lux/keys/)
		if key, err := LoadKeyByName(keyName); err == nil {
			return key, nil
		}
		// Try as a file path
		if data, err := os.ReadFile(keyName); err == nil {
			return parseKeyData(data)
		}
	}

	// 5. Try default key
	if key, err := LoadKeyByName("default"); err == nil {
		return key, nil
	}

	return nil, fmt.Errorf("no private key provided")
}

// LoadKeyByName loads a key from ~/.lux/keys/<name>/
// It first tries the EC private key file, then falls back to mnemonic
func LoadKeyByName(name string) (*secp256k1.PrivateKey, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	baseDir := filepath.Join(home, LuxKeysDir, name)

	// Try EC private key first (faster, no derivation needed)
	ecKeyPath := filepath.Join(baseDir, ECPrivateKeyFile)
	if data, err := os.ReadFile(ecKeyPath); err == nil {
		return parseHexKey(string(data))
	}

	// Fall back to mnemonic
	mnemonicPath := filepath.Join(baseDir, MnemonicFile)
	if data, err := os.ReadFile(mnemonicPath); err == nil {
		return keyFromMnemonic(string(data))
	}

	// Try as legacy .pk file (older format)
	pkPath := filepath.Join(home, LuxKeysDir, name+".pk")
	if data, err := os.ReadFile(pkPath); err == nil {
		return parseHexKey(string(data))
	}

	return nil, fmt.Errorf("key %q not found in ~/.lux/keys/", name)
}

// keyFromMnemonic derives a private key from a BIP39 mnemonic phrase.
func keyFromMnemonic(mnemonic string) (*secp256k1.PrivateKey, error) {
	mnemonic = strings.TrimSpace(mnemonic)
	words := strings.Fields(mnemonic)
	if len(words) < 12 {
		return nil, fmt.Errorf("invalid mnemonic: need at least 12 words, got %d", len(words))
	}

	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid BIP39 mnemonic")
	}

	// Generate seed from mnemonic (no passphrase)
	seed := bip39.NewSeed(mnemonic, "")

	// Use first 32 bytes of seed as private key
	// Note: For full BIP44 derivation, use the CLI's key package
	if len(seed) < 32 {
		return nil, fmt.Errorf("seed too short")
	}

	return secp256k1.ToPrivateKey(seed[:32])
}

// parseKeyData tries to parse key data as hex or mnemonic
func parseKeyData(data []byte) (*secp256k1.PrivateKey, error) {
	s := strings.TrimSpace(string(data))

	// If it looks like a mnemonic (contains spaces, multiple words)
	if strings.Contains(s, " ") && len(strings.Fields(s)) >= 12 {
		return keyFromMnemonic(s)
	}

	// Try as hex
	return parseHexKey(s)
}

// parseHexKey parses a hex-encoded private key string.
func parseHexKey(s string) (*secp256k1.PrivateKey, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")

	keyBytes, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key: %w", err)
	}

	if len(keyBytes) != secp256k1.PrivateKeyLen {
		return nil, fmt.Errorf("invalid key length: got %d bytes, want %d", len(keyBytes), secp256k1.PrivateKeyLen)
	}

	return secp256k1.ToPrivateKey(keyBytes)
}

// listAvailableKeys prints available keys in ~/.lux/keys/
func listAvailableKeys() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	keysDir := filepath.Join(home, LuxKeysDir)
	entries, err := os.ReadDir(keysDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  (could not read keys directory)\n")
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Check if it has mnemonic or ec key
			baseDir := filepath.Join(keysDir, entry.Name())
			hasMnemonic := fileExists(filepath.Join(baseDir, MnemonicFile))
			hasEC := fileExists(filepath.Join(baseDir, ECPrivateKeyFile))
			if hasMnemonic || hasEC {
				fmt.Fprintf(os.Stderr, "  - %s\n", entry.Name())
			}
		} else if strings.HasSuffix(entry.Name(), ".pk") {
			// Legacy .pk file
			name := strings.TrimSuffix(entry.Name(), ".pk")
			fmt.Fprintf(os.Stderr, "  - %s (legacy)\n", name)
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
