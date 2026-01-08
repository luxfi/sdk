// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package platformvm

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"

		"github.com/luxfi/geth/accounts/abi/bind"
		"github.com/luxfi/geth/common"
		"github.com/luxfi/geth/ethclient"
	"github.com/luxfi/log"
	
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
)

var (
	errNoNFTOwned           = errors.New("validator does not own required NFT")
	errInvalidNFTContract   = errors.New("invalid NFT contract address")
	errNFTNotInValidTier    = errors.New("NFT not in valid tier for staking")
	errNFTAlreadyStaked     = errors.New("NFT already staked by another validator")
)

// NFTStakingManager manages NFT-based validator staking
type NFTStakingManager struct {
	config      *NFTStakingConfig
	ethClient   *ethclient.Client
	nftContract *ValidatorNFTContract
	stakedNFTs  map[uint64]ids.NodeID // tokenID -> NodeID mapping
	mu          sync.RWMutex
	log         log.Logger
}

// NFTStakingConfig contains configuration for NFT staking
type NFTStakingConfig struct {
	Enabled         bool
	ContractAddress common.Address
	RequiredBalance uint64
	ValidatorTiers  []ValidatorTier
	EthRPCEndpoint  string
}

// ValidatorTier represents a tier of validator NFTs
type ValidatorTier struct {
	Name              string
	MinTokenID        uint64
	MaxTokenID        uint64
	StakingMultiplier uint32
}

// ValidatorNFTContract interface for interacting with NFT contract
type ValidatorNFTContract interface {
	OwnerOf(opts *bind.CallOpts, tokenId *big.Int) (common.Address, error)
	BalanceOf(opts *bind.CallOpts, owner common.Address) (*big.Int, error)
	TokenURI(opts *bind.CallOpts, tokenId *big.Int) (string, error)
}

// NewNFTStakingManager creates a new NFT staking manager
func NewNFTStakingManager(config *NFTStakingConfig, log log.Logger) (*NFTStakingManager, error) {
	if !config.Enabled {
		return nil, nil
	}
	
	// Connect to Ethereum client
	client, err := ethclient.Dial(config.EthRPCEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum: %w", err)
	}
	
	// TODO: Initialize NFT contract binding
	// This would use abigen-generated bindings in production
	
	manager := &NFTStakingManager{
		config:     config,
		ethClient:  client,
		stakedNFTs: make(map[uint64]ids.NodeID),
		log:        log,
	}
	
	return manager, nil
}

// ValidateNFTOwnership checks if a node owns a valid NFT for staking
func (m *NFTStakingManager) ValidateNFTOwnership(
	ctx context.Context,
	nodeID ids.NodeID,
	ethAddress common.Address,
	tokenID uint64,
) error {
	if m == nil || !m.config.Enabled {
		return nil // NFT staking not enabled
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if NFT is already staked
	if existingNode, staked := m.stakedNFTs[tokenID]; staked {
		if existingNode != nodeID {
			return errNFTAlreadyStaked
		}
		// Already staked by this node, which is fine
		return nil
	}
	
	// Verify ownership on-chain
	if m.nftContract != nil {
		owner, err := (*m.nftContract).OwnerOf(&bind.CallOpts{Context: ctx}, big.NewInt(int64(tokenID)))
		if err != nil {
			return fmt.Errorf("failed to check NFT ownership: %w", err)
		}
		
		if owner != ethAddress {
			return errNoNFTOwned
		}
	}
	
	// Check if token is in valid tier
	validTier := false
	for _, tier := range m.config.ValidatorTiers {
		if tokenID >= tier.MinTokenID && tokenID <= tier.MaxTokenID {
			validTier = true
			break
		}
	}
	
	if !validTier {
		return errNFTNotInValidTier
	}
	
	// Record NFT as staked
	m.stakedNFTs[tokenID] = nodeID
	
	m.log.Info("NFT staked for validation",
		log.String("nodeID", nodeID.String()),
		log.Uint64("tokenID", tokenID),
		log.String("ethAddress", ethAddress.Hex()),
	)
	
	return nil
}

// GetStakingMultiplier returns the staking multiplier for a given NFT
func (m *NFTStakingManager) GetStakingMultiplier(tokenID uint64) uint32 {
	if m == nil || !m.config.Enabled {
		return 100 // Default multiplier
	}
	
	for _, tier := range m.config.ValidatorTiers {
		if tokenID >= tier.MinTokenID && tokenID <= tier.MaxTokenID {
			return tier.StakingMultiplier
		}
	}
	
	return 100 // Default multiplier
}

// ReleaseNFT releases an NFT from staking
func (m *NFTStakingManager) ReleaseNFT(nodeID ids.NodeID, tokenID uint64) error {
	if m == nil || !m.config.Enabled {
		return nil
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if stakedNode, exists := m.stakedNFTs[tokenID]; exists {
		if stakedNode != nodeID {
			return errors.New("NFT not staked by this node")
		}
		delete(m.stakedNFTs, tokenID)
		
		m.log.Info("NFT released from staking",
			log.String("nodeID", nodeID.String()),
			log.Uint64("tokenID", tokenID),
		)
	}
	
	return nil
}

// GetStakedNFTs returns all currently staked NFTs
func (m *NFTStakingManager) GetStakedNFTs() map[uint64]ids.NodeID {
	if m == nil || !m.config.Enabled {
		return nil
	}
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to prevent external modification
	result := make(map[uint64]ids.NodeID)
	for tokenID, nodeID := range m.stakedNFTs {
		result[tokenID] = nodeID
	}
	
	return result
}

// ValidateNFTStaking validates NFT staking for a staker
func ValidateNFTStaking(
	ctx context.Context,
	staker *state.Staker,
	nftManager *NFTStakingManager,
) error {
	if nftManager == nil || staker.ValidatorNFT == nil {
		return nil // No NFT staking required
	}
	
	// Parse Ethereum address from staker
	ethAddr := common.HexToAddress(staker.ValidatorNFT.ContractAddress)
	
	// Validate NFT ownership
	return nftManager.ValidateNFTOwnership(
		ctx,
		staker.NodeID,
		ethAddr,
		staker.ValidatorNFT.TokenID,
	)
}

// ApplyNFTMultiplier applies the NFT staking multiplier to rewards
func ApplyNFTMultiplier(
	baseReward uint64,
	tokenID uint64,
	nftManager *NFTStakingManager,
) uint64 {
	if nftManager == nil {
		return baseReward
	}
	
	multiplier := nftManager.GetStakingMultiplier(tokenID)
	return (baseReward * uint64(multiplier)) / 100
}
