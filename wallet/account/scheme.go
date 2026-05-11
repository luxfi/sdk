// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package account implements PQ-native wallet account types.
//
// A wallet account is a triple (scheme, role, key material) together with a
// 48-byte AccountID. AccountIDs are derived deterministically from the
// public key and the chain-id under cSHAKE-256 with NIST SP 800-185
// customization strings, so the same key under different (chain, scheme)
// pairs yields different AccountIDs. This is the safety property that lets
// a strict-PQ chain refuse a key that was issued for a permissive chain
// without re-running classification.
//
// The scheme enum mirrors consensus/config.SigSchemeID byte assignments so
// the wire-level identifier of a wallet's signing scheme matches what the
// consensus profile pins. The wallet package declares its own enum (rather
// than importing config.WalletSchemeID) because the consensus
// security_profile.go has not yet committed the WalletSchemeID type; when
// that lands, this package's IDs will map 1:1 by byte value.
//
// Recovery accounts live in a separate enum (RecoverySchemeID) because the
// signature scheme used for cold-key recovery is stateless hash-based
// (SLH-DSA / FIPS 205) and is intentionally NOT in the wallet's hot
// signing path. RecoverySchemeID values are disjoint from WalletSchemeID
// values so a recovery key cannot be mistaken for a hot key at the type
// boundary.
package account

import "fmt"

// WalletSchemeID is the wire byte that identifies which signature scheme a
// wallet account signs under. Byte values intentionally match the raw
// ML-DSA block (0x40..) defined in
// github.com/luxfi/consensus/config.SigSchemeID so a wallet's declared
// scheme is wire-compatible with the consensus identity-scheme byte.
//
// 0x00..0x3F — reserved (classical / non-PQ space, kept disjoint from PQ).
//
//	0x10  — secp256k1 (classical, allowed only under the classical-compat
//	        unsafe flag; strict-PQ chains refuse this).
//
// 0x40..0x4F — raw FIPS 204 ML-DSA, single-party identity signatures.
//
//	0x41  — ML-DSA-44 (NIST Cat 2; dev/testnet only).
//	0x42  — ML-DSA-65 (NIST Cat 3; production default).
//	0x43  — ML-DSA-87 (NIST Cat 5; high-value identities).
type WalletSchemeID uint8

const (
	WalletSchemeNone      WalletSchemeID = 0x00
	WalletSchemeSecp256k1 WalletSchemeID = 0x10 // classical-compat (unsafe on strict-PQ chains)

	WalletSchemeMLDSA44 WalletSchemeID = 0x41
	WalletSchemeMLDSA65 WalletSchemeID = 0x42 // production default
	WalletSchemeMLDSA87 WalletSchemeID = 0x43
)

// String returns the canonical lowercase scheme name.
func (s WalletSchemeID) String() string {
	switch s {
	case WalletSchemeNone:
		return "none"
	case WalletSchemeSecp256k1:
		return "secp256k1"
	case WalletSchemeMLDSA44:
		return "ml-dsa-44"
	case WalletSchemeMLDSA65:
		return "ml-dsa-65"
	case WalletSchemeMLDSA87:
		return "ml-dsa-87"
	default:
		return fmt.Sprintf("wallet-scheme(0x%02x)", uint8(s))
	}
}

// IsPostQuantum reports whether this scheme is a NIST-aligned PQ signature.
// Strict-PQ chains MUST refuse accounts whose scheme is not post-quantum.
func (s WalletSchemeID) IsPostQuantum() bool {
	return s == WalletSchemeMLDSA44 ||
		s == WalletSchemeMLDSA65 ||
		s == WalletSchemeMLDSA87
}

// IsClassicalCompat reports whether this scheme is the legacy classical
// signature path. Allowed only on chains that opt into the
// classical-compat unsafe flag at genesis time; strict-PQ chains refuse it.
func (s WalletSchemeID) IsClassicalCompat() bool {
	return s == WalletSchemeSecp256k1
}

// RecoverySchemeID is the wire byte for stateless hash-based recovery
// signatures. Disjoint from WalletSchemeID so a recovery key cannot be
// confused with a hot signing key at the type boundary.
//
// 0x60..0x6F — SLH-DSA (FIPS 205), SHAKE-based parameter sets only.
//
//	0x61  — SLH-DSA-SHAKE-128s (NIST Cat 1)
//	0x62  — SLH-DSA-SHAKE-192s (NIST Cat 3; production default for cold recovery)
//	0x63  — SLH-DSA-SHAKE-256s (NIST Cat 5)
type RecoverySchemeID uint8

const (
	RecoverySchemeNone       RecoverySchemeID = 0x00
	RecoverySchemeSLHDSA128s RecoverySchemeID = 0x61
	RecoverySchemeSLHDSA192s RecoverySchemeID = 0x62 // production default
	RecoverySchemeSLHDSA256s RecoverySchemeID = 0x63
)

// String returns the canonical lowercase scheme name.
func (s RecoverySchemeID) String() string {
	switch s {
	case RecoverySchemeNone:
		return "none"
	case RecoverySchemeSLHDSA128s:
		return "slh-dsa-shake-128s"
	case RecoverySchemeSLHDSA192s:
		return "slh-dsa-shake-192s"
	case RecoverySchemeSLHDSA256s:
		return "slh-dsa-shake-256s"
	default:
		return fmt.Sprintf("recovery-scheme(0x%02x)", uint8(s))
	}
}

// AccountRole identifies what an account is for. Roles map to distinct HD
// derivation indices so the same master seed cannot accidentally produce
// the same key under two roles, and so a chain can refuse a key whose
// declared role doesn't match the operation it signs.
type AccountRole string

const (
	AccountRoleIdentity AccountRole = "identity"
	AccountRoleTx       AccountRole = "tx"
	AccountRoleRecovery AccountRole = "recovery"
	AccountRoleSession  AccountRole = "session"
)

// HDIndex returns the conventional HD-derivation role index for this
// role. Callers of NewPQAccount may pass this verbatim as the roleIdx
// argument, or pin a different index for application-specific reasons
// (multiple tx keys per identity, etc.). Indices are stable across
// implementations; changing them is a wallet-incompatible rewrite.
//
// AccountRoleRecovery returns an error because SLH-DSA recovery keys are
// NOT derived on the wallet ML-DSA branch; they have their own derivation
// path (see recovery.go).
func (r AccountRole) HDIndex() (uint32, error) {
	switch r {
	case AccountRoleIdentity:
		return 0, nil
	case AccountRoleTx:
		return 1, nil
	case AccountRoleSession:
		return 2, nil
	case AccountRoleRecovery:
		return 0, fmt.Errorf("account: role %q is for SLH-DSA recovery accounts; use NewSLHDSARecoveryAccount", r)
	default:
		return 0, fmt.Errorf("account: unknown role %q", r)
	}
}

// cshakeCustomization returns the NIST SP 800-185 cSHAKE customization
// string bound to this role. The customization string is what makes child
// seeds derived for one role unusable as seeds for another, even if an
// attacker recovers a parent BIP-32 child seed.
func (r AccountRole) cshakeCustomization() (string, error) {
	switch r {
	case AccountRoleIdentity:
		return CSHAKECustomizationIdentity, nil
	case AccountRoleTx:
		return CSHAKECustomizationTx, nil
	case AccountRoleSession:
		return CSHAKECustomizationSession, nil
	case AccountRoleRecovery:
		return CSHAKECustomizationRecovery, nil
	default:
		return "", fmt.Errorf("account: unknown role %q", r)
	}
}
