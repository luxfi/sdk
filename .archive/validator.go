// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
	"github.com/luxfi/set"

	"github.com/luxfi/sdk/blockchain"
	"github.com/luxfi/sdk/wallet"
)

// Status represents validator status
type Status string

const (
	StatusPending Status = "pending"
	StatusActive  Status = "active"
	StatusExpired Status = "expired"
	StatusRevoked Status = "revoked"
)

// Validator represents a blockchain validator
type Validator struct {
	// Identity
	NodeID       ids.NodeID     `json:"nodeId"`
	Address      ids.ShortID    `json:"address"`
	BLSPublicKey *bls.PublicKey `json:"blsPublicKey"`

	// Staking info
	StakeAmount  uint64    `json:"stakeAmount"`
	StakeAssetID ids.ID    `json:"stakeAssetId"`
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime"`

	// Delegation
	DelegationFee float64     `json:"delegationFee"` // Percentage
	Delegators    []Delegator `json:"delegators"`

	// Status
	Status Status `json:"status"`

	// Performance
	Uptime       float64 `json:"uptime"` // Percentage
	BlocksSigned uint64  `json:"blocksSigned"`

	// Rewards
	RewardsEarned uint64 `json:"rewardsEarned"`

	// Metadata
	Metadata map[string]string `json:"metadata"`
}

// Delegator represents a validator delegator
type Delegator struct {
	Address     ids.ShortID `json:"address"`
	Amount      uint64      `json:"amount"`
	StartTime   time.Time   `json:"startTime"`
	EndTime     time.Time   `json:"endTime"`
	RewardShare float64     `json:"rewardShare"` // Percentage after fee
}

// Manager manages validator operations
type Manager struct {
	wallet      *wallet.Wallet
	validators  map[ids.NodeID]*Validator
	myValidator *Validator
}

// NewManager creates a new validator manager
func NewManager(wallet *wallet.Wallet) *Manager {
	return &Manager{
		wallet:     wallet,
		validators: make(map[ids.NodeID]*Validator),
	}
}

// Register registers as a validator
func (m *Manager) Register(ctx context.Context, config RegisterConfig) (*Validator, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Check if already registered
	if m.myValidator != nil {
		return nil, errors.New("already registered as validator")
	}

	// Get wallet address
	address, err := m.wallet.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet address: %w", err)
	}

	// Check balance
	balance := m.wallet.GetBalance(config.StakeAssetID)
	if balance < config.StakeAmount {
		return nil, fmt.Errorf("insufficient balance: have %d, need %d", balance, config.StakeAmount)
	}

	// Get or generate BLS key
	blsKey, err := m.wallet.GetBLSKey()
	if err != nil {
		// Generate new BLS key
		blsKey, err = bls.NewSecretKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate BLS key: %w", err)
		}
		m.wallet.SetBLSKey(blsKey)
	}

	// Create validator
	validator := &Validator{
		NodeID:        config.NodeID,
		Address:       address,
		BLSPublicKey:  bls.PublicKeyFromSecretKey(blsKey),
		StakeAmount:   config.StakeAmount,
		StakeAssetID:  config.StakeAssetID,
		StartTime:     config.StartTime,
		EndTime:       config.EndTime,
		DelegationFee: config.DelegationFee,
		Status:        StatusPending,
		Metadata:      make(map[string]string),
	}

	// TODO: Submit validator registration transaction
	// This would interact with the P-Chain or equivalent

	m.myValidator = validator
	m.validators[validator.NodeID] = validator

	return validator, nil
}

// StartValidating starts the validation process
func (m *Manager) StartValidating(ctx context.Context) error {
	if m.myValidator == nil {
		return errors.New("not registered as validator")
	}

	if m.myValidator.Status != StatusPending {
		return fmt.Errorf("invalid status: %s", m.myValidator.Status)
	}

	// Check if start time has passed
	if time.Now().Before(m.myValidator.StartTime) {
		return errors.New("start time not reached")
	}

	// Update status
	m.myValidator.Status = StatusActive

	// Start validation loop
	go m.validationLoop(ctx)

	return nil
}

// StopValidating stops validation
func (m *Manager) StopValidating(ctx context.Context) error {
	if m.myValidator == nil {
		return errors.New("not registered as validator")
	}

	if m.myValidator.Status != StatusActive {
		return fmt.Errorf("not actively validating: %s", m.myValidator.Status)
	}

	// Update status
	m.myValidator.Status = StatusExpired

	// TODO: Submit exit transaction

	return nil
}

// AddDelegator adds a delegator to the validator
func (m *Manager) AddDelegator(ctx context.Context, delegator Delegator) error {
	if m.myValidator == nil {
		return errors.New("not registered as validator")
	}

	// Validate delegator
	if delegator.Amount == 0 {
		return errors.New("delegation amount must be greater than 0")
	}

	if delegator.EndTime.Before(delegator.StartTime) {
		return errors.New("delegation end time must be after start time")
	}

	// Check if delegator already exists
	for _, d := range m.myValidator.Delegators {
		if d.Address == delegator.Address {
			return errors.New("delegator already exists")
		}
	}

	// Calculate reward share
	delegator.RewardShare = 100 - m.myValidator.DelegationFee

	m.myValidator.Delegators = append(m.myValidator.Delegators, delegator)

	// TODO: Process delegation transaction

	return nil
}

// RemoveDelegator removes a delegator
func (m *Manager) RemoveDelegator(ctx context.Context, address ids.ShortID) error {
	if m.myValidator == nil {
		return errors.New("not registered as validator")
	}

	found := false
	delegators := make([]Delegator, 0, len(m.myValidator.Delegators))
	for _, d := range m.myValidator.Delegators {
		if d.Address != address {
			delegators = append(delegators, d)
		} else {
			found = true
		}
	}

	if !found {
		return errors.New("delegator not found")
	}

	m.myValidator.Delegators = delegators

	// TODO: Process undelegation transaction

	return nil
}

// GetValidator returns a validator by node ID
func (m *Manager) GetValidator(nodeID ids.NodeID) (*Validator, error) {
	validator, exists := m.validators[nodeID]
	if !exists {
		return nil, errors.New("validator not found")
	}
	return validator, nil
}

// GetMyValidator returns the current validator
func (m *Manager) GetMyValidator() (*Validator, error) {
	if m.myValidator == nil {
		return nil, errors.New("not registered as validator")
	}
	return m.myValidator, nil
}

// ListValidators returns all known validators
func (m *Manager) ListValidators() []*Validator {
	validators := make([]*Validator, 0, len(m.validators))
	for _, v := range m.validators {
		validators = append(validators, v)
	}
	return validators
}

// GetValidatorSet returns the current validator set
func (m *Manager) GetValidatorSet(ctx context.Context) (set.Set[ids.NodeID], error) {
	validatorSet := set.NewSet[ids.NodeID](len(m.validators))

	now := time.Now()
	for nodeID, v := range m.validators {
		if v.Status == StatusActive &&
			now.After(v.StartTime) &&
			now.Before(v.EndTime) {
			validatorSet.Add(nodeID)
		}
	}

	return validatorSet, nil
}

// CalculateRewards calculates validator rewards
func (m *Manager) CalculateRewards(ctx context.Context) (uint64, error) {
	if m.myValidator == nil {
		return 0, errors.New("not registered as validator")
	}

	// Base reward calculation
	baseReward := m.calculateBaseReward()

	// Add delegation rewards
	delegationReward := m.calculateDelegationReward()

	totalReward := baseReward + delegationReward
	m.myValidator.RewardsEarned = totalReward

	return totalReward, nil
}

// ClaimRewards claims validator rewards
func (m *Manager) ClaimRewards(ctx context.Context) error {
	rewards, err := m.CalculateRewards(ctx)
	if err != nil {
		return err
	}

	if rewards == 0 {
		return errors.New("no rewards to claim")
	}

	// TODO: Submit claim rewards transaction

	// Reset rewards
	m.myValidator.RewardsEarned = 0

	return nil
}

// validationLoop runs the validation process
func (m *Manager) validationLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Update metrics
			m.updateMetrics()

			// Check if validation period ended
			if time.Now().After(m.myValidator.EndTime) {
				m.myValidator.Status = StatusExpired
				return
			}
		}
	}
}

// updateMetrics updates validator metrics
func (m *Manager) updateMetrics() {
	// Update uptime
	// In production, this would track actual node uptime
	m.myValidator.Uptime = 99.9

	// Update blocks signed
	m.myValidator.BlocksSigned++
}

// calculateBaseReward calculates base validation reward
func (m *Manager) calculateBaseReward() uint64 {
	// Simplified reward calculation
	// In production, this would use actual protocol parameters
	annualRewardRate := 0.05 // 5% annual
	duration := m.myValidator.EndTime.Sub(m.myValidator.StartTime)
	years := duration.Hours() / (24 * 365)

	return uint64(float64(m.myValidator.StakeAmount) * annualRewardRate * years * m.myValidator.Uptime / 100)
}

// calculateDelegationReward calculates delegation fee reward
func (m *Manager) calculateDelegationReward() uint64 {
	var totalDelegationReward uint64

	for _, d := range m.myValidator.Delegators {
		// Calculate delegator's base reward
		duration := d.EndTime.Sub(d.StartTime)
		years := duration.Hours() / (24 * 365)
		delegatorReward := uint64(float64(d.Amount) * 0.05 * years)

		// Validator gets delegation fee percentage
		validatorShare := uint64(float64(delegatorReward) * m.myValidator.DelegationFee / 100)
		totalDelegationReward += validatorShare
	}

	return totalDelegationReward
}

// RegisterConfig represents validator registration configuration
type RegisterConfig struct {
	NodeID        ids.NodeID
	StakeAmount   uint64
	StakeAssetID  ids.ID
	StartTime     time.Time
	EndTime       time.Time
	DelegationFee float64
}

// Validate validates registration configuration
func (c *RegisterConfig) Validate() error {
	if c.StakeAmount == 0 {
		return errors.New("stake amount must be greater than 0")
	}

	if c.StakeAssetID == ids.Empty {
		return errors.New("stake asset ID required")
	}

	if c.EndTime.Before(c.StartTime) {
		return errors.New("end time must be after start time")
	}

	minStakeDuration := 14 * 24 * time.Hour // 14 days
	if c.EndTime.Sub(c.StartTime) < minStakeDuration {
		return fmt.Errorf("stake duration must be at least %s", minStakeDuration)
	}

	if c.DelegationFee < 0 || c.DelegationFee > 100 {
		return errors.New("delegation fee must be between 0 and 100")
	}

	return nil
}
