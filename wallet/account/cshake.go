// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package account

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/sha3"
)

// CSHAKE customization strings, per NIST SP 800-185 §3.3 (cSHAKE). The
// customization string ("S") is what makes two cSHAKE invocations with the
// same input message produce independent outputs. We use it three ways:
//
//   1. Per-role child-seed expansion. The BIP-32 child seed at
//      m/44'/9000'/<chain>'/0'/<role>'/<account>' is expanded through
//      cSHAKE-256 with the role's customization string into the 32-byte ξ
//      that FIPS 204 §5.1 KeyGen consumes. Without the customization, a
//      single leaked child seed would be re-usable across roles.
//
//   2. AccountID derivation. AccountIDs hash the (chain, scheme, pubkey)
//      triple under cSHAKE-256 with the "ACCOUNT_ID" customization so the
//      same pubkey under different chains/schemes produces independent
//      AccountIDs.
//
//   3. Recovery-seed expansion. SLH-DSA recovery accounts use the same
//      role-customized cSHAKE-256 to expand the BIP-32 child seed into the
//      three n-byte seeds (skSeed, skPrf, pkSeed) that FIPS 205 KeyGen
//      consumes.
//
// The strings are versioned ("/V1") so a future scheme rotation (e.g.,
// migration to ML-DSA-87 by default or to a new derivation tree shape) can
// land without invalidating already-issued V1 accounts.
const (
	CSHAKECustomizationIdentity  = "LUX/WALLET/IDENTITY/V1"
	CSHAKECustomizationTx        = "LUX/WALLET/TX/V1"
	CSHAKECustomizationSession   = "LUX/WALLET/SESSION/V1"
	CSHAKECustomizationRecovery  = "LUX/WALLET/RECOVERY/V1"
	CSHAKECustomizationAccountID = "LUX/WALLET/ACCOUNT_ID/V1"
)

// accountIDLabel is the function-name (N) prefix passed to cSHAKE-256 for
// AccountID derivation. NIST SP 800-185 §3.3 reserves N for the calling
// function's name; AccountID is a Lux-specific function.
const accountIDLabel = "LUX_ACCOUNT_V1"

// AccountIDSize is the fixed 48-byte length of an AccountID. 48 bytes ==
// 384 bits == matches the SHAKE256-384 output the consensus layer expects
// for identity hashes (ChainSecurityProfile.MinHashOutputBits >= 384 on
// strict-PQ). Bigger AccountIDs would not give more collision resistance
// (we're already past the SHA-384 birthday bound for any realistic chain
// size) and smaller ones would force a hash-suite mismatch with strict-PQ.
const AccountIDSize = 48

// AccountID is the 48-byte deterministic identifier for a wallet account.
// Computed as cSHAKE-256(N="LUX_ACCOUNT_V1", S="LUX/WALLET/ACCOUNT_ID/V1",
// msg = u32be(chain_id) || u8(scheme) || pubkey, outlen = 48 bytes).
type AccountID [AccountIDSize]byte

// DeriveAccountID returns the canonical 48-byte AccountID for a (chain,
// scheme, pubkey) triple. Pure function; same inputs → same output.
//
// Errors only on pubkey == nil. Empty pubkey is permitted (yields a
// well-defined "uninitialised account" AccountID); callers that want to
// reject empty pubkeys should do so before calling.
func DeriveAccountID(chainID uint32, scheme WalletSchemeID, pubkey []byte) (AccountID, error) {
	if pubkey == nil {
		return AccountID{}, fmt.Errorf("account: pubkey is nil")
	}

	h := sha3.NewCShake256([]byte(accountIDLabel), []byte(CSHAKECustomizationAccountID))

	var chainBytes [4]byte
	binary.BigEndian.PutUint32(chainBytes[:], chainID)
	if _, err := h.Write(chainBytes[:]); err != nil {
		return AccountID{}, fmt.Errorf("cshake write chain: %w", err)
	}
	if _, err := h.Write([]byte{byte(scheme)}); err != nil {
		return AccountID{}, fmt.Errorf("cshake write scheme: %w", err)
	}
	if _, err := h.Write(pubkey); err != nil {
		return AccountID{}, fmt.Errorf("cshake write pubkey: %w", err)
	}

	var out AccountID
	if _, err := h.Read(out[:]); err != nil {
		return AccountID{}, fmt.Errorf("cshake read account-id: %w", err)
	}
	return out, nil
}

// expandChildSeed expands a BIP-32 child seed into a fixed-length keygen
// input under a role-bound cSHAKE-256 customization. The function-name (N)
// is "LUX_PQ_KEYGEN_V1" so the same role customization is unambiguous
// across (a) AccountID derivation and (b) keygen seed derivation: callers
// reading the cSHAKE transcript can tell which Lux function produced an
// output by looking at N, even if the customization (S) is the same.
//
// outLen must be > 0. The output is suitable for direct use as a FIPS-204
// ξ (outLen = 32) or as the random-stream input to a deterministic
// FIPS-205 keygen (outLen = 3 * n).
func expandChildSeed(childSeed []byte, customization string, outLen int) ([]byte, error) {
	if len(childSeed) == 0 {
		return nil, fmt.Errorf("account: child seed is empty")
	}
	if outLen <= 0 {
		return nil, fmt.Errorf("account: outLen=%d must be positive", outLen)
	}
	if customization == "" {
		return nil, fmt.Errorf("account: customization is empty")
	}

	h := sha3.NewCShake256([]byte("LUX_PQ_KEYGEN_V1"), []byte(customization))
	if _, err := h.Write(childSeed); err != nil {
		return nil, fmt.Errorf("cshake write child seed: %w", err)
	}
	out := make([]byte, outLen)
	if _, err := h.Read(out); err != nil {
		return nil, fmt.Errorf("cshake read: %w", err)
	}
	return out, nil
}
