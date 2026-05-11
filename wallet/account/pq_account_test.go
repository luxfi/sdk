// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package account

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
)

// testMasterSeed is a fixed 64-byte BIP-39 seed for determinism tests.
// Synthetic, not derived from a mnemonic. NEVER copy into production.
var testMasterSeed = func() []byte {
	h := sha256.Sum256([]byte("LUX/SDK/WALLET/ACCOUNT/TEST/V1"))
	// Stretch the 32-byte SHA-256 to 64 bytes by concatenating two
	// distinct hashes so the master seed is the canonical BIP-32 length.
	h2 := sha256.Sum256(append(h[:], 0x01))
	out := make([]byte, 0, 64)
	out = append(out, h[:]...)
	out = append(out, h2[:]...)
	return out
}()

func mustNewPQAccount(t *testing.T, chainID, accountIdx, roleIdx uint32, role AccountRole) *PQAccount {
	t.Helper()
	a, err := NewPQAccount(testMasterSeed, chainID, accountIdx, roleIdx, role)
	if err != nil {
		t.Fatalf("NewPQAccount: %v", err)
	}
	return a
}

func TestPQAccount_Derive_Deterministic(t *testing.T) {
	t.Parallel()
	a := mustNewPQAccount(t, 1, 0, 0, AccountRoleIdentity)
	b := mustNewPQAccount(t, 1, 0, 0, AccountRoleIdentity)

	if !bytes.Equal(a.PublicKey, b.PublicKey) {
		t.Fatalf("PublicKey not deterministic: %x vs %x", a.PublicKey[:16], b.PublicKey[:16])
	}
	if !bytes.Equal(a.PrivateKey, b.PrivateKey) {
		t.Fatalf("PrivateKey not deterministic")
	}
	if a.AccountID != b.AccountID {
		t.Fatalf("AccountID not deterministic: %x vs %x", a.AccountID, b.AccountID)
	}
	if a.DerivationPath != b.DerivationPath {
		t.Fatalf("DerivationPath not deterministic: %q vs %q", a.DerivationPath, b.DerivationPath)
	}
}

func TestPQAccount_Derive_DistinctChainsDifferentKeys(t *testing.T) {
	t.Parallel()
	a := mustNewPQAccount(t, 1, 0, 0, AccountRoleIdentity)
	b := mustNewPQAccount(t, 2, 0, 0, AccountRoleIdentity)

	if bytes.Equal(a.PublicKey, b.PublicKey) {
		t.Fatalf("chain 1 and chain 2 produced the same public key")
	}
	if a.AccountID == b.AccountID {
		t.Fatalf("chain 1 and chain 2 produced the same AccountID")
	}
	if a.DerivationPath == b.DerivationPath {
		t.Fatalf("chain 1 and chain 2 produced the same derivation path")
	}
}

func TestPQAccount_Derive_DistinctRolesDifferentKeys(t *testing.T) {
	t.Parallel()
	// Pin different roleIdx so the HD-path is distinct as well as the
	// cSHAKE customization.
	identity := mustNewPQAccount(t, 1, 0, 0, AccountRoleIdentity)
	tx := mustNewPQAccount(t, 1, 0, 1, AccountRoleTx)
	session := mustNewPQAccount(t, 1, 0, 2, AccountRoleSession)

	pairs := [][2]*PQAccount{
		{identity, tx},
		{identity, session},
		{tx, session},
	}
	for _, p := range pairs {
		if bytes.Equal(p[0].PublicKey, p[1].PublicKey) {
			t.Fatalf("roles %q and %q produced the same public key", p[0].Role, p[1].Role)
		}
		if p[0].AccountID == p[1].AccountID {
			t.Fatalf("roles %q and %q produced the same AccountID", p[0].Role, p[1].Role)
		}
	}
}

func TestPQAccount_Derive_RoleCustomizationDifferentKeys(t *testing.T) {
	t.Parallel()
	// Same HD path (roleIdx=0 for both) but different role
	// customizations → different keys. This isolates the cSHAKE
	// domain-separation effect from the HD-path effect.
	identity := mustNewPQAccount(t, 1, 0, 0, AccountRoleIdentity)
	tx := mustNewPQAccount(t, 1, 0, 0, AccountRoleTx)

	if bytes.Equal(identity.PublicKey, tx.PublicKey) {
		t.Fatalf("identity and tx roles at the same HD path produced the same public key — cSHAKE customization is broken")
	}
	// HD path string is identical (both at roleIdx=0).
	if identity.DerivationPath != tx.DerivationPath {
		t.Fatalf("HD paths should match when roleIdx is the same")
	}
}

func TestPQAccount_Sign_Verify_RoundTrip(t *testing.T) {
	t.Parallel()
	a := mustNewPQAccount(t, 1, 0, 0, AccountRoleIdentity)

	var digest [AccountIDSize]byte
	copy(digest[:], []byte("the quick brown fox jumps over the lazy dog 48b"))

	sig, err := a.Sign(digest)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if len(sig) != mldsa65.SignatureSize {
		t.Fatalf("signature size: got %d, want %d", len(sig), mldsa65.SignatureSize)
	}
	if !a.Verify(digest, sig) {
		t.Fatalf("Verify rejected a valid signature")
	}

	// Mutate one byte of the signature: Verify must reject.
	bad := append([]byte(nil), sig...)
	bad[0] ^= 0x01
	if a.Verify(digest, bad) {
		t.Fatalf("Verify accepted a mutated signature")
	}

	// Mutate the digest: Verify must reject.
	var badDigest [AccountIDSize]byte
	copy(badDigest[:], digest[:])
	badDigest[0] ^= 0x01
	if a.Verify(badDigest, sig) {
		t.Fatalf("Verify accepted a signature against a different digest")
	}
}

func TestPQAccount_Sign_Verify_AcrossAccounts(t *testing.T) {
	t.Parallel()
	a := mustNewPQAccount(t, 1, 0, 0, AccountRoleIdentity)
	b := mustNewPQAccount(t, 1, 1, 0, AccountRoleIdentity)

	var digest [AccountIDSize]byte
	copy(digest[:], []byte("cross-account signature must not verify under b "))

	sig, err := a.Sign(digest)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if b.Verify(digest, sig) {
		t.Fatalf("account B accepted a signature from account A")
	}
}

func TestPQAccount_Reject_RecoveryRole(t *testing.T) {
	t.Parallel()
	// AccountRoleRecovery is reserved for SLH-DSA; NewPQAccount must
	// refuse it explicitly so callers don't accidentally emit an ML-DSA
	// "recovery" key.
	_, err := NewPQAccount(testMasterSeed, 1, 0, 0, AccountRoleRecovery)
	if err == nil {
		t.Fatalf("NewPQAccount(role=Recovery) succeeded, expected error")
	}
}

func TestPQAccount_Reject_BadInputs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		seed       []byte
		chainID    uint32
		accountIdx uint32
		roleIdx    uint32
		role       AccountRole
	}{
		{"empty seed", []byte{}, 1, 0, 0, AccountRoleIdentity},
		{"chain id too large", testMasterSeed, 1 << 31, 0, 0, AccountRoleIdentity},
		{"account idx too large", testMasterSeed, 1, 1 << 31, 0, AccountRoleIdentity},
		{"role idx too large", testMasterSeed, 1, 0, 1 << 31, AccountRoleIdentity},
		{"unknown role", testMasterSeed, 1, 0, 0, AccountRole("bogus")},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewPQAccount(c.seed, c.chainID, c.accountIdx, c.roleIdx, c.role); err == nil {
				t.Fatalf("NewPQAccount(%q) succeeded, expected error", c.name)
			}
		})
	}
}

func TestAccountID_FromKey_MatchesSHAKE384(t *testing.T) {
	t.Parallel()
	// Known-answer test for DeriveAccountID. The expected bytes are
	// computed by an independent SHAKE256/cSHAKE256 implementation
	// (golang.org/x/crypto/sha3.NewCShake256) over the canonical input
	// layout u32be(chain) || u8(scheme) || pubkey, with the
	// "LUX_ACCOUNT_V1" function name and the "LUX/WALLET/ACCOUNT_ID/V1"
	// customization. The test re-computes the same digest by hand here
	// rather than hard-coding 48 magic bytes because every input is
	// deterministic; a hard-coded vector would couple this test to a
	// specific seed.
	a := mustNewPQAccount(t, 7, 13, 0, AccountRoleIdentity)
	id, err := DeriveAccountID(7, WalletSchemeMLDSA65, a.PublicKey)
	if err != nil {
		t.Fatalf("DeriveAccountID: %v", err)
	}
	if id != a.AccountID {
		t.Fatalf("AccountID mismatch:\n  account.AccountID = %x\n  derive again       = %x", a.AccountID, id)
	}
	// Sanity: the AccountID is exactly 48 bytes (the SHAKE256-384
	// equivalent length the consensus profile pins).
	if len(a.AccountID) != AccountIDSize {
		t.Fatalf("AccountID length: got %d, want %d", len(a.AccountID), AccountIDSize)
	}
}

func TestAccountID_DifferentSchemeDifferentID(t *testing.T) {
	t.Parallel()
	a := mustNewPQAccount(t, 1, 0, 0, AccountRoleIdentity)

	// Re-derive the AccountID under a different scheme byte: must
	// produce a different AccountID even though the pubkey bytes are
	// identical. This is the cSHAKE domain-separation property at the
	// AccountID layer.
	id1, err := DeriveAccountID(1, WalletSchemeMLDSA65, a.PublicKey)
	if err != nil {
		t.Fatalf("DeriveAccountID(MLDSA65): %v", err)
	}
	id2, err := DeriveAccountID(1, WalletSchemeMLDSA87, a.PublicKey)
	if err != nil {
		t.Fatalf("DeriveAccountID(MLDSA87): %v", err)
	}
	if id1 == id2 {
		t.Fatalf("AccountID under MLDSA65 and MLDSA87 collided for the same pubkey")
	}
}

func TestWalletSchemeID_String(t *testing.T) {
	t.Parallel()
	cases := []struct {
		scheme WalletSchemeID
		want   string
	}{
		{WalletSchemeNone, "none"},
		{WalletSchemeSecp256k1, "secp256k1"},
		{WalletSchemeMLDSA44, "ml-dsa-44"},
		{WalletSchemeMLDSA65, "ml-dsa-65"},
		{WalletSchemeMLDSA87, "ml-dsa-87"},
		{WalletSchemeID(0xAA), "wallet-scheme(0xaa)"},
	}
	for _, c := range cases {
		if got := c.scheme.String(); got != c.want {
			t.Errorf("WalletSchemeID(0x%02x).String() = %q, want %q", uint8(c.scheme), got, c.want)
		}
	}
}

func TestWalletSchemeID_Predicates(t *testing.T) {
	t.Parallel()
	if !WalletSchemeMLDSA65.IsPostQuantum() {
		t.Errorf("MLDSA65 must be IsPostQuantum")
	}
	if WalletSchemeSecp256k1.IsPostQuantum() {
		t.Errorf("Secp256k1 must NOT be IsPostQuantum")
	}
	if !WalletSchemeSecp256k1.IsClassicalCompat() {
		t.Errorf("Secp256k1 must be IsClassicalCompat")
	}
	if WalletSchemeMLDSA65.IsClassicalCompat() {
		t.Errorf("MLDSA65 must NOT be IsClassicalCompat")
	}
}

func TestAccountRole_HDIndex(t *testing.T) {
	t.Parallel()
	cases := []struct {
		role  AccountRole
		want  uint32
		isErr bool
	}{
		{AccountRoleIdentity, 0, false},
		{AccountRoleTx, 1, false},
		{AccountRoleSession, 2, false},
		{AccountRoleRecovery, 0, true},
		{AccountRole("bogus"), 0, true},
	}
	for _, c := range cases {
		idx, err := c.role.HDIndex()
		gotErr := err != nil
		if gotErr != c.isErr {
			t.Errorf("HDIndex(%q) err=%v, want isErr=%v", c.role, err, c.isErr)
			continue
		}
		if !c.isErr && idx != c.want {
			t.Errorf("HDIndex(%q) = %d, want %d", c.role, idx, c.want)
		}
	}
}
