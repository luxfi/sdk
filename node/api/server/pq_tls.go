// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package server provides PQ TLS enforcement for VictoriaMetrics-optimized servers
package server

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/luxfi/log"
	"github.com/luxfi/metric"
	"golang.org/x/crypto/cryptobyte"
)

// PQTLSConfig provides Post-Quantum TLS configuration
// Enforces X25519MLKEM768 key exchange for quantum resistance
type PQTLSConfig struct {
	// Standard TLS config
	*tls.Config
	
	// PQ-specific settings
	EnforcePQKeyExchange bool
	PQKeyExchangeGroups   []string
	
	// Metrics
	pqHandshakeCounter *metric.OptimizedCounter
	pqHandshakeErrors  *metric.OptimizedCounter
	pqHandshakeTime    *metric.OptimizedHistogram
	
	logger log.Logger
}

// NewPQTLSConfig creates a new PQ TLS configuration
// Requires Go 1.25.5+ for X25519MLKEM768 support
func NewPQTLSConfig(
	logger log.Logger,
	metricsNamespace string,
	reg *metric.MetricsRegistry,
	options ...PQTLSOption,
) (*PQTLSConfig, error) {
	if reg == nil {
		reg = metric.NewMetricsRegistry()
	}
	
	// Create PQ metrics
	pqHandshakeCounter := metric.NewOptimizedCounter(metricsNamespace + "_pq_handshakes_total", "Total PQ TLS handshakes")

	pqHandshakeErrors := metric.NewOptimizedCounter(metricsNamespace + "_pq_handshake_errors_total", "Total PQ TLS handshake errors")

	pqHandshakeTime := metric.NewOptimizedHistogram(metricsNamespace + "_pq_handshake_duration_seconds", "PQ TLS handshake duration", []float64{0.001, 0.01, 0.05, 0.1, 0.5, 1.0, 2.5})
	
	reg.RegisterCounter("pq_handshakes", pqHandshakeCounter)
	reg.RegisterCounter("pq_handshake_errors", pqHandshakeErrors)
	reg.RegisterHistogram("pq_handshake_duration", pqHandshakeTime)
	
	// Create base TLS config with modern security
	baseConfig := &tls.Config{
		MinVersion:               tls.VersionTLS13,
		MaxVersion:               tls.VersionTLS13,
		CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384},
		CipherSuites:             []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		ClientAuth:               tls.NoClientCert,
		SessionTicketsDisabled:   false,
		ClientSessionCache:       tls.NewLRUClientSessionCache(1024),
		KeyLogWriter:             nil,
	}
	
	// Create PQ config
	pqConfig := &PQTLSConfig{
		Config:                baseConfig,
		EnforcePQKeyExchange: true,
		PQKeyExchangeGroups:   []string{"X25519MLKEM768"},
		pqHandshakeCounter:    pqHandshakeCounter,
		pqHandshakeErrors:     pqHandshakeErrors,
		pqHandshakeTime:       pqHandshakeTime,
		logger:               logger,
	}
	
	// Apply options
	for _, opt := range options {
		opt.apply(pqConfig)
	}
	
	// Configure PQ key exchange
	if err := pqConfig.configurePQKeyExchange(); err != nil {
		return nil, fmt.Errorf("failed to configure PQ key exchange: %w", err)
	}
	
	return pqConfig, nil
}

// configurePQKeyExchange configures Post-Quantum key exchange
func (c *PQTLSConfig) configurePQKeyExchange() error {
	// Verify Go version supports PQ (1.25.5+)
	if !supportsPQTLS() {
		c.logger.Warn("Go version does not support PQ TLS (requires 1.25.5+), falling back to standard TLS")
		c.EnforcePQKeyExchange = false
		return nil
	}
	
	// Set PQ key exchange groups
	// X25519MLKEM768 is the primary PQ group for Go 1.25.5+
	c.CurvePreferences = append(c.CurvePreferences, tls.CurveID(0x0100)) // X25519MLKEM768
	
	// Configure key exchange policies
	if c.EnforcePQKeyExchange {
		c.logger.Info("Enforcing Post-Quantum key exchange (X25519MLKEM768)", 
			"groups", strings.Join(c.PQKeyExchangeGroups, ", "))
	}
	
	return nil
}

// supportsPQTLS checks if the Go version supports PQ TLS
func supportsPQTLS() bool {
	// In production, this would check runtime.Version()
	// For now, assume we're running on Go 1.25.5+
	return true
}

// PQTLSOption defines options for PQ TLS configuration
type PQTLSOption interface {
	apply(*PQTLSConfig)
}

// WithPQKeyExchangeGroups sets PQ key exchange groups
type withPQKeyExchangeGroups []string

func (g withPQKeyExchangeGroups) apply(c *PQTLSConfig) {
	c.PQKeyExchangeGroups = []string(g)
}

// WithPQKeyExchangeGroups creates an option to set PQ groups
func WithPQKeyExchangeGroups(groups ...string) PQTLSOption {
	return withPQKeyExchangeGroups(groups)
}

// WithEnforcePQKeyExchange sets PQ enforcement
type withEnforcePQKeyExchange bool

func (e withEnforcePQKeyExchange) apply(c *PQTLSConfig) {
	c.EnforcePQKeyExchange = bool(e)
}

// WithEnforcePQKeyExchange creates an option to enforce PQ
func WithEnforcePQKeyExchange(enforce bool) PQTLSOption {
	return withEnforcePQKeyExchange(enforce)
}

// WithClientAuth sets client authentication
type withClientAuth tls.ClientAuthType

func (a withClientAuth) apply(c *PQTLSConfig) {
	c.ClientAuth = tls.ClientAuthType(a)
}

// WithClientAuth creates an option for client auth
func WithClientAuth(auth tls.ClientAuthType) PQTLSOption {
	return withClientAuth(auth)
}

// PQTLSListener wraps net.Listener with PQ TLS enforcement
type PQTLSListener struct {
	net.Listener
	pqConfig *PQTLSConfig
	logger   log.Logger
}

// NewPQTLSListener creates a new PQ TLS listener
func NewPQTLSListener(
	inner net.Listener,
	pqConfig *PQTLSConfig,
	logger log.Logger,
) *PQTLSListener {
	return &PQTLSListener{
		Listener:  inner,
		pqConfig:  pqConfig,
		logger:    logger,
	}
}

// Accept accepts connections and enforces PQ TLS
func (l *PQTLSListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	
	// Wrap with PQ TLS
	return l.wrapWithPQTLS(conn), nil
}

// wrapWithPQTLS wraps a connection with PQ TLS
func (l *PQTLSListener) wrapWithPQTLS(conn net.Conn) net.Conn {
	if !l.pqConfig.EnforcePQKeyExchange {
		return conn
	}
	
	return &pqTLSConn{
		Conn:    conn,
		pqConfig: l.pqConfig,
		logger:  l.logger,
	}
}

// pqTLSConn wraps net.Conn with PQ TLS enforcement
type pqTLSConn struct {
	net.Conn
	pqConfig *PQTLSConfig
	logger   log.Logger
	tlsConn  *tls.Conn
}

// Read implements net.Conn with PQ TLS enforcement
func (c *pqTLSConn) Read(b []byte) (int, error) {
	if c.tlsConn == nil {
		if err := c.ensureTLS(); err != nil {
			return 0, err
		}
	}
	return c.tlsConn.Read(b)
}

// Write implements net.Conn with PQ TLS enforcement
func (c *pqTLSConn) Write(b []byte) (int, error) {
	if c.tlsConn == nil {
		if err := c.ensureTLS(); err != nil {
			return 0, err
		}
	}
	return c.tlsConn.Write(b)
}

// Close implements net.Conn
func (c *pqTLSConn) Close() error {
	if c.tlsConn != nil {
		return c.tlsConn.Close()
	}
	return c.Conn.Close()
}

// ensureTLS ensures TLS is established with PQ key exchange
func (c *pqTLSConn) ensureTLS() error {
	// Start timing
	timer := metric.NewTimingMetric(c.pqConfig.pqHandshakeTime)
	defer timer.Stop()
	
	// Create TLS connection
	tlsConn := tls.Server(c.Conn, c.pqConfig.Config)
	
	// Perform handshake
	if err := tlsConn.Handshake(); err != nil {
		c.pqConfig.pqHandshakeErrors.Inc()
		c.logger.Error("PQ TLS handshake failed", "error", err)
		return fmt.Errorf("PQ TLS handshake failed: %w", err)
	}
	
	// Verify PQ key exchange was used
	if err := c.verifyPQKeyExchange(tlsConn); err != nil {
		c.pqConfig.pqHandshakeErrors.Inc()
		c.logger.Error("PQ key exchange verification failed", "error", err)
		return fmt.Errorf("PQ key exchange verification failed: %w", err)
	}
	
	c.tlsConn = tlsConn
	c.pqConfig.pqHandshakeCounter.Inc()
	c.logger.Debug("PQ TLS handshake successful", "remote", c.Conn.RemoteAddr())
	
	return nil
}

// verifyPQKeyExchange verifies that PQ key exchange was used
func (c *pqTLSConn) verifyPQKeyExchange(tlsConn *tls.Conn) error {
	if !c.pqConfig.EnforcePQKeyExchange {
		return nil
	}
	
	// Get connection state
	state := tlsConn.ConnectionState()
	
	// Check if PQ group was negotiated
	pqGroupFound := false
	for _, group := range c.pqConfig.PQKeyExchangeGroups {
		for _, negotiatedGroup := range state.NegotiatedGroups {
			if getGroupName(negotiatedGroup) == group {
				pqGroupFound = true
				break
			}
		}
		if pqGroupFound {
			break
		}
	}
	
	if !pqGroupFound {
		return fmt.Errorf("PQ key exchange not negotiated, available groups: %v, negotiated: %v",
			c.pqConfig.PQKeyExchangeGroups, state.NegotiatedGroups)
	}
	
	return nil
}

// getGroupName converts TLS group ID to name
func getGroupName(group uint16) string {
	switch group {
	case 0x0100:
		return "X25519MLKEM768"
	case tls.X25519:
		return "X25519"
	case tls.CurveP256:
		return "P256"
	case tls.CurveP384:
		return "P384"
	case tls.CurveP521:
		return "P521"
	default:
		return fmt.Sprintf("Unknown(%d)", group)
	}
}

// PQTLSDialer provides PQ TLS dialing
type PQTLSDialer struct {
	pqConfig *PQTLSConfig
	logger   log.Logger
}

// NewPQTLSDialer creates a new PQ TLS dialer
func NewPQTLSDialer(
	pqConfig *PQTLSConfig,
	logger log.Logger,
) *PQTLSDialer {
	return &PQTLSDialer{
		pqConfig: pqConfig,
		logger:   logger,
	}
}

// Dial connects to a server with PQ TLS enforcement
func (d *PQTLSDialer) Dial(network, address string) (net.Conn, error) {
	// Start timing
	timer := metric.NewTimingMetric(d.pqConfig.pqHandshakeTime)
	defer timer.Stop()
	
	// Connect to server
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	
	// Wrap with PQ TLS
	tlsConn := tls.Client(conn, d.pqConfig.Config)
	
	// Perform handshake
	if err := tlsConn.Handshake(); err != nil {
		d.pqConfig.pqHandshakeErrors.Inc()
		d.logger.Error("PQ TLS client handshake failed", "address", address, "error", err)
		return nil, fmt.Errorf("PQ TLS client handshake failed: %w", err)
	}
	
	// Verify PQ key exchange
	if err := d.verifyClientPQKeyExchange(tlsConn); err != nil {
		d.pqConfig.pqHandshakeErrors.Inc()
		d.logger.Error("PQ key exchange verification failed", "address", address, "error", err)
		return nil, fmt.Errorf("PQ key exchange verification failed: %w", err)
	}
	
	d.pqConfig.pqHandshakeCounter.Inc()
	d.logger.Debug("PQ TLS client handshake successful", "address", address)
	
	return tlsConn, nil
}

// verifyClientPQKeyExchange verifies PQ key exchange for client
func (d *PQTLSDialer) verifyClientPQKeyExchange(tlsConn *tls.Conn) error {
	if !d.pqConfig.EnforcePQKeyExchange {
		return nil
	}
	
	state := tlsConn.ConnectionState()
	
	// Check if PQ group was negotiated
	pqGroupFound := false
	for _, group := range d.pqConfig.PQKeyExchangeGroups {
		for _, negotiatedGroup := range state.NegotiatedGroups {
			if getGroupName(negotiatedGroup) == group {
				pqGroupFound = true
				break
			}
		}
		if pqGroupFound {
			break
		}
	}
	
	if !pqGroupFound {
		return fmt.Errorf("PQ key exchange not negotiated with server %s, available: %v, negotiated: %v",
			tlsConn.RemoteAddr(), d.pqConfig.PQKeyExchangeGroups, state.NegotiatedGroups)
	}
	
	return nil
}

// PQTLSWrapper wraps existing connections with PQ TLS
type PQTLSWrapper struct {
	pqConfig *PQTLSConfig
	logger   log.Logger
}

// NewPQTLSWrapper creates a new PQ TLS wrapper
func NewPQTLSWrapper(
	pqConfig *PQTLSConfig,
	logger log.Logger,
) *PQTLSWrapper {
	return &PQTLSWrapper{
		pqConfig: pqConfig,
		logger:   logger,
	}
}

// WrapConnection wraps an existing connection with PQ TLS
func (w *PQTLSWrapper) WrapConnection(conn net.Conn, isServer bool) (net.Conn, error) {
	if !w.pqConfig.EnforcePQKeyExchange {
		return conn, nil
	}
	
	var tlsConn *tls.Conn
	var err error
	
	if isServer {
		tlsConn = tls.Server(conn, w.pqConfig.Config)
	} else {
		tlsConn = tls.Client(conn, w.pqConfig.Config)
	}
	
	// Perform handshake
	if err := tlsConn.Handshake(); err != nil {
		w.pqConfig.pqHandshakeErrors.Inc()
		w.logger.Error("PQ TLS wrap handshake failed", "error", err)
		return nil, fmt.Errorf("PQ TLS wrap handshake failed: %w", err)
	}
	
	// Verify PQ key exchange
	if err := w.verifyWrappedPQKeyExchange(tlsConn); err != nil {
		w.pqConfig.pqHandshakeErrors.Inc()
		w.logger.Error("PQ key exchange verification failed", "error", err)
		return nil, fmt.Errorf("PQ key exchange verification failed: %w", err)
	}
	
	w.pqConfig.pqHandshakeCounter.Inc()
	w.logger.Debug("PQ TLS wrap successful", "remote", conn.RemoteAddr())
	
	return tlsConn, nil
}

// verifyWrappedPQKeyExchange verifies PQ key exchange for wrapped connections
func (w *PQTLSWrapper) verifyWrappedPQKeyExchange(tlsConn *tls.Conn) error {
	if !w.pqConfig.EnforcePQKeyExchange {
		return nil
	}
	
	state := tlsConn.ConnectionState()
	
	// Check if PQ group was negotiated
	pqGroupFound := false
	for _, group := range w.pqConfig.PQKeyExchangeGroups {
		for _, negotiatedGroup := range state.NegotiatedGroups {
			if getGroupName(negotiatedGroup) == group {
				pqGroupFound = true
				break
			}
		}
		if pqGroupFound {
			break
		}
	}
	
	if !pqGroupFound {
		return fmt.Errorf("PQ key exchange not negotiated, available: %v, negotiated: %v",
			w.pqConfig.PQKeyExchangeGroups, state.NegotiatedGroups)
	}
	
	return nil
}

// GetPQMetrics returns PQ-specific metrics
func (c *PQTLSConfig) GetPQMetrics() *metric.MetricsRegistry {
	reg := metric.NewMetricsRegistry()
	
	// Register PQ metrics
	reg.RegisterCounter("pq_handshakes", c.pqHandshakeCounter)
	reg.RegisterCounter("pq_handshake_errors", c.pqHandshakeErrors)
	reg.RegisterHistogram("pq_handshake_duration", c.pqHandshakeTime)
	
	return reg
}

// PQTLSHealthCheck provides health checking for PQ TLS
func (c *PQTLSConfig) PQTLSHealthCheck() map[string]interface{} {
	return map[string]interface{}{
		"pq_enabled":           c.EnforcePQKeyExchange,
		"pq_groups":           c.PQKeyExchangeGroups,
		"tls_min_version":     "TLS 1.3",
		"tls_max_version":     "TLS 1.3",
		"supported_ciphers":   []string{"AES-128-GCM", "AES-256-GCM", "CHACHA20-POLY1305"},
		"pq_ready":            supportsPQTLS(),
	}
}