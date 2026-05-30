// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package account

import (
	"bytes"
	"testing"
)

func mustNewRecovery(t *testing.T, chainID, accountIdx uint32) *RecoveryAccount {
	t.Helper()
	r, err := NewSLHDSARecoveryAccount(testMasterSeed, chainID, accountIdx)
	if err != nil {
		t.Fatalf("NewSLHDSARecoveryAccount: %v", err)
	}
	return r
}

func TestRecoveryAccount_Derive_Deterministic(t *testing.T) {
	t.Parallel()
	a := mustNewRecovery(t, 1, 0)
	b := mustNewRecovery(t, 1, 0)

	if !bytes.Equal(a.PublicKey, b.PublicKey) {
		t.Fatalf("RecoveryAccount.PublicKey not deterministic")
	}
	if !bytes.Equal(a.PrivateKey, b.PrivateKey) {
		t.Fatalf("RecoveryAccount.PrivateKey not deterministic")
	}
	if a.AccountID != b.AccountID {
		t.Fatalf("RecoveryAccount.AccountID not deterministic")
	}
	if a.DerivationPath != b.DerivationPath {
		t.Fatalf("RecoveryAccount.DerivationPath not deterministic")
	}
}

func TestRecoveryAccount_Derive_DistinctChainsDifferentKeys(t *testing.T) {
	t.Parallel()
	a := mustNewRecovery(t, 1, 0)
	b := mustNewRecovery(t, 2, 0)
	if bytes.Equal(a.PublicKey, b.PublicKey) {
		t.Fatalf("recovery key was the same across chain 1 and chain 2")
	}
	if a.AccountID == b.AccountID {
		t.Fatalf("recovery AccountID was the same across chain 1 and chain 2")
	}
}

func TestRecoveryAccount_Derive_DistinctAccountIdxDifferentKeys(t *testing.T) {
	t.Parallel()
	a := mustNewRecovery(t, 1, 0)
	b := mustNewRecovery(t, 1, 1)
	if bytes.Equal(a.PublicKey, b.PublicKey) {
		t.Fatalf("recovery key was the same across accountIdx 0 and 1")
	}
}

func TestRecoveryAccount_Sign_Verify_RoundTrip(t *testing.T) {
	t.Parallel()
	r := mustNewRecovery(t, 1, 0)

	var digest [AccountIDSize]byte
	copy(digest[:], []byte("recovery-test-digest-48-bytes-long----exact-len-"))

	sig, err := r.Sign(digest)
	if err != nil {
		t.Fatalf("RecoveryAccount.Sign: %v", err)
	}
	if !r.Verify(digest, sig) {
		t.Fatalf("RecoveryAccount.Verify rejected a valid signature")
	}

	// Mutate one byte of the signature: Verify must reject.
	bad := append([]byte(nil), sig...)
	bad[0] ^= 0x01
	if r.Verify(digest, bad) {
		t.Fatalf("RecoveryAccount.Verify accepted a mutated signature")
	}

	// Mutate the digest: Verify must reject.
	var badDigest [AccountIDSize]byte
	copy(badDigest[:], digest[:])
	badDigest[AccountIDSize-1] ^= 0x01
	if r.Verify(badDigest, sig) {
		t.Fatalf("RecoveryAccount.Verify accepted a signature against a different digest")
	}
}

func TestRecoveryAccount_DeterministicSignature(t *testing.T) {
	t.Parallel()
	// SLH-DSA SignDeterministic must produce identical signatures for
	// identical (key, digest) pairs. This is the property air-gapped
	// recovery hosts depend on (no entropy source needed).
	r := mustNewRecovery(t, 1, 0)
	var digest [AccountIDSize]byte
	copy(digest[:], []byte("deterministic-recovery-signature-test-vector-48b"))

	sig1, err := r.Sign(digest)
	if err != nil {
		t.Fatalf("Sign 1: %v", err)
	}
	sig2, err := r.Sign(digest)
	if err != nil {
		t.Fatalf("Sign 2: %v", err)
	}
	if !bytes.Equal(sig1, sig2) {
		t.Fatalf("SLH-DSA SignDeterministic produced different signatures for the same (key, digest)")
	}
}

func TestRecoveryAccount_Reject_BadInputs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		seed       []byte
		chainID    uint32
		accountIdx uint32
	}{
		{"empty seed", []byte{}, 1, 0},
		{"chain id too large", testMasterSeed, 1 << 31, 0},
		{"account idx too large", testMasterSeed, 1, 1 << 31},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewSLHDSARecoveryAccount(c.seed, c.chainID, c.accountIdx); err == nil {
				t.Fatalf("NewSLHDSARecoveryAccount(%q) succeeded, expected error", c.name)
			}
		})
	}
}

func TestRecoveryAccount_AccountIDDistinctFromHotKey(t *testing.T) {
	t.Parallel()
	// A recovery account at (chain=1, idx=0) MUST have a different
	// AccountID from a hot ML-DSA-65 account at any (chain=1, role,
	// idx) — even if some hostile construction tried to make the
	// pubkeys coincide. We verify domain separation by checking that
	// the recovery's AccountID is not a member of the ML-DSA-65
	// AccountID space for the same chain. We can't enumerate the
	// entire ML-DSA space, but we can verify that the recovery's
	// AccountID derived under WalletSchemeMLDSA65 (instead of the
	// recovery scheme byte) is different.
	r := mustNewRecovery(t, 1, 0)
	hot, err := DeriveAccountID(1, WalletSchemeMLDSA65, r.PublicKey)
	if err != nil {
		t.Fatalf("DeriveAccountID: %v", err)
	}
	if r.AccountID == hot {
		t.Fatalf("recovery AccountID collides with hot ML-DSA-65 AccountID — domain separation is broken")
	}
}

func TestRecoverySchemeID_String(t *testing.T) {
	t.Parallel()
	cases := []struct {
		scheme RecoverySchemeID
		want   string
	}{
		{RecoverySchemeNone, "none"},
		{RecoverySchemeSLHDSA128s, "slh-dsa-shake-128s"},
		{RecoverySchemeSLHDSA192s, "slh-dsa-shake-192s"},
		{RecoverySchemeSLHDSA256s, "slh-dsa-shake-256s"},
		{RecoverySchemeID(0xAA), "recovery-scheme(0xaa)"},
	}
	for _, c := range cases {
		if got := c.scheme.String(); got != c.want {
			t.Errorf("RecoverySchemeID(0x%02x).String() = %q, want %q", uint8(c.scheme), got, c.want)
		}
	}
}
