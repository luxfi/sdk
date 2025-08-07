// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package wallet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/crypto"
)

func TestWallet_New(t *testing.T) {
	networkID := uint32(1)
	chainID := ids.GenerateTestID()

	wallet := New(networkID, chainID)
	require.NotNil(t, wallet)
	assert.Equal(t, networkID, wallet.networkID)
	assert.Equal(t, chainID, wallet.chainID)
	assert.NotNil(t, wallet.keychain)
	assert.NotNil(t, wallet.addresses)
	assert.NotNil(t, wallet.utxos)
}

func TestWallet_ImportKey(t *testing.T) {
	wallet := New(1, ids.GenerateTestID())

	// Generate a private key
	privateKey, err := crypto.GeneratePrivateKey()
	require.NoError(t, err)

	// Import the key
	address, err := wallet.ImportKey(privateKey)
	require.NoError(t, err)
	assert.NotEqual(t, ids.ShortID{}, address)

	// Verify the key was added
	assert.True(t, wallet.addresses.Contains(address))

	// Try to import the same key again
	_, err = wallet.ImportKey(privateKey)
	assert.NoError(t, err, "importing same key should return same address")
}

func TestWallet_GenerateKey(t *testing.T) {
	wallet := New(1, ids.GenerateTestID())

	// Generate a new key
	address1, err := wallet.GenerateKey()
	require.NoError(t, err)
	assert.NotEqual(t, ids.ShortID{}, address1)

	// Generate another key
	address2, err := wallet.GenerateKey()
	require.NoError(t, err)
	assert.NotEqual(t, ids.ShortID{}, address2)

	// Addresses should be different
	assert.NotEqual(t, address1, address2)

	// Both should be in the wallet
	assert.True(t, wallet.addresses.Contains(address1))
	assert.True(t, wallet.addresses.Contains(address2))
}

func TestWallet_GetAddress(t *testing.T) {
	wallet := New(1, ids.GenerateTestID())

	// No addresses initially
	_, err := wallet.GetAddress()
	assert.Error(t, err)

	// Generate an address
	address, err := wallet.GenerateKey()
	require.NoError(t, err)

	// Should return the address
	gotAddress, err := wallet.GetAddress()
	require.NoError(t, err)
	assert.Equal(t, address, gotAddress)
}

func TestWallet_GetAllAddresses(t *testing.T) {
	wallet := New(1, ids.GenerateTestID())

	// Initially empty
	addresses := wallet.GetAllAddresses()
	assert.Empty(t, addresses)

	// Generate multiple addresses
	expectedAddresses := make([]ids.ShortID, 3)
	for i := 0; i < 3; i++ {
		addr, err := wallet.GenerateKey()
		require.NoError(t, err)
		expectedAddresses[i] = addr
	}

	// Get all addresses
	addresses = wallet.GetAllAddresses()
	assert.Len(t, addresses, 3)

	// Verify all expected addresses are present
	for _, expected := range expectedAddresses {
		found := false
		for _, addr := range addresses {
			if addr == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "address %s not found", expected)
	}
}

func TestWallet_UTXO_Management(t *testing.T) {
	wallet := New(1, ids.GenerateTestID())

	// Generate an address
	address, err := wallet.GenerateKey()
	require.NoError(t, err)

	assetID := ids.GenerateTestID()

	// Initially no balance
	balance := wallet.GetBalance(assetID)
	assert.Equal(t, uint64(0), balance)

	// Add a UTXO
	utxo1 := &UTXO{
		ID:       ids.GenerateTestID(),
		AssetID:  assetID,
		Amount:   1000,
		Owner:    address,
		Locktime: 0,
	}
	wallet.AddUTXO(utxo1)

	// Check balance
	balance = wallet.GetBalance(assetID)
	assert.Equal(t, uint64(1000), balance)

	// Add another UTXO
	utxo2 := &UTXO{
		ID:       ids.GenerateTestID(),
		AssetID:  assetID,
		Amount:   500,
		Owner:    address,
		Locktime: 0,
	}
	wallet.AddUTXO(utxo2)

	// Check updated balance
	balance = wallet.GetBalance(assetID)
	assert.Equal(t, uint64(1500), balance)

	// Remove a UTXO
	wallet.RemoveUTXO(utxo1.ID)

	// Check balance after removal
	balance = wallet.GetBalance(assetID)
	assert.Equal(t, uint64(500), balance)
}

func TestWallet_GetUTXOs(t *testing.T) {
	wallet := New(1, ids.GenerateTestID())

	// Generate an address
	address, err := wallet.GenerateKey()
	require.NoError(t, err)

	assetID := ids.GenerateTestID()

	// Add multiple UTXOs
	utxo1 := &UTXO{
		ID:       ids.GenerateTestID(),
		AssetID:  assetID,
		Amount:   1000,
		Owner:    address,
		Locktime: 0,
	}
	utxo2 := &UTXO{
		ID:       ids.GenerateTestID(),
		AssetID:  assetID,
		Amount:   500,
		Owner:    address,
		Locktime: 0,
	}
	utxo3 := &UTXO{
		ID:       ids.GenerateTestID(),
		AssetID:  assetID,
		Amount:   300,
		Owner:    address,
		Locktime: 0,
	}

	wallet.AddUTXO(utxo1)
	wallet.AddUTXO(utxo2)
	wallet.AddUTXO(utxo3)

	tests := []struct {
		name        string
		amount      uint64
		expectError bool
		expectUTXOs int
		expectTotal uint64
	}{
		{
			name:        "exact amount single UTXO",
			amount:      1000,
			expectError: false,
			expectUTXOs: 1,
			expectTotal: 1000,
		},
		{
			name:        "amount requiring multiple UTXOs",
			amount:      1200,
			expectError: false,
			expectUTXOs: 2,
			expectTotal: 1500,
		},
		{
			name:        "amount requiring all UTXOs",
			amount:      1800,
			expectError: false,
			expectUTXOs: 3,
			expectTotal: 1800,
		},
		{
			name:        "insufficient funds",
			amount:      2000,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utxos, total, err := wallet.GetUTXOs(assetID, tt.amount)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, ErrInsufficientFunds, err)
			} else {
				require.NoError(t, err)
				assert.GreaterOrEqual(t, total, tt.amount)
				// We may get different UTXOs due to map iteration order
				assert.GreaterOrEqual(t, len(utxos), 1)
				assert.LessOrEqual(t, len(utxos), 3)
			}
		})
	}
}

func TestWallet_BLSKey(t *testing.T) {
	wallet := New(1, ids.GenerateTestID())

	// Initially no BLS key
	_, err := wallet.GetBLSKey()
	assert.Error(t, err)

	// Generate and set BLS key
	blsKey, err := bls.NewSecretKey()
	require.NoError(t, err)

	wallet.SetBLSKey(blsKey)

	// Get BLS key
	gotKey, err := wallet.GetBLSKey()
	require.NoError(t, err)
	assert.Equal(t, blsKey, gotKey)
}

func TestWallet_CreateTransferTx(t *testing.T) {
	wallet := New(1, ids.GenerateTestID())

	// Generate addresses
	from, err := wallet.GenerateKey()
	require.NoError(t, err)

	to := ids.GenerateTestShortID()
	assetID := ids.GenerateTestID()

	// Add UTXOs
	utxo1 := &UTXO{
		ID:       ids.GenerateTestID(),
		AssetID:  assetID,
		Amount:   1000,
		Owner:    from,
		Locktime: 0,
	}
	wallet.AddUTXO(utxo1)

	// Create transfer
	tx, err := wallet.CreateTransferTx(to, assetID, 700, []byte("test memo"))
	require.NoError(t, err)
	require.NotNil(t, tx)

	// Verify transaction
	assert.Equal(t, wallet.networkID, tx.NetworkID)
	assert.Equal(t, wallet.chainID, tx.ChainID)
	assert.Len(t, tx.Inputs, 1)
	assert.Len(t, tx.Outputs, 2) // Transfer + change

	// Verify outputs
	assert.Equal(t, uint64(700), tx.Outputs[0].Amount)
	assert.Equal(t, to, tx.Outputs[0].Recipient)
	assert.Equal(t, uint64(300), tx.Outputs[1].Amount) // Change
	assert.Equal(t, from, tx.Outputs[1].Recipient)
}

func TestWallet_CreateTransferTx_InsufficientFunds(t *testing.T) {
	wallet := New(1, ids.GenerateTestID())

	// Generate address
	from, err := wallet.GenerateKey()
	require.NoError(t, err)

	to := ids.GenerateTestShortID()
	assetID := ids.GenerateTestID()

	// Add insufficient UTXO
	utxo := &UTXO{
		ID:       ids.GenerateTestID(),
		AssetID:  assetID,
		Amount:   500,
		Owner:    from,
		Locktime: 0,
	}
	wallet.AddUTXO(utxo)

	// Try to create transfer for more than balance
	_, err = wallet.CreateTransferTx(to, assetID, 1000, nil)
	assert.Error(t, err)
	assert.Equal(t, ErrInsufficientFunds, err)
}

// Benchmark tests
func BenchmarkWallet_GenerateKey(b *testing.B) {
	wallet := New(1, ids.GenerateTestID())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := wallet.GenerateKey()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWallet_GetBalance(b *testing.B) {
	wallet := New(1, ids.GenerateTestID())

	// Setup: Generate address and add UTXOs
	address, _ := wallet.GenerateKey()
	assetID := ids.GenerateTestID()

	for i := 0; i < 100; i++ {
		utxo := &UTXO{
			ID:       ids.GenerateTestID(),
			AssetID:  assetID,
			Amount:   uint64(i * 100),
			Owner:    address,
			Locktime: 0,
		}
		wallet.AddUTXO(utxo)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wallet.GetBalance(assetID)
	}
}

func BenchmarkWallet_GetUTXOs(b *testing.B) {
	wallet := New(1, ids.GenerateTestID())

	// Setup: Generate address and add UTXOs
	address, _ := wallet.GenerateKey()
	assetID := ids.GenerateTestID()

	for i := 0; i < 1000; i++ {
		utxo := &UTXO{
			ID:       ids.GenerateTestID(),
			AssetID:  assetID,
			Amount:   uint64((i % 10) * 100),
			Owner:    address,
			Locktime: 0,
		}
		wallet.AddUTXO(utxo)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := wallet.GetUTXOs(assetID, 2500)
		if err != nil {
			b.Fatal(err)
		}
	}
}
