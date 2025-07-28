// (c) 2020-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package iface

import (
	"context"
	"fmt"
	"github.com/luxfi/geth/common"
)

// NodeConsensus provides chain-specific consensus information for node integration
type NodeConsensus interface {
	// GetConsensusParamsAt returns consensus parameters at a given block height
	GetConsensusParamsAt(ctx context.Context, blockHeight uint64) (*NodeConsensusParams, error)
	
	// IsUpgradeActive checks if an upgrade is active at a given timestamp
	IsUpgradeActive(upgradeID string, timestamp uint64) bool
}

// NodeConsensusParams represents consensus parameters for node integration
type NodeConsensusParams struct {
	// BlockGasCost is the base gas cost for a block
	BlockGasCost uint64
	
	// BlockGasLimit is the maximum gas allowed in a block
	BlockGasLimit uint64
	
	// MinBaseFee is the minimum base fee
	MinBaseFee uint64
	
	// TargetBlockRate is the target rate for block production
	TargetBlockRate uint64
	
	// BaseFeeChangeDenominator controls base fee adjustment rate
	BaseFeeChangeDenominator uint64
	
	// MinBlockGasCost is the minimum gas cost for a block
	MinBlockGasCost uint64
	
	// MaxBlockGasCost is the maximum gas cost for a block
	MaxBlockGasCost uint64
	
	// BlockGasCostStep is the step size for block gas cost changes
	BlockGasCostStep uint64
}

// Validator represents a validator
type Validator interface {
	// NodeID returns the validator's node ID
	NodeID() NodeID
	
	// Weight returns the validator's weight
	Weight() uint64
}

// ValidatorState provides access to validator information
type ValidatorState interface {
	// GetCurrentHeight returns the current P-chain height
	GetCurrentHeight(ctx context.Context) (uint64, error)
	
	// GetValidatorSet returns the validator set at a given height
	GetValidatorSet(ctx context.Context, height uint64, subnetID common.Hash) (map[common.Hash]*ValidatorOutput, error)
	
	// GetMinimumHeight returns the minimum height
	GetMinimumHeight(ctx context.Context) (uint64, error)
	
	// GetSubnetID returns the subnet ID for a given chain ID
	GetSubnetID(ctx context.Context, chainID common.Hash) (common.Hash, error)
}

// ValidatorData contains validator information
type ValidatorData struct {
	NodeID    NodeID
	PublicKey []byte
	Weight    uint64
}

// Choice represents the status of a decision
type Choice uint32

const (
	// Undecided means the decision hasn't been made yet
	Undecided Choice = iota
	// Processing means the decision is being processed
	Processing
	// Accepted means the decision was accepted
	Accepted
	// Rejected means the decision was rejected
	Rejected
)

func (c Choice) String() string {
	switch c {
	case Undecided:
		return "Undecided"
	case Processing:
		return "Processing"
	case Accepted:
		return "Accepted"
	case Rejected:
		return "Rejected"
	default:
		return fmt.Sprintf("Choice(%d)", c)
	}
}

// Decidable represents an element that can be decided on
type Decidable interface {
	// ID returns the ID of this element
	ID() BlockID
	
	// Status returns the current status
	Status() Choice
	
	// Accept accepts this element
	Accept(context.Context) error
	
	// Reject rejects this element
	Reject(context.Context) error
}

// State provides access to validator state
type State interface {
	// GetCurrentHeight returns the current blockchain height
	GetCurrentHeight(ctx context.Context) (uint64, error)
	
	// GetMinimumHeight returns the minimum height
	GetMinimumHeight(ctx context.Context) (uint64, error)
	
	// GetSubnetID returns the subnet ID for a chain
	GetSubnetID(ctx context.Context, chainID ID) (ID, error)
	
	// GetValidatorSet returns the validator set at a given height
	GetValidatorSet(ctx context.Context, height uint64, subnetID ID) (map[NodeID]*GetValidatorOutput, error)
}

// GetValidatorOutput represents validator information
type GetValidatorOutput struct {
	NodeID    NodeID
	PublicKey []byte
	Weight    uint64
}

// ConsensusConstants for node consensus
const (
	// DefaultMinBlockGasCost is the default minimum block gas cost
	DefaultMinBlockGasCost = 0
	
	// DefaultMaxBlockGasCost is the default maximum block gas cost
	DefaultMaxBlockGasCost = 10_000_000
	
	// DefaultTargetBlockRate is the default target block rate (2 seconds)
	DefaultTargetBlockRate = 2
	
	// DefaultBlockGasCostStep is the default block gas cost step
	DefaultBlockGasCostStep = 200_000
)