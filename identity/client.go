// Copyright (C) 2024, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/luxfi/sdk/chain"
	"github.com/luxfi/sdk/crypto"
	"github.com/luxfi/sdk/types"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/scrypt"
)

// Client provides access to Lux identity chains (I-Chain, M-Chain, K-Chain)
type Client struct {
	iChain *IChainClient
	mChain *MPCChainClient
	kChain *KMSChainClient
	
	// User's master key (derived from seed phrase)
	masterKey []byte
}

// NewClient creates a new identity client
func NewClient(endpoint string, seedPhrase string) (*Client, error) {
	// Derive master key from seed phrase
	masterKey, err := deriveMasterKey(seedPhrase)
	if err != nil {
		return nil, fmt.Errorf("failed to derive master key: %w", err)
	}
	
	return &Client{
		iChain: NewIChainClient(endpoint + "/ext/bc/I"),
		mChain: NewMPCChainClient(endpoint + "/ext/bc/M"),
		kChain: NewKMSChainClient(endpoint + "/ext/bc/K"),
		masterKey: masterKey,
	}, nil
}

// IChainClient handles identity chain operations
type IChainClient struct {
	endpoint string
	client   *chain.Client
}

// NewIChainClient creates a new I-Chain client
func NewIChainClient(endpoint string) *IChainClient {
	return &IChainClient{
		endpoint: endpoint,
		client:   chain.NewClient(endpoint),
	}
}

// CreateDID creates a new decentralized identifier
func (c *IChainClient) CreateDID(publicKey []byte) (string, error) {
	// Generate DID from public key
	did := fmt.Sprintf("did:lux:i:%s", hex.EncodeToString(publicKey[:20]))
	
	// Register on I-Chain
	tx := &types.Transaction{
		Type: "create_did",
		Data: map[string]interface{}{
			"did":       did,
			"publicKey": hex.EncodeToString(publicKey),
		},
	}
	
	_, err := c.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return "", err
	}
	
	return did, nil
}

// StoreEncrypted stores encrypted data on I-Chain (only user can decrypt)
func (c *IChainClient) StoreEncrypted(did string, dataType string, data []byte, encryptionKey []byte) error {
	// Encrypt data client-side
	encrypted, err := encryptData(data, encryptionKey)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}
	
	// Store on I-Chain (chain cannot decrypt)
	tx := &types.Transaction{
		Type: "store_encrypted",
		Data: map[string]interface{}{
			"did":       did,
			"dataType":  dataType,
			"encrypted": hex.EncodeToString(encrypted),
		},
	}
	
	_, err = c.client.SendTransaction(context.Background(), tx)
	return err
}

// GetEncrypted retrieves encrypted data (must decrypt client-side)
func (c *IChainClient) GetEncrypted(did string, dataType string) ([]byte, error) {
	result, err := c.client.Query(context.Background(), "get_encrypted", map[string]interface{}{
		"did":      did,
		"dataType": dataType,
	})
	if err != nil {
		return nil, err
	}
	
	encryptedHex, ok := result["encrypted"].(string)
	if !ok {
		return nil, errors.New("invalid response format")
	}
	
	return hex.DecodeString(encryptedHex)
}

// MPCChainClient handles multi-party computation operations
type MPCChainClient struct {
	endpoint string
	client   *chain.Client
}

// NewMPCChainClient creates a new M-Chain client
func NewMPCChainClient(endpoint string) *MPCChainClient {
	return &MPCChainClient{
		endpoint: endpoint,
		client:   chain.NewClient(endpoint),
	}
}

// CreateMPCSession initiates a new MPC session
func (c *MPCChainClient) CreateMPCSession(sessionType string, threshold, total int) (string, error) {
	tx := &types.Transaction{
		Type: "create_mpc_session",
		Data: map[string]interface{}{
			"type":      sessionType,
			"threshold": threshold,
			"total":     total,
		},
	}
	
	result, err := c.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return "", err
	}
	
	sessionID, ok := result["sessionId"].(string)
	if !ok {
		return "", errors.New("invalid session ID in response")
	}
	
	return sessionID, nil
}

// ThresholdSign performs threshold signing
func (c *MPCChainClient) ThresholdSign(sessionID string, message []byte) ([]byte, error) {
	tx := &types.Transaction{
		Type: "threshold_sign",
		Data: map[string]interface{}{
			"sessionId": sessionID,
			"message":   hex.EncodeToString(message),
		},
	}
	
	result, err := c.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return nil, err
	}
	
	signatureHex, ok := result["signature"].(string)
	if !ok {
		return nil, errors.New("invalid signature in response")
	}
	
	return hex.DecodeString(signatureHex)
}

// DistributedKeyGen performs distributed key generation
func (c *MPCChainClient) DistributedKeyGen(threshold, total int) (string, []byte, error) {
	sessionID, err := c.CreateMPCSession("keygen", threshold, total)
	if err != nil {
		return "", nil, err
	}
	
	// Wait for DKG completion
	result, err := c.client.Query(context.Background(), "get_session", map[string]interface{}{
		"sessionId": sessionID,
	})
	if err != nil {
		return "", nil, err
	}
	
	publicKeyHex, ok := result["publicKey"].(string)
	if !ok {
		return "", nil, errors.New("DKG not completed")
	}
	
	publicKey, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return "", nil, err
	}
	
	return sessionID, publicKey, nil
}

// KMSChainClient handles key management operations
type KMSChainClient struct {
	endpoint string
	client   *chain.Client
}

// NewKMSChainClient creates a new K-Chain client
func NewKMSChainClient(endpoint string) *KMSChainClient {
	return &KMSChainClient{
		endpoint: endpoint,
		client:   chain.NewClient(endpoint),
	}
}

// CreateKey creates a new managed key
func (c *KMSChainClient) CreateKey(name, keyType, algorithm string, useHSM bool) (string, error) {
	tx := &types.Transaction{
		Type: "create_key",
		Data: map[string]interface{}{
			"name":      name,
			"type":      keyType,
			"algorithm": algorithm,
			"useHSM":    useHSM,
		},
	}
	
	result, err := c.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return "", err
	}
	
	keyID, ok := result["keyId"].(string)
	if !ok {
		return "", errors.New("invalid key ID in response")
	}
	
	return keyID, nil
}

// Encrypt encrypts data with a managed key
func (c *KMSChainClient) Encrypt(keyID string, plaintext []byte) ([]byte, error) {
	tx := &types.Transaction{
		Type: "encrypt",
		Data: map[string]interface{}{
			"keyId":     keyID,
			"plaintext": hex.EncodeToString(plaintext),
		},
	}
	
	result, err := c.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return nil, err
	}
	
	ciphertextHex, ok := result["ciphertext"].(string)
	if !ok {
		return nil, errors.New("invalid ciphertext in response")
	}
	
	return hex.DecodeString(ciphertextHex)
}

// Decrypt decrypts data with a managed key
func (c *KMSChainClient) Decrypt(keyID string, ciphertext []byte) ([]byte, error) {
	tx := &types.Transaction{
		Type: "decrypt",
		Data: map[string]interface{}{
			"keyId":      keyID,
			"ciphertext": hex.EncodeToString(ciphertext),
		},
	}
	
	result, err := c.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return nil, err
	}
	
	plaintextHex, ok := result["plaintext"].(string)
	if !ok {
		return nil, errors.New("invalid plaintext in response")
	}
	
	return hex.DecodeString(plaintextHex)
}

// CreateSecret creates a new managed secret
func (c *KMSChainClient) CreateSecret(name string, value []byte, secretType string) (string, error) {
	tx := &types.Transaction{
		Type: "create_secret",
		Data: map[string]interface{}{
			"name":  name,
			"value": hex.EncodeToString(value),
			"type":  secretType,
		},
	}
	
	result, err := c.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return "", err
	}
	
	secretID, ok := result["secretId"].(string)
	if !ok {
		return "", errors.New("invalid secret ID in response")
	}
	
	return secretID, nil
}

// GetSecret retrieves a managed secret
func (c *KMSChainClient) GetSecret(secretID string) ([]byte, error) {
	result, err := c.client.Query(context.Background(), "get_secret", map[string]interface{}{
		"secretId": secretID,
	})
	if err != nil {
		return nil, err
	}
	
	valueHex, ok := result["value"].(string)
	if !ok {
		return nil, errors.New("invalid secret value in response")
	}
	
	return hex.DecodeString(valueHex)
}

// Identity represents a user's sovereign identity
type Identity struct {
	DID        string
	PublicKeys map[string][]byte
	PrivateKey []byte // Never sent to chain
}

// NewIdentity generates a new identity
func NewIdentity() (*Identity, error) {
	// Generate key pair
	privKey, err := crypto.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	
	pubKey := privKey.PublicKey()
	
	// Generate DID
	did := fmt.Sprintf("did:lux:i:%s", hex.EncodeToString(pubKey.Bytes()[:20]))
	
	return &Identity{
		DID: did,
		PublicKeys: map[string][]byte{
			"master": pubKey.Bytes(),
		},
		PrivateKey: privKey.Bytes(),
	}, nil
}

// EncryptForSelf encrypts data that only the identity owner can decrypt
func (id *Identity) EncryptForSelf(data []byte) ([]byte, error) {
	return encryptData(data, id.PrivateKey[:32])
}

// DecryptForSelf decrypts data encrypted for this identity
func (id *Identity) DecryptForSelf(encrypted []byte) ([]byte, error) {
	return decryptData(encrypted, id.PrivateKey[:32])
}

// Helper functions

func deriveMasterKey(seedPhrase string) ([]byte, error) {
	// Use scrypt for key derivation
	salt := []byte("lux-identity-v1")
	return scrypt.Key([]byte(seedPhrase), salt, 32768, 8, 1, 32)
}

func encryptData(plaintext, key []byte) ([]byte, error) {
	// Use XChaCha20-Poly1305 for AEAD
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	
	// Generate nonce
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	
	// Encrypt and prepend nonce
	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func decryptData(ciphertext, key []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	
	if len(ciphertext) < aead.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	
	// Extract nonce
	nonce, ciphertext := ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():]
	
	// Decrypt
	return aead.Open(nil, nonce, ciphertext, nil)
}

// VerifiableCredential represents a W3C verifiable credential
type VerifiableCredential struct {
	Context           []string               `json:"@context"`
	ID                string                 `json:"id"`
	Type              []string               `json:"type"`
	Issuer            string                 `json:"issuer"`
	IssuanceDate      string                 `json:"issuanceDate"`
	CredentialSubject map[string]interface{} `json:"credentialSubject"`
	Proof             map[string]interface{} `json:"proof"`
}

// CreateVC creates a new verifiable credential
func CreateVC(issuerDID string, subjectDID string, claims map[string]interface{}) *VerifiableCredential {
	return &VerifiableCredential{
		Context: []string{
			"https://www.w3.org/2018/credentials/v1",
			"https://lux.network/credentials/v1",
		},
		ID:   fmt.Sprintf("urn:uuid:%s", generateUUID()),
		Type: []string{"VerifiableCredential"},
		Issuer: issuerDID,
		IssuanceDate: time.Now().Format(time.RFC3339),
		CredentialSubject: map[string]interface{}{
			"id": subjectDID,
		},
	}
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}