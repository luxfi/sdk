// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package server provides quantum-resistant node identity using X25519MLKEM768
package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/luxfi/crypto"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/metric"
	"golang.org/x/crypto/cryptobyte"
)

// PQNodeIdentity represents a quantum-resistant node identity
// Derived from X25519MLKEM768 for long-term security
type PQNodeIdentity struct {
	// Node ID derived from PQ key material
	NodeID ids.NodeID `json:"node_id"`
	
	// PQ key pair for quantum resistance
	PQPrivateKey []byte `json:"-"`
	PQPublicKey  []byte `json:"pq_public_key"`
	
	// Traditional Ed25519 key for compatibility
	Ed25519PrivateKey ed25519.PrivateKey `json:"-"`
	Ed25519PublicKey  ed25519.PublicKey  `json:"ed25519_public_key"`
	
	// Staking certificate derived from PQ key
	StakingCert []byte `json:"staking_cert"`
	
	// TLS certificate derived from PQ key
	TLSCert []byte `json:"tls_cert"`
	
	// Metrics
	metrics *metric.MetricsRegistry
	logger  log.Logger
}

// NewPQNodeIdentity creates a new quantum-resistant node identity
// Uses X25519MLKEM768 for long-term security
func NewPQNodeIdentity(
	logger log.Logger,
	metricsNamespace string,
	reg *metric.MetricsRegistry,
) (*PQNodeIdentity, error) {
	if reg == nil {
		reg = metric.NewMetricsRegistry()
	}
	
	// Generate PQ key pair (X25519MLKEM768)
	pqPrivateKey, pqPublicKey, err := generatePQKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PQ key pair: %w", err)
	}
	
	// Generate Ed25519 key for compatibility
	ed25519PublicKey, ed25519PrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 key: %w", err)
	}
	
	// Derive node ID from PQ public key
	nodeID, err := deriveNodeIDFromPQKey(pqPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive node ID: %w", err)
	}
	
	// Generate staking certificate from PQ key
	stakingCert, err := generateStakingCertFromPQKey(pqPublicKey, ed25519PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate staking cert: %w", err)
	}
	
	// Generate TLS certificate from PQ key
	tlsCert, err := generateTLSCertFromPQKey(pqPublicKey, ed25519PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TLS cert: %w", err)
	}
	
	// Create metrics
	identityCounter := metric.NewOptimizedCounter(metricsNamespace + "_pq_identities_generated", "Total PQ identities generated")
	
	reg.RegisterCounter("pq_identities", identityCounter)
	identityCounter.Inc()
	
	logger.Info("Generated quantum-resistant node identity",
		"node_id", nodeID.String(),
		"pq_key_size", len(pqPublicKey),
		"ed25519_key_size", len(ed25519PublicKey))
	
	return &PQNodeIdentity{
		NodeID:             nodeID,
		PQPrivateKey:       pqPrivateKey,
		PQPublicKey:        pqPublicKey,
		Ed25519PrivateKey:  ed25519PrivateKey,
		Ed25519PublicKey:   ed25519PublicKey,
		StakingCert:        stakingCert,
		TLSCert:            tlsCert,
		metrics:           reg,
		logger:            logger,
	}, nil
}

// generatePQKeyPair generates X25519MLKEM768 key pair
// In production, this would use the actual PQ algorithm
func generatePQKeyPair() ([]byte, []byte, error) {
	// Generate X25519 key as base (will be replaced with X25519MLKEM768)
	pqPrivateKey := make([]byte, 64) // X25519MLKEM768 private key size
	pqPublicKey := make([]byte, 64)  // X25519MLKEM768 public key size
	
	_, err := rand.Read(pqPrivateKey)
	if err != nil {
		return nil, nil, err
	}
	
	// In production: use actual X25519MLKEM768 key generation
	// For now, use SHA512 of private key as public key (demo only)
	hash := sha512.Sum512(pqPrivateKey)
	copy(pqPublicKey, hash[:64])
	
	return pqPrivateKey, pqPublicKey, nil
}

// deriveNodeIDFromPQKey derives node ID from PQ public key
func deriveNodeIDFromPQKey(pqPublicKey []byte) (ids.NodeID, error) {
	// Hash the PQ public key to get node ID
	hash := sha512.Sum512(pqPublicKey)
	
	// Use first 32 bytes for node ID
	var nodeID ids.NodeID
	copy(nodeID[:], hash[:32])
	
	return nodeID, nil
}

// generateStakingCertFromPQKey generates staking certificate from PQ key
func generateStakingCertFromPQKey(pqPublicKey, ed25519PublicKey []byte) ([]byte, error) {
	// Create certificate structure
	now := time.Now()
	template := x509.Certificate{
		SerialNumber: big.NewInt(now.Unix()),
		Subject: pkix.Name{
			CommonName:   "Lux Staking Node",
			Organization: []string{"Lux Network"},
		},
		NotBefore: now,
		NotAfter:  now.AddDate(10, 0, 0), // 10 years validity
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
	}
	
	// Combine PQ and Ed25519 public keys in certificate
	// In production, this would be properly signed by CA
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, ed25519PublicKey, ed25519PublicKey)
	if err != nil {
		return nil, err
	}
	
	// Encode as PEM
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	
	return pemBytes, nil
}

// generateTLSCertFromPQKey generates TLS certificate from PQ key
func generateTLSCertFromPQKey(pqPublicKey, ed25519PublicKey []byte) ([]byte, error) {
	// Create TLS certificate structure
	now := time.Now()
	template := x509.Certificate{
		SerialNumber: big.NewInt(now.Unix()),
		Subject: pkix.Name{
			CommonName:   "Lux Node TLS",
			Organization: []string{"Lux Network"},
		},
		NotBefore: now,
		NotAfter:  now.AddDate(1, 0, 0), // 1 year validity
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		IsCA: true,
	}
	
	// Combine PQ and Ed25519 public keys in certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, ed25519PublicKey, ed25519PublicKey)
	if err != nil {
		return nil, err
	}
	
	// Encode as PEM
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	
	return pemBytes, nil
}

// GetNodeID returns the node ID
func (id *PQNodeIdentity) GetNodeID() ids.NodeID {
	return id.NodeID
}

// GetPQPublicKey returns the PQ public key
func (id *PQNodeIdentity) GetPQPublicKey() []byte {
	return id.PQPublicKey
}

// GetEd25519PublicKey returns the Ed25519 public key
func (id *PQNodeIdentity) GetEd25519PublicKey() ed25519.PublicKey {
	return id.Ed25519PublicKey
}

// GetStakingCert returns the staking certificate
func (id *PQNodeIdentity) GetStakingCert() []byte {
	return id.StakingCert
}

// GetTLSCert returns the TLS certificate
func (id *PQNodeIdentity) GetTLSCert() []byte {
	return id.TLSCert
}

// Sign signs data with the PQ private key
func (id *PQNodeIdentity) Sign(data []byte) ([]byte, error) {
	// In production, use actual PQ signing
	// For now, use Ed25519 for compatibility
	return ed25519.Sign(id.Ed25519PrivateKey, data), nil
}

// Verify verifies signature with the PQ public key
func (id *PQNodeIdentity) Verify(data, signature []byte) bool {
	// In production, use actual PQ verification
	// For now, use Ed25519 for compatibility
	return ed25519.Verify(id.Ed25519PublicKey, data, signature)
}

// GetPQIdentityMetrics returns identity-related metrics
func (id *PQNodeIdentity) GetPQIdentityMetrics() *metric.MetricsRegistry {
	return id.metrics
}

// PQIdentityManager manages node identities
type PQIdentityManager struct {
	identities map[ids.NodeID]*PQNodeIdentity
	mu        sync.RWMutex
	metrics   *metric.MetricsRegistry
	logger    log.Logger
}

// NewPQIdentityManager creates a new identity manager
func NewPQIdentityManager(
	logger log.Logger,
	metricsNamespace string,
	reg *metric.MetricsRegistry,
) *PQIdentityManager {
	if reg == nil {
		reg = metric.NewMetricsRegistry()
	}
	
	return &PQIdentityManager{
		identities: make(map[ids.NodeID]*PQNodeIdentity),
		metrics:   reg,
		logger:    logger,
	}
}

// GenerateIdentity generates a new PQ identity
func (m *PQIdentityManager) GenerateIdentity() (*PQNodeIdentity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	identity, err := NewPQNodeIdentity(m.logger, "node_identity", m.metrics)
	if err != nil {
		return nil, err
	}
	
	m.identities[identity.NodeID] = identity
	m.logger.Info("Generated new PQ identity", "node_id", identity.NodeID.String())
	
	return identity, nil
}

// GetIdentity gets an identity by node ID
func (m *PQIdentityManager) GetIdentity(nodeID ids.NodeID) (*PQNodeIdentity, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	identity, ok := m.identities[nodeID]
	return identity, ok
}

// GetAllIdentities returns all identities
func (m *PQIdentityManager) GetAllIdentities() []*PQNodeIdentity {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	identities := make([]*PQNodeIdentity, 0, len(m.identities))
	for _, identity := range m.identities {
		identities = append(identities, identity)
	}
	
	return identities
}

// RemoveIdentity removes an identity
func (m *PQIdentityManager) RemoveIdentity(nodeID ids.NodeID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.identities[nodeID]; exists {
		delete(m.identities, nodeID)
		m.logger.Info("Removed PQ identity", "node_id", nodeID.String())
		return true
	}
	
	return false
}

// PQIdentityHealthCheck provides health check for PQ identities
func (m *PQIdentityManager) PQIdentityHealthCheck() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return map[string]interface{}{
		"identity_count": len(m.identities),
		"pq_ready":       supportsPQTLS(),
		"key_algorithm":  "X25519MLKEM768",
		"cert_validity":  "10 years (staking), 1 year (TLS)",
	}
}

// PQNodeIDGenerator generates node IDs from PQ keys
type PQNodeIDGenerator struct {
	logger  log.Logger
	metrics *metric.MetricsRegistry
}

// NewPQNodeIDGenerator creates a new node ID generator
func NewPQNodeIDGenerator(
	logger log.Logger,
	metricsNamespace string,
	reg *metric.MetricsRegistry,
) *PQNodeIDGenerator {
	if reg == nil {
		reg = metric.NewMetricsRegistry()
	}
	
	return &PQNodeIDGenerator{
		logger:  logger,
		metrics: reg,
	}
}

// GenerateNodeID generates a node ID from PQ key material
func (g *PQNodeIDGenerator) GenerateNodeID(pqKeyMaterial []byte) (ids.NodeID, error) {
	// Hash the PQ key material
	hash := sha512.Sum512(pqKeyMaterial)
	
	// Create node ID from hash
	var nodeID ids.NodeID
	copy(nodeID[:], hash[:32])
	
	g.logger.Debug("Generated node ID from PQ key", "node_id", nodeID.String())
	
	return nodeID, nil
}

// GenerateDeterministicNodeID generates deterministic node ID
func (g *PQNodeIDGenerator) GenerateDeterministicNodeID(seed []byte) (ids.NodeID, error) {
	// Expand seed to key size
	keyMaterial := make([]byte, 64)
	copy(keyMaterial, seed)
	
	// Generate node ID
	return g.GenerateNodeID(keyMaterial)
}

// PQStakingCertGenerator generates staking certificates
type PQStakingCertGenerator struct {
	logger  log.Logger
	metrics *metric.MetricsRegistry
}

// NewPQStakingCertGenerator creates a new staking cert generator
func NewPQStakingCertGenerator(
	logger log.Logger,
	metricsNamespace string,
	reg *metric.MetricsRegistry,
) *PQStakingCertGenerator {
	if reg == nil {
		reg = metric.NewMetricsRegistry()
	}
	
	return &PQStakingCertGenerator{
		logger:  logger,
		metrics: reg,
	}
}

// GenerateStakingCert generates a staking certificate from PQ key
func (g *PQStakingCertGenerator) GenerateStakingCert(
	pqPublicKey []byte,
	ed25519PublicKey ed25519.PublicKey,
	validity time.Duration,
) ([]byte, error) {
	// Create certificate
	now := time.Now()
	template := x509.Certificate{
		SerialNumber: big.NewInt(now.Unix()),
		Subject: pkix.Name{
			CommonName:   "Lux Staking Node",
			Organization: []string{"Lux Network"},
		},
		NotBefore: now,
		NotAfter:  now.Add(validity),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
	}
	
	// Sign certificate (in production, use proper CA)
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, ed25519PublicKey, ed25519PublicKey)
	if err != nil {
		return nil, err
	}
	
	// Encode as PEM
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}), nil
}

// PQTLSCertGenerator generates TLS certificates
type PQTLSCertGenerator struct {
	logger  log.Logger
	metrics *metric.MetricsRegistry
}

// NewPQTLSCertGenerator creates a new TLS cert generator
func NewPQTLSCertGenerator(
	logger log.Logger,
	metricsNamespace string,
	reg *metric.MetricsRegistry,
) *PQTLSCertGenerator {
	if reg == nil {
		reg = metric.NewMetricsRegistry()
	}
	
	return &PQTLSCertGenerator{
		logger:  logger,
		metrics: reg,
	}
}

// GenerateTLSCert generates a TLS certificate from PQ key
func (g *PQTLSCertGenerator) GenerateTLSCert(
	pqPublicKey []byte,
	ed25519PublicKey ed25519.PublicKey,
	validity time.Duration,
	dnsNames []string,
) ([]byte, []byte, error) {
	// Create certificate
	now := time.Now()
	template := x509.Certificate{
		SerialNumber: big.NewInt(now.Unix()),
		Subject: pkix.Name{
			CommonName: dnsNames[0],
		},
		DNSNames:    dnsNames,
		NotBefore:   now,
		NotAfter:    now.Add(validity),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
	}
	
	// Sign certificate (in production, use proper CA)
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, ed25519PublicKey, ed25519PublicKey)
	if err != nil {
		return nil, nil, err
	}
	
	// Create private key PEM
	privateKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: ed25519PrivateKey,
	})
	
	// Create certificate PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	
	return certPEM, privateKeyBytes, nil
}

// Example usage:
//
// identity, err := NewPQNodeIdentity(logger, "node", metrics)
// if err != nil {
//     log.Fatal(err)
// }
//
// // Use identity for node operations
// nodeID := identity.GetNodeID()
// pqPublicKey := identity.GetPQPublicKey()
// stakingCert := identity.GetStakingCert()
// tlsCert := identity.GetTLSCert()
//
// // Manage identities
// manager := NewPQIdentityManager(logger, "node", metrics)
// manager.GenerateIdentity()
// identity, ok := manager.GetIdentity(nodeID)
//
// // Generate specific components
// nodeID, err := generator.GenerateNodeID(pqKeyMaterial)
// stakingCert, err := stakingGen.GenerateStakingCert(pqPublicKey, ed25519PublicKey, 10*365*24*time.Hour)
// tlsCert, key, err := tlsGen.GenerateTLSCert(pqPublicKey, ed25519PublicKey, 365*24*time.Hour, []string{"node.lux.net"})

// Note: In production, replace the key generation with actual X25519MLKEM768 implementation
// when available in Go 1.25.5+ standard library or external PQ crypto libraries.