// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package keys provides key derivation and management utilities for LUX.
package keys

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/go-bip32"
	"github.com/luxfi/go-bip39"
	"github.com/luxfi/vm/vms/secp256k1fx"
)

const (
	// LUXCoinType is the BIP-44 coin type for LUX (9000')
	LUXCoinType = 9000

	// DefaultDerivationPath is the standard BIP-44 path for LUX: m/44'/9000'/0'/0/0
	DefaultDerivationPath = "m/44'/9000'/0'/0/0"
)

// DeriveKeyFromMnemonic derives a private key from a BIP-39 mnemonic using BIP-44 path.
// Default path: m/44'/9000'/0'/0/0
func DeriveKeyFromMnemonic(mnemonic string, accountIndex uint32) ([]byte, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic phrase")
	}
	seed := bip39.NewSeed(mnemonic, "")
	return DeriveKeyFromSeed(seed, accountIndex)
}

// DeriveKeyFromSeed derives a private key from a BIP-39 seed using BIP-44 path.
// Path: m/44'/9000'/0'/0/{accountIndex}
func DeriveKeyFromSeed(seed []byte, accountIndex uint32) ([]byte, error) {
	// Create master key from seed
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to create master key: %w", err)
	}

	// BIP-44 path: m/44'/9000'/0'/0/{accountIndex}
	// m/44' (purpose)
	key, err := masterKey.NewChildKey(bip32.FirstHardenedChild + 44)
	if err != nil {
		return nil, fmt.Errorf("failed to derive purpose: %w", err)
	}

	// m/44'/9000' (coin type for LUX)
	key, err = key.NewChildKey(bip32.FirstHardenedChild + LUXCoinType)
	if err != nil {
		return nil, fmt.Errorf("failed to derive coin type: %w", err)
	}

	// m/44'/9000'/0' (account)
	key, err = key.NewChildKey(bip32.FirstHardenedChild + 0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive account: %w", err)
	}

	// m/44'/9000'/0'/0 (change)
	key, err = key.NewChildKey(0)
	if err != nil {
		return nil, fmt.Errorf("failed to derive change: %w", err)
	}

	// m/44'/9000'/0'/0/{accountIndex} (address index)
	key, err = key.NewChildKey(accountIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to derive address index: %w", err)
	}

	return key.Key, nil
}

// LoadKeyFromFile reads a hex-encoded private key from a file.
func LoadKeyFromFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}
	keyHex := strings.TrimSpace(string(data))
	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key hex: %w", err)
	}
	return keyBytes, nil
}

// LoadKeyFromEnv loads a key from environment variables.
// Priority: 1) LUX_PRIVATE_KEY (hex), 2) LUX_MNEMONIC (24 words)
func LoadKeyFromEnv() ([]byte, error) {
	if envKey := os.Getenv("LUX_PRIVATE_KEY"); envKey != "" {
		keyBytes, err := hex.DecodeString(strings.TrimSpace(envKey))
		if err != nil {
			return nil, fmt.Errorf("failed to decode LUX_PRIVATE_KEY: %w", err)
		}
		return keyBytes, nil
	}

	if mnemonic := os.Getenv("LUX_MNEMONIC"); mnemonic != "" {
		return DeriveKeyFromMnemonic(mnemonic, 0)
	}

	return nil, fmt.Errorf("no key found: set LUX_PRIVATE_KEY or LUX_MNEMONIC environment variable")
}

// LoadKey loads a key from file, environment, or returns an error.
// Priority: 1) keyPath if non-empty, 2) LUX_PRIVATE_KEY env, 3) LUX_MNEMONIC env
func LoadKey(keyPath string) ([]byte, error) {
	if keyPath != "" {
		return LoadKeyFromFile(keyPath)
	}
	return LoadKeyFromEnv()
}

// ToPrivateKey converts raw key bytes to a secp256k1 private key.
func ToPrivateKey(keyBytes []byte) (*secp256k1.PrivateKey, error) {
	return secp256k1.ToPrivateKey(keyBytes)
}

// NewKeychain creates a secp256k1fx.Keychain from raw key bytes.
func NewKeychain(keyBytes []byte) (*secp256k1fx.Keychain, error) {
	sk, err := ToPrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	return secp256k1fx.NewKeychain(sk), nil
}

// MustNewKeychain creates a secp256k1fx.Keychain from raw key bytes, panics on error.
func MustNewKeychain(keyBytes []byte) *secp256k1fx.Keychain {
	kc, err := NewKeychain(keyBytes)
	if err != nil {
		panic(err)
	}
	return kc
}
