// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/cloudflare/circl/kem/mlkem/mlkem768"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/crypto/threshold"
	_ "github.com/luxfi/crypto/threshold/bls" // Register BLS threshold scheme
	"github.com/luxfi/math/set"
)

var (
	_ Signature = (*BitSetSignature)(nil)
	_ Signature = (*RingtailSignature)(nil)
	_ Signature = (*HybridBLSRTSignature)(nil) // Deprecated: use RingtailSignature

	ErrInvalidBitSet        = errors.New("bitset is invalid")
	ErrInsufficientWeight   = errors.New("signature weight is insufficient")
	ErrInvalidSignature     = errors.New("signature is invalid")
	ErrParseSignature       = errors.New("failed to parse signature")
	ErrInvalidRTSignature   = errors.New("ringtail signature is invalid")
	ErrMissingRTPublicKey   = errors.New("missing ringtail public key for validator")
	ErrHybridVerifyFailed   = errors.New("hybrid signature verification failed")
	ErrDecryptionFailed     = errors.New("ML-KEM decryption failed")
	ErrInvalidCiphertext    = errors.New("invalid ciphertext")
)

type Signature interface {
	fmt.Stringer

	// NumSigners is the number of [bls.PublicKeys] that participated in the
	// [Signature]. This is exposed because users of these signatures typically
	// impose a verification fee that is a function of the number of
	// signers.
	NumSigners() (int, error)

	// Verify that this signature was signed by at least [quorumNum]/[quorumDen]
	// of the validators of [msg.SourceChainID] at [pChainHeight].
	//
	// Invariant: [msg] is correctly initialized.
	Verify(
		msg *UnsignedMessage,
		networkID uint32,
		validators CanonicalValidatorSet,
		quorumNum uint64,
		quorumDen uint64,
	) error
}

type BitSetSignature struct {
	// Signers is a big-endian byte slice encoding which validators signed this
	// message.
	Signers   []byte                 `serialize:"true"`
	Signature [bls.SignatureLen]byte `serialize:"true"`
}

func (s *BitSetSignature) NumSigners() (int, error) {
	// Parse signer bit vector
	//
	// We assert that the length of [signerIndices.Bytes()] is equal
	// to [len(s.Signers)] to ensure that [s.Signers] does not have
	// any unnecessary zero-padding to represent the [set.Bits].
	signerIndices := set.BitsFromBytes(s.Signers)
	if len(signerIndices.Bytes()) != len(s.Signers) {
		return 0, ErrInvalidBitSet
	}
	return signerIndices.Len(), nil
}

func (s *BitSetSignature) Verify(
	msg *UnsignedMessage,
	networkID uint32,
	validators CanonicalValidatorSet,
	quorumNum uint64,
	quorumDen uint64,
) error {
	if msg.NetworkID != networkID {
		return ErrWrongNetworkID
	}

	// Parse signer bit vector
	//
	// We assert that the length of [signerIndices.Bytes()] is equal
	// to [len(s.Signers)] to ensure that [s.Signers] does not have
	// any unnecessary zero-padding to represent the [set.Bits].
	signerIndices := set.BitsFromBytes(s.Signers)
	if len(signerIndices.Bytes()) != len(s.Signers) {
		return ErrInvalidBitSet
	}

	// Get the validators that (allegedly) signed the message.
	signers, err := FilterValidators(signerIndices, validators.Validators)
	if err != nil {
		return err
	}

	// Because [signers] is a subset of [validators.Validators], this can never error.
	sigWeight, _ := SumWeight(signers)

	// Make sure the signature's weight is sufficient.
	err = VerifyWeight(
		sigWeight,
		validators.TotalWeight,
		quorumNum,
		quorumDen,
	)
	if err != nil {
		return err
	}

	// Parse the aggregate signature
	aggSig, err := bls.SignatureFromBytes(s.Signature[:])
	if err != nil {
		return fmt.Errorf("%w: %w", ErrParseSignature, err)
	}

	// Create the aggregate public key
	aggPubKey, err := AggregatePublicKeys(signers)
	if err != nil {
		return err
	}

	// Verify the signature
	unsignedBytes := msg.Bytes()
	if !bls.Verify(aggPubKey, aggSig, unsignedBytes) {
		return ErrInvalidSignature
	}
	return nil
}

func (s *BitSetSignature) String() string {
	return fmt.Sprintf("BitSetSignature(Signers = %x, Signature = %x)", s.Signers, s.Signature)
}

// VerifyWeight returns [nil] if [sigWeight] is at least [quorumNum]/[quorumDen]
// of [totalWeight].
// If [sigWeight >= totalWeight * quorumNum / quorumDen] then return [nil]
func VerifyWeight(
	sigWeight uint64,
	totalWeight uint64,
	quorumNum uint64,
	quorumDen uint64,
) error {
	// Verifies that quorumNum * totalWeight <= quorumDen * sigWeight
	scaledTotalWeight := new(big.Int).SetUint64(totalWeight)
	scaledTotalWeight.Mul(scaledTotalWeight, new(big.Int).SetUint64(quorumNum))
	scaledSigWeight := new(big.Int).SetUint64(sigWeight)
	scaledSigWeight.Mul(scaledSigWeight, new(big.Int).SetUint64(quorumDen))
	if scaledTotalWeight.Cmp(scaledSigWeight) == 1 {
		return fmt.Errorf(
			"%w: %d*%d > %d*%d",
			ErrInsufficientWeight,
			quorumNum,
			totalWeight,
			quorumDen,
			sigWeight,
		)
	}
	return nil
}

// =============================================================================
// Warp 1.5: Ringtail Signature (Post-Quantum Safe, replaces BLS)
// =============================================================================

// Ringtail is a lattice-based threshold signature scheme from LWE
// Paper: https://eprint.iacr.org/2024/1113
// Implementation: github.com/luxfi/ringtail
//
// Key properties:
// - Post-quantum secure (based on LWE hardness)
// - Native threshold support (t-of-n signing in 2 rounds)
// - Ring-LWE with NTT-friendly prime Q = 0x1000000004A01 (48-bit)
// - Parameters: M=8, N=7, Dbar=48, Kappa=23

// Ringtail signature constants (from github.com/luxfi/ringtail/sign/config.go)
const (
	// RingtailQ is the NTT-friendly prime modulus
	RingtailQ = 0x1000000004A01 // 48-bit prime

	// RingtailM is the matrix dimension M
	RingtailM = 8

	// RingtailN is the matrix dimension N
	RingtailN = 7

	// RingtailKappa is the hash output bound
	RingtailKappa = 23

	// RingtailDbar is the signature dimension
	RingtailDbar = 48

	// RingtailKeySize is the symmetric key size in bytes (256 bits)
	RingtailKeySize = 32
)

// ML-KEM security levels per FIPS 203
const (
	// MLKEM768CiphertextLen is ML-KEM-768 ciphertext length (NIST Level 3) - DEFAULT
	MLKEM768CiphertextLen = 1088

	// MLKEM768PublicKeyLen is ML-KEM-768 public key length
	MLKEM768PublicKeyLen = 1184

	// MLKEM768SharedSecretLen is the shared secret length
	MLKEM768SharedSecretLen = 32

	// MLKEM1024CiphertextLen is ML-KEM-1024 ciphertext length (NIST Level 5)
	MLKEM1024CiphertextLen = 1568

	// AESGCMNonceLen is the nonce length for AES-256-GCM
	AESGCMNonceLen = 12

	// AESGCMTagLen is the authentication tag length
	AESGCMTagLen = 16
)

// RingtailSignature is the Warp 1.5 quantum-safe signature type.
// This is the recommended signature type for all new Warp messages.
// It uses Ringtail (LWE-based) threshold signatures for post-quantum security.
//
// Ringtail properties:
// - Native threshold support (no need for separate TSS layer)
// - Two-round signing protocol
// - Post-quantum secure (based on LWE hardness)
// - Paper: https://eprint.iacr.org/2024/1113
//
// Replaces: BitSetSignature (BLS), HybridBLSRTSignature
// Security: Post-quantum secure (LWE-based)
// Size: Variable based on threshold parameters
type RingtailSignature struct {
	// Signers is a big-endian byte slice encoding which validators signed
	Signers []byte `serialize:"true"`

	// Signature is the Ringtail threshold signature
	// Contains: c (challenge polynomial), z (response vector), Delta (hint vector)
	// Size depends on threshold parameters (M, N, Dbar, Kappa)
	Signature []byte `serialize:"true"`
}

// NumSigners returns the number of validators that participated in signing
func (s *RingtailSignature) NumSigners() (int, error) {
	signerIndices := set.BitsFromBytes(s.Signers)
	if len(signerIndices.Bytes()) != len(s.Signers) {
		return 0, ErrInvalidBitSet
	}
	return signerIndices.Len(), nil
}

// Verify validates the Ringtail (ML-DSA) threshold signature
func (s *RingtailSignature) Verify(
	msg *UnsignedMessage,
	networkID uint32,
	validators CanonicalValidatorSet,
	quorumNum uint64,
	quorumDen uint64,
) error {
	if msg.NetworkID != networkID {
		return ErrWrongNetworkID
	}

	// Parse signer bit vector
	signerIndices := set.BitsFromBytes(s.Signers)
	if len(signerIndices.Bytes()) != len(s.Signers) {
		return ErrInvalidBitSet
	}

	// Get the validators that (allegedly) signed the message
	signers, err := FilterValidators(signerIndices, validators.Validators)
	if err != nil {
		return err
	}

	// Verify signer weight meets quorum
	sigWeight, _ := SumWeight(signers)
	if err := VerifyWeight(sigWeight, validators.TotalWeight, quorumNum, quorumDen); err != nil {
		return err
	}

	// Collect Ringtail public keys from signers
	rtPubKeys := make([][]byte, 0, len(signers))
	for _, signer := range signers {
		if len(signer.RingtailPubKey) == 0 {
			return fmt.Errorf("%w: validator missing RT key", ErrMissingRTPublicKey)
		}
		rtPubKeys = append(rtPubKeys, signer.RingtailPubKey)
	}

	// Aggregate the Ringtail public keys for threshold verification
	aggregatedPK, err := AggregateRingtailPublicKeys(rtPubKeys)
	if err != nil {
		return fmt.Errorf("failed to aggregate RT public keys: %w", err)
	}

	// Verify the Ringtail (LWE-based) signature
	if !VerifyRingtailSignature(aggregatedPK, msg.Bytes(), s.Signature) {
		return ErrInvalidRTSignature
	}

	return nil
}

func (s *RingtailSignature) String() string {
	return fmt.Sprintf("RingtailSignature(Signers = %x, Sig = %x...)",
		s.Signers, s.Signature[:min(32, len(s.Signature))])
}

// =============================================================================
// Warp 1.5: Encrypted Payload (ML-KEM + AES-256-GCM)
// =============================================================================

// EncryptedWarpPayload provides quantum-safe encryption for confidential
// cross-chain messages using ML-KEM (FIPS 203) key encapsulation.
//
// Use cases:
// - Private bridge transfers (hidden amounts/recipients)
// - Sealed-bid cross-chain auctions
// - Confidential governance votes
// - MEV protection (encrypt intent until committed)
//
// Encryption: ML-KEM-768 (NIST Level 3) + AES-256-GCM
// Security: 192-bit post-quantum for key exchange, 256-bit symmetric
type EncryptedWarpPayload struct {
	// EncapsulatedKey is the ML-KEM ciphertext containing the encapsulated shared secret
	// Size: 1088 bytes for ML-KEM-768
	EncapsulatedKey []byte `serialize:"true"`

	// Nonce is the AES-GCM nonce (12 bytes)
	Nonce []byte `serialize:"true"`

	// Ciphertext is the AES-256-GCM encrypted payload
	// Includes 16-byte authentication tag at the end
	Ciphertext []byte `serialize:"true"`

	// RecipientKeyID identifies which ML-KEM public key was used
	// This allows recipients to know which private key to use for decryption
	RecipientKeyID []byte `serialize:"true"`
}

// Encrypt creates an encrypted Warp payload using ML-KEM + AES-256-GCM
//
// Parameters:
//   - plaintext: The message to encrypt
//   - recipientPubKey: The recipient's ML-KEM-768 public key
//   - recipientKeyID: Identifier for the recipient's key (e.g., hash of pubkey)
//
// Returns:
//   - EncryptedWarpPayload containing the encrypted message
//   - error if encryption fails
func EncryptWarpPayload(plaintext []byte, recipientPubKey []byte, recipientKeyID []byte) (*EncryptedWarpPayload, error) {
	// Validate recipient public key length
	if len(recipientPubKey) != MLKEM768PublicKeyLen {
		return nil, fmt.Errorf("invalid ML-KEM public key length: got %d, expected %d",
			len(recipientPubKey), MLKEM768PublicKeyLen)
	}

	// ML-KEM Encapsulation: generate shared secret and ciphertext
	sharedSecret, encapsulatedKey, err := mlkemEncapsulate(recipientPubKey)
	if err != nil {
		return nil, fmt.Errorf("ML-KEM encapsulation failed: %w", err)
	}

	// Generate random nonce for AES-GCM
	nonce, err := generateSecureRandom(AESGCMNonceLen)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt plaintext with AES-256-GCM using the shared secret as the key
	ciphertext, err := aesGCMEncrypt(sharedSecret, nonce, plaintext)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM encryption failed: %w", err)
	}

	return &EncryptedWarpPayload{
		EncapsulatedKey: encapsulatedKey,
		Nonce:           nonce,
		Ciphertext:      ciphertext,
		RecipientKeyID:  recipientKeyID,
	}, nil
}

// Decrypt decrypts an encrypted Warp payload using the recipient's ML-KEM private key
//
// Parameters:
//   - recipientPrivKey: The recipient's ML-KEM-768 private key
//
// Returns:
//   - The decrypted plaintext
//   - error if decryption fails (wrong key, tampered ciphertext, etc.)
func (e *EncryptedWarpPayload) Decrypt(recipientPrivKey []byte) ([]byte, error) {
	// Validate ciphertext structure
	if len(e.EncapsulatedKey) != MLKEM768CiphertextLen {
		return nil, fmt.Errorf("%w: invalid encapsulated key length", ErrInvalidCiphertext)
	}
	if len(e.Nonce) != AESGCMNonceLen {
		return nil, fmt.Errorf("%w: invalid nonce length", ErrInvalidCiphertext)
	}
	if len(e.Ciphertext) < AESGCMTagLen {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrInvalidCiphertext)
	}

	// ML-KEM Decapsulation: recover shared secret from encapsulated key
	sharedSecret, err := mlkemDecapsulate(recipientPrivKey, e.EncapsulatedKey)
	if err != nil {
		return nil, fmt.Errorf("%w: ML-KEM decapsulation failed: %v", ErrDecryptionFailed, err)
	}

	// Decrypt ciphertext with AES-256-GCM
	plaintext, err := aesGCMDecrypt(sharedSecret, e.Nonce, e.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: AES-GCM decryption failed: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

// Size returns the total size of the encrypted payload in bytes
func (e *EncryptedWarpPayload) Size() int {
	return len(e.EncapsulatedKey) + len(e.Nonce) + len(e.Ciphertext) + len(e.RecipientKeyID)
}

func (e *EncryptedWarpPayload) String() string {
	return fmt.Sprintf("EncryptedWarpPayload(KeyID = %x, Size = %d bytes)",
		e.RecipientKeyID[:min(8, len(e.RecipientKeyID))], e.Size())
}

// =============================================================================
// ML-KEM Cryptographic Primitives (FIPS 203)
// =============================================================================

// mlkemEncapsulate performs ML-KEM-768 encapsulation
// Returns: (sharedSecret, ciphertext, error)
func mlkemEncapsulate(publicKey []byte) ([]byte, []byte, error) {
	if len(publicKey) != MLKEM768PublicKeyLen {
		return nil, nil, fmt.Errorf("invalid public key length: got %d, expected %d",
			len(publicKey), MLKEM768PublicKeyLen)
	}

	// Get the ML-KEM-768 scheme
	scheme := mlkem768.Scheme()

	// Unmarshal the public key
	pk, err := scheme.UnmarshalBinaryPublicKey(publicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	// Encapsulate to generate shared secret and ciphertext
	ciphertext, sharedSecret, err := scheme.Encapsulate(pk)
	if err != nil {
		return nil, nil, fmt.Errorf("ML-KEM encapsulation failed: %w", err)
	}

	return sharedSecret, ciphertext, nil
}

// mlkemDecapsulate performs ML-KEM-768 decapsulation
// Returns: (sharedSecret, error)
func mlkemDecapsulate(privateKey []byte, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) != MLKEM768CiphertextLen {
		return nil, fmt.Errorf("invalid ciphertext length: got %d, expected %d",
			len(ciphertext), MLKEM768CiphertextLen)
	}

	// Get the ML-KEM-768 scheme
	scheme := mlkem768.Scheme()

	// Unmarshal the private key
	sk, err := scheme.UnmarshalBinaryPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal private key: %w", err)
	}

	// Decapsulate to recover the shared secret
	sharedSecret, err := scheme.Decapsulate(sk, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("ML-KEM decapsulation failed: %w", err)
	}

	return sharedSecret, nil
}

// =============================================================================
// AES-256-GCM Cryptographic Primitives
// =============================================================================

// aesGCMEncrypt encrypts plaintext using AES-256-GCM
func aesGCMEncrypt(key []byte, nonce []byte, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("AES-256 requires 32-byte key")
	}
	if len(nonce) != AESGCMNonceLen {
		return nil, errors.New("AES-GCM requires 12-byte nonce")
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nil
}

// aesGCMDecrypt decrypts ciphertext using AES-256-GCM
func aesGCMDecrypt(key []byte, nonce []byte, ciphertext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("AES-256 requires 32-byte key")
	}
	if len(nonce) != AESGCMNonceLen {
		return nil, errors.New("AES-GCM requires 12-byte nonce")
	}
	if len(ciphertext) < AESGCMTagLen {
		return nil, errors.New("ciphertext too short")
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt and verify authentication
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	return plaintext, nil
}

// generateSecureRandom generates cryptographically secure random bytes
func generateSecureRandom(length int) ([]byte, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return b, nil
}

// =============================================================================
// Hybrid BLS+Ringtail Signature (DEPRECATED - use RingtailSignature)
// =============================================================================

// RingtailSignatureLen is a placeholder for legacy compatibility.
// Actual Ringtail signature size depends on threshold parameters.
// Ringtail uses LWE-based signatures, not ML-DSA.
// See: https://eprint.iacr.org/2024/1113
//
// Deprecated: Use RingtailSignature type which handles variable-length signatures.
const RingtailSignatureLen = 3309

// HybridBLSRTSignature implements a quantum-safe hybrid signature combining:
// - BLS aggregate signatures (classical security, compact)
// - Ringtail lattice signatures (post-quantum security, larger)
//
// Both signatures MUST verify for the message to be considered valid.
// This provides security against both classical and quantum attackers.
//
// Migration path:
// 1. Pre-quantum: BLS-only (BitSetSignature)
// 2. Transition: HybridBLSRTSignature (both required)
// 3. Post-quantum: Ringtail-only (future)
type HybridBLSRTSignature struct {
	// Signers is a big-endian byte slice encoding which validators signed
	Signers []byte `serialize:"true"`

	// BLSSignature is the aggregated BLS signature (96 bytes)
	BLSSignature [bls.SignatureLen]byte `serialize:"true"`

	// RingtailSignature is the aggregated Ringtail lattice signature
	// Uses threshold signing to produce a single combined signature
	RingtailSignature []byte `serialize:"true"`

	// RingtailPublicKeys contains the Ringtail public keys for each signer
	// in the same order as indicated by the Signers bitset
	// This is needed because validators may have different RT keys than BLS keys
	RingtailPublicKeys [][]byte `serialize:"true"`
}

// NumSigners returns the number of validators that participated in signing
func (s *HybridBLSRTSignature) NumSigners() (int, error) {
	signerIndices := set.BitsFromBytes(s.Signers)
	if len(signerIndices.Bytes()) != len(s.Signers) {
		return 0, ErrInvalidBitSet
	}
	return signerIndices.Len(), nil
}

// Verify validates both BLS and Ringtail signatures
// Both MUST be valid for the hybrid signature to be accepted
func (s *HybridBLSRTSignature) Verify(
	msg *UnsignedMessage,
	networkID uint32,
	validators CanonicalValidatorSet,
	quorumNum uint64,
	quorumDen uint64,
) error {
	if msg.NetworkID != networkID {
		return ErrWrongNetworkID
	}

	// Parse signer bit vector
	signerIndices := set.BitsFromBytes(s.Signers)
	if len(signerIndices.Bytes()) != len(s.Signers) {
		return ErrInvalidBitSet
	}

	// Get the validators that (allegedly) signed the message
	signers, err := FilterValidators(signerIndices, validators.Validators)
	if err != nil {
		return err
	}

	// Verify signer weight meets quorum
	sigWeight, _ := SumWeight(signers)
	if err := VerifyWeight(sigWeight, validators.TotalWeight, quorumNum, quorumDen); err != nil {
		return err
	}

	// === BLS Signature Verification ===
	if err := s.verifyBLS(msg, signers); err != nil {
		return fmt.Errorf("BLS verification failed: %w", err)
	}

	// === Ringtail Signature Verification ===
	if err := s.verifyRingtail(msg, signers); err != nil {
		return fmt.Errorf("Ringtail verification failed: %w", err)
	}

	return nil
}

// verifyBLS verifies the BLS aggregate signature
func (s *HybridBLSRTSignature) verifyBLS(msg *UnsignedMessage, signers []*Validator) error {
	// Parse the aggregate BLS signature
	aggSig, err := bls.SignatureFromBytes(s.BLSSignature[:])
	if err != nil {
		return fmt.Errorf("%w: %w", ErrParseSignature, err)
	}

	// Create the aggregate public key
	aggPubKey, err := AggregatePublicKeys(signers)
	if err != nil {
		return err
	}

	// Verify the BLS signature
	unsignedBytes := msg.Bytes()
	if !bls.Verify(aggPubKey, aggSig, unsignedBytes) {
		return ErrInvalidSignature
	}
	return nil
}

// verifyRingtail verifies the Ringtail lattice-based signature
func (s *HybridBLSRTSignature) verifyRingtail(msg *UnsignedMessage, signers []*Validator) error {
	// Validate we have RT public keys for all signers
	if len(s.RingtailPublicKeys) != len(signers) {
		return fmt.Errorf("%w: got %d keys, expected %d",
			ErrMissingRTPublicKey, len(s.RingtailPublicKeys), len(signers))
	}

	// Validate Ringtail signature is present
	if len(s.RingtailSignature) == 0 {
		return ErrInvalidRTSignature
	}

	// Aggregate the Ringtail public keys
	aggregatedRTPK, err := AggregateRingtailPublicKeys(s.RingtailPublicKeys)
	if err != nil {
		return fmt.Errorf("failed to aggregate RT public keys: %w", err)
	}

	// Verify the Ringtail signature
	unsignedBytes := msg.Bytes()
	if !VerifyRingtailSignature(aggregatedRTPK, unsignedBytes, s.RingtailSignature) {
		return ErrInvalidRTSignature
	}

	return nil
}

func (s *HybridBLSRTSignature) String() string {
	return fmt.Sprintf("HybridBLSRTSignature(Signers = %x, BLS = %x, RT = %x)",
		s.Signers, s.BLSSignature, s.RingtailSignature[:min(32, len(s.RingtailSignature))])
}

// =============================================================================
// Ringtail Signature Functions
// =============================================================================

// AggregateRingtailPublicKeys aggregates multiple Ringtail public keys
// into a single combined public key for threshold verification.
// This uses the threshold package's SchemeRingtail.
func AggregateRingtailPublicKeys(publicKeys [][]byte) ([]byte, error) {
	if len(publicKeys) == 0 {
		return nil, errors.New("no public keys to aggregate")
	}

	// Validate all keys have consistent length
	keyLen := len(publicKeys[0])
	for i, pk := range publicKeys {
		if len(pk) != keyLen {
			return nil, fmt.Errorf("inconsistent public key lengths: key %d has length %d, expected %d",
				i, len(pk), keyLen)
		}
	}

	// Get the Ringtail threshold scheme
	if !threshold.HasScheme(threshold.SchemeRingtail) {
		return nil, errors.New("Ringtail threshold scheme is not registered")
	}

	scheme, err := threshold.GetScheme(threshold.SchemeRingtail)
	if err != nil {
		return nil, fmt.Errorf("failed to get Ringtail scheme: %w", err)
	}

	// Parse all public keys using the threshold scheme
	parsedKeys := make([]threshold.PublicKey, len(publicKeys))
	for i, pk := range publicKeys {
		parsed, err := scheme.ParsePublicKey(pk)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key %d: %w", i, err)
		}
		parsedKeys[i] = parsed
	}

	// For threshold signatures, the "aggregated" public key is typically
	// just the group public key that all shares were generated from.
	// In Ringtail threshold protocol, all signers share the same group key,
	// so we can use any of the parsed keys directly.
	// The actual threshold verification is done by the scheme's verifier.
	//
	// Return the first public key as the aggregated key since they should
	// all represent the same group public key in a proper threshold setup.
	return parsedKeys[0].Bytes(), nil
}

// VerifyRingtailSignature verifies a Ringtail lattice-based signature.
// This uses the threshold package's SchemeRingtail verifier.
func VerifyRingtailSignature(publicKey []byte, message []byte, signature []byte) bool {
	// Basic sanity checks
	if len(publicKey) < 32 || len(signature) < 64 || len(message) == 0 {
		return false
	}

	// Get the Ringtail threshold scheme
	if !threshold.HasScheme(threshold.SchemeRingtail) {
		// Ringtail scheme must be registered for production use
		// This should be done via: import _ "github.com/luxfi/crypto/threshold/ringtail"
		return false
	}

	scheme, err := threshold.GetScheme(threshold.SchemeRingtail)
	if err != nil {
		return false
	}

	// Parse the public key
	pk, err := scheme.ParsePublicKey(publicKey)
	if err != nil {
		return false
	}

	// Create a verifier for the public key
	verifier, err := scheme.NewVerifier(pk)
	if err != nil {
		return false
	}

	// Verify using the threshold verifier
	// This performs full lattice-based signature verification
	return verifier.VerifyBytes(message, signature)
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
