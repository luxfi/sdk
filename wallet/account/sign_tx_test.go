// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package account

import (
	"bytes"
	"errors"
	"testing"
)

// fakeEnvelope is a minimal TxAuthEnvelope for testing SignTx wiring.
type fakeEnvelope struct {
	digest       [AccountIDSize]byte
	digestErr    error
	attachErr    error
	attachedSig  []byte
	attachedPK   []byte
	attachedSch  WalletSchemeID
}

func (e *fakeEnvelope) SigningDigest() ([AccountIDSize]byte, error) {
	return e.digest, e.digestErr
}

func (e *fakeEnvelope) AttachSignature(scheme WalletSchemeID, pubkey, sig []byte) error {
	e.attachedSch = scheme
	e.attachedPK = append([]byte(nil), pubkey...)
	e.attachedSig = append([]byte(nil), sig...)
	return e.attachErr
}

func TestSignTx_Happy(t *testing.T) {
	t.Parallel()
	a := mustNewPQAccount(t, 1, 0, 0, AccountRoleIdentity)
	env := &fakeEnvelope{}
	copy(env.digest[:], []byte("signtx-happy-path-digest-48-bytes-exact-length-"))

	if err := SignTx(a, env); err != nil {
		t.Fatalf("SignTx: %v", err)
	}
	if env.attachedSch != a.SchemeID {
		t.Errorf("attached scheme = %s, want %s", env.attachedSch, a.SchemeID)
	}
	if !bytes.Equal(env.attachedPK, a.PublicKey) {
		t.Errorf("attached pubkey mismatch")
	}
	if !a.Verify(env.digest, env.attachedSig) {
		t.Errorf("attached signature does not verify under the account's public key")
	}
}

func TestSignTx_PropagatesDigestError(t *testing.T) {
	t.Parallel()
	a := mustNewPQAccount(t, 1, 0, 0, AccountRoleIdentity)
	sentinel := errors.New("digest broken")
	env := &fakeEnvelope{digestErr: sentinel}

	if err := SignTx(a, env); err == nil || !errors.Is(err, sentinel) {
		t.Fatalf("SignTx error = %v, want errors.Is(err, sentinel)", err)
	}
}

func TestSignTx_PropagatesAttachError(t *testing.T) {
	t.Parallel()
	a := mustNewPQAccount(t, 1, 0, 0, AccountRoleIdentity)
	sentinel := errors.New("attach broken")
	env := &fakeEnvelope{attachErr: sentinel}

	if err := SignTx(a, env); err == nil || !errors.Is(err, sentinel) {
		t.Fatalf("SignTx error = %v, want errors.Is(err, sentinel)", err)
	}
}

func TestSignTx_RejectsNil(t *testing.T) {
	t.Parallel()
	if err := SignTx(nil, &fakeEnvelope{}); err == nil {
		t.Fatalf("SignTx(nil account) succeeded, expected error")
	}
	a := mustNewPQAccount(t, 1, 0, 0, AccountRoleIdentity)
	if err := SignTx(a, nil); err == nil {
		t.Fatalf("SignTx(nil env) succeeded, expected error")
	}
}
