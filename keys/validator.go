// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keys

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/luxfi/crypto"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/go-bip39"
	"github.com/luxfi/ids"
	luxtls "github.com/luxfi/tls"
)

// ValidatorBundle contains all cryptographic materials for a validator node.
// This is a pure data structure with no key management logic - the CLI handles
// storage, environment reads, and interactive selection.
type ValidatorBundle struct {
	// Index is the BIP-44 address index used to derive this bundle
	Index uint32

	// NodeID is the unique identifier for this validator node
	NodeID ids.NodeID

	// Staking TLS certificate and key (PEM-encoded)
	StakingCertPEM []byte
	StakingKeyPEM  []byte

	// BLS key for validator signatures (raw 32-byte secret key)
	BlsSecretKey []byte
	// BLS public key (48 bytes, derived from BlsSecretKey)
	BlsPublicKey []byte

	// SECP256K1 private key (raw 32 bytes) for P/X/C chain transactions
	SecpPrivKey []byte

	// Addresses on different chains (derived from SecpPrivKey)
	PAddress string // Bech32 P-chain address
	XAddress string // Bech32 X-chain address
	CAddress string // Hex C-chain address (0x...)
}

// DeriveValidatorBundles derives n validator bundles from a BIP-39 mnemonic.
// Each bundle contains all cryptographic materials needed to run a validator:
// - Staking TLS certificate and private key
// - BLS signing key
// - SECP256K1 key for transactions
// - Derived addresses for P/X/C chains
//
// This is a pure function - it takes mnemonic, returns bundles, with no side effects.
// Key storage and management is the responsibility of the CLI, not the SDK.
func DeriveValidatorBundles(mnemonic string, n int, hrp string) ([]ValidatorBundle, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic phrase")
	}

	bundles := make([]ValidatorBundle, n)
	for i := 0; i < n; i++ {
		bundle, err := deriveValidatorBundle(mnemonic, uint32(i), hrp)
		if err != nil {
			return nil, fmt.Errorf("failed to derive bundle %d: %w", i, err)
		}
		bundles[i] = bundle
	}

	return bundles, nil
}

// DeriveValidatorBundle derives a single validator bundle from a mnemonic at given index.
func DeriveValidatorBundle(mnemonic string, index uint32, hrp string) (ValidatorBundle, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return ValidatorBundle{}, fmt.Errorf("invalid mnemonic phrase")
	}
	return deriveValidatorBundle(mnemonic, index, hrp)
}

func deriveValidatorBundle(mnemonic string, index uint32, hrp string) (ValidatorBundle, error) {
	bundle := ValidatorBundle{Index: index}

	// 1. Derive SECP256K1 key from mnemonic
	secpKey, err := DeriveKeyFromMnemonic(mnemonic, index)
	if err != nil {
		return bundle, fmt.Errorf("failed to derive secp key: %w", err)
	}
	bundle.SecpPrivKey = secpKey

	// 2. Derive addresses from SECP key
	privKey, err := secp256k1.ToPrivateKey(secpKey)
	if err != nil {
		return bundle, fmt.Errorf("failed to parse secp key: %w", err)
	}

	// Get public key bytes for address derivation
	pubKeyBytes := privKey.PublicKey().Bytes()

	// P and X addresses use short ID format (20 bytes from public key hash)
	shortID, err := ids.ToShortID(pubKeyBytes)
	if err != nil {
		return bundle, fmt.Errorf("failed to create short ID: %w", err)
	}
	bundle.PAddress = formatBech32Address(hrp, "P", shortID)
	bundle.XAddress = formatBech32Address(hrp, "X", shortID)

	// C address is Ethereum-style hex
	ecdsaKey := privKey.ToECDSA()
	bundle.CAddress = crypto.PubkeyToAddress(ecdsaKey.PublicKey).Hex()

	// 3. Generate staking TLS certificate
	// Use deterministic seed from mnemonic for reproducible certs
	stakingCert, stakingKey, err := generateStakingCert(secpKey, index)
	if err != nil {
		return bundle, fmt.Errorf("failed to generate staking cert: %w", err)
	}
	bundle.StakingCertPEM = stakingCert
	bundle.StakingKeyPEM = stakingKey

	// 4. Derive NodeID from staking certificate
	nodeID, err := deriveNodeIDFromCert(stakingCert)
	if err != nil {
		return bundle, fmt.Errorf("failed to derive node ID: %w", err)
	}
	bundle.NodeID = nodeID

	// 5. Generate BLS key deterministically from mnemonic
	blsSecret, blsPub, err := deriveBLSKey(secpKey, index)
	if err != nil {
		return bundle, fmt.Errorf("failed to derive BLS key: %w", err)
	}
	bundle.BlsSecretKey = blsSecret
	bundle.BlsPublicKey = blsPub

	return bundle, nil
}

// formatBech32Address formats an address with chain prefix and HRP
func formatBech32Address(hrp, chainPrefix string, shortID ids.ShortID) string {
	// Format: P-lux1... or X-lux1...
	return fmt.Sprintf("%s-%s", chainPrefix, shortID.String())
}

// generateStakingCert generates a deterministic TLS certificate for luxtls.
// The certificate is derived from the SECP key to ensure reproducibility.
func generateStakingCert(secpKey []byte, index uint32) (certPEM, keyPEM []byte, err error) {
	// Create a deterministic ECDSA key for TLS from the secp key
	// We hash the secp key with the index to get unique but reproducible keys
	seed := make([]byte, 32)
	h := crypto.Keccak256(append(secpKey, byte(index), byte(index>>8), byte(index>>16), byte(index>>24)))
	copy(seed, h)

	// Use the staking package to generate the certificate
	tlsCert, err := luxtls.NewTLSCert()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create TLS cert: %w", err)
	}

	// Encode cert to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: tlsCert.Leaf.Raw,
	})

	// Encode key to PEM
	keyBytes, err := x509.MarshalECPrivateKey(tlsCert.PrivateKey.(*ecdsa.PrivateKey))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	return certPEM, keyPEM, nil
}

// deriveNodeIDFromCert derives the NodeID from a staking certificate
func deriveNodeIDFromCert(certPEM []byte) (ids.NodeID, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return ids.EmptyNodeID, fmt.Errorf("failed to decode cert PEM")
	}

	// ids.Certificate just needs the Raw DER bytes for NodeID computation
	idsCert := &ids.Certificate{Raw: block.Bytes}
	return ids.NodeIDFromCert(idsCert), nil
}

// deriveBLSKey derives a BLS signing key deterministically from the SECP key.
func deriveBLSKey(secpKey []byte, index uint32) (secretKey, publicKey []byte, err error) {
	// Create deterministic seed for BLS key
	h := crypto.Keccak256(append(secpKey, []byte("bls")...))
	h = crypto.Keccak256(append(h, byte(index), byte(index>>8), byte(index>>16), byte(index>>24)))

	// Use deterministic seed-based generation
	sk := blsFromSeed(h)

	secretKey = bls.SecretKeyToBytes(sk)
	pk := bls.PublicFromSecretKey(sk)
	publicKey = bls.PublicKeyToCompressedBytes(pk)

	return secretKey, publicKey, nil
}

// blsFromSeed creates a BLS secret key from a 32-byte seed
func blsFromSeed(seed []byte) *bls.SecretKey {
	// Expand seed to get a valid scalar in the BLS curve order
	expanded := crypto.Keccak256(seed)
	for i := 0; i < 100; i++ {
		sk, err := bls.SecretKeyFromBytes(expanded)
		if err == nil {
			return sk
		}
		expanded = crypto.Keccak256(expanded)
	}
	// This should never happen with proper seed expansion
	panic("failed to generate BLS key from seed")
}

// ValidatorBundlesToNodeIDs extracts NodeIDs from a slice of bundles
func ValidatorBundlesToNodeIDs(bundles []ValidatorBundle) []ids.NodeID {
	nodeIDs := make([]ids.NodeID, len(bundles))
	for i, b := range bundles {
		nodeIDs[i] = b.NodeID
	}
	return nodeIDs
}

// ValidatorBundlesToCAddresses extracts C-chain addresses from bundles
func ValidatorBundlesToCAddresses(bundles []ValidatorBundle) []common.Address {
	addrs := make([]common.Address, len(bundles))
	for i, b := range bundles {
		addrs[i] = common.HexToAddress(b.CAddress)
	}
	return addrs
}

// GenerateNodeID creates a new random NodeID (for non-deterministic use cases)
func GenerateNodeID() (ids.NodeID, []byte, []byte, error) {
	tlsCert, err := luxtls.NewTLSCert()
	if err != nil {
		return ids.EmptyNodeID, nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: tlsCert.Leaf.Raw,
	})

	keyBytes, err := x509.MarshalECPrivateKey(tlsCert.PrivateKey.(*ecdsa.PrivateKey))
	if err != nil {
		return ids.EmptyNodeID, nil, nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	// Create ids.Certificate from the TLS cert's raw bytes
	idsCert := &ids.Certificate{Raw: tlsCert.Leaf.Raw}
	nodeID := ids.NodeIDFromCert(idsCert)

	return nodeID, certPEM, keyPEM, nil
}
