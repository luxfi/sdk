// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package wallet

import (
	"context"
	"errors"
	"fmt"

	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
	"github.com/luxfi/set"

	"github.com/luxfi/sdk/chain"
	"github.com/luxfi/sdk/crypto"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrNoUTXOs           = errors.New("no UTXOs available")
	ErrInvalidAddress    = errors.New("invalid address")
)

// UTXO represents an unspent transaction output
type UTXO struct {
	ID       ids.ID
	AssetID  ids.ID
	Amount   uint64
	Owner    ids.ShortID
	Locktime uint64
}

// Wallet manages keys and transactions for personal usage
type Wallet struct {
	// Network configuration
	networkID uint32
	chainID   ids.ID

	// Key management
	keychain  *Keychain
	addresses set.Set[ids.ShortID]

	// UTXO management
	utxos map[ids.ID]*UTXO

	// BLS key for validator operations
	blsKey *bls.SecretKey
}

// New creates a new wallet instance
func New(networkID uint32, chainID ids.ID) *Wallet {
	return &Wallet{
		networkID: networkID,
		chainID:   chainID,
		keychain:  NewKeychain(),
		addresses: set.NewSet[ids.ShortID](10),
		utxos:     make(map[ids.ID]*UTXO),
	}
}

// ImportKey imports a private key into the wallet
func (w *Wallet) ImportKey(privateKey crypto.PrivateKey) (ids.ShortID, error) {
	pubKey := privateKey.PublicKey()
	address := pubKey.Address()

	if err := w.keychain.Add(privateKey); err != nil {
		return ids.ShortID{}, err
	}

	w.addresses.Add(address)
	return address, nil
}

// GenerateKey generates a new key and adds it to the wallet
func (w *Wallet) GenerateKey() (ids.ShortID, error) {
	privateKey, err := crypto.GeneratePrivateKey()
	if err != nil {
		return ids.ShortID{}, err
	}

	return w.ImportKey(privateKey)
}

// GetAddress returns a wallet address
func (w *Wallet) GetAddress() (ids.ShortID, error) {
	addresses := w.addresses.List()
	if len(addresses) == 0 {
		return ids.ShortID{}, errors.New("no addresses in wallet")
	}
	return addresses[0], nil
}

// GetAllAddresses returns all wallet addresses
func (w *Wallet) GetAllAddresses() []ids.ShortID {
	return w.addresses.List()
}

// GetBalance returns the balance for a specific asset
func (w *Wallet) GetBalance(assetID ids.ID) uint64 {
	var balance uint64
	for _, utxo := range w.utxos {
		if utxo.AssetID == assetID && w.addresses.Contains(utxo.Owner) {
			balance += utxo.Amount
		}
	}
	return balance
}

// AddUTXO adds a UTXO to the wallet
func (w *Wallet) AddUTXO(utxo *UTXO) {
	w.utxos[utxo.ID] = utxo
}

// RemoveUTXO removes a UTXO from the wallet
func (w *Wallet) RemoveUTXO(utxoID ids.ID) {
	delete(w.utxos, utxoID)
}

// GetUTXOs returns UTXOs for spending
func (w *Wallet) GetUTXOs(assetID ids.ID, amount uint64) ([]*UTXO, uint64, error) {
	var (
		utxos    []*UTXO
		totalAmt uint64
	)

	for _, utxo := range w.utxos {
		if utxo.AssetID != assetID {
			continue
		}

		// Check if we own this UTXO
		if !w.addresses.Contains(utxo.Owner) {
			continue
		}

		utxos = append(utxos, utxo)
		totalAmt += utxo.Amount

		if totalAmt >= amount {
			return utxos, totalAmt, nil
		}
	}

	if totalAmt < amount {
		return nil, 0, ErrInsufficientFunds
	}

	return utxos, totalAmt, nil
}

// Sign signs a transaction with the wallet's keys
func (w *Wallet) Sign(ctx context.Context, tx chain.Transaction) error {
	// Get the required signers for this transaction
	signers := tx.Auth().Actor()

	// Find the appropriate key and sign
	for _, signer := range []ids.ShortID{signers} {
		if w.addresses.Contains(signer) {
			privateKey, err := w.keychain.Get(signer)
			if err != nil {
				return fmt.Errorf("failed to get key for %s: %w", signer, err)
			}

			// Sign the transaction
			if err := tx.Sign(privateKey); err != nil {
				return fmt.Errorf("failed to sign transaction: %w", err)
			}

			return nil
		}
	}

	return errors.New("no signing key found for transaction")
}

// SetBLSKey sets the BLS key for validator operations
func (w *Wallet) SetBLSKey(key *bls.SecretKey) {
	w.blsKey = key
}

// GetBLSKey returns the BLS key for validator operations
func (w *Wallet) GetBLSKey() (*bls.SecretKey, error) {
	if w.blsKey == nil {
		return nil, errors.New("no BLS key set")
	}
	return w.blsKey, nil
}

// TransferInput represents an input to a transfer transaction
type TransferInput struct {
	UTXOID  ids.ID
	AssetID ids.ID
	Amount  uint64
}

// TransferOutput represents an output from a transfer transaction
type TransferOutput struct {
	AssetID   ids.ID
	Amount    uint64
	Recipient ids.ShortID
	Locktime  uint64
}

// CreateTransferTx creates a transfer transaction
func (w *Wallet) CreateTransferTx(
	to ids.ShortID,
	assetID ids.ID,
	amount uint64,
	memo []byte,
) (*TransferTx, error) {
	// Get UTXOs for the transfer
	utxos, totalAmt, err := w.GetUTXOs(assetID, amount)
	if err != nil {
		return nil, err
	}

	// Create inputs
	var inputs []TransferInput
	for _, utxo := range utxos {
		input := TransferInput{
			UTXOID:  utxo.ID,
			AssetID: assetID,
			Amount:  utxo.Amount,
		}
		inputs = append(inputs, input)
	}

	// Create outputs
	outputs := []TransferOutput{
		{
			AssetID:   assetID,
			Amount:    amount,
			Recipient: to,
		},
	}

	// Add change output if necessary
	if totalAmt > amount {
		from, err := w.GetAddress()
		if err != nil {
			return nil, err
		}

		changeOutput := TransferOutput{
			AssetID:   assetID,
			Amount:    totalAmt - amount,
			Recipient: from,
		}
		outputs = append(outputs, changeOutput)
	}

	return &TransferTx{
		NetworkID: w.networkID,
		ChainID:   w.chainID,
		Inputs:    inputs,
		Outputs:   outputs,
		Memo:      memo,
	}, nil
}

// TransferTx represents a transfer transaction
type TransferTx struct {
	NetworkID uint32
	ChainID   ids.ID
	Inputs    []TransferInput
	Outputs   []TransferOutput
	Memo      []byte
}
