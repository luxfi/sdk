// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package account

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestKeystore_SealOpen_RoundTrip(t *testing.T) {
	t.Parallel()
	a := mustNewPQAccount(t, 9000, 0, 0, AccountRoleIdentity)
	passphrase := []byte("correct horse battery staple — non-zero entropy")

	blob, err := SealPQAccount(a, passphrase)
	if err != nil {
		t.Fatalf("SealPQAccount: %v", err)
	}
	// Sanity: the blob is valid JSON.
	var sanity map[string]any
	if err := json.Unmarshal(blob, &sanity); err != nil {
		t.Fatalf("blob is not JSON: %v", err)
	}

	b, err := OpenPQAccount(blob, passphrase)
	if err != nil {
		t.Fatalf("OpenPQAccount: %v", err)
	}

	if !bytes.Equal(a.PublicKey, b.PublicKey) {
		t.Fatalf("PublicKey mismatch after seal/open round-trip")
	}
	if !bytes.Equal(a.PrivateKey, b.PrivateKey) {
		t.Fatalf("PrivateKey mismatch after seal/open round-trip")
	}
	if a.AccountID != b.AccountID {
		t.Fatalf("AccountID mismatch after seal/open round-trip")
	}
	if a.SchemeID != b.SchemeID {
		t.Fatalf("SchemeID mismatch after seal/open round-trip")
	}
	if a.Role != b.Role {
		t.Fatalf("Role mismatch after seal/open round-trip: %q vs %q", a.Role, b.Role)
	}
	if a.DerivationPath != b.DerivationPath {
		t.Fatalf("DerivationPath mismatch after seal/open round-trip: %q vs %q", a.DerivationPath, b.DerivationPath)
	}
}

func TestKeystore_Open_RejectsWrongPassphrase(t *testing.T) {
	t.Parallel()
	a := mustNewPQAccount(t, 1, 0, 0, AccountRoleIdentity)
	blob, err := SealPQAccount(a, []byte("right-passphrase"))
	if err != nil {
		t.Fatalf("SealPQAccount: %v", err)
	}
	if _, err := OpenPQAccount(blob, []byte("wrong-passphrase")); err == nil {
		t.Fatalf("OpenPQAccount with wrong passphrase succeeded, expected GCM open failure")
	}
}

func TestKeystore_Seal_RejectsBadInputs(t *testing.T) {
	t.Parallel()
	a := mustNewPQAccount(t, 1, 0, 0, AccountRoleIdentity)

	if _, err := SealPQAccount(nil, []byte("p")); err == nil {
		t.Fatalf("nil account accepted")
	}
	if _, err := SealPQAccount(a, nil); err == nil {
		t.Fatalf("nil passphrase accepted")
	}
	if _, err := SealPQAccount(a, []byte{}); err == nil {
		t.Fatalf("empty passphrase accepted")
	}
}

func TestKeystore_Open_RejectsTamperedBlob(t *testing.T) {
	t.Parallel()
	a := mustNewPQAccount(t, 1, 0, 0, AccountRoleIdentity)
	blob, err := SealPQAccount(a, []byte("p"))
	if err != nil {
		t.Fatalf("SealPQAccount: %v", err)
	}

	// Unmarshal → mutate the ciphertext field → re-marshal. This
	// guarantees we hit the AEAD-protected bytes (not an unrelated
	// JSON whitespace byte or a base64 padding character).
	var parsed KeystoreBlob
	if err := json.Unmarshal(blob, &parsed); err != nil {
		t.Fatalf("unmarshal blob: %v", err)
	}
	if len(parsed.Ciphertext) == 0 {
		t.Fatalf("blob has no ciphertext field")
	}
	parsed.Ciphertext[0] ^= 0x01
	mut, err := json.Marshal(parsed)
	if err != nil {
		t.Fatalf("re-marshal blob: %v", err)
	}
	if _, err := OpenPQAccount(mut, []byte("p")); err == nil {
		t.Fatalf("OpenPQAccount accepted a tampered blob")
	}
}

func TestKeystore_Open_RejectsBadMagic(t *testing.T) {
	t.Parallel()
	bogus, _ := json.Marshal(KeystoreBlob{
		Magic:   [8]byte{'X', 'X', 'X', 'X', 'X', 'X', 'X', 'X'},
		Version: 1,
	})
	if _, err := OpenPQAccount(bogus, []byte("p")); err == nil {
		t.Fatalf("OpenPQAccount accepted a blob with bad magic")
	}
}

func TestPQAccount_ChainID(t *testing.T) {
	t.Parallel()
	a := mustNewPQAccount(t, 12345, 0, 0, AccountRoleIdentity)
	if got := a.ChainID(); got != 12345 {
		t.Errorf("ChainID = %d, want 12345", got)
	}
	// An account with no derivation path returns 0.
	empty := &PQAccount{}
	if got := empty.ChainID(); got != 0 {
		t.Errorf("empty.ChainID = %d, want 0", got)
	}
}
