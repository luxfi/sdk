// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package wallet

import (
	"errors"
	"fmt"
	"sync"

	"github.com/luxfi/sdk/wallet/account"
)

// PQWallet is the post-quantum native wallet facade. It stores
// PQAccounts (ML-DSA-65) and RecoveryAccounts (SLH-DSA-SHAKE-192s) by
// their canonical 48-byte AccountID, exposes Sign on accounts by ID,
// and is safe for concurrent use across goroutines.
//
// PQWallet is intentionally separate from the secp256k1 Wallet type in
// this package: classical-compat keys live in a strictly disjoint
// keyring so a chain that refuses WalletSchemeSecp256k1 cannot
// accidentally see one. Callers that mix classical and PQ keys hold
// both Wallet and PQWallet instances.
type PQWallet struct {
	mu         sync.RWMutex
	accounts   map[account.AccountID]*account.PQAccount
	recoveries map[account.AccountID]*account.RecoveryAccount
}

// NewPQWallet returns an empty PQWallet. The wallet is in-memory only;
// persistence is the caller's responsibility (typically through
// account.SealPQAccount + a file/keychain store).
func NewPQWallet() *PQWallet {
	return &PQWallet{
		accounts:   make(map[account.AccountID]*account.PQAccount),
		recoveries: make(map[account.AccountID]*account.RecoveryAccount),
	}
}

// AddAccount inserts a PQ account into the wallet. Returns an error if
// an account with the same AccountID is already present (idempotent
// inserts are NOT allowed — callers must choose to overwrite by
// calling RemoveAccount first).
func (w *PQWallet) AddAccount(a *account.PQAccount) error {
	if a == nil {
		return errors.New("wallet: AddAccount called with nil account")
	}
	if !a.SchemeID.IsPostQuantum() {
		return fmt.Errorf("wallet: PQWallet refuses non-PQ scheme %s", a.SchemeID)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, exists := w.accounts[a.AccountID]; exists {
		return fmt.Errorf("wallet: account %x already exists", a.AccountID[:6])
	}
	w.accounts[a.AccountID] = a
	return nil
}

// AddRecovery inserts an SLH-DSA recovery account into the wallet.
func (w *PQWallet) AddRecovery(r *account.RecoveryAccount) error {
	if r == nil {
		return errors.New("wallet: AddRecovery called with nil account")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, exists := w.recoveries[r.AccountID]; exists {
		return fmt.Errorf("wallet: recovery account %x already exists", r.AccountID[:6])
	}
	w.recoveries[r.AccountID] = r
	return nil
}

// Account returns the PQ account with the given AccountID. The second
// return value is false if no such account is present.
func (w *PQWallet) Account(id account.AccountID) (*account.PQAccount, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	a, ok := w.accounts[id]
	return a, ok
}

// Recovery returns the SLH-DSA recovery account with the given
// AccountID.
func (w *PQWallet) Recovery(id account.AccountID) (*account.RecoveryAccount, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	r, ok := w.recoveries[id]
	return r, ok
}

// RemoveAccount removes the account with the given AccountID. Returns
// false if no such account was present.
func (w *PQWallet) RemoveAccount(id account.AccountID) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, exists := w.accounts[id]; !exists {
		return false
	}
	delete(w.accounts, id)
	return true
}

// AccountIDs returns a slice of all hot (PQ) AccountIDs known to the
// wallet. Order is not stable across calls.
func (w *PQWallet) AccountIDs() []account.AccountID {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]account.AccountID, 0, len(w.accounts))
	for id := range w.accounts {
		out = append(out, id)
	}
	return out
}

// SignTx signs the envelope under the account identified by id. Thin
// wrapper around account.SignTx; returns ErrAccountNotFound if no such
// account exists. Mirrors the existing Wallet.Sign shape for callers
// who treat the wallet as the single signing entry point.
func (w *PQWallet) SignTx(id account.AccountID, env account.TxAuthEnvelope) error {
	w.mu.RLock()
	a, ok := w.accounts[id]
	w.mu.RUnlock()
	if !ok {
		return ErrAccountNotFound
	}
	return account.SignTx(a, env)
}

// ErrAccountNotFound is returned by PQWallet.SignTx when the supplied
// AccountID is not in the wallet.
var ErrAccountNotFound = errors.New("wallet: account not found")
