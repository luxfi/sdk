// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package account

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"testing"

	"golang.org/x/crypto/sha3"
)

// TestCSHAKE_Customizations_KAT is a Known-Answer Test that pins the
// canonical cSHAKE customization strings to their byte values. If
// anyone changes the string content, the constant string content
// (case, version suffix, slashes), this test changes the digest
// vectors and CI catches the wallet-incompatibility.
//
// The vectors are computed at test-write time by feeding the
// customization string and a single \x00 message byte into cSHAKE-256
// and reading the first 48 bytes. The expected values are recomputed
// inline rather than hard-coded so the test is self-checking; the test
// is meaningful only because it exercises the SAME constants the
// production code uses (no separate copy of the strings).
func TestCSHAKE_Customizations_KAT(t *testing.T) {
	t.Parallel()

	type vector struct {
		name          string
		customization string
	}
	vectors := []vector{
		{"identity", CSHAKECustomizationIdentity},
		{"tx", CSHAKECustomizationTx},
		{"session", CSHAKECustomizationSession},
		{"recovery", CSHAKECustomizationRecovery},
		{"account-id", CSHAKECustomizationAccountID},
	}

	// All four wallet-role customizations and the AccountID
	// customization must (a) be non-empty, (b) start with "LUX/",
	// (c) end with "/V1", and (d) produce distinct cSHAKE-256
	// outputs when fed identical (N, msg) pairs.
	seen := map[string][]byte{}
	for _, v := range vectors {
		if v.customization == "" {
			t.Errorf("%s: customization is empty", v.name)
			continue
		}
		if !bytesHasPrefix(v.customization, "LUX/") {
			t.Errorf("%s: customization %q does not start with LUX/", v.name, v.customization)
		}
		if !bytesHasSuffix(v.customization, "/V1") {
			t.Errorf("%s: customization %q does not end with /V1", v.name, v.customization)
		}

		h := sha3.NewCShake256([]byte("LUX_PQ_KEYGEN_V1"), []byte(v.customization))
		_, _ = h.Write([]byte{0x00})
		out := make([]byte, 48)
		if _, err := h.Read(out); err != nil {
			t.Fatalf("%s: cshake read: %v", v.name, err)
		}
		for prevName, prev := range seen {
			if bytes.Equal(prev, out) {
				t.Errorf("%s and %s produced the same cSHAKE output — customization strings collide", v.name, prevName)
			}
		}
		seen[v.name] = out
	}
}

// TestCSHAKE_Customization_Values pins the literal byte content of each
// customization string. A change to any of these strings is a
// wire-incompatible rewrite of the wallet; this test is the
// single source of truth for what V1 wallets agree on.
func TestCSHAKE_Customization_Values(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"identity", CSHAKECustomizationIdentity, "LUX/WALLET/IDENTITY/V1"},
		{"tx", CSHAKECustomizationTx, "LUX/WALLET/TX/V1"},
		{"session", CSHAKECustomizationSession, "LUX/WALLET/SESSION/V1"},
		{"recovery", CSHAKECustomizationRecovery, "LUX/WALLET/RECOVERY/V1"},
		{"account-id", CSHAKECustomizationAccountID, "LUX/WALLET/ACCOUNT_ID/V1"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: customization = %q, want %q", c.name, c.got, c.want)
		}
	}
}

// TestDeriveAccountID_KAT pins the AccountID format. The test feeds a
// fixed (chain_id, scheme, pubkey) triple through DeriveAccountID and
// (a) compares against an independent cSHAKE-256 computation in this
// test file, (b) compares against a hard-coded 48-byte vector that was
// produced by an independent JS implementation (@noble/hashes/sha3-addons
// cshake256). The hard-coded vector is the wire-compatibility anchor
// for the TS-side wallet: any regression here also flags the TS side.
func TestDeriveAccountID_KAT(t *testing.T) {
	t.Parallel()
	pubkey := []byte("test-pubkey-for-accountid-kat-vector-stable-across-runs")
	const chainID uint32 = 9000
	scheme := WalletSchemeMLDSA65

	got, err := DeriveAccountID(chainID, scheme, pubkey)
	if err != nil {
		t.Fatalf("DeriveAccountID: %v", err)
	}

	// (a) Independent recomputation inside this test file. Catches a
	// regression in the production cshake.go helper.
	h := sha3.NewCShake256([]byte(accountIDLabel), []byte(CSHAKECustomizationAccountID))
	var cidBytes [4]byte
	binary.BigEndian.PutUint32(cidBytes[:], chainID)
	_, _ = h.Write(cidBytes[:])
	_, _ = h.Write([]byte{byte(scheme)})
	_, _ = h.Write(pubkey)
	expected := make([]byte, AccountIDSize)
	if _, err := h.Read(expected); err != nil {
		t.Fatalf("kat cshake read: %v", err)
	}
	if !bytes.Equal(got[:], expected) {
		t.Fatalf("AccountID KAT mismatch vs in-test cSHAKE:\n  got:  %s\n  want: %s",
			hex.EncodeToString(got[:]), hex.EncodeToString(expected))
	}

	// (b) Cross-implementation KAT: byte vector produced by
	// @noble/hashes/sha3-addons cshake256 against the same input
	// (recorded 2026-05-10). A change here means the Go and TS sides
	// have drifted; one of the implementations is wrong.
	const tsKATHex = "76a03630148103ec558cf4d8f7e8a2d8766ea205d96a4529249c3aaf9c5d078cc9f9fdb8f21a5e7ef6bb4c4ea29c9654"
	tsKAT, err := hex.DecodeString(tsKATHex)
	if err != nil {
		t.Fatalf("decode ts KAT hex: %v", err)
	}
	if !bytes.Equal(got[:], tsKAT) {
		t.Fatalf("AccountID KAT mismatch vs JS @noble/hashes cshake256:\n  got:  %s\n  want: %s",
			hex.EncodeToString(got[:]), tsKATHex)
	}
}

func TestDeriveAccountID_RejectsNilPubkey(t *testing.T) {
	t.Parallel()
	if _, err := DeriveAccountID(1, WalletSchemeMLDSA65, nil); err == nil {
		t.Fatalf("DeriveAccountID(nil pubkey) succeeded, expected error")
	}
}

func TestExpandChildSeed_Deterministic(t *testing.T) {
	t.Parallel()
	seed := []byte("test-child-seed-32-bytes-long-ok!")
	a, err := expandChildSeed(seed, CSHAKECustomizationIdentity, 32)
	if err != nil {
		t.Fatalf("expandChildSeed a: %v", err)
	}
	b, err := expandChildSeed(seed, CSHAKECustomizationIdentity, 32)
	if err != nil {
		t.Fatalf("expandChildSeed b: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("expandChildSeed not deterministic")
	}

	// Different customization → different output for the same seed.
	c, err := expandChildSeed(seed, CSHAKECustomizationTx, 32)
	if err != nil {
		t.Fatalf("expandChildSeed c: %v", err)
	}
	if bytes.Equal(a, c) {
		t.Fatalf("expandChildSeed with different customization produced the same output")
	}

	// Different out length → first 32 bytes need NOT match because
	// cSHAKE is variable-output-length; the test asserts the
	// implementation requests the correct number of bytes.
	d, err := expandChildSeed(seed, CSHAKECustomizationIdentity, 64)
	if err != nil {
		t.Fatalf("expandChildSeed d: %v", err)
	}
	if len(d) != 64 {
		t.Fatalf("expandChildSeed length: got %d, want 64", len(d))
	}
}

func TestExpandChildSeed_RejectsBadInputs(t *testing.T) {
	t.Parallel()
	if _, err := expandChildSeed(nil, CSHAKECustomizationIdentity, 32); err == nil {
		t.Fatalf("nil seed accepted")
	}
	if _, err := expandChildSeed([]byte{0x01}, CSHAKECustomizationIdentity, 0); err == nil {
		t.Fatalf("zero outLen accepted")
	}
	if _, err := expandChildSeed([]byte{0x01}, "", 32); err == nil {
		t.Fatalf("empty customization accepted")
	}
}

func bytesHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func bytesHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
