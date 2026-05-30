// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package wallet

import (
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/luxfi/sdk/wallet/account"
)

// pqTestSeed is a local synthetic seed for the pq_wallet tests; the
// account package has its own testMasterSeed but it isn't exported.
var pqTestSeed = func() []byte {
	h := sha256.Sum256([]byte("LUX/SDK/WALLET/PQWALLET/TEST/V1"))
	h2 := sha256.Sum256(append(h[:], 0x02))
	out := make([]byte, 0, 64)
	out = append(out, h[:]...)
	out = append(out, h2[:]...)
	return out
}()

func mustAccount(t *testing.T, chainID, accountIdx, roleIdx uint32, role account.AccountRole) *account.PQAccount {
	t.Helper()
	a, err := account.NewPQAccount(pqTestSeed, chainID, accountIdx, roleIdx, role)
	if err != nil {
		t.Fatalf("NewPQAccount: %v", err)
	}
	return a
}

func TestPQWallet_AddAccount(t *testing.T) {
	t.Parallel()
	w := NewPQWallet()
	a := mustAccount(t, 1, 0, 0, account.AccountRoleIdentity)

	if err := w.AddAccount(a); err != nil {
		t.Fatalf("AddAccount: %v", err)
	}
	got, ok := w.Account(a.AccountID)
	if !ok {
		t.Fatalf("Account(%x) not found after AddAccount", a.AccountID[:6])
	}
	if got != a {
		t.Errorf("Account returned a different pointer than AddAccount stored")
	}
}

func TestPQWallet_AddAccount_RejectsDuplicate(t *testing.T) {
	t.Parallel()
	w := NewPQWallet()
	a := mustAccount(t, 1, 0, 0, account.AccountRoleIdentity)

	if err := w.AddAccount(a); err != nil {
		t.Fatalf("AddAccount: %v", err)
	}
	if err := w.AddAccount(a); err == nil {
		t.Fatalf("AddAccount(duplicate) succeeded, expected error")
	}
}

func TestPQWallet_AddAccount_RejectsNil(t *testing.T) {
	t.Parallel()
	if err := NewPQWallet().AddAccount(nil); err == nil {
		t.Fatalf("AddAccount(nil) succeeded, expected error")
	}
}

func TestPQWallet_RemoveAccount(t *testing.T) {
	t.Parallel()
	w := NewPQWallet()
	a := mustAccount(t, 1, 0, 0, account.AccountRoleIdentity)
	if err := w.AddAccount(a); err != nil {
		t.Fatalf("AddAccount: %v", err)
	}
	if !w.RemoveAccount(a.AccountID) {
		t.Fatalf("RemoveAccount returned false for present account")
	}
	if _, ok := w.Account(a.AccountID); ok {
		t.Fatalf("Account still present after RemoveAccount")
	}
	// Second remove is idempotent-false.
	if w.RemoveAccount(a.AccountID) {
		t.Fatalf("RemoveAccount returned true for already-removed account")
	}
}

func TestPQWallet_SignTx(t *testing.T) {
	t.Parallel()
	w := NewPQWallet()
	a := mustAccount(t, 1, 0, 0, account.AccountRoleIdentity)
	if err := w.AddAccount(a); err != nil {
		t.Fatalf("AddAccount: %v", err)
	}

	env := &pqWalletEnvelope{}
	copy(env.digest[:], []byte("pq-wallet-sign-test-digest-48-bytes-exact-len--!"))

	if err := w.SignTx(a.AccountID, env); err != nil {
		t.Fatalf("SignTx: %v", err)
	}
	if !a.Verify(env.digest, env.sig) {
		t.Errorf("PQWallet.SignTx produced a signature that does not verify under the account")
	}
}

func TestPQWallet_SignTx_UnknownAccount(t *testing.T) {
	t.Parallel()
	w := NewPQWallet()
	var bogus account.AccountID
	err := w.SignTx(bogus, &pqWalletEnvelope{})
	if !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("SignTx(unknown) = %v, want ErrAccountNotFound", err)
	}
}

func TestPQWallet_AccountIDs(t *testing.T) {
	t.Parallel()
	w := NewPQWallet()
	if len(w.AccountIDs()) != 0 {
		t.Errorf("empty wallet should have 0 AccountIDs")
	}
	a := mustAccount(t, 1, 0, 0, account.AccountRoleIdentity)
	b := mustAccount(t, 1, 1, 0, account.AccountRoleIdentity)
	if err := w.AddAccount(a); err != nil {
		t.Fatalf("AddAccount a: %v", err)
	}
	if err := w.AddAccount(b); err != nil {
		t.Fatalf("AddAccount b: %v", err)
	}
	got := w.AccountIDs()
	if len(got) != 2 {
		t.Errorf("AccountIDs len = %d, want 2", len(got))
	}
}

func TestPQWallet_RecoveryAccount(t *testing.T) {
	t.Parallel()
	w := NewPQWallet()
	r, err := account.NewSLHDSARecoveryAccount(pqTestSeed, 1, 0)
	if err != nil {
		t.Fatalf("NewSLHDSARecoveryAccount: %v", err)
	}
	if err := w.AddRecovery(r); err != nil {
		t.Fatalf("AddRecovery: %v", err)
	}
	got, ok := w.Recovery(r.AccountID)
	if !ok {
		t.Fatalf("Recovery not found after AddRecovery")
	}
	if got != r {
		t.Errorf("Recovery returned a different pointer than AddRecovery stored")
	}
}

// pqWalletEnvelope is a local TxAuthEnvelope test helper.
type pqWalletEnvelope struct {
	digest [account.AccountIDSize]byte
	sig    []byte
}

func (e *pqWalletEnvelope) SigningDigest() ([account.AccountIDSize]byte, error) {
	return e.digest, nil
}

func (e *pqWalletEnvelope) AttachSignature(_ account.WalletSchemeID, _, sig []byte) error {
	e.sig = append([]byte(nil), sig...)
	return nil
}
