// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package account

import (
	"fmt"
	"io"

	"github.com/cloudflare/circl/sign/slhdsa"
	"github.com/luxfi/go-bip32"
)

// HD derivation path for SLH-DSA recovery accounts. Distinct from the
// ML-DSA wallet path so a leaked wallet branch (0') cannot be used to
// derive recovery keys, and vice versa. Uses a fresh hardened branch (2')
// that is reserved exclusively for SLH-DSA recovery.
//
//	m/44'/9000'/<chain_id>'/2'/<account_idx>'
//
// Branch 0' = ML-DSA wallet (identity/tx/session per role)
// Branch 1' = secp256k1 classical-compat (genesis F35)
// Branch 2' = SLH-DSA recovery (this file)
const (
	branchRecovery uint32 = 2

	// slhdsaSHAKE192sSeedSize is the total number of random bytes
	// FIPS-205 §10.1 KeyGen reads from its random source: skSeed ||
	// skPrf || pkSeed, each n=24 bytes for the 192-bit parameter sets.
	slhdsaSHAKE192sSeedSize = 3 * 24
)

// RecoveryAccount is an SLH-DSA-SHAKE-192s (FIPS 205, NIST Cat 3) account
// used as a stateless cold-key recovery anchor. Recovery accounts NEVER
// sign hot-path transactions; they sign account rotation / re-keying
// envelopes that the consensus layer pins under a higher policy
// (typically a multisig of N recovery accounts).
//
// Invariants:
//   - SchemeID == RecoverySchemeSLHDSA192s.
//   - PublicKey is the packed FIPS-205 public key (2 * n = 48 bytes for 192s).
//   - PrivateKey is the packed FIPS-205 private key (4 * n = 96 bytes for 192s).
//   - AccountID == DeriveAccountID(chainID, WalletSchemeNone(...).
//     Recovery accounts are tagged with a synthetic scheme byte that is
//     NOT a hot WalletSchemeID, so AccountIDs of recovery vs hot keys are
//     domain-separated even if they share a public key value space.
type RecoveryAccount struct {
	AccountID      AccountID
	SchemeID       RecoverySchemeID
	PublicKey      []byte
	PrivateKey     []byte
	DerivationPath string
}

// NewSLHDSARecoveryAccount derives a fresh SLH-DSA-SHAKE-192s keypair from
// a domain-separated child seed under a dedicated HD path
// (m/44'/9000'/<chainID>'/2'/<accountIdx>'). The child seed is expanded
// through cSHAKE-256 with the "LUX/WALLET/RECOVERY/V1" customization into
// the 3 * n = 72-byte deterministic stream that FIPS-205 §10.1 KeyGen
// consumes as (skSeed, skPrf, pkSeed). The result is fully deterministic
// for a fixed (masterSeed, chainID, accountIdx).
//
// Private keys produced here MUST be moved to an air-gapped store
// immediately; the in-memory copy returned on the struct is for the
// constructor's caller to wrap through the keystore before persisting.
func NewSLHDSARecoveryAccount(masterSeed []byte, chainID uint32, accountIdx uint32) (*RecoveryAccount, error) {
	if len(masterSeed) == 0 {
		return nil, fmt.Errorf("account: masterSeed is empty")
	}
	if chainID >= bip32.FirstHardenedChild {
		return nil, fmt.Errorf("account: chainID %d must be < 2^31 (BIP-32 hardening limit)", chainID)
	}
	if accountIdx >= bip32.FirstHardenedChild {
		return nil, fmt.Errorf("account: accountIdx %d must be < 2^31 (BIP-32 hardening limit)", accountIdx)
	}

	childSeed, derivationPath, err := deriveRecoveryChildSeed(masterSeed, chainID, accountIdx)
	if err != nil {
		return nil, err
	}

	// Expand the BIP-32 child seed into the 72-byte deterministic stream
	// FIPS-205 §10.1 KeyGen wants. cSHAKE-256 with the RECOVERY
	// customization makes this stream non-replayable as a wallet (hot)
	// stream — the customization string is bound into the cSHAKE state
	// before the seed bytes.
	stream, err := expandChildSeed(childSeed, CSHAKECustomizationRecovery, slhdsaSHAKE192sSeedSize)
	if err != nil {
		return nil, fmt.Errorf("account: expand recovery child seed: %w", err)
	}

	pub, priv, err := slhdsa.GenerateKey(newDeterministicReader(stream), slhdsa.SHAKE_192s)
	if err != nil {
		return nil, fmt.Errorf("account: slh-dsa keygen: %w", err)
	}

	pubBytes, err := pub.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("account: marshal slh-dsa public key: %w", err)
	}
	privBytes, err := priv.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("account: marshal slh-dsa private key: %w", err)
	}

	// AccountID derivation uses the synthetic scheme byte
	// WalletSchemeID(RecoverySchemeSLHDSA192s) so the recovery AccountID
	// space is domain-separated from any hot wallet AccountID with the
	// same chain/pubkey. The cast is safe because the byte values
	// (0x61..0x63 for recovery) cannot collide with the WalletSchemeID
	// blocks (0x00..0x4F) — see scheme.go for the byte map.
	accID, err := DeriveAccountID(chainID, WalletSchemeID(RecoverySchemeSLHDSA192s), pubBytes)
	if err != nil {
		return nil, fmt.Errorf("account: derive recovery AccountID: %w", err)
	}

	return &RecoveryAccount{
		AccountID:      accID,
		SchemeID:       RecoverySchemeSLHDSA192s,
		PublicKey:      pubBytes,
		PrivateKey:     privBytes,
		DerivationPath: derivationPath,
	}, nil
}

// Sign signs a 48-byte digest under the recovery key. Uses FIPS-205
// SignDeterministic (no per-signature randomness) so recovery signatures
// are reproducible from the private key alone — a deliberate choice for
// air-gapped recovery flows where the signing host has no entropy source.
//
// Context binds to "LUX/WALLET/RECOVERY/V1" so a recovery signature
// cannot be replayed as a hot wallet signature even on the same digest.
func (r *RecoveryAccount) Sign(digest [AccountIDSize]byte) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("account: RecoveryAccount is nil")
	}
	if r.SchemeID != RecoverySchemeSLHDSA192s {
		return nil, fmt.Errorf("account: RecoveryAccount.Sign currently supports only SLH-DSA-SHAKE-192s (got %s)", r.SchemeID)
	}
	priv := slhdsa.PrivateKey{ID: slhdsa.SHAKE_192s}
	if err := priv.UnmarshalBinary(r.PrivateKey); err != nil {
		return nil, fmt.Errorf("account: unmarshal slh-dsa private key: %w", err)
	}
	msg := slhdsa.NewMessage(digest[:])
	sig, err := slhdsa.SignDeterministic(&priv, msg, []byte(CSHAKECustomizationRecovery))
	if err != nil {
		return nil, fmt.Errorf("account: slh-dsa sign: %w", err)
	}
	return sig, nil
}

// Verify returns true iff sig is a valid SLH-DSA-SHAKE-192s signature of
// digest under r.PublicKey with the "LUX/WALLET/RECOVERY/V1" context.
func (r *RecoveryAccount) Verify(digest [AccountIDSize]byte, sig []byte) bool {
	if r == nil {
		return false
	}
	if r.SchemeID != RecoverySchemeSLHDSA192s {
		return false
	}
	pub := slhdsa.PublicKey{ID: slhdsa.SHAKE_192s}
	if err := pub.UnmarshalBinary(r.PublicKey); err != nil {
		return false
	}
	msg := slhdsa.NewMessage(digest[:])
	return slhdsa.Verify(&pub, msg, sig, []byte(CSHAKECustomizationRecovery))
}

// deriveRecoveryChildSeed walks m/44'/9000'/<chainID>'/2'/<accountIdx>'.
// Every level is hardened.
func deriveRecoveryChildSeed(masterSeed []byte, chainID uint32, accountIdx uint32) ([]byte, string, error) {
	master, err := bip32.NewMasterKey(masterSeed)
	if err != nil {
		return nil, "", fmt.Errorf("account: derive master key: %w", err)
	}
	purpose, err := master.NewChildKey(bip32.FirstHardenedChild + purposeBIP44)
	if err != nil {
		return nil, "", fmt.Errorf("account: derive purpose 44': %w", err)
	}
	coin, err := purpose.NewChildKey(bip32.FirstHardenedChild + coinTypeLux)
	if err != nil {
		return nil, "", fmt.Errorf("account: derive coin 9000': %w", err)
	}
	chain, err := coin.NewChildKey(bip32.FirstHardenedChild + chainID)
	if err != nil {
		return nil, "", fmt.Errorf("account: derive chain %d': %w", chainID, err)
	}
	branch, err := chain.NewChildKey(bip32.FirstHardenedChild + branchRecovery)
	if err != nil {
		return nil, "", fmt.Errorf("account: derive branch 2' (recovery): %w", err)
	}
	leaf, err := branch.NewChildKey(bip32.FirstHardenedChild + accountIdx)
	if err != nil {
		return nil, "", fmt.Errorf("account: derive account %d': %w", accountIdx, err)
	}
	path := fmt.Sprintf("m/44'/9000'/%d'/2'/%d'", chainID, accountIdx)
	return leaf.Key, path, nil
}

// deterministicReader feeds a fixed byte slice to FIPS-205 §10.1 KeyGen
// in place of a random source. The reader returns io.EOF once the seed
// stream is exhausted; KeyGen reads exactly 3 * n bytes so the stream
// length is sized to match.
type deterministicReader struct {
	buf []byte
	pos int
}

func newDeterministicReader(seed []byte) io.Reader {
	return &deterministicReader{buf: seed}
}

func (r *deterministicReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.buf) {
		return 0, io.EOF
	}
	n := copy(p, r.buf[r.pos:])
	r.pos += n
	return n, nil
}
