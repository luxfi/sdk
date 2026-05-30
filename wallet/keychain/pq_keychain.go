// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keychain

import (
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"

	// "github.com/luxfi/crypto/bls"      // TODO: Implement BLS support
	// "github.com/luxfi/crypto/mlkem"    // TODO: Implement ML-KEM support
	// "github.com/luxfi/crypto/corona" // TODO: Implement corona support
	"github.com/luxfi/crypto/mldsa"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/crypto/slhdsa"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
)

var (
	ErrInvalidKeyType = errors.New("invalid key type")
	ErrKeyNotFound    = errors.New("key not found")

	// ErrHybridSigTruncated is returned when a hybrid signature blob is
	// too short to carry the two length-prefixed halves.
	ErrHybridSigTruncated = errors.New("hybrid signature truncated")

	// ErrHybridSigClassicalFailed is returned when the classical half of
	// a hybrid signature does not verify under the supplied classical
	// public key. Under AND-semantics, this fails the whole hybrid
	// signature — even a valid PQ half cannot rescue it.
	//
	// Closes red-team F112 on the classical axis: previously the
	// implicit "either half verifies" semantics let a forged hybrid
	// pass as long as ONE half was correct; now both must pass.
	ErrHybridSigClassicalFailed = errors.New("hybrid signature: classical half failed to verify")

	// ErrHybridSigPQFailed is returned when the post-quantum half of a
	// hybrid signature does not verify under the supplied PQ public
	// key. Closes F112 on the PQ axis.
	ErrHybridSigPQFailed = errors.New("hybrid signature: post-quantum half failed to verify")
)

// KeyType represents the type of cryptographic key
type KeyType uint8

const (
	// Classical cryptography
	KeyTypeSecp256k1 KeyType = iota
	KeyTypeBLS               // BLS signatures for consensus

	// Post-quantum cryptography (NIST FIPS standards)
	KeyTypeMLDSA44   // FIPS 204 - ML-DSA-44
	KeyTypeMLDSA65   // FIPS 204 - ML-DSA-65
	KeyTypeMLDSA87   // FIPS 204 - ML-DSA-87
	KeyTypeSLHDSA128 // FIPS 205 - SLH-DSA-128
	KeyTypeSLHDSA192 // FIPS 205 - SLH-DSA-192
	KeyTypeSLHDSA256 // FIPS 205 - SLH-DSA-256

	// Key encapsulation (FIPS 203)
	KeyTypeMLKEM512  // ML-KEM-512
	KeyTypeMLKEM768  // ML-KEM-768
	KeyTypeMLKEM1024 // ML-KEM-1024

	// Privacy-preserving
	KeyTypeRingSig // Ring signatures

	// Hybrid modes (classical + post-quantum)
	KeyTypeHybridSecp256k1MLDSA44
	KeyTypeHybridSecp256k1SLHDSA128
	KeyTypeHybridBLSMLDSA44
)

// PQSigner implements Signer with post-quantum support
type PQSigner struct {
	keyType KeyType
	address ids.ShortID

	// Classical keys
	secp256k1Key *secp256k1.PrivateKey

	// Post-quantum keys
	mldsaKey  interface{} // Can be *mldsa.PrivateKey44/65/87
	slhdsaKey interface{} // Can be *slhdsa.PrivateKey128/192/256
	// ringSigKey *corona.PrivateKey // TODO: implement when available

	// For hybrid modes, we store both
	hybridClassical *secp256k1.PrivateKey
	hybridPQ        interface{}
}

// SignHash signs a hash with the appropriate algorithm
func (s *PQSigner) SignHash(hash []byte) ([]byte, error) {
	switch s.keyType {
	case KeyTypeSecp256k1:
		if s.secp256k1Key == nil {
			return nil, ErrInvalidKeyType
		}
		return s.secp256k1Key.SignHash(hash)

	case KeyTypeMLDSA44, KeyTypeMLDSA65, KeyTypeMLDSA87:
		if key, ok := s.mldsaKey.(*mldsa.PrivateKey); ok {
			sig, err := key.Sign(rand.Reader, hash, crypto.Hash(0))
			if err != nil {
				return nil, err
			}
			return sig, nil
		}
		return nil, ErrInvalidKeyType

	case KeyTypeSLHDSA128, KeyTypeSLHDSA192, KeyTypeSLHDSA256:
		if key, ok := s.slhdsaKey.(*slhdsa.PrivateKey); ok {
			sig, err := key.Sign(rand.Reader, hash, crypto.Hash(0))
			if err != nil {
				return nil, err
			}
			return sig, nil
		}
		return nil, ErrInvalidKeyType

	// TODO: implement corona when available
	// case KeyTypeRingSig:
	//	if s.ringSigKey == nil {
	//		return nil, ErrInvalidKeyType
	//	}
	//	// Corona requires ring members for signing
	//	// For now, we'll use a simple signature
	//	return s.ringSigKey.Sign(hash), nil

	case KeyTypeHybridSecp256k1MLDSA44:
		// Hybrid mode: concatenate both signatures
		if s.hybridClassical == nil || s.hybridPQ == nil {
			return nil, ErrInvalidKeyType
		}

		classicalSig, err := s.hybridClassical.SignHash(hash)
		if err != nil {
			return nil, err
		}

		if key, ok := s.hybridPQ.(*mldsa.PrivateKey); ok {
			pqSig, err := key.Sign(rand.Reader, hash, crypto.Hash(0))
			if err != nil {
				return nil, err
			}
			// Concatenate signatures with length prefixes
			result := make([]byte, 0, 2+len(classicalSig)+2+len(pqSig))
			result = append(result, byte(len(classicalSig)>>8), byte(len(classicalSig)))
			result = append(result, classicalSig...)
			result = append(result, byte(len(pqSig)>>8), byte(len(pqSig)))
			result = append(result, pqSig...)
			return result, nil
		}
		return nil, ErrInvalidKeyType

	default:
		return nil, ErrInvalidKeyType
	}
}

// Sign signs a message with the appropriate algorithm
func (s *PQSigner) Sign(msg []byte) ([]byte, error) {
	switch s.keyType {
	case KeyTypeSecp256k1:
		// secp256k1 needs a 32-byte hash
		if s.secp256k1Key == nil {
			return nil, ErrInvalidKeyType
		}
		hash := sha256.New()
		hash.Write(msg)
		return s.secp256k1Key.SignHash(hash.Sum(nil))

	case KeyTypeHybridSecp256k1MLDSA44, KeyTypeHybridSecp256k1SLHDSA128:
		// For hybrid, we need to hash for the classical part
		hash := sha256.New()
		hash.Write(msg)
		hashBytes := hash.Sum(nil)

		// Sign with both algorithms
		if s.hybridClassical == nil || s.hybridPQ == nil {
			return nil, ErrInvalidKeyType
		}

		classicalSig, err := s.hybridClassical.SignHash(hashBytes)
		if err != nil {
			return nil, err
		}

		var pqSig []byte
		switch pq := s.hybridPQ.(type) {
		case *mldsa.PrivateKey:
			pqSig, err = pq.Sign(rand.Reader, msg, crypto.Hash(0))
		case *slhdsa.PrivateKey:
			pqSig, err = pq.Sign(rand.Reader, msg, crypto.Hash(0))
		default:
			return nil, ErrInvalidKeyType
		}
		if err != nil {
			return nil, err
		}

		// Concatenate signatures with length prefixes
		result := make([]byte, 0, 2+len(classicalSig)+2+len(pqSig))
		result = append(result, byte(len(classicalSig)>>8), byte(len(classicalSig)))
		result = append(result, classicalSig...)
		result = append(result, byte(len(pqSig)>>8), byte(len(pqSig)))
		result = append(result, pqSig...)
		return result, nil

	default:
		// PQ algorithms sign the message directly
		return s.SignHash(msg)
	}
}

// Address returns the address associated with this signer
func (s *PQSigner) Address() ids.ShortID {
	return s.address
}

// PQKeychain implements Keychain with post-quantum support
type PQKeychain struct {
	keysByAddress map[ids.ShortID]*PQSigner
	addressSet    set.Set[ids.ShortID]
	defaultType   KeyType
}

// NewPQKeychain creates a new post-quantum keychain
func NewPQKeychain(defaultType KeyType) *PQKeychain {
	return &PQKeychain{
		keysByAddress: make(map[ids.ShortID]*PQSigner),
		addressSet:    set.NewSet[ids.ShortID](0),
		defaultType:   defaultType,
	}
}

// AddSecp256k1 adds a secp256k1 key to the keychain
func (kc *PQKeychain) AddSecp256k1(key *secp256k1.PrivateKey) ids.ShortID {
	pk := key.PublicKey()
	addr := pk.Address()
	shortAddr, _ := ids.ToShortID(addr[:])

	signer := &PQSigner{
		keyType:      KeyTypeSecp256k1,
		address:      shortAddr,
		secp256k1Key: key,
	}

	kc.keysByAddress[shortAddr] = signer
	kc.addressSet.Add(shortAddr)
	return shortAddr
}

// AddMLDSA adds an ML-DSA key to the keychain
func (kc *PQKeychain) AddMLDSA(key *mldsa.PrivateKey, keyType KeyType) ids.ShortID {
	// Generate address from public key bytes
	pubKeyBytes := key.Bytes()
	addrBytes := ids.ShortID{}
	copy(addrBytes[:], pubKeyBytes[:20]) // Use first 20 bytes as address

	signer := &PQSigner{
		keyType:  keyType,
		address:  addrBytes,
		mldsaKey: key,
	}

	kc.keysByAddress[addrBytes] = signer
	kc.addressSet.Add(addrBytes)
	return addrBytes
}

// AddSLHDSA adds an SLH-DSA key to the keychain
func (kc *PQKeychain) AddSLHDSA(key *slhdsa.PrivateKey, keyType KeyType) ids.ShortID {
	pubKeyBytes := key.Bytes()
	addrBytes := ids.ShortID{}
	copy(addrBytes[:], pubKeyBytes[:20])

	signer := &PQSigner{
		keyType:   keyType,
		address:   addrBytes,
		slhdsaKey: key,
	}

	kc.keysByAddress[addrBytes] = signer
	kc.addressSet.Add(addrBytes)
	return addrBytes
}

// TODO: implement when corona is available
// AddRingSig adds a corona key to the keychain
// func (kc *PQKeychain) AddRingSig(key *corona.PrivateKey) ids.ShortID {
//	pubKey := key.PublicKey()
//	addrBytes := ids.ShortID{}
//	copy(addrBytes[:], pubKey.Bytes()[:20])
//
//	signer := &PQSigner{
//		keyType:     KeyTypeRingSig,
//		address:     addrBytes,
//		ringSigKey: key,
//	}
//
//	kc.keysByAddress[addrBytes] = signer
//	kc.addressSet.Add(addrBytes)
//	return addrBytes
// }

// AddHybrid adds a hybrid classical+PQ key pair
func (kc *PQKeychain) AddHybrid(classical *secp256k1.PrivateKey, pq interface{}) ids.ShortID {
	// Generate address from classical key for compatibility
	pk := classical.PublicKey()
	addr := pk.Address()
	shortAddr, _ := ids.ToShortID(addr[:])

	var keyType KeyType
	switch pq.(type) {
	case *mldsa.PrivateKey:
		keyType = KeyTypeHybridSecp256k1MLDSA44
	case *slhdsa.PrivateKey:
		keyType = KeyTypeHybridSecp256k1SLHDSA128
	default:
		return ids.ShortEmpty
	}

	signer := &PQSigner{
		keyType:         keyType,
		address:         shortAddr,
		hybridClassical: classical,
		hybridPQ:        pq,
	}

	kc.keysByAddress[shortAddr] = signer
	kc.addressSet.Add(shortAddr)
	return shortAddr
}

// GenerateKey generates a new key of the default type
func (kc *PQKeychain) GenerateKey() (ids.ShortID, error) {
	switch kc.defaultType {
	case KeyTypeSecp256k1:
		key, err := secp256k1.NewPrivateKey()
		if err != nil {
			return ids.ShortEmpty, err
		}
		return kc.AddSecp256k1(key), nil

	case KeyTypeMLDSA44:
		key, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA44)
		if err != nil {
			return ids.ShortEmpty, err
		}
		return kc.AddMLDSA(key, KeyTypeMLDSA44), nil

	case KeyTypeMLDSA65:
		key, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA65)
		if err != nil {
			return ids.ShortEmpty, err
		}
		return kc.AddMLDSA(key, KeyTypeMLDSA65), nil

	case KeyTypeMLDSA87:
		key, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA87)
		if err != nil {
			return ids.ShortEmpty, err
		}
		return kc.AddMLDSA(key, KeyTypeMLDSA87), nil

	case KeyTypeSLHDSA128:
		key, err := slhdsa.GenerateKey(rand.Reader, slhdsa.SHA2_128s)
		if err != nil {
			return ids.ShortEmpty, err
		}
		return kc.AddSLHDSA(key, KeyTypeSLHDSA128), nil

	case KeyTypeSLHDSA192:
		key, err := slhdsa.GenerateKey(rand.Reader, slhdsa.SHA2_192s)
		if err != nil {
			return ids.ShortEmpty, err
		}
		return kc.AddSLHDSA(key, KeyTypeSLHDSA192), nil

	case KeyTypeSLHDSA256:
		key, err := slhdsa.GenerateKey(rand.Reader, slhdsa.SHA2_256s)
		if err != nil {
			return ids.ShortEmpty, err
		}
		return kc.AddSLHDSA(key, KeyTypeSLHDSA256), nil

	// TODO: implement when corona is available
	// case KeyTypeRingSig:
	//	key, err := corona.GenerateKey(rand.Reader)
	//	if err != nil {
	//		return ids.ShortEmpty, err
	//	}
	//	return kc.AddRingSig(key), nil

	case KeyTypeHybridSecp256k1MLDSA44:
		classical, err := secp256k1.NewPrivateKey()
		if err != nil {
			return ids.ShortEmpty, err
		}
		pq, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA44)
		if err != nil {
			return ids.ShortEmpty, err
		}
		return kc.AddHybrid(classical, pq), nil

	case KeyTypeHybridSecp256k1SLHDSA128:
		classical, err := secp256k1.NewPrivateKey()
		if err != nil {
			return ids.ShortEmpty, err
		}
		pq, err := slhdsa.GenerateKey(rand.Reader, slhdsa.SHA2_128s)
		if err != nil {
			return ids.ShortEmpty, err
		}
		return kc.AddHybrid(classical, pq), nil

	default:
		return ids.ShortEmpty, fmt.Errorf("unsupported key type: %v", kc.defaultType)
	}
}

// Addresses returns all addresses in the keychain
func (kc *PQKeychain) Addresses() []ids.ShortID {
	addrs := make([]ids.ShortID, 0, kc.addressSet.Len())
	for addr := range kc.addressSet {
		addrs = append(addrs, addr)
	}
	return addrs
}

// Get returns the signer for the given address
func (kc *PQKeychain) Get(addr ids.ShortID) (Signer, bool) {
	signer, exists := kc.keysByAddress[addr]
	if !exists {
		return nil, false
	}
	return signer, true
}

// GetPQSigner returns the PQ signer for advanced operations
func (kc *PQKeychain) GetPQSigner(addr ids.ShortID) (*PQSigner, bool) {
	signer, exists := kc.keysByAddress[addr]
	return signer, exists
}

// SetDefaultType sets the default key type for new keys
func (kc *PQKeychain) SetDefaultType(keyType KeyType) {
	kc.defaultType = keyType
}

// =============================================================================
// Hybrid signature AND-verification (closes red-team F112).
//
// PQSigner.SignHash / PQSigner.Sign emit a hybrid signature on a
// KeyTypeHybridSecp256k1MLDSA44 / KeyTypeHybridSecp256k1SLHDSA128
// signer as the concatenation:
//
//	[u16 BE classicalLen] || classicalSig || [u16 BE pqLen] || pqSig
//
// The previous verifier surface treated this as an OR — a forged
// hybrid carrying a valid ECDSA half and a random "PQ" blob would
// pass any "either half verifies" verifier, reducing hybrid security
// to the weaker of the two halves (ECDSA).
//
// The verifiers below enforce AND: both halves MUST verify under
// their respective public keys, or the entire hybrid signature is
// refused. Returns typed sentinels (ErrHybridSig*) so call sites can
// route on the exact failure axis without parsing error strings.
//
// One-way ordering: classical is checked first, PQ second. Both halves
// must produce ok=true; any non-true result fails the hybrid. There is
// no short-circuit on the classical refusal — verifiers still run the
// PQ check so timing-side channels do not leak which half failed.

// SplitHybridSignature parses a hybrid signature blob into its
// classical and PQ halves using the length-prefix layout produced by
// PQSigner. Returns ErrHybridSigTruncated on any structural failure.
//
// Exported so callers that want to dispatch verification through their
// own keying setup (e.g. a wallet that holds the two public keys
// separately) can reuse the parsing without taking the verify path.
func SplitHybridSignature(sig []byte) (classical, pq []byte, err error) {
	if len(sig) < 2 {
		return nil, nil, ErrHybridSigTruncated
	}
	classicalLen := int(sig[0])<<8 | int(sig[1])
	if 2+classicalLen+2 > len(sig) {
		return nil, nil, ErrHybridSigTruncated
	}
	classical = sig[2 : 2+classicalLen]
	pqLenOff := 2 + classicalLen
	pqLen := int(sig[pqLenOff])<<8 | int(sig[pqLenOff+1])
	pqStart := pqLenOff + 2
	if pqStart+pqLen != len(sig) {
		return nil, nil, ErrHybridSigTruncated
	}
	pq = sig[pqStart : pqStart+pqLen]
	return classical, pq, nil
}

// VerifyHybridSecp256k1MLDSA verifies a hybrid signature produced by a
// KeyTypeHybridSecp256k1MLDSA{44,65,87} signer over message msg. BOTH
// halves must verify; either half failing fails the whole signature.
// Closes F112.
//
// classicalPub is the secp256k1 public key. pqPub is the ML-DSA
// public key carrying its own internal Mode setting. The classical
// half is verified against the SHA-256 hash of msg (matching
// PQSigner.Sign's hash-then-sign for the classical leg); the PQ half
// is verified against the raw message (ML-DSA hashes internally).
func VerifyHybridSecp256k1MLDSA(
	classicalPub *secp256k1.PublicKey,
	pqPub *mldsa.PublicKey,
	msg []byte,
	hybridSig []byte,
) error {
	classical, pq, err := SplitHybridSignature(hybridSig)
	if err != nil {
		return err
	}
	if classicalPub == nil {
		return ErrInvalidKeyType
	}
	if pqPub == nil {
		return ErrInvalidKeyType
	}

	hashed := sha256.Sum256(msg)
	classicalOK := classicalPub.VerifyHash(hashed[:], classical)
	pqOK := pqPub.VerifySignature(msg, pq)

	// AND-semantics: BOTH must pass. We compute both halves before
	// branching so a side-channel observer can't infer which half
	// failed from short-circuit timing.
	if !classicalOK {
		return ErrHybridSigClassicalFailed
	}
	if !pqOK {
		return ErrHybridSigPQFailed
	}
	return nil
}

// VerifyHybridSecp256k1SLHDSA verifies a hybrid signature produced by
// a KeyTypeHybridSecp256k1SLHDSA{128,192,256} signer over message msg.
// AND-semantics: BOTH halves must verify. Closes F112 on the
// SLH-DSA axis.
func VerifyHybridSecp256k1SLHDSA(
	classicalPub *secp256k1.PublicKey,
	pqPub *slhdsa.PublicKey,
	msg []byte,
	hybridSig []byte,
) error {
	classical, pq, err := SplitHybridSignature(hybridSig)
	if err != nil {
		return err
	}
	if classicalPub == nil {
		return ErrInvalidKeyType
	}
	if pqPub == nil {
		return ErrInvalidKeyType
	}

	hashed := sha256.Sum256(msg)
	classicalOK := classicalPub.VerifyHash(hashed[:], classical)
	pqOK := pqPub.VerifySignature(msg, pq)

	if !classicalOK {
		return ErrHybridSigClassicalFailed
	}
	if !pqOK {
		return ErrHybridSigPQFailed
	}
	return nil
}
