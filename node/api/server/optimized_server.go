// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package server provides optimized server implementations using VictoriaMetrics-style optimizations
package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/luxfi/fasthttp"
	"github.com/luxfi/log"
	"github.com/luxfi/metric"
	"github.com/luxfi/pool"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// OptimizedServer provides a high-performance server using VictoriaMetrics-style optimizations
type OptimizedServer struct {
	*server // Embed standard server for compatibility
	
	// Optimized components
	fasthttpServer *fasthttp.Server
	pqTLSConfig    *PQTLSConfig
	metrics       *metric.MetricsRegistry
	
	// Configuration
	useFastHTTP bool
	usePQTLS    bool
	maxConns    int
	
	// Metrics
	requestCounter  *metric.OptimizedCounter
	requestDuration *metric.OptimizedHistogram
	activeConns     *metric.OptimizedGauge
}

// NewOptimizedServer creates a new optimized server
func NewOptimizedServer(
	ctx context.Context,
	logger log.Logger,
	metricsNamespace string,
	useFastHTTP bool,
	usePQTLS bool,
	options ...ServerOption,
) (*OptimizedServer, error) {
	// Create standard server first
	baseServer, err := NewServer(ctx, logger, metricsNamespace, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create base server: %w", err)
	}
	
	// Create metrics registry
	reg := metric.NewMetricsRegistry()
	
	// Create optimized metrics
	requestCounter := metric.NewOptimizedCounter(metricsNamespace + "_requests_total", "Total requests received by optimized server")

	requestDuration := metric.NewOptimizedHistogram(metricsNamespace + "_request_duration_seconds", "Request processing duration", []float64{0.001, 0.01, 0.05, 0.1, 0.5, 1.0, 2.5, 5.0, 10.0})

	activeConns := metric.NewOptimizedGauge(metricsNamespace + "_active_connections", "Currently active connections")
	
	reg.RegisterCounter("requests", requestCounter)
	reg.RegisterHistogram("request_duration", requestDuration)
	reg.RegisterGauge("active_connections", activeConns)
	
	// Create PQ TLS config if enabled
	var pqConfig *PQTLSConfig
	if usePQTLS {
		var err error
		pqConfig, err = NewPQTLSConfig(
			logger,
			metricsNamespace,
			reg,
			WithEnforcePQKeyExchange(true),
			WithPQKeyExchangeGroups("X25519MLKEM768"),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create PQ TLS config: %w", err)
		}
		
		// Register PQ metrics
		pqMetrics := pqConfig.GetPQMetrics()
		for _, metric := range pqMetrics.Collect() {
			reg.Register(metric)
		}
	}
	
	// Create optimized server
	s := &OptimizedServer{
		server:         baseServer.(*server),
		useFastHTTP:    useFastHTTP,
		usePQTLS:      usePQTLS,
		pqTLSConfig:    pqConfig,
		metrics:       reg,
		requestCounter: requestCounter,
		requestDuration: requestDuration,
		activeConns:    activeConns,
	}
	
	// Configure based on options
	for _, opt := range options {
		opt.apply(s)
	}
	
	// Set reasonable defaults for FastHTTP
	if s.useFastHTTP {
		s.maxConns = 10000
	}
	
	return s, nil
}

// Start starts the optimized server
func (s *OptimizedServer) Start() error {
	if s.useFastHTTP {
		return s.startFastHTTP()
	}
	return s.server.Start()
}

// startFastHTTP starts the FastHTTP server
func (s *OptimizedServer) startFastHTTP() error {
	// Create FastHTTP handler wrapper
	handler := s.createFastHTTPHandler()
	
	// Create FastHTTP server
	s.fasthttpServer = fasthttp.NewServer(handler)
	
	// Configure server
	s.fasthttpServer.Server.TCPKeepalive = 3 * time.Minute
	s.fasthttpServer.Server.ReadTimeout = s.server.config.readTimeout
	s.fasthttpServer.Server.WriteTimeout = s.server.config.writeTimeout
	s.fasthttpServer.Server.IdleTimeout = s.server.config.idleTimeout
	s.fasthttpServer.Server.MaxConnsPerIP = s.maxConns
	s.fasthttpServer.Server.MaxRequestsPerConn = 1000
	
	// Start server with PQ TLS if enabled
	addr := net.JoinHostPort(s.server.config.host, fmt.Sprintf("%d", s.server.config.port))
	
	if s.usePQTLS && s.pqTLSConfig != nil {
		s.server.logger.Info("Starting optimized FastHTTP server with PQ TLS", 
			"address", addr,
			"pq_groups", strings.Join(s.pqTLSConfig.PQKeyExchangeGroups, ", "))
		return s.startFastHTTPWithPQTLS(addr)
	}
	
	s.server.logger.Info("Starting optimized FastHTTP server", "address", addr)
	return s.fasthttpServer.ListenAndServe(addr)
}

// startFastHTTPWithPQTLS starts FastHTTP server with PQ TLS
func (s *OptimizedServer) startFastHTTPWithPQTLS(addr string) error {
	// Create standard listener
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	
	// Wrap with PQ TLS
	pqListener := NewPQTLSListener(ln, s.pqTLSConfig, s.server.logger)
	
	// Start FastHTTP server with PQ listener
	return s.fasthttpServer.Server.Serve(pqListener)
}

// createFastHTTPHandler creates a FastHTTP handler that wraps the standard HTTP handler
func (s *OptimizedServer) createFastHTTPHandler() http.Handler {
	// Wrap standard handler with optimization middleware
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Start timing
		timer := metric.NewTimingMetric(s.requestDuration)
		defer timer.Stop()
		
		// Increment counters
		s.requestCounter.Inc()
		s.activeConns.Inc()
		defer s.activeConns.Dec()
		
		// Use pooled buffer for request processing
		buf := pool.GetByteSlice()
		defer pool.PutByteSlice(buf)
		
		// Wrap response writer for pooling
		pw := &pooledResponseWriter{
			ResponseWriter: w,
			buffer:         pool.GetFastBuffer(),
		}
		defer pool.PutFastBuffer(pw.buffer)
		
		// Call standard handler
		s.server.ServeHTTP(pw, r)
	})
}

// pooledResponseWriter wraps http.ResponseWriter with pooling optimizations
type pooledResponseWriter struct {
	http.ResponseWriter
	buffer *pool.FastBuffer
	
	status int
	header http.Header
}

func (w *pooledResponseWriter) Write(b []byte) (int, error) {
	// Use pooled buffer for response
	w.buffer.Write(b)
	return len(b), nil
}

func (w *pooledResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *pooledResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *pooledResponseWriter) Flush() {
	// Write buffered data
	if w.buffer.Len() > 0 {
		w.ResponseWriter.Write(w.buffer.Bytes())
		w.buffer.Reset()
	}
	
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Stop stops the optimized server
func (s *OptimizedServer) Stop() error {
	if s.fasthttpServer != nil {
		s.server.logger.Info("Stopping optimized FastHTTP server")
		return s.fasthttpServer.Shutdown()
	}
	return s.server.Stop()
}

// GetMetrics returns the metrics registry for monitoring
func (s *OptimizedServer) GetMetrics() *metric.MetricsRegistry {
	return s.metrics
}

// GetHealthCheck returns health check information
func (s *OptimizedServer) GetHealthCheck() map[string]interface{} {
	health := map[string]interface{}{
		"server_type": "optimized",
		"fast_http":   s.useFastHTTP,
		"pq_tls":      s.usePQTLS,
		"max_conns":   s.maxConns,
	}
	
	if s.usePQTLS && s.pqTLSConfig != nil {
		health["pq_config"] = s.pqTLSConfig.PQTLSHealthCheck()
	}
	
	return health
}

// GetPQTLSConfig returns the PQ TLS configuration
func (s *OptimizedServer) GetPQTLSConfig() *PQTLSConfig {
	return s.pqTLSConfig
}

// OptimizedServerOption defines options for the optimized server
type OptimizedServerOption interface {
	apply(*OptimizedServer)
}

// WithMaxConnections sets the maximum number of connections
type withMaxConnections int

func (m withMaxConnections) apply(s *OptimizedServer) {
	s.maxConns = int(m)
}

// WithMaxConnections creates an option to set max connections
func WithMaxConnections(max int) OptimizedServerOption {
	return withMaxConnections(max)
}

// OptimizedHandler provides an optimized HTTP handler wrapper
type OptimizedHandler struct {
	handler http.Handler
	metrics *metric.MetricsRegistry
	
	// Metrics
	handlerCounter  *metric.OptimizedCounter
	handlerDuration *metric.OptimizedHistogram
	handlerErrors   *metric.OptimizedCounter
}

// NewOptimizedHandler creates a new optimized handler
func NewOptimizedHandler(
	handler http.Handler,
	metricsNamespace string,
	reg *metric.MetricsRegistry,
) *OptimizedHandler {
	if reg == nil {
		reg = metric.NewMetricsRegistry()
	}
	
	handlerCounter := metric.NewOptimizedCounter(metricsNamespace + "_handler_requests_total", "Total requests handled")

	handlerDuration := metric.NewOptimizedHistogram(metricsNamespace + "_handler_duration_seconds", "Handler processing duration", []float64{0.001, 0.01, 0.05, 0.1, 0.5, 1.0, 2.5, 5.0})

	handlerErrors := metric.NewOptimizedCounter(metricsNamespace + "_handler_errors_total", "Total handler errors")
	
	reg.RegisterCounter("handler_requests", handlerCounter)
	reg.RegisterHistogram("handler_duration", handlerDuration)
	reg.RegisterCounter("handler_errors", handlerErrors)
	
	return &OptimizedHandler{
		handler:         handler,
		metrics:        reg,
		handlerCounter: handlerCounter,
		handlerDuration: handlerDuration,
		handlerErrors:  handlerErrors,
	}
}

// ServeHTTP implements http.Handler with optimizations
func (h *OptimizedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Start timing
	timer := metric.NewTimingMetric(h.handlerDuration)
	defer timer.Stop()
	
	// Increment counter
	h.handlerCounter.Inc()
	
	// Use pooled buffer for request body
	if r.Body != nil {
		buf := pool.GetByteSlice()
		defer pool.PutByteSlice(buf)
		
		// Read body into pooled buffer
		_, err := r.Body.Read(buf)
		if err != nil && err.Error() != "EOF" {
			h.handlerErrors.Inc()
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
	}
	
	// Wrap response writer
	pw := &pooledResponseWriter{
		ResponseWriter: w,
		buffer:         pool.GetFastBuffer(),
	}
	defer pool.PutFastBuffer(pw.buffer)
	
	// Call underlying handler
	h.handler.ServeHTTP(pw, r)
}

// GetMetrics returns the metrics registry
func (h *OptimizedHandler) GetMetrics() *metric.MetricsRegistry {
	return h.metrics
}

// OptimizedCORSHandler provides CORS handling with optimizations
type OptimizedCORSHandler struct {
	corsHandler cors.CorsHandler
	metrics    *metric.MetricsRegistry
	
	// Metrics
	corsCounter  *metric.OptimizedCounter
	corsDuration *metric.OptimizedHistogram
}

// NewOptimizedCORSHandler creates a new optimized CORS handler
func NewOptimizedCORSHandler(
	corsHandler cors.CorsHandler,
	metricsNamespace string,
	reg *metric.MetricsRegistry,
) *OptimizedCORSHandler {
	if reg == nil {
		reg = metric.NewMetricsRegistry()
	}
	
	corsCounter := metric.NewOptimizedCounter(metricsNamespace + "_cors_requests_total", "Total CORS requests")

	corsDuration := metric.NewOptimizedHistogram(metricsNamespace + "_cors_duration_seconds", "CORS processing duration", []float64{0.001, 0.01, 0.05, 0.1, 0.5})
	
	reg.RegisterCounter("cors_requests", corsCounter)
	reg.RegisterHistogram("cors_duration", corsDuration)
	
	return &OptimizedCORSHandler{
		corsHandler: corsHandler,
		metrics:    reg,
		corsCounter: corsCounter,
		corsDuration: corsDuration,
	}
}

// ServeHTTP implements http.Handler with CORS optimizations
func (h *OptimizedCORSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Start timing
	timer := metric.NewTimingMetric(h.corsDuration)
	defer timer.Stop()
	
	// Increment counter
	h.corsCounter.Inc()
	
	// Use pooled buffer for CORS processing
	pw := &pooledResponseWriter{
		ResponseWriter: w,
		buffer:         pool.GetFastBuffer(),
	}
	defer pool.PutFastBuffer(pw.buffer)
	
	// Call CORS handler
	h.corsHandler.ServeHTTP(pw, r)
}

// GetMetrics returns the metrics registry
func (h *OptimizedCORSHandler) GetMetrics() *metric.MetricsRegistry {
	return h.metrics
}

// OptimizedHTTP2Handler provides HTTP/2 handling with optimizations
type OptimizedHTTP2Handler struct {
	handler http.Handler
	metrics *metric.MetricsRegistry
	
	// Metrics
	http2Counter  *metric.OptimizedCounter
	http2Duration *metric.OptimizedHistogram
}

// NewOptimizedHTTP2Handler creates a new optimized HTTP/2 handler
func NewOptimizedHTTP2Handler(
	handler http.Handler,
	metricsNamespace string,
	reg *metric.MetricsRegistry,
) *OptimizedHTTP2Handler {
	if reg == nil {
		reg = metric.NewMetricsRegistry()
	}
	
	http2Counter := metric.NewOptimizedCounter(metricsNamespace + "_http2_requests_total", "Total HTTP/2 requests")

	http2Duration := metric.NewOptimizedHistogram(metricsNamespace + "_http2_duration_seconds", "HTTP/2 request duration", []float64{0.001, 0.01, 0.05, 0.1, 0.5, 1.0})
	
	reg.RegisterCounter("http2_requests", http2Counter)
	reg.RegisterHistogram("http2_duration", http2Duration)
	
	return &OptimizedHTTP2Handler{
		handler:      handler,
		metrics:      reg,
		http2Counter: http2Counter,
		http2Duration: http2Duration,
	}
}

// ServeHTTP implements http.Handler with HTTP/2 optimizations
func (h *OptimizedHTTP2Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Start timing
	timer := metric.NewTimingMetric(h.http2Duration)
	defer timer.Stop()
	
	// Increment counter
	h.http2Counter.Inc()
	
	// Use pooled buffer
	pw := &pooledResponseWriter{
		ResponseWriter: w,
		buffer:         pool.GetFastBuffer(),
	}
	defer pool.PutFastBuffer(pw.buffer)
	
	// Call underlying handler
	h.handler.ServeHTTP(pw, r)
}

// GetMetrics returns the metrics registry
func (h *OptimizedHTTP2Handler) GetMetrics() *metric.MetricsRegistry {
	return h.metrics
}