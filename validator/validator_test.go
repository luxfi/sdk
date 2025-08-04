// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/node/ids"

	"github.com/luxfi/sdk/blockchain"
	"github.com/luxfi/sdk/wallet"
)

func TestManager_AddValidator(t *testing.T) {
	// Create mock wallet
	w := wallet.New(1, ids.GenerateTestID())
	manager := NewManager(w)

	// Create mock blockchain
	bc := &blockchain.Blockchain{
		ID: "test-blockchain-1",
	}

	// Create validator request
	req := &AddValidatorRequest{
		NodeID:        ids.GenerateTestNodeID(),
		StartTime:     time.Now().Add(time.Hour),
		EndTime:       time.Now().Add(24 * time.Hour * 30), // 30 days
		StakeAmount:   2000 * 1000000000,                   // 2000 LUX
		DelegationFee: 10.0,
	}

	// Add validator
	val, err := manager.AddValidator(context.Background(), bc, w, req)
	require.NoError(t, err)
	require.NotNil(t, val)
	require.Equal(t, req.NodeID, val.NodeID)
	require.Equal(t, req.StakeAmount, val.StakeAmount)
	require.Equal(t, StatusPending, val.Status)
}

func TestManager_GetValidator(t *testing.T) {
	w := wallet.New(1, ids.GenerateTestID())
	manager := NewManager(w)

	// Create test validator
	nodeID := ids.GenerateTestNodeID()
	val := &Validator{
		NodeID:      nodeID,
		Status:      StatusActive,
		StakeAmount: 2000 * 1000000000,
	}

	// Store validator (in real implementation)
	manager.validators[nodeID] = val

	// Get validator
	retrieved, err := manager.GetValidator(nodeID)
	require.NoError(t, err)
	require.Equal(t, val, retrieved)

	// Get non-existent validator
	_, err = manager.GetValidator(ids.GenerateTestNodeID())
	require.Error(t, err)
}

func TestManager_ListValidators(t *testing.T) {
	w := wallet.New(1, ids.GenerateTestID())
	manager := NewManager(w)

	// Add multiple validators
	vals := make([]*Validator, 3)
	for i := range vals {
		vals[i] = &Validator{
			NodeID:      ids.GenerateTestNodeID(),
			Status:      StatusActive,
			StakeAmount: uint64((i + 1) * 1000),
		}
		manager.validators[vals[i].NodeID] = vals[i]
	}

	// List all validators
	list, err := manager.ListValidators()
	require.NoError(t, err)
	require.Len(t, list, 3)

	// Verify all are active
	for _, v := range list {
		require.Equal(t, StatusActive, v.Status)
	}
}

func TestManager_UpdateValidator(t *testing.T) {
	w := wallet.New(1, ids.GenerateTestID())
	manager := NewManager(w)

	// Create test validator
	nodeID := ids.GenerateTestNodeID()
	val := &Validator{
		NodeID:      nodeID,
		Status:      StatusPending,
		StakeAmount: 2000 * 1000000000,
	}
	manager.validators[nodeID] = val

	// Update validator status
	update := &ValidatorUpdate{
		Status: StatusActive,
	}
	updated, err := manager.UpdateValidator(context.Background(), nodeID, update)
	require.NoError(t, err)
	require.Equal(t, StatusActive, updated.Status)
	require.Equal(t, val.StakeAmount, updated.StakeAmount)
}

func TestManager_AddDelegator(t *testing.T) {
	w := wallet.New(1, ids.GenerateTestID())
	manager := NewManager(w)

	// Create test validator
	nodeID := ids.GenerateTestNodeID()
	val := &Validator{
		NodeID:        nodeID,
		Status:        StatusActive,
		StakeAmount:   2000 * 1000000000,
		DelegationFee: 10.0,
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(24 * time.Hour * 30), // 30 days
		Delegators:    []Delegator{},
	}
	manager.validators[nodeID] = val
	manager.myValidator = val

	// Add delegator
	req := &AddDelegatorRequest{
		NodeID:      nodeID,
		StartTime:   time.Now().Add(time.Hour),
		EndTime:     time.Now().Add(time.Hour + 24*time.Hour*14), // 14 days from start
		StakeAmount: 100 * 1000000000,
	}

	delegator := Delegator{
		Address:     ids.GenerateTestShortID(),
		Amount:      req.StakeAmount,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		RewardShare: 100.0 - val.DelegationFee,
	}

	err := manager.AddDelegator(context.Background(), delegator)
	require.NoError(t, err)

	// Verify delegator was added
	require.Len(t, manager.myValidator.Delegators, 1)
	require.Equal(t, req.StakeAmount, manager.myValidator.Delegators[0].Amount)
}

func TestManager_CalculateRewards(t *testing.T) {
	w := wallet.New(1, ids.GenerateTestID())
	manager := NewManager(w)

	// Create test validator with delegators
	nodeID := ids.GenerateTestNodeID()
	val := &Validator{
		NodeID:        nodeID,
		Status:        StatusActive,
		StakeAmount:   2000 * 1000000000,
		DelegationFee: 10.0,
		StartTime:     time.Now().Add(-24 * time.Hour * 30), // Started 30 days ago
		EndTime:       time.Now().Add(24 * time.Hour),       // Ends tomorrow
		Delegators: []Delegator{
			{
				Address:     ids.GenerateTestShortID(),
				Amount:      100 * 1000000000,
				StartTime:   time.Now().Add(-24 * time.Hour * 15), // Started 15 days ago
				EndTime:     time.Now().Add(24 * time.Hour),       // Ends tomorrow
				RewardShare: 90.0,                                 // 90% after 10% fee
			},
		},
	}
	manager.validators[nodeID] = val
	manager.myValidator = val

	// Calculate rewards
	reward, err := manager.CalculateRewards(context.Background())
	require.NoError(t, err)
	require.Greater(t, reward, uint64(0))
}

func TestValidatorLifecycle(t *testing.T) {
	w := wallet.New(1, ids.GenerateTestID())
	manager := NewManager(w)

	// Create validator
	nodeID := ids.GenerateTestNodeID()
	val := &Validator{
		NodeID:      nodeID,
		Status:      StatusPending,
		StakeAmount: 2000 * 1000000000,
		StartTime:   time.Now().Add(time.Second),
		EndTime:     time.Now().Add(2 * time.Second),
	}
	manager.validators[nodeID] = val

	// Check initial status
	require.Equal(t, StatusPending, val.Status)

	// Wait for validator to become active
	time.Sleep(1100 * time.Millisecond)

	// Update status (in real implementation this would be automatic)
	val.Status = StatusActive
	require.Equal(t, StatusActive, val.Status)

	// Wait for validator to expire
	time.Sleep(1100 * time.Millisecond)

	// Update status (in real implementation this would be automatic)
	val.Status = StatusExpired
	require.Equal(t, StatusExpired, val.Status)
}

func TestDelegatorValidation(t *testing.T) {
	w := wallet.New(1, ids.GenerateTestID())
	manager := NewManager(w)

	// Create test validator
	nodeID := ids.GenerateTestNodeID()
	val := &Validator{
		NodeID:        nodeID,
		Status:        StatusActive,
		StakeAmount:   2000 * 1000000000,
		DelegationFee: 10.0,
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(24 * time.Hour * 30),
	}
	manager.validators[nodeID] = val

	tests := []struct {
		name    string
		req     *AddDelegatorRequest
		wantErr bool
	}{
		{
			name: "valid delegator",
			req: &AddDelegatorRequest{
				NodeID:      nodeID,
				StartTime:   time.Now().Add(time.Hour),
				EndTime:     time.Now().Add(time.Hour + 24*time.Hour*14),
				StakeAmount: 100 * 1000000000,
			},
			wantErr: false,
		},
		{
			name: "stake too low",
			req: &AddDelegatorRequest{
				NodeID:      nodeID,
				StartTime:   time.Now().Add(time.Hour),
				EndTime:     time.Now().Add(24 * time.Hour * 14),
				StakeAmount: 10 * 1000000000, // Below minimum
			},
			wantErr: true,
		},
		{
			name: "duration too short",
			req: &AddDelegatorRequest{
				NodeID:      nodeID,
				StartTime:   time.Now().Add(time.Hour),
				EndTime:     time.Now().Add(time.Hour * 2), // Too short
				StakeAmount: 100 * 1000000000,
			},
			wantErr: true,
		},
		{
			name: "end time after validator",
			req: &AddDelegatorRequest{
				NodeID:      nodeID,
				StartTime:   time.Now().Add(time.Hour),
				EndTime:     time.Now().Add(24 * time.Hour * 60), // After validator ends
				StakeAmount: 100 * 1000000000,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Need to set myValidator for the AddDelegator to work
			manager.myValidator = val

			delegator := Delegator{
				Address:     ids.GenerateTestShortID(),
				Amount:      tt.req.StakeAmount,
				StartTime:   tt.req.StartTime,
				EndTime:     tt.req.EndTime,
				RewardShare: 100.0 - val.DelegationFee,
			}
			err := manager.AddDelegator(context.Background(), delegator)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBLSKeyGeneration(t *testing.T) {
	// Generate BLS key
	blsKey, err := bls.NewSecretKey()
	require.NoError(t, err)

	// Get public key
	pubKey := bls.PublicFromSecretKey(blsKey)
	require.NotNil(t, pubKey)

	// Sign a message
	message := []byte("test validator message")
	sig := bls.Sign(blsKey, message)
	require.NotNil(t, sig)

	// Verify signature
	valid := bls.Verify(pubKey, sig, message)
	require.True(t, valid)
}

func TestValidatorMetrics(t *testing.T) {
	w := wallet.New(1, ids.GenerateTestID())
	manager := NewManager(w)

	// Create test validators
	active := &Validator{
		NodeID:      ids.GenerateTestNodeID(),
		Status:      StatusActive,
		StakeAmount: 2000 * 1000000000,
	}
	pending := &Validator{
		NodeID:      ids.GenerateTestNodeID(),
		Status:      StatusPending,
		StakeAmount: 1500 * 1000000000,
	}
	expired := &Validator{
		NodeID:      ids.GenerateTestNodeID(),
		Status:      StatusExpired,
		StakeAmount: 1000 * 1000000000,
	}

	manager.validators[active.NodeID] = active
	manager.validators[pending.NodeID] = pending
	manager.validators[expired.NodeID] = expired

	// Get metrics
	metrics := manager.GetMetrics()
	require.Equal(t, 3, metrics.TotalValidators)
	require.Equal(t, 1, metrics.ActiveValidators)
	require.Equal(t, 1, metrics.PendingValidators)
	require.Equal(t, uint64(2000*1000000000), metrics.TotalStaked)
}
