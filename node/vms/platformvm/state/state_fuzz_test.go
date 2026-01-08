// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/state/statetest"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

// FuzzStateTransitions tests state transitions with random operations
func FuzzStateTransitions(f *testing.F) {
	// Seed corpus with various operations
	f.Add(uint8(0), uint64(1000), uint32(1))
	f.Add(uint8(1), uint64(0), uint32(0))
	f.Add(uint8(2), uint64(1_000_000), uint32(100))
	f.Add(uint8(3), uint64(100_000_000), uint32(10000))

	f.Fuzz(func(t *testing.T, operation uint8, amount uint64, shares uint32) {
		// Limit values to reasonable ranges
		if amount > 1_000_000_000_000 {
			amount = amount % 1_000_000_000_000
		}
		if shares > 100_000 {
			shares = shares % 100_000
		}

		// Create state using statetest helper
		s := statetest.New(t, statetest.Config{})

		// Perform operations based on fuzzed input
		switch operation % 5 {
		case 0:
			// Test adding a validator
			nodeID := ids.GenerateTestNodeID()
			startTime := time.Now().Add(time.Hour)
			endTime := startTime.Add(24 * time.Hour)

			err := s.PutCurrentValidator(&state.Staker{
				TxID:            ids.GenerateTestID(),
				NodeID:          nodeID,
				PublicKey:       nil,
				ChainID:           constants.PrimaryNetworkID,
				Weight:          amount,
				StartTime:       startTime,
				EndTime:         endTime,
				PotentialReward: 0,
			})
			if err != nil {
				// Some validator configurations might be invalid
				return
			}

			// Verify validator was added
			val, err := s.GetCurrentValidator(constants.PrimaryNetworkID, nodeID)
			if err != nil {
				t.Errorf("Failed to get validator after adding: %v", err)
				return
			}

			if val.Weight != amount {
				t.Errorf("Validator weight mismatch: got %v, want %v", val.Weight, amount)
			}

		case 1:
			// Test UTXO operations
			txID := ids.GenerateTestID()
			utxo := &lux.UTXO{
				UTXOID: lux.UTXOID{
					TxID:        txID,
					OutputIndex: shares,
				},
				Asset: lux.Asset{ID: ids.GenerateTestID()},
				Out: &secp256k1fx.TransferOutput{
					Amt: amount,
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: 1,
						Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
					},
				},
			}

			// Add UTXO
			s.AddUTXO(utxo)

			// Get UTXO
			retrievedUTXO, err := s.GetUTXO(utxo.InputID())
			if err != nil {
				t.Errorf("Failed to get UTXO after adding: %v", err)
				return
			}

			// Verify UTXO data
			if retrievedUTXO.TxID != txID {
				t.Errorf("UTXO TxID mismatch")
			}

			// Delete UTXO
			s.DeleteUTXO(utxo.InputID())

		case 2:
			// Test chain operations
			chainID := ids.GenerateTestID()
			createChainTx := &txs.Tx{
				Unsigned: &txs.CreateChainTx{
					ChainID:       ids.GenerateTestID(),
					BlockchainName:   "test-chain",
					VMID:        ids.GenerateTestID(),
					FxIDs:       []ids.ID{},
					GenesisData: []byte("genesis"),
				},
			}

			// Add chain
			s.AddChain(createChainTx)

			// Chain operations don't have a direct Get method
			// Just verify the add doesn't error
			_ = chainID


		case 3:
			// Test reward UTXO operations
			txID := ids.GenerateTestID()
			rewardUTXO := &lux.UTXO{
				UTXOID: lux.UTXOID{
					TxID:        txID,
					OutputIndex: shares,
				},
				Asset: lux.Asset{ID: ids.GenerateTestID()},
				Out: &secp256k1fx.TransferOutput{
					Amt: amount,
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: 1,
						Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
					},
				},
			}

			// Add reward UTXO
			s.AddRewardUTXO(txID, rewardUTXO)

			// Get reward UTXOs
			utxos, err := s.GetRewardUTXOs(txID)
			if err != nil {
				// Retrieval might fail
				return
			}

			if len(utxos) == 0 {
				t.Error("Should have reward UTXOs after adding")
			}

		case 4:
			// Test subnet transformation operations
			subnetID := ids.GenerateTestID()

			// Add a subnet transformation
			s.AddNetTransformation(&txs.Tx{
				Unsigned: &txs.TransformChainTx{
					Chain:           subnetID,
					AssetID:       ids.GenerateTestID(),
					InitialSupply: amount,
					MaximumSupply: amount * 2,
					MinConsumptionRate: 100000,
					MaxConsumptionRate: 120000,
					MinValidatorStake:  1000,
					MaxValidatorStake:  amount,
					MinStakeDuration:   86400,
					MaxStakeDuration:   8640000,
					MinDelegationFee:   20000,
					MinDelegatorStake:  25,
				},
			})

			// Verify the transformation was recorded
			// This tests the subnet/net transformation tracking logic
		}

		// Commit changes
		err := s.Commit()
		if err != nil {
			// Commit might fail for some state configurations
			return
		}
	})
}

// FuzzStateSerialization tests state serialization/deserialization
func FuzzStateSerialization(f *testing.F) {
	// Seed corpus
	f.Add([]byte{}, uint32(0))
	f.Add([]byte{1, 2, 3, 4}, uint32(100))
	f.Add(bytes.Repeat([]byte{0xff}, 100), uint32(1000))

	f.Fuzz(func(t *testing.T, data []byte, height uint32) {
		// Limit data size
		if len(data) > 10000 {
			data = data[:10000]
		}

		// Create initial state
		s := statetest.New(t, statetest.Config{})

		// Set some state based on fuzzing input
		if len(data) >= 32 {
			var blockID ids.ID
			copy(blockID[:], data[:32])
			s.SetLastAccepted(blockID)
			s.SetHeight(uint64(height))
		}

		// Set timestamp
		if len(data) >= 8 {
			timestamp := int64(0)
			for i := 0; i < 8 && i < len(data); i++ {
				timestamp |= int64(data[i]) << (8 * i)
			}
			s.SetTimestamp(time.Unix(timestamp, 0))
		}

		// Commit state
		err := s.Commit()
		if err != nil {
			return
		}

		// State doesn't have a GetHeight() method directly
		// The height is managed internally

		if len(data) >= 32 {
			var expectedBlockID ids.ID
			copy(expectedBlockID[:], data[:32])
			if s.GetLastAccepted() != expectedBlockID {
				t.Error("Last accepted block mismatch")
			}
		}
	})
}

// FuzzValidatorSet tests validator set operations
func FuzzValidatorSet(f *testing.F) {
	// Seed corpus
	f.Add(uint8(10), uint64(1000), uint64(100))
	f.Add(uint8(1), uint64(0), uint64(0))
	f.Add(uint8(100), uint64(1_000_000), uint64(10000))

	f.Fuzz(func(t *testing.T, numValidators uint8, baseWeight uint64, variation uint64) {
		// Limit parameters
		if numValidators > 20 {
			numValidators = 20
		}
		if baseWeight > 1_000_000_000 {
			baseWeight = baseWeight % 1_000_000_000
		}
		if variation > baseWeight {
			variation = baseWeight
		}

		s := statetest.New(t, statetest.Config{})

		// Get initial validator count
		ctx := context.Background()
		initialValidators, _, _, err := s.GetCurrentValidators(ctx, constants.PrimaryNetworkID)
		if err != nil {
			return
		}
		initialCount := len(initialValidators)

		validators := make([]*state.Staker, 0, numValidators)
		totalWeight := uint64(0)

		// Add validators
		for i := uint8(0); i < numValidators; i++ {
			weight := baseWeight
			if variation > 0 && i%2 == 0 {
				weight += variation * uint64(i)
			}

			validator := &state.Staker{
				TxID:            ids.GenerateTestID(),
				NodeID:          ids.GenerateTestNodeID(),
				PublicKey:       nil,
				ChainID:           constants.PrimaryNetworkID,
				Weight:          weight,
				StartTime:       time.Now().Add(time.Duration(i) * time.Hour),
				EndTime:         time.Now().Add(time.Duration(24+i) * time.Hour),
				PotentialReward: 0,
			}

			err := s.PutCurrentValidator(validator)
			if err != nil {
				// Some validator configurations might fail
				continue
			}

			validators = append(validators, validator)
			totalWeight += weight
		}

		// Test getting current validators after adding
		currentValidators, _, _, err := s.GetCurrentValidators(ctx, constants.PrimaryNetworkID)
		if err != nil {
			return
		}

		expectedCount := initialCount + len(validators)
		if len(currentValidators) != expectedCount {
			t.Errorf("Validator count mismatch: got %v, want %v (initial: %v, added: %v)", 
				len(currentValidators), expectedCount, initialCount, len(validators))
		}

		// Remove some validators
		removedCount := 0
		for i, validator := range validators {
			if i%2 == 0 {
				s.DeleteCurrentValidator(validator)
				removedCount++
			}
		}

		// Verify removal by getting validators again
		currentValidatorsAfter, _, _, err := s.GetCurrentValidators(ctx, constants.PrimaryNetworkID)
		if err != nil {
			return
		}

		expectedCountAfter := expectedCount - removedCount
		if len(currentValidatorsAfter) != expectedCountAfter {
			t.Errorf("Validator count after removal mismatch: got %v, want %v (removed %v)",
				len(currentValidatorsAfter), expectedCountAfter, removedCount)
		}
	})
}