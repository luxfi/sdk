// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package account

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

// keystoreMagic identifies a Lux PQ wallet keystore blob on disk.
// "LUXKSV1\x00" → 8 bytes. Allows a future LUXKSV2 with a different
// wrapper (ML-KEM-768 key wrap) to coexist with V1 blobs.
var keystoreMagic = [8]byte{'L', 'U', 'X', 'K', 'S', 'V', '1', 0x00}

const (
	// argonTime is the Argon2id time cost. Conservative for a desktop
	// keystore unlock (~250ms on a 2024-era laptop with default memory
	// cost). Tune up before shipping a server-side keystore.
	argonTime uint32 = 3
	// argonMemoryKB is the Argon2id memory cost in kibibytes.
	// 64 MiB. RFC 9106 §7.4 recommendation for password-hashing.
	argonMemoryKB uint32 = 64 * 1024
	// argonThreads is the Argon2id parallelism cost.
	argonThreads uint8 = 4
	// argonKeyLen is the wrapping-key length (AES-256-GCM key).
	argonKeyLen uint32 = 32
	// argonSaltLen is the Argon2id salt length. 16 bytes per RFC 9106.
	argonSaltLen = 16
	// aesGCMNonceLen is the AES-GCM nonce length (96 bits per NIST SP
	// 800-38D).
	aesGCMNonceLen = 12
)

// KeystoreBlob is the persisted on-disk shape. JSON-encoded so the blob
// is human-inspectable for audit purposes; the actual private-key
// material is inside Ciphertext and is AES-256-GCM sealed under a key
// derived from the user's passphrase via Argon2id.
//
// Forward-compat: a future V2 blob would have a different Magic and a
// distinct field set (e.g., a KEM-wrapped DEK). Reading code dispatches
// on Magic before unmarshalling.
type KeystoreBlob struct {
	Magic     [8]byte `json:"magic"`      // keystoreMagic
	Version   uint8   `json:"version"`    // 1
	SchemeID  uint8   `json:"scheme_id"`  // WalletSchemeID byte
	Role      string  `json:"role"`       // AccountRole string
	ChainID   uint32  `json:"chain_id"`   // for AccountID re-derivation
	Path      string  `json:"path"`       // derivation path, audit only
	AccountID string  `json:"account_id"` // hex(48), for human lookup

	Salt       []byte `json:"salt"`       // 16 bytes
	Nonce      []byte `json:"nonce"`      // 12 bytes (AES-GCM)
	Ciphertext []byte `json:"ciphertext"` // GCM(pubkey || privkey)
	PubkeyLen  uint32 `json:"pubkey_len"` // splits the plaintext

	// Argon2id parameters frozen at seal-time so a passphrase that
	// took 64 MiB to derive in 2026 still unlocks in 2032 even if
	// defaults shift.
	ArgonTime    uint32 `json:"argon_time"`
	ArgonMemory  uint32 `json:"argon_memory_kib"`
	ArgonThreads uint8  `json:"argon_threads"`
}

// SealPQAccount AEAD-seals an account's private key under a passphrase
// using Argon2id (RFC 9106) → AES-256-GCM. Returns a JSON-encoded
// keystore blob suitable for persistence. The passphrase is consumed via
// a copy; callers SHOULD zeroize their own copy after the call returns.
//
// Future migration: a V2 SealPQAccountKEM will land that wraps the DEK
// under ML-KEM-768 derived from passphrase+salt, so the Argon2id step
// becomes the KEM-shared-secret step. The blob Magic switches from
// LUXKSV1 to LUXKSV2 then; readers are dispatched by Magic.
func SealPQAccount(account *PQAccount, passphrase []byte) ([]byte, error) {
	if account == nil {
		return nil, fmt.Errorf("keystore: SealPQAccount called with nil account")
	}
	if len(passphrase) == 0 {
		return nil, fmt.Errorf("keystore: passphrase is empty")
	}
	if len(account.PublicKey) == 0 || len(account.PrivateKey) == 0 {
		return nil, fmt.Errorf("keystore: account has empty key material")
	}

	salt := make([]byte, argonSaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("keystore: read salt: %w", err)
	}
	nonce := make([]byte, aesGCMNonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("keystore: read nonce: %w", err)
	}

	dek := argon2.IDKey(passphrase, salt, argonTime, argonMemoryKB, argonThreads, argonKeyLen)
	gcm, err := newGCM(dek)
	if err != nil {
		return nil, err
	}

	// Plaintext layout: pubkey || privkey. The PubkeyLen field on the
	// blob is the split offset; no length prefixes in the ciphertext
	// itself so the AEAD tag covers exactly (header || pub || priv).
	plain := make([]byte, 0, len(account.PublicKey)+len(account.PrivateKey))
	plain = append(plain, account.PublicKey...)
	plain = append(plain, account.PrivateKey...)

	// Bind the AEAD additional-data to the blob header so an attacker
	// cannot swap a ciphertext from one (chainID, scheme, role) into
	// another without invalidating the tag.
	aad := authData(account.SchemeID, account.Role, account.ChainID(), account.DerivationPath)
	ct := gcm.Seal(nil, nonce, plain, aad)

	// Zeroize the in-memory DEK before returning.
	for i := range dek {
		dek[i] = 0
	}

	blob := KeystoreBlob{
		Magic:        keystoreMagic,
		Version:      1,
		SchemeID:     uint8(account.SchemeID),
		Role:         string(account.Role),
		ChainID:      account.ChainID(),
		Path:         account.DerivationPath,
		AccountID:    fmt.Sprintf("%x", account.AccountID[:]),
		Salt:         salt,
		Nonce:        nonce,
		Ciphertext:   ct,
		PubkeyLen:    uint32(len(account.PublicKey)),
		ArgonTime:    argonTime,
		ArgonMemory:  argonMemoryKB,
		ArgonThreads: argonThreads,
	}
	return json.Marshal(blob)
}

// OpenPQAccount inverts SealPQAccount. The caller supplies the passphrase
// and the blob bytes; on success the account is fully reconstructed
// including its AccountID (re-derived from the unsealed pubkey rather
// than trusted from the blob, so tampering with the AccountID hex field
// is detected at unseal time).
func OpenPQAccount(blobBytes []byte, passphrase []byte) (*PQAccount, error) {
	if len(passphrase) == 0 {
		return nil, fmt.Errorf("keystore: passphrase is empty")
	}

	var blob KeystoreBlob
	if err := json.Unmarshal(blobBytes, &blob); err != nil {
		return nil, fmt.Errorf("keystore: unmarshal blob: %w", err)
	}
	if blob.Magic != keystoreMagic {
		return nil, fmt.Errorf("keystore: bad magic (want LUXKSV1, got %q)", blob.Magic[:])
	}
	if blob.Version != 1 {
		return nil, fmt.Errorf("keystore: unsupported version %d", blob.Version)
	}
	if len(blob.Salt) != argonSaltLen {
		return nil, fmt.Errorf("keystore: bad salt length %d", len(blob.Salt))
	}
	if len(blob.Nonce) != aesGCMNonceLen {
		return nil, fmt.Errorf("keystore: bad nonce length %d", len(blob.Nonce))
	}

	dek := argon2.IDKey(passphrase, blob.Salt, blob.ArgonTime, blob.ArgonMemory, blob.ArgonThreads, argonKeyLen)
	gcm, err := newGCM(dek)
	if err != nil {
		return nil, err
	}

	aad := authData(WalletSchemeID(blob.SchemeID), AccountRole(blob.Role), blob.ChainID, blob.Path)
	plain, err := gcm.Open(nil, blob.Nonce, blob.Ciphertext, aad)
	for i := range dek {
		dek[i] = 0
	}
	if err != nil {
		return nil, fmt.Errorf("keystore: gcm open: %w", err)
	}
	if uint32(len(plain)) < blob.PubkeyLen {
		return nil, fmt.Errorf("keystore: plaintext shorter than pubkey length")
	}

	pub := make([]byte, blob.PubkeyLen)
	priv := make([]byte, uint32(len(plain))-blob.PubkeyLen)
	copy(pub, plain[:blob.PubkeyLen])
	copy(priv, plain[blob.PubkeyLen:])

	// Re-derive the AccountID rather than trust the blob's hex field.
	scheme := WalletSchemeID(blob.SchemeID)
	accID, err := DeriveAccountID(blob.ChainID, scheme, pub)
	if err != nil {
		return nil, fmt.Errorf("keystore: re-derive AccountID: %w", err)
	}

	return &PQAccount{
		AccountID:      accID,
		SchemeID:       scheme,
		PublicKey:      pub,
		PrivateKey:     priv,
		Role:           AccountRole(blob.Role),
		DerivationPath: blob.Path,
	}, nil
}

// ChainID parses the chain id back out of the derivation path. Returns 0
// for an account with no path (an account that was constructed by hand
// rather than through NewPQAccount). The path always starts with
// "m/44'/9000'/<chainID>'/" so a single Scanf suffices.
func (a *PQAccount) ChainID() uint32 {
	var cid uint32
	// "m/44'/9000'/<cid>'/0'/..." — Sscanf reads up to the first '/'
	// after <cid>'.
	if _, err := fmt.Sscanf(a.DerivationPath, "m/44'/9000'/%d", &cid); err != nil {
		return 0
	}
	return cid
}

// newGCM builds an AES-256-GCM AEAD from a 32-byte key.
func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("keystore: AES key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("keystore: aes new: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("keystore: gcm new: %w", err)
	}
	return gcm, nil
}

// authData is the AEAD additional-authenticated-data bound to a sealed
// account. Includes the on-wire scheme byte, the role string, the chain
// id, and the derivation path — all the metadata that lives unencrypted
// on the blob header. Binding them through AAD means an attacker cannot
// move a sealed ciphertext from one account-shape to another.
func authData(scheme WalletSchemeID, role AccountRole, chainID uint32, path string) []byte {
	var cidBytes [4]byte
	binary.BigEndian.PutUint32(cidBytes[:], chainID)
	buf := make([]byte, 0, 1+len(role)+4+len(path)+len("LUXKSV1-AAD"))
	buf = append(buf, []byte("LUXKSV1-AAD")...)
	buf = append(buf, byte(scheme))
	buf = append(buf, []byte(role)...)
	buf = append(buf, cidBytes[:]...)
	buf = append(buf, []byte(path)...)
	return buf
}
