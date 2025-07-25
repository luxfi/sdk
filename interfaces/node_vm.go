// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package interfaces

import (
	"context"
	"fmt"
	"time"

	"github.com/luxfi/geth/common"
)

// NodeID represents a node identifier
type NodeID [32]byte

// BlockID represents a block identifier
type BlockID [32]byte

// ID represents a generic identifier
type ID [32]byte

// String returns the string representation of an ID
func (id ID) String() string {
	return fmt.Sprintf("%x", id[:])
}

// String returns the string representation of a NodeID
func (id NodeID) String() string {
	return fmt.Sprintf("%x", id[:])
}

// String returns the string representation of a BlockID
func (id BlockID) String() string {
	return fmt.Sprintf("%x", id[:])
}

// RequestID represents a request identifier
type RequestID uint32

// EmptyID is an empty ID
var EmptyID = ID{}

// BuildTestNodeID creates a test node ID for testing
func BuildTestNodeID(key string) NodeID {
	var id NodeID
	copy(id[:], []byte(key))
	return id
}

// GenerateTestID creates a test ID from pseudo-random data
func GenerateTestID() ID {
	var id ID
	// Use current time and some constant for deterministic "randomness"
	for i := 0; i < 32; i++ {
		id[i] = byte(i * 7)
	}
	return id
}

// GenerateTestNodeID creates a test NodeID from pseudo-random data
func GenerateTestNodeID() NodeID {
	var id NodeID
	// Use current time and some constant for deterministic "randomness"
	for i := 0; i < 32; i++ {
		id[i] = byte(i * 11)
	}
	return id
}

// NodeBlock represents a blockchain block interface for node integration
type NodeBlock interface {
	ID() BlockID
	Parent() BlockID
	Height() uint64
	Timestamp() uint64
	Bytes() []byte
	Verify(context.Context) error
	Accept(context.Context) error
	Reject(context.Context) error
}

// NodeVM defines the interface for the node virtual machine
type NodeVM interface {
	// Initialize initializes the VM
	Initialize(
		ctx context.Context,
		chainCtx *ChainContext,
		db Database,
		genesisBytes []byte,
		upgradeBytes []byte,
		configBytes []byte,
		toEngine chan<- EngineMessage,
		fxs []Fx,
		appSender AppSender,
	) error

	// SetState sets the VM state
	SetState(ctx context.Context, state State) error

	// Shutdown shuts down the VM
	Shutdown(context.Context) error

	// Version returns the VM version
	Version(context.Context) (string, error)

	// CreateHandlers returns HTTP handlers
	CreateHandlers(context.Context) (map[string]HTTPHandler, error)

	// HealthCheck returns the VM health
	HealthCheck(context.Context) (interface{}, error)
}

// ChainVM extends NodeVM with chain-specific operations
type ChainVM interface {
	NodeVM
	BlockGetter
	BlockParser

	// BuildBlock attempts to build a new block
	BuildBlock(context.Context) (NodeBlock, error)

	// SetPreference sets the preferred block ID
	SetPreference(ctx context.Context, blkID BlockID) error

	// LastAccepted returns the last accepted block ID
	LastAccepted(context.Context) (BlockID, error)
}

// BlockGetter defines block retrieval operations
type BlockGetter interface {
	// GetBlock retrieves a block by ID
	GetBlock(ctx context.Context, blkID BlockID) (NodeBlock, error)
}

// BlockParser defines block parsing operations
type BlockParser interface {
	// ParseBlock parses block bytes into a Block
	ParseBlock(ctx context.Context, blockBytes []byte) (NodeBlock, error)
}

// BuildBlockWithContextChainVM defines VMs that can build blocks with context
type BuildBlockWithContextChainVM interface {
	ChainVM
	// BuildBlockWithContext builds a block with additional context
	BuildBlockWithContext(ctx context.Context, blockContext *BlockBuildContext) (NodeBlock, error)
}

// StateSyncableVM defines VMs that support state sync
type StateSyncableVM interface {
	ChainVM
	// StateSyncEnabled returns if state sync is enabled
	StateSyncEnabled(context.Context) (bool, error)
	// GetOngoingSyncStateSummary returns the ongoing sync state summary
	GetOngoingSyncStateSummary(context.Context) (StateSummary, error)
	// GetLastStateSummary returns the last state summary
	GetLastStateSummary(context.Context) (StateSummary, error)
	// ParseStateSummary parses state summary bytes
	ParseStateSummary(ctx context.Context, summaryBytes []byte) (StateSummary, error)
	// GetStateSummary returns a state summary at the given height
	GetStateSummary(ctx context.Context, height uint64) (StateSummary, error)
	// VerifyHeightIndex verifies the height index
	VerifyHeightIndex(context.Context) error
	// AcceptStateSummary accepts a state summary
	AcceptStateSummary(ctx context.Context, stateSummary StateSummary) error
}

// ChainContext provides chain context
type ChainContext struct {
	NetworkID uint32
	SubnetID  SubnetID
	ChainID   ChainID
	NodeID    NodeID

	// Node version
	AppVersion uint32

	// Chain configuration
	ChainDataDir string

	// Network upgrades configuration
	NetworkUpgrades Config
}

// SubnetID represents a subnet identifier
type SubnetID [32]byte

// ChainID represents a chain identifier
type ChainID [32]byte

// BlockBuildContext provides context for building blocks
type BlockBuildContext struct {
	PChainHeight uint64
}

// StateSummary represents a state summary
type StateSummary interface {
	ID() BlockID
	Height() uint64
	Bytes() []byte
	Accept(context.Context) error
}

// EngineMessage represents a message to the consensus engine
type EngineMessage struct {
	InboundMessage
	EngineType
}

// InboundMessage represents an inbound message
type InboundMessage interface{}

// EngineType represents the consensus engine type
type EngineType uint32

// VMState represents VM states
type VMState uint32

// Status represents the status of a block
type Status int

// Timestamp represents a block timestamp
type Timestamp struct {
	Unix int64
}

const (
	Bootstrapping VMState = iota
	NormalOp
)

// Fx represents a feature extension
type Fx interface{}

// AppSender sends application messages
type AppSender interface {
	SendAppRequest(ctx context.Context, nodeIDs Set, requestID uint32, request []byte) error
	SendAppResponse(ctx context.Context, nodeID NodeID, requestID uint32, response []byte) error
	SendAppGossip(ctx context.Context, config SendConfig, appGossipBytes []byte) error
}

// Set represents a set of node IDs
type Set interface {
	Contains(NodeID) bool
	Len() int
	List() []NodeID
	Add(...NodeID)
	Remove(...NodeID)
}

// SendConfig configures message sending
type SendConfig struct {
	NodeIDs       Set
	Validators    int
	NonValidators int
	Peers         int
}

// HTTPHandler represents an HTTP handler
type HTTPHandler interface{}

// Timer provides timer functionality
type Timer interface {
	NewTimer(func()) *TimerID
	Dispatch()
}

// TimerID represents a timer ID
type TimerID struct{}

// Clock provides time functionality
type Clock interface {
	Time() time.Time
	Unix() int64
}

// WarpMessage represents a warp message
type WarpMessage struct {
	SourceChainID       common.Hash
	OriginSenderAddress common.Address
	Payload             []byte
}

// WarpSignature represents a warp signature
type WarpSignature struct {
	Signers   []byte
	Signature []byte
}

// WarpUnsignedMessage represents an unsigned warp message
type WarpUnsignedMessage struct {
	NetworkID     uint32
	SourceChainID common.Hash
	Payload       []byte
}

// WarpBackend provides warp functionality
type WarpBackend interface {
	GetBlockSignature(ctx context.Context, blockID BlockID) (*WarpSignature, error)
	GetMessageSignature(ctx context.Context, msg *WarpUnsignedMessage) (*WarpSignature, error)
	AddMessage(ctx context.Context, msg *WarpUnsignedMessage) error
}