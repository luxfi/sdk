// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package sdk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/luxfi/bft"
	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
)

// BFTConfig contains configuration for BFT consensus engine
type BFTConfig struct {
	// Deployment type
	DeploymentType DeploymentType

	// Network configuration
	NetworkID uint32
	ChainID   ids.ID
	NodeID    ids.NodeID

	// Consensus parameters
	MaxProposalWait    time.Duration
	MaxRebroadcastWait time.Duration

	// Quantum-safe mode (for Quasar Protocol)
	QuantumSafeMode bool

	// BLS signature aggregation
	BLSAggregation bool

	// Epoch configuration
	EpochNumber uint64
	StartTime   time.Time

	// Validator configuration
	Validators []ids.NodeID

	// Storage
	DB  database.Database
	WAL bft.WriteAheadLog

	// Logging
	Logger log.Logger

	// Enable replication protocol
	ReplicationEnabled bool
}

// DeploymentType indicates where the BFT consensus is deployed
type DeploymentType int

const (
	// MainnetDeployment is for Lux mainnet
	MainnetDeployment DeploymentType = iota

	// TestnetDeployment is for Lux testnet
	TestnetDeployment

	// SovereignL1Deployment is for sovereign quantum-safe L1 chains
	// secured by Quasar Protocol (Lux Quantum Consensus)
	SovereignL1Deployment
)

func (dt DeploymentType) String() string {
	switch dt {
	case MainnetDeployment:
		return "mainnet"
	case TestnetDeployment:
		return "testnet"
	case SovereignL1Deployment:
		return "sovereign-l1"
	default:
		return "unknown"
	}
}

// BFTConsensusEngine implements ConsensusEngine interface using the BFT package
type BFTConsensusEngine struct {
	config *BFTConfig
	epoch  *bft.Epoch
	logger log.Logger

	// VM integration
	blockBuilder    bft.BlockBuilder
	storage         bft.Storage
	signer          bft.Signer
	verifier        bft.SignatureVerifier
	aggregator      bft.SignatureAggregator
	communication   bft.Communication
	qcDeserializer  bft.QCDeserializer
	blockDeserializ bft.BlockDeserializer

	// State
	mu      sync.RWMutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewBFTConsensusEngine creates a new BFT consensus engine
func NewBFTConsensusEngine(config *BFTConfig) (*BFTConsensusEngine, error) {
	if config == nil {
		return nil, fmt.Errorf("BFT config is required")
	}

	if config.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	engine := &BFTConsensusEngine{
		config: config,
		logger: config.Logger,
	}

	// Set defaults based on deployment type
	engine.setDefaults()

	return engine, nil
}

// setDefaults configures defaults based on deployment type
func (e *BFTConsensusEngine) setDefaults() {
	cfg := e.config

	switch cfg.DeploymentType {
	case MainnetDeployment:
		// Mainnet: production settings
		if cfg.MaxProposalWait == 0 {
			cfg.MaxProposalWait = 10 * time.Second
		}
		if cfg.MaxRebroadcastWait == 0 {
			cfg.MaxRebroadcastWait = 30 * time.Second
		}
		// BLS aggregation recommended for mainnet
		cfg.BLSAggregation = true

	case TestnetDeployment:
		// Testnet: faster for testing
		if cfg.MaxProposalWait == 0 {
			cfg.MaxProposalWait = 5 * time.Second
		}
		if cfg.MaxRebroadcastWait == 0 {
			cfg.MaxRebroadcastWait = 15 * time.Second
		}
		cfg.BLSAggregation = true

	case SovereignL1Deployment:
		// Sovereign L1: quantum-safe mode enabled
		if cfg.MaxProposalWait == 0 {
			cfg.MaxProposalWait = 8 * time.Second
		}
		if cfg.MaxRebroadcastWait == 0 {
			cfg.MaxRebroadcastWait = 20 * time.Second
		}
		// Quantum-safe mode for Quasar Protocol
		cfg.QuantumSafeMode = true
		cfg.BLSAggregation = true
		cfg.ReplicationEnabled = true
	}

	e.logger.Info("BFT consensus engine configured",
		log.String("deployment_type", cfg.DeploymentType.String()),
		log.Duration("max_proposal_wait", cfg.MaxProposalWait),
		log.Duration("max_rebroadcast_wait", cfg.MaxRebroadcastWait),
		log.Bool("quantum_safe", cfg.QuantumSafeMode),
		log.Bool("bls_aggregation", cfg.BLSAggregation),
	)
}

// Initialize initializes the BFT epoch with the provided components
func (e *BFTConsensusEngine) Initialize(
	blockBuilder bft.BlockBuilder,
	storage bft.Storage,
	signer bft.Signer,
	verifier bft.SignatureVerifier,
	aggregator bft.SignatureAggregator,
	communication bft.Communication,
	qcDeserializer bft.QCDeserializer,
	blockDeserializer bft.BlockDeserializer,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("BFT engine already running")
	}

	e.blockBuilder = blockBuilder
	e.storage = storage
	e.signer = signer
	e.verifier = verifier
	e.aggregator = aggregator
	e.communication = communication
	e.qcDeserializer = qcDeserializer
	e.blockDeserializ = blockDeserializer

	// Create logger adapter
	bftLogger := &loggerAdapter{logger: e.logger}

	// Create epoch config
	epochConfig := bft.EpochConfig{
		ID:                  nodeIDToBFT(e.config.NodeID),
		MaxProposalWait:     e.config.MaxProposalWait,
		MaxRebroadcastWait:  e.config.MaxRebroadcastWait,
		Logger:              bftLogger,
		BlockBuilder:        blockBuilder,
		Storage:             storage,
		Comm:                communication,
		Signer:              signer,
		Verifier:            verifier,
		SignatureAggregator: aggregator,
		QCDeserializer:      qcDeserializer,
		BlockDeserializer:   blockDeserializer,
		Epoch:               e.config.EpochNumber,
		StartTime:           e.config.StartTime,
		ReplicationEnabled:  e.config.ReplicationEnabled,
	}

	if e.config.WAL != nil {
		epochConfig.WAL = e.config.WAL
	}

	// Create BFT epoch
	var err error
	e.epoch, err = bft.NewEpoch(epochConfig)
	if err != nil {
		return fmt.Errorf("failed to create BFT epoch: %w", err)
	}

	e.logger.Info("BFT consensus engine initialized",
		log.String("node_id", e.config.NodeID.String()),
		log.Int("num_validators", len(e.config.Validators)),
		log.String("deployment", e.config.DeploymentType.String()),
		log.Uint64("epoch", e.config.EpochNumber),
	)

	return nil
}

// Start starts the BFT consensus engine
func (e *BFTConsensusEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("BFT engine already running")
	}

	if e.epoch == nil {
		return fmt.Errorf("BFT engine not initialized, call Initialize first")
	}

	e.ctx, e.cancel = context.WithCancel(ctx)
	e.running = true

	// Start the epoch
	if err := e.epoch.Start(); err != nil {
		e.running = false
		e.cancel()
		return fmt.Errorf("failed to start BFT epoch: %w", err)
	}

	e.logger.Info("BFT consensus engine started")
	return nil
}

// Stop stops the BFT consensus engine
func (e *BFTConsensusEngine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	e.running = false
	if e.cancel != nil {
		e.cancel()
	}

	if e.epoch != nil {
		e.epoch.Stop()
	}

	e.logger.Info("BFT consensus engine stopped")
	return nil
}

// GetID returns the chain ID
func (e *BFTConsensusEngine) GetID() ids.ID {
	return e.config.ChainID
}

// SetQuantumMode enables or disables quantum-safe mode
func (e *BFTConsensusEngine) SetQuantumMode(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.config.QuantumSafeMode = enabled
	e.logger.Info("Quantum-safe mode updated", log.Bool("enabled", enabled))
}

// SetBLSAggregation enables or disables BLS signature aggregation
func (e *BFTConsensusEngine) SetBLSAggregation(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.config.BLSAggregation = enabled
	e.logger.Info("BLS aggregation updated", log.Bool("enabled", enabled))
}

// SetVerkleWitnesses enables or disables Verkle tree witnesses
func (e *BFTConsensusEngine) SetVerkleWitnesses(enabled bool) {
	e.logger.Info("Verkle witnesses not yet supported in BFT", log.Bool("requested", enabled))
}

// HandleMessage handles incoming consensus messages
func (e *BFTConsensusEngine) HandleMessage(msg *bft.Message, from ids.NodeID) error {
	e.mu.RLock()
	if !e.running || e.epoch == nil {
		e.mu.RUnlock()
		return fmt.Errorf("BFT engine not running")
	}
	epoch := e.epoch
	e.mu.RUnlock()

	return epoch.HandleMessage(msg, nodeIDToBFT(from))
}

// AdvanceTime hints to the engine that time has passed
func (e *BFTConsensusEngine) AdvanceTime(duration time.Duration) {
	e.mu.RLock()
	if !e.running || e.epoch == nil {
		e.mu.RUnlock()
		return
	}
	e.mu.RUnlock()

	// The epoch's timeout handler will automatically trigger
	// empty votes and leader rotation as needed
}

// GetMetadata returns the current consensus metadata
func (e *BFTConsensusEngine) GetMetadata() *bft.ProtocolMetadata {
	e.mu.RLock()
	if e.epoch == nil {
		e.mu.RUnlock()
		return nil
	}
	epoch := e.epoch
	e.mu.RUnlock()

	metadata := epoch.Metadata()
	return &metadata
}

// IsRunning returns whether the engine is running
func (e *BFTConsensusEngine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// Config returns the engine's configuration
func (e *BFTConsensusEngine) Config() *BFTConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Return a copy to prevent external modification
	configCopy := *e.config
	return &configCopy
}

// Epoch returns the underlying BFT epoch (for advanced use cases)
func (e *BFTConsensusEngine) Epoch() *bft.Epoch {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.epoch
}

// loggerAdapter adapts log.Logger to bft.Logger interface
type loggerAdapter struct {
	logger log.Logger
}

func (l *loggerAdapter) Fatal(msg string, fields ...log.Field) {
	l.logger.Fatal(msg, fields...)
}

func (l *loggerAdapter) Error(msg string, fields ...log.Field) {
	ctx := make([]interface{}, len(fields))
	for i, f := range fields {
		ctx[i] = f
	}
	l.logger.Error(msg, ctx...)
}

func (l *loggerAdapter) Warn(msg string, fields ...log.Field) {
	ctx := make([]interface{}, len(fields))
	for i, f := range fields {
		ctx[i] = f
	}
	l.logger.Warn(msg, ctx...)
}

func (l *loggerAdapter) Info(msg string, fields ...log.Field) {
	ctx := make([]interface{}, len(fields))
	for i, f := range fields {
		ctx[i] = f
	}
	l.logger.Info(msg, ctx...)
}

func (l *loggerAdapter) Trace(msg string, fields ...log.Field) {
	ctx := make([]interface{}, len(fields))
	for i, f := range fields {
		ctx[i] = f
	}
	l.logger.Trace(msg, ctx...)
}

func (l *loggerAdapter) Debug(msg string, fields ...log.Field) {
	ctx := make([]interface{}, len(fields))
	for i, f := range fields {
		ctx[i] = f
	}
	l.logger.Debug(msg, ctx...)
}

func (l *loggerAdapter) Verbo(msg string, fields ...log.Field) {
	l.logger.Verbo(msg, fields...)
}

// nodeIDToBFT converts ids.NodeID to bft.NodeID
func nodeIDToBFT(id ids.NodeID) bft.NodeID {
	return bft.NodeID(id[:])
}

// bftToNodeID converts bft.NodeID to ids.NodeID
func bftToNodeID(id bft.NodeID) ids.NodeID {
	var nodeID ids.NodeID
	copy(nodeID[:], id)
	return nodeID
}
