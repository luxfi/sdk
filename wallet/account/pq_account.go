// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package account

import (
	"fmt"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	"github.com/luxfi/go-bip32"
)

// HD derivation path used for ML-DSA wallet accounts:
//
//	m/44'/9000'/<chain_id>'/0'/<role_idx>'/<account_idx>'
//
// Mirrors the F35 hardening landed in
// genesis/pkg/genesis/keys.go:
//
//	device_pq_key[i]  = m/44'/9000'/nid'/0'/i'  (ML-DSA-65)
//	device_lux_key[i] = m/44'/9000'/nid'/1'/i'  (secp256k1)
//
// with one extra hardened level (<role_idx>) so identity / tx / session
// keys are siblings under the ML-DSA branch but cannot be confused with
// each other. Five hardened levels above; <chain_id> must be < 2^31
// because BIP-32 hardening folds the high bit.
const (
	// purposeBIP44 is the BIP-44 purpose constant.
	purposeBIP44 uint32 = 44
	// coinTypeLux is Lux's registered coin type. 9000 chosen at HIP-0077
	// rollout and pinned across all luxfi wallets.
	coinTypeLux uint32 = 9000
	// branchPQ is the hardened child index for the ML-DSA branch.
	branchPQ uint32 = 0
	// branchClassical is the hardened child index for the secp256k1 branch.
	// Kept here as a named constant for documentation; PQAccount itself
	// only ever uses branchPQ.
	branchClassical uint32 = 1
)

// PQAccount is an ML-DSA-65 native wallet account.
//
// Invariants:
//   - SchemeID is one of {WalletSchemeMLDSA44, WalletSchemeMLDSA65,
//     WalletSchemeMLDSA87}. The constructor refuses any other scheme.
//   - PublicKey is the packed FIPS-204 public key bytes; never nil.
//   - PrivateKey is the packed FIPS-204 private key bytes; never nil for
//     accounts produced by NewPQAccount. Callers MUST wrap it through the
//     keystore before persisting to disk.
//   - AccountID == DeriveAccountID(chainID, SchemeID, PublicKey).
//
// The struct is a deliberately flat data carrier — no methods that mutate
// state. Sign/Verify operate on a value receiver so the type is safe to
// pass by value or by pointer.
type PQAccount struct {
	AccountID      AccountID
	SchemeID       WalletSchemeID
	PublicKey      []byte
	PrivateKey     []byte
	Role           AccountRole
	DerivationPath string
}

// NewPQAccount derives a fresh ML-DSA-65 keypair from a domain-separated
// child seed under the F35 HD path.
//
//	masterSeed   — the BIP-39 seed (caller decides mnemonic provenance).
//	chainID      — uint32 network id, < 2^31 (BIP-32 hardening).
//	accountIdx   — leaf account index, < 2^31.
//	roleIdx      — hardened role index on branch 0' (< 2^31). Orthogonal
//	               from `role`: the HD path uses this verbatim, the cSHAKE
//	               customization is driven by `role`. Conventional indices
//	               (0=identity, 1=tx, 2=session) are returned by
//	               AccountRole.HDIndex but the caller may pass any value.
//	role         — identity | tx | session. Recovery role is refused; use
//	               NewSLHDSARecoveryAccount instead. Drives the cSHAKE
//	               customization string and is persisted on the struct.
//
// The keypair is generated as:
//
//	child_seed = m/44'/9000'/<chainID>'/0'/<roleIdx>'/<accountIdx>'
//	ξ          = cSHAKE-256(N="LUX_PQ_KEYGEN_V1",
//	                        S="LUX/WALLET/<ROLE>/V1",
//	                        child_seed,
//	                        outlen = 32)
//	(pk, sk)   = ML-DSA-65.KeyGen(ξ)
//
// The cSHAKE customization binds the keypair to its role. A leaked child
// seed cannot be re-used to derive a key of a different role because the
// customization string is mixed into the cSHAKE state before the seed.
func NewPQAccount(
	masterSeed []byte,
	chainID uint32,
	accountIdx uint32,
	roleIdx uint32,
	role AccountRole,
) (*PQAccount, error) {
	if len(masterSeed) == 0 {
		return nil, fmt.Errorf("account: masterSeed is empty")
	}
	if chainID >= bip32.FirstHardenedChild {
		return nil, fmt.Errorf("account: chainID %d must be < 2^31 (BIP-32 hardening limit)", chainID)
	}
	if accountIdx >= bip32.FirstHardenedChild {
		return nil, fmt.Errorf("account: accountIdx %d must be < 2^31 (BIP-32 hardening limit)", accountIdx)
	}
	if roleIdx >= bip32.FirstHardenedChild {
		return nil, fmt.Errorf("account: roleIdx %d must be < 2^31 (BIP-32 hardening limit)", roleIdx)
	}

	// Recovery role is reserved for SLH-DSA on a separate HD branch;
	// refuse it here so callers don't accidentally emit an ML-DSA
	// "recovery" key on the wallet branch.
	if role == AccountRoleRecovery {
		return nil, fmt.Errorf("account: role %q is reserved for SLH-DSA recovery accounts; use NewSLHDSARecoveryAccount", role)
	}

	customization, err := role.cshakeCustomization()
	if err != nil {
		return nil, err
	}

	// Walk the HD path m/44'/9000'/<chainID>'/0'/<roleIdx>'/<accountIdx>'.
	childSeed, derivationPath, err := derivePQChildSeed(masterSeed, chainID, roleIdx, accountIdx)
	if err != nil {
		return nil, err
	}

	// Domain-separate the child seed into ξ for FIPS-204 KeyGen.
	xiBytes, err := expandChildSeed(childSeed, customization, mldsa65.SeedSize)
	if err != nil {
		return nil, fmt.Errorf("account: expand child seed: %w", err)
	}
	var xi [mldsa65.SeedSize]byte
	copy(xi[:], xiBytes)

	pk, sk := mldsa65.NewKeyFromSeed(&xi)
	pubBytes := pk.Bytes()
	privBytes := sk.Bytes()

	accID, err := DeriveAccountID(chainID, WalletSchemeMLDSA65, pubBytes)
	if err != nil {
		return nil, fmt.Errorf("account: derive AccountID: %w", err)
	}

	return &PQAccount{
		AccountID:      accID,
		SchemeID:       WalletSchemeMLDSA65,
		PublicKey:      pubBytes,
		PrivateKey:     privBytes,
		Role:           role,
		DerivationPath: derivationPath,
	}, nil
}

// Sign signs a 48-byte digest (typically from TxAuthEnvelope.SigningDigest()).
//
// The 48-byte length is enforced — callers MUST hash to 384 bits before
// calling Sign, both to keep the on-wire transcript at the
// MinHashOutputBits floor that strict-PQ profiles enforce and to keep
// signature inputs deterministic-sized.
//
// Signing uses FIPS-204's "pure" Sign with a Lux-specific context string
// ("LUX/WALLET/TX/V1") so signatures issued by this wallet cannot be
// replayed against a different protocol/version. Signing is randomized
// (the FIPS-204 default).
func (a *PQAccount) Sign(digest [AccountIDSize]byte) ([]byte, error) {
	if a == nil {
		return nil, fmt.Errorf("account: PQAccount is nil")
	}
	if a.SchemeID != WalletSchemeMLDSA65 {
		return nil, fmt.Errorf("account: PQAccount.Sign currently supports only ML-DSA-65 (got %s)", a.SchemeID)
	}
	if len(a.PrivateKey) != mldsa65.PrivateKeySize {
		return nil, fmt.Errorf("account: private key is %d bytes, want %d", len(a.PrivateKey), mldsa65.PrivateKeySize)
	}

	sk, err := unpackMLDSA65Private(a.PrivateKey)
	if err != nil {
		return nil, err
	}

	ctx := []byte(CSHAKECustomizationTx)
	sig := make([]byte, mldsa65.SignatureSize)
	if err := mldsa65.SignTo(sk, digest[:], ctx, true, sig); err != nil {
		return nil, fmt.Errorf("account: ml-dsa-65 sign: %w", err)
	}
	return sig, nil
}

// Verify is a thin wrapper around the ML-DSA-65 verify routine. Returns
// true iff sig is a valid signature of digest under a.PublicKey with the
// "LUX/WALLET/TX/V1" context.
func (a *PQAccount) Verify(digest [AccountIDSize]byte, sig []byte) bool {
	if a == nil {
		return false
	}
	if a.SchemeID != WalletSchemeMLDSA65 {
		return false
	}
	if len(a.PublicKey) != mldsa65.PublicKeySize {
		return false
	}
	if len(sig) != mldsa65.SignatureSize {
		return false
	}
	pk, err := unpackMLDSA65Public(a.PublicKey)
	if err != nil {
		return false
	}
	return mldsa65.Verify(pk, digest[:], []byte(CSHAKECustomizationTx), sig)
}

// derivePQChildSeed walks m/44'/9000'/<chainID>'/0'/<roleIdx>'/<accountIdx>'
// and returns the 32-byte BIP-32 child key together with the textual
// derivation path for record-keeping. Every level is hardened; chainID,
// roleIdx, and accountIdx must each be < 2^31.
func derivePQChildSeed(masterSeed []byte, chainID uint32, roleIdx uint32, accountIdx uint32) ([]byte, string, error) {
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
	branch, err := chain.NewChildKey(bip32.FirstHardenedChild + branchPQ)
	if err != nil {
		return nil, "", fmt.Errorf("account: derive branch 0' (ML-DSA): %w", err)
	}
	roleNode, err := branch.NewChildKey(bip32.FirstHardenedChild + roleIdx)
	if err != nil {
		return nil, "", fmt.Errorf("account: derive role %d': %w", roleIdx, err)
	}
	leaf, err := roleNode.NewChildKey(bip32.FirstHardenedChild + accountIdx)
	if err != nil {
		return nil, "", fmt.Errorf("account: derive account %d': %w", accountIdx, err)
	}
	path := fmt.Sprintf("m/44'/9000'/%d'/0'/%d'/%d'", chainID, roleIdx, accountIdx)
	return leaf.Key, path, nil
}

// unpackMLDSA65Public unpacks a packed ML-DSA-65 public key. Wraps the
// circl Unpack call so callers can reason about errors instead of having
// to know unpack has no failure mode.
func unpackMLDSA65Public(b []byte) (*mldsa65.PublicKey, error) {
	if len(b) != mldsa65.PublicKeySize {
		return nil, fmt.Errorf("account: public key is %d bytes, want %d", len(b), mldsa65.PublicKeySize)
	}
	var buf [mldsa65.PublicKeySize]byte
	copy(buf[:], b)
	var pk mldsa65.PublicKey
	pk.Unpack(&buf)
	return &pk, nil
}

// unpackMLDSA65Private unpacks a packed ML-DSA-65 private key.
func unpackMLDSA65Private(b []byte) (*mldsa65.PrivateKey, error) {
	if len(b) != mldsa65.PrivateKeySize {
		return nil, fmt.Errorf("account: private key is %d bytes, want %d", len(b), mldsa65.PrivateKeySize)
	}
	var buf [mldsa65.PrivateKeySize]byte
	copy(buf[:], b)
	var sk mldsa65.PrivateKey
	sk.Unpack(&buf)
	return &sk, nil
}
