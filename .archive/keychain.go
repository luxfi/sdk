// Copyright (C) 2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package wallet

import (
	"errors"
	"sync"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/crypto"
)

// Keychain manages a collection of private keys
type Keychain struct {
	mu   sync.RWMutex
	keys map[ids.ShortID]crypto.PrivateKey
}

// NewKeychain creates a new keychain instance
func NewKeychain() *Keychain {
	return &Keychain{
		keys: make(map[ids.ShortID]crypto.PrivateKey),
	}
}

// Add adds a private key to the keychain
func (k *Keychain) Add(privateKey crypto.PrivateKey) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	address := privateKey.PublicKey().Address()
	if _, exists := k.keys[address]; exists {
		return errors.New("key already exists in keychain")
	}

	k.keys[address] = privateKey
	return nil
}

// Get retrieves a private key by address
func (k *Keychain) Get(address ids.ShortID) (crypto.PrivateKey, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	key, exists := k.keys[address]
	if !exists {
		return nil, errors.New("key not found in keychain")
	}

	return key, nil
}

// Remove removes a private key from the keychain
func (k *Keychain) Remove(address ids.ShortID) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if _, exists := k.keys[address]; !exists {
		return errors.New("key not found in keychain")
	}

	delete(k.keys, address)
	return nil
}

// List returns all addresses in the keychain
func (k *Keychain) List() []ids.ShortID {
	k.mu.RLock()
	defer k.mu.RUnlock()

	addresses := make([]ids.ShortID, 0, len(k.keys))
	for addr := range k.keys {
		addresses = append(addresses, addr)
	}

	return addresses
}

// Size returns the number of keys in the keychain
func (k *Keychain) Size() int {
	k.mu.RLock()
	defer k.mu.RUnlock()

	return len(k.keys)
}

// Clear removes all keys from the keychain
func (k *Keychain) Clear() {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.keys = make(map[ids.ShortID]crypto.PrivateKey)
}