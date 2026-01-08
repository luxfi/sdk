// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package airdrop

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"sync"
	"time"

		"github.com/luxfi/geth/common"
		"github.com/luxfi/geth/ethclient"

	"github.com/luxfi/sdk/node/config"
	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
)

var (
	errAirdropNotEnabled     = errors.New("airdrop not enabled")
	errAirdropAlreadyClaimed = errors.New("airdrop already claimed")
	errAirdropExpired        = errors.New("airdrop claim period expired")
	errInvalidREQLBalance    = errors.New("no REQL balance at snapshot")
)

// Manager manages the REQL to LUX airdrop
type Manager struct {
	config        *config.AirdropConfig
	tokenomics    *config.TokenomicsConfig
	db            database.Database
	ethClient     *ethclient.Client
	reqlSnapshots map[common.Address]*REQLSnapshot
	claims        map[common.Address]*AirdropClaim
	mu            sync.RWMutex
	log           log.Logger
}

// REQLSnapshot represents a REQL holder's balance at snapshot time
type REQLSnapshot struct {
	Address       common.Address `json:"address"`
	REQLBalance   *big.Int       `json:"reqlBalance"`
	LUXAllocation *big.Int       `json:"luxAllocation"`
	SnapshotBlock uint64         `json:"snapshotBlock"`
	SnapshotTime  time.Time      `json:"snapshotTime"`
}

// AirdropClaim represents a claim on the airdrop
type AirdropClaim struct {
	Address        common.Address `json:"address"`
	LuxAddress     ids.ShortID    `json:"luxAddress"`
	AmountClaimed  *big.Int       `json:"amountClaimed"`
	ClaimTime      time.Time      `json:"claimTime"`
	VestingEndTime time.Time      `json:"vestingEndTime"`
	TxID           ids.ID         `json:"txId"`
}

// NewManager creates a new airdrop manager
func NewManager(
	config *config.AirdropConfig,
	tokenomics *config.TokenomicsConfig,
	db database.Database,
	ethClient *ethclient.Client,
	log log.Logger,
) (*Manager, error) {
	if !config.Enabled {
		return nil, errAirdropNotEnabled
	}

	manager := &Manager{
		config:        config,
		tokenomics:    tokenomics,
		db:            db,
		ethClient:     ethClient,
		reqlSnapshots: make(map[common.Address]*REQLSnapshot),
		claims:        make(map[common.Address]*AirdropClaim),
		log:           log,
	}

	// Load existing claims from database
	if err := manager.loadClaims(); err != nil {
		return nil, err
	}

	// Load REQL snapshot data
	if err := manager.loadSnapshot(); err != nil {
		return nil, err
	}

	return manager, nil
}

// CheckEligibility checks if an address is eligible for the airdrop
func (m *Manager) CheckEligibility(ethAddress common.Address) (*REQLSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if already claimed
	if _, claimed := m.claims[ethAddress]; claimed {
		return nil, errAirdropAlreadyClaimed
	}

	// Check if in snapshot
	snapshot, exists := m.reqlSnapshots[ethAddress]
	if !exists {
		return nil, errInvalidREQLBalance
	}

	// Check if claim period has expired
	snapshotTime, _ := time.Parse(time.RFC3339, m.config.SnapshotDate)
	claimDeadline := snapshotTime.Add(time.Duration(m.config.ClaimPeriod) * time.Second)
	if time.Now().After(claimDeadline) {
		return nil, errAirdropExpired
	}

	return snapshot, nil
}

// ClaimAirdrop processes an airdrop claim
func (m *Manager) ClaimAirdrop(
	ctx context.Context,
	ethAddress common.Address,
	luxAddress ids.ShortID,
	signature []byte,
) (*AirdropClaim, error) {
	// Verify eligibility
	snapshot, err := m.CheckEligibility(ethAddress)
	if err != nil {
		return nil, err
	}

	// Verify signature (proves ownership of ETH address)
	if err := m.verifySignature(ethAddress, luxAddress, signature); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Create claim record
	claim := &AirdropClaim{
		Address:        ethAddress,
		LuxAddress:     luxAddress,
		AmountClaimed:  snapshot.LUXAllocation,
		ClaimTime:      time.Now(),
		VestingEndTime: time.Now().Add(time.Duration(m.config.VestingPeriod) * time.Second),
	}

	// Process the claim (mint tokens with vesting)
	txID, err := m.processClaim(ctx, claim)
	if err != nil {
		return nil, err
	}
	claim.TxID = txID

	// Save claim to database
	if err := m.saveClaim(claim); err != nil {
		return nil, err
	}

	m.claims[ethAddress] = claim

	m.log.Info("Airdrop claimed",
		"ethAddress", ethAddress.Hex(),
		"luxAddress", luxAddress,
		"amount", claim.AmountClaimed.String(),
		"txID", txID,
	)

	return claim, nil
}

// GetAirdropStats returns statistics about the airdrop
func (m *Manager) GetAirdropStats() *AirdropStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalAllocated := big.NewInt(0)
	totalClaimed := big.NewInt(0)

	for _, snapshot := range m.reqlSnapshots {
		totalAllocated.Add(totalAllocated, snapshot.LUXAllocation)
	}

	for _, claim := range m.claims {
		totalClaimed.Add(totalClaimed, claim.AmountClaimed)
	}

	return &AirdropStats{
		TotalREQLHolders:   uint64(len(m.reqlSnapshots)),
		TotalLUXAllocated:  totalAllocated,
		TotalClaims:        uint64(len(m.claims)),
		TotalLUXClaimed:    totalClaimed,
		ConversionRatio:    m.config.ConversionRatio,
		ClaimPeriodEnds:    m.getClaimDeadline(),
	}
}

// AirdropStats contains statistics about the airdrop
type AirdropStats struct {
	TotalREQLHolders  uint64    `json:"totalREQLHolders"`
	TotalLUXAllocated *big.Int  `json:"totalLUXAllocated"`
	TotalClaims       uint64    `json:"totalClaims"`
	TotalLUXClaimed   *big.Int  `json:"totalLUXClaimed"`
	ConversionRatio   float64   `json:"conversionRatio"`
	ClaimPeriodEnds   time.Time `json:"claimPeriodEnds"`
}

// loadSnapshot loads the REQL holder snapshot from file or chain
func (m *Manager) loadSnapshot() error {
	// In production, this would load from a verified snapshot file
	// or query historical blockchain state
	// For now, we'll create a mock snapshot

	// Example snapshot entries
	mockSnapshots := []REQLSnapshot{
		{
			Address:       common.HexToAddress("0x1234567890123456789012345678901234567890"),
			REQLBalance:   new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)), // 1M REQL
			LUXAllocation: new(big.Int).Mul(big.NewInt(1000000000), big.NewInt(1e9)), // 1B LUX
		},
		// Add more snapshot entries...
	}

	for _, snapshot := range mockSnapshots {
		snapshot.SnapshotTime, _ = time.Parse(time.RFC3339, m.config.SnapshotDate)
		m.reqlSnapshots[snapshot.Address] = &snapshot
	}

	return nil
}

// loadClaims loads existing claims from the database
func (m *Manager) loadClaims() error {
	// Load claims from database
	claimsData, err := m.db.Get([]byte("airdrop_claims"))
	if err != nil {
		if err == database.ErrNotFound {
			return nil // No existing claims
		}
		return err
	}

	var claims map[common.Address]*AirdropClaim
	if err := json.Unmarshal(claimsData, &claims); err != nil {
		return err
	}

	m.claims = claims
	return nil
}

// saveClaim saves a claim to the database
func (m *Manager) saveClaim(claim *AirdropClaim) error {
	claimsData, err := json.Marshal(m.claims)
	if err != nil {
		return err
	}

	return m.db.Put([]byte("airdrop_claims"), claimsData)
}

// verifySignature verifies ownership of the Ethereum address
func (m *Manager) verifySignature(ethAddress common.Address, luxAddress ids.ShortID, signature []byte) error {
	// In production, implement proper signature verification
	// to prove ownership of the Ethereum address
	return nil
}

// processClaim processes the actual token minting and vesting
func (m *Manager) processClaim(ctx context.Context, claim *AirdropClaim) (ids.ID, error) {
	// In production, this would:
	// 1. Create a vesting transaction on P-Chain
	// 2. Mint tokens to the specified LUX address
	// 3. Apply vesting schedule
	
	// For now, return a mock transaction ID
	return ids.GenerateTestID(), nil
}

// getClaimDeadline returns the deadline for claiming airdrops
func (m *Manager) getClaimDeadline() time.Time {
	snapshotTime, _ := time.Parse(time.RFC3339, m.config.SnapshotDate)
	return snapshotTime.Add(time.Duration(m.config.ClaimPeriod) * time.Second)
}

// GetClaimStatus returns the claim status for an address
func (m *Manager) GetClaimStatus(ethAddress common.Address) (*ClaimStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := &ClaimStatus{
		Address: ethAddress,
	}

	// Check if claimed
	if claim, exists := m.claims[ethAddress]; exists {
		status.Claimed = true
		status.Claim = claim
		return status, nil
	}

	// Check eligibility
	if snapshot, exists := m.reqlSnapshots[ethAddress]; exists {
		status.Eligible = true
		status.Snapshot = snapshot
		status.ClaimDeadline = m.getClaimDeadline()
	}

	return status, nil
}

// ClaimStatus represents the airdrop claim status for an address
type ClaimStatus struct {
	Address       common.Address `json:"address"`
	Eligible      bool           `json:"eligible"`
	Claimed       bool           `json:"claimed"`
	Snapshot      *REQLSnapshot  `json:"snapshot,omitempty"`
	Claim         *AirdropClaim  `json:"claim,omitempty"`
	ClaimDeadline time.Time      `json:"claimDeadline,omitempty"`
}