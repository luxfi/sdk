// Copyright (C) 2019-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keychain

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/luxfi/crypto/mldsa"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/crypto/slhdsa"
)

// TestVerifyHybridMLDSA_AndSemantics_AcceptsBoth proves the hybrid
// verifier accepts a signature where BOTH halves verify. Baseline
// happy-path check for the AND verification path.
func TestVerifyHybridMLDSA_AndSemantics_AcceptsBoth(t *testing.T) {
	classical, err := secp256k1.NewPrivateKey()
	if err != nil {
		t.Fatalf("secp256k1 keygen: %v", err)
	}
	pq, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA44)
	if err != nil {
		t.Fatalf("mldsa keygen: %v", err)
	}

	kc := NewPQKeychain(KeyTypeHybridSecp256k1MLDSA44)
	addr := kc.AddHybrid(classical, pq)
	signer, _ := kc.GetPQSigner(addr)

	msg := []byte("AND-verify happy path message")
	sig, err := signer.Sign(msg)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	classicalPub := classical.PublicKey()
	pqPub := pq.PublicKey
	if err := VerifyHybridSecp256k1MLDSA(classicalPub, pqPub, msg, sig); err != nil {
		t.Fatalf("VerifyHybridSecp256k1MLDSA rejected a valid hybrid signature: %v", err)
	}
}

// TestVerifyHybridMLDSA_AndSemantics_RejectsTamperedClassical proves
// the F112 closure: a hybrid signature with a real PQ half but a
// tampered classical half MUST fail. The previous OR-semantics would
// have accepted this as long as the PQ half verified — that loophole
// is now closed.
func TestVerifyHybridMLDSA_AndSemantics_RejectsTamperedClassical(t *testing.T) {
	classical, err := secp256k1.NewPrivateKey()
	if err != nil {
		t.Fatalf("secp256k1 keygen: %v", err)
	}
	pq, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA44)
	if err != nil {
		t.Fatalf("mldsa keygen: %v", err)
	}

	kc := NewPQKeychain(KeyTypeHybridSecp256k1MLDSA44)
	addr := kc.AddHybrid(classical, pq)
	signer, _ := kc.GetPQSigner(addr)

	msg := []byte("AND-verify rejects tampered classical")
	sig, err := signer.Sign(msg)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	// Flip a byte in the classical half (offset 5: 2-byte len prefix + 3
	// bytes into the secp256k1 signature). The PQ half remains valid.
	tampered := append([]byte(nil), sig...)
	tampered[5] ^= 0xFF

	classicalPub := classical.PublicKey()
	pqPub := pq.PublicKey
	err = VerifyHybridSecp256k1MLDSA(classicalPub, pqPub, msg, tampered)
	if err == nil {
		t.Fatal("AND-verifier accepted a hybrid with tampered classical half: F112 regressed")
	}
	if !errors.Is(err, ErrHybridSigClassicalFailed) {
		t.Errorf("expected ErrHybridSigClassicalFailed, got: %v", err)
	}
}

// TestVerifyHybridMLDSA_AndSemantics_RejectsTamperedPQ proves the
// other axis of F112: a hybrid with a valid classical half but a
// tampered PQ half MUST fail. Previously the OR-semantics would have
// accepted because classical alone passed; AND now refuses.
func TestVerifyHybridMLDSA_AndSemantics_RejectsTamperedPQ(t *testing.T) {
	classical, err := secp256k1.NewPrivateKey()
	if err != nil {
		t.Fatalf("secp256k1 keygen: %v", err)
	}
	pq, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA44)
	if err != nil {
		t.Fatalf("mldsa keygen: %v", err)
	}

	kc := NewPQKeychain(KeyTypeHybridSecp256k1MLDSA44)
	addr := kc.AddHybrid(classical, pq)
	signer, _ := kc.GetPQSigner(addr)

	msg := []byte("AND-verify rejects tampered PQ")
	sig, err := signer.Sign(msg)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	// Parse to find the PQ-half offset then flip a byte at the midpoint.
	classicalSig, pqSig, err := SplitHybridSignature(sig)
	if err != nil {
		t.Fatalf("SplitHybridSignature: %v", err)
	}
	pqOffset := 2 + len(classicalSig) + 2 // [len][cl][len][pq]
	tampered := append([]byte(nil), sig...)
	tampered[pqOffset+len(pqSig)/2] ^= 0xFF

	classicalPub := classical.PublicKey()
	pqPub := pq.PublicKey
	err = VerifyHybridSecp256k1MLDSA(classicalPub, pqPub, msg, tampered)
	if err == nil {
		t.Fatal("AND-verifier accepted a hybrid with tampered PQ half: F112 regressed")
	}
	if !errors.Is(err, ErrHybridSigPQFailed) {
		t.Errorf("expected ErrHybridSigPQFailed, got: %v", err)
	}
}

// TestVerifyHybridMLDSA_AndSemantics_RejectsRandomPQ is the canonical
// F112 anti-regression: a real ECDSA signature concatenated with
// random bytes labelled as "PQ" must be rejected. Previously the OR
// semantics let this through.
func TestVerifyHybridMLDSA_AndSemantics_RejectsRandomPQ(t *testing.T) {
	classical, err := secp256k1.NewPrivateKey()
	if err != nil {
		t.Fatalf("secp256k1 keygen: %v", err)
	}
	pq, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA44)
	if err != nil {
		t.Fatalf("mldsa keygen: %v", err)
	}

	kc := NewPQKeychain(KeyTypeHybridSecp256k1MLDSA44)
	addr := kc.AddHybrid(classical, pq)
	signer, _ := kc.GetPQSigner(addr)

	msg := []byte("AND-verify rejects random PQ half")
	sig, err := signer.Sign(msg)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	// Replace the entire PQ half with random bytes of the same length.
	classicalSig, pqSig, err := SplitHybridSignature(sig)
	if err != nil {
		t.Fatalf("SplitHybridSignature: %v", err)
	}
	pqOffset := 2 + len(classicalSig) + 2
	tampered := append([]byte(nil), sig...)
	if _, err := rand.Read(tampered[pqOffset : pqOffset+len(pqSig)]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	classicalPub := classical.PublicKey()
	pqPub := pq.PublicKey
	err = VerifyHybridSecp256k1MLDSA(classicalPub, pqPub, msg, tampered)
	if err == nil {
		t.Fatal("AND-verifier accepted ECDSA || random: F112 regressed (the headline attack)")
	}
	if !errors.Is(err, ErrHybridSigPQFailed) {
		t.Errorf("expected ErrHybridSigPQFailed, got: %v", err)
	}
}

// TestVerifyHybridMLDSA_AndSemantics_RejectsWrongMessage proves the
// verifier refuses a valid hybrid signature against a different
// message — basic signature integrity.
func TestVerifyHybridMLDSA_AndSemantics_RejectsWrongMessage(t *testing.T) {
	classical, _ := secp256k1.NewPrivateKey()
	pq, _ := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA44)

	kc := NewPQKeychain(KeyTypeHybridSecp256k1MLDSA44)
	addr := kc.AddHybrid(classical, pq)
	signer, _ := kc.GetPQSigner(addr)

	msg := []byte("original message")
	sig, _ := signer.Sign(msg)

	wrongMsg := []byte("different message")
	classicalPub := classical.PublicKey()
	pqPub := pq.PublicKey
	err := VerifyHybridSecp256k1MLDSA(classicalPub, pqPub, wrongMsg, sig)
	if err == nil {
		t.Fatal("AND-verifier accepted a signature against wrong message")
	}
	// Either half can fail first; both are acceptable failure modes.
	if !errors.Is(err, ErrHybridSigClassicalFailed) && !errors.Is(err, ErrHybridSigPQFailed) {
		t.Errorf("expected ErrHybridSig{Classical,PQ}Failed, got: %v", err)
	}
}

// TestVerifyHybridMLDSA_AndSemantics_RejectsTruncated proves the
// verifier refuses a truncated signature blob.
func TestVerifyHybridMLDSA_AndSemantics_RejectsTruncated(t *testing.T) {
	classical, _ := secp256k1.NewPrivateKey()
	pq, _ := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA44)
	classicalPub := classical.PublicKey()
	pqPub := pq.PublicKey

	cases := map[string][]byte{
		"empty":    {},
		"one byte": {0x00},
		"len only": {0x00, 0x40},
	}
	for name, sig := range cases {
		err := VerifyHybridSecp256k1MLDSA(classicalPub, pqPub, []byte("msg"), sig)
		if err == nil {
			t.Errorf("%s: AND-verifier accepted a truncated signature", name)
			continue
		}
		if !errors.Is(err, ErrHybridSigTruncated) {
			t.Errorf("%s: expected ErrHybridSigTruncated, got: %v", name, err)
		}
	}
}

// TestVerifyHybridSLHDSA_AndSemantics_AcceptsBoth covers the
// secp256k1+SLH-DSA hybrid happy path. F112 closure is symmetric
// across the two PQ choices.
func TestVerifyHybridSLHDSA_AndSemantics_AcceptsBoth(t *testing.T) {
	classical, _ := secp256k1.NewPrivateKey()
	pq, err := slhdsa.GenerateKey(rand.Reader, slhdsa.SHA2_128s)
	if err != nil {
		t.Fatalf("slhdsa keygen: %v", err)
	}

	kc := NewPQKeychain(KeyTypeHybridSecp256k1SLHDSA128)
	addr := kc.AddHybrid(classical, pq)
	signer, _ := kc.GetPQSigner(addr)

	msg := []byte("SLH-DSA AND-verify happy path")
	sig, err := signer.Sign(msg)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	classicalPub := classical.PublicKey()
	pqPub := pq.PublicKey
	if err := VerifyHybridSecp256k1SLHDSA(classicalPub, pqPub, msg, sig); err != nil {
		t.Fatalf("VerifyHybridSecp256k1SLHDSA rejected a valid hybrid: %v", err)
	}
}

// TestVerifyHybridSLHDSA_AndSemantics_RejectsRandomPQ — F112 closure
// on the SLH-DSA axis.
func TestVerifyHybridSLHDSA_AndSemantics_RejectsRandomPQ(t *testing.T) {
	classical, _ := secp256k1.NewPrivateKey()
	pq, _ := slhdsa.GenerateKey(rand.Reader, slhdsa.SHA2_128s)

	kc := NewPQKeychain(KeyTypeHybridSecp256k1SLHDSA128)
	addr := kc.AddHybrid(classical, pq)
	signer, _ := kc.GetPQSigner(addr)

	msg := []byte("SLH-DSA AND-verify rejects random PQ")
	sig, _ := signer.Sign(msg)

	classicalSig, pqSig, err := SplitHybridSignature(sig)
	if err != nil {
		t.Fatalf("SplitHybridSignature: %v", err)
	}
	pqOffset := 2 + len(classicalSig) + 2
	tampered := append([]byte(nil), sig...)
	if _, err := rand.Read(tampered[pqOffset : pqOffset+len(pqSig)]); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	classicalPub := classical.PublicKey()
	pqPub := pq.PublicKey
	err = VerifyHybridSecp256k1SLHDSA(classicalPub, pqPub, msg, tampered)
	if err == nil {
		t.Fatal("SLH-DSA AND-verifier accepted ECDSA || random: F112 regressed")
	}
	if !errors.Is(err, ErrHybridSigPQFailed) {
		t.Errorf("expected ErrHybridSigPQFailed, got: %v", err)
	}
}

// TestSplitHybridSignature_Roundtrip proves the parsing primitive
// matches the producer's layout — a regression check on the wire
// format that the verifier depends on.
func TestSplitHybridSignature_Roundtrip(t *testing.T) {
	classical, _ := secp256k1.NewPrivateKey()
	pq, _ := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA44)

	kc := NewPQKeychain(KeyTypeHybridSecp256k1MLDSA44)
	addr := kc.AddHybrid(classical, pq)
	signer, _ := kc.GetPQSigner(addr)

	msg := []byte("roundtrip")
	sig, _ := signer.Sign(msg)

	cl, pqHalf, err := SplitHybridSignature(sig)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if len(cl) == 0 || len(pqHalf) == 0 {
		t.Fatalf("Split produced empty halves: cl=%d pq=%d", len(cl), len(pqHalf))
	}
	// Both halves must equal the directly-signed primitives.
	hashed := sha256.Sum256(msg)
	if !classical.PublicKey().VerifyHash(hashed[:], cl) {
		t.Errorf("classical half didn't verify directly")
	}
	if !pq.PublicKey.VerifySignature(msg, pqHalf) {
		t.Errorf("PQ half didn't verify directly")
	}
}
