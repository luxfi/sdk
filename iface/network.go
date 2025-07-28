// (c) 2020-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package iface

import (
	"context"
	"time"
)

// Network provides network operations
type Network interface {
	// SendAppRequest sends an application request to specified nodes
	SendAppRequest(ctx context.Context, nodeIDs Set, requestID uint32, request []byte) error
	
	// SendAppResponse sends an application response to a specific node
	SendAppResponse(ctx context.Context, nodeID NodeID, requestID uint32, response []byte) error
	
	// SendAppGossip gossips application data to peers
	SendAppGossip(ctx context.Context, config SendConfig, appGossipBytes []byte) error
	
	// SendCrossChainAppRequest sends a cross-chain app request
	SendCrossChainAppRequest(ctx context.Context, chainID ChainID, requestID uint32, appRequestBytes []byte) error
	
	// SendCrossChainAppResponse sends a cross-chain app response
	SendCrossChainAppResponse(ctx context.Context, chainID ChainID, requestID uint32, appResponseBytes []byte) error
}

// GossipHandler handles gossip messages
type GossipHandler interface {
	// HandleEthTxs handles ethereum transaction gossip
	HandleEthTxs(nodeID NodeID, msg GossipMessage) error
}

// GossipMessage represents a gossip message
type GossipMessage interface {
	// Bytes returns the message bytes
	Bytes() []byte
}

// RequestHandler handles network requests
type RequestHandler interface {
	// HandleTrieLeafsRequest handles trie leafs requests
	HandleTrieLeafsRequest(ctx context.Context, nodeID NodeID, requestID uint32, request LeafsRequest) ([]byte, error)
	
	// HandleBlockRequest handles block requests
	HandleBlockRequest(ctx context.Context, nodeID NodeID, requestID uint32, request BlockRequest) ([]byte, error)
	
	// HandleCodeRequest handles code requests
	HandleCodeRequest(ctx context.Context, nodeID NodeID, requestID uint32, request CodeRequest) ([]byte, error)
	
	// HandleMessageSignatureRequest handles message signature requests
	HandleMessageSignatureRequest(ctx context.Context, nodeID NodeID, requestID uint32, request MessageSignatureRequest) ([]byte, error)
	
	// HandleBlockSignatureRequest handles block signature requests
	HandleBlockSignatureRequest(ctx context.Context, nodeID NodeID, requestID uint32, request BlockSignatureRequest) ([]byte, error)
}

// CrossChainRequestHandler handles cross-chain requests
type CrossChainRequestHandler interface {
	// HandleEthCallRequest handles ethereum call requests
	HandleEthCallRequest(ctx context.Context, requestingChainID ChainID, requestID uint32, ethCallRequest EthCallRequest) ([]byte, error)
}

// LeafsRequest represents a request for trie leafs
type LeafsRequest struct {
	Root     []byte
	Account  []byte
	Start    []byte
	End      []byte
	Limit    uint16
	NodeType uint
}

// BlockRequest represents a request for blocks
type BlockRequest struct {
	Hash    []byte
	Number  uint64
	Parents uint16
}

// CodeRequest represents a request for code
type CodeRequest struct {
	Hashes [][]byte
}

// MessageSignatureRequest represents a request for message signature
type MessageSignatureRequest struct {
	MessageID []byte
}

// BlockSignatureRequest represents a request for block signature
type BlockSignatureRequest struct {
	BlockID []byte
}

// EthCallRequest represents an ethereum call request
type EthCallRequest struct {
	RequestArgs []byte
}

// Metrics provides metric collection
type Metrics interface {
	// IncCounter increments a counter metric
	IncCounter(name string)
	
	// AddSample adds a sample to a metric
	AddSample(name string, value float64)
	
	// SetGauge sets a gauge metric
	SetGauge(name string, value float64)
}

// P2PClient provides peer-to-peer communication
type P2PClient interface {
	// SendRequest sends a request to a peer
	SendRequest(ctx context.Context, nodeID NodeID, request []byte, responseHandler func([]byte, error)) error
	
	// TrackBandwidth tracks bandwidth usage
	TrackBandwidth(nodeID NodeID, bandwidth float64)
}

// PeerTracker tracks peer information
type PeerTracker interface {
	// Connected notifies that a peer connected
	Connected(nodeID NodeID, version *Version)
	
	// Disconnected notifies that a peer disconnected
	Disconnected(nodeID NodeID)
	
	// NumPeers returns the number of connected peers
	NumPeers() int
	
	// ConnectedPeers returns the set of connected peers
	ConnectedPeers() []NodeID
}

// Version represents peer version information
type Version struct {
	// Application version
	App uint32
	
	// Application name
	AppName string
	
	// Version timestamp
	VersionTime time.Time
	
	// Git commit
	GitCommit string
}