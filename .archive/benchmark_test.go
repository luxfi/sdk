// Copyright (C) 2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package sdk

import (
	"context"
	"math/big"
	"testing"

	"github.com/luxfi/sdk/blockchain"
	"github.com/luxfi/sdk/network"
	"github.com/luxfi/ids"
)

// BenchmarkCreateNetwork benchmarks network creation
func BenchmarkCreateNetwork(b *testing.B) {
	sdk, err := New(
		WithLogLevel("error"),
		WithDataDir(b.TempDir()),
	)
	if err != nil {
		b.Fatal(err)
	}
	defer sdk.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		params := &network.NetworkParams{
			Name:     "bench-network",
			Type:     network.NetworkTypeLocal,
			NumNodes: 5,
		}
		
		network, err := sdk.CreateNetwork(ctx, params)
		if err != nil {
			b.Fatal(err)
		}
		
		// Clean up
		_ = sdk.DeleteNetwork(ctx, network.ID)
	}
}

// BenchmarkCreateBlockchain benchmarks blockchain creation
func BenchmarkCreateBlockchain(b *testing.B) {
	sdk, err := New(
		WithLogLevel("error"),
		WithDataDir(b.TempDir()),
	)
	if err != nil {
		b.Fatal(err)
	}
	defer sdk.Close()

	ctx := context.Background()

	b.Run("L1-EVM", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := sdk.CreateL1(ctx, "bench-l1", &blockchain.L1Params{
				VMType: blockchain.VMTypeEVM,
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("L2-EVM", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := sdk.CreateL2(ctx, "bench-l2", &blockchain.L2Params{
				VMType:          blockchain.VMTypeEVM,
				SequencerType:   "centralized",
				DALayer:         "celestia",
				SettlementChain: "chain-123",
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("L3-WASM", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := sdk.CreateL3(ctx, "bench-l3", &blockchain.L3Params{
				VMType:  blockchain.VMTypeWASM,
				L2Chain: "l2-chain",
				AppType: "gaming",
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkChainOperations benchmarks chain operations
func BenchmarkChainOperations(b *testing.B) {
	// This benchmark requires mocked chain clients
	// since we can't connect to real chains in benchmarks
	
	b.Run("GetBalance", func(b *testing.B) {
		// Mock implementation
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate balance query
			_ = big.NewInt(1000000)
		}
	})

	b.Run("CreateAsset", func(b *testing.B) {
		// Mock implementation
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate asset creation
			_ = ids.GenerateTestID()
		}
	})

	b.Run("CrossChainTransfer", func(b *testing.B) {
		// Mock implementation
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate cross-chain transfer
			_ = ids.GenerateTestID()
		}
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent operations
func BenchmarkConcurrentOperations(b *testing.B) {
	sdk, err := New(
		WithLogLevel("error"),
		WithDataDir(b.TempDir()),
	)
	if err != nil {
		b.Fatal(err)
	}
	defer sdk.Close()

	ctx := context.Background()

	b.Run("ConcurrentBlockchainCreation", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			counter := 0
			for pb.Next() {
				params := &blockchain.CreateParams{
					Name:    "concurrent-bench",
					Type:    blockchain.TypeL1,
					VMType:  blockchain.VMTypeEVM,
					ChainID: big.NewInt(int64(10000 + counter)),
				}
				counter++
				
				_, err := sdk.CreateBlockchain(ctx, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

// BenchmarkMemoryAllocation benchmarks memory allocation
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("NetworkParams", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = &network.NetworkParams{
				Name:     "test",
				Type:     network.NetworkTypeLocal,
				NumNodes: 10,
			}
		}
	})

	b.Run("BlockchainParams", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = &blockchain.CreateParams{
				Name:    "test",
				Type:    blockchain.TypeL1,
				VMType:  blockchain.VMTypeEVM,
				ChainID: big.NewInt(12345),
			}
		}
	})
}

// BenchmarkValidation benchmarks validation operations
func BenchmarkValidation(b *testing.B) {
	sdk, err := New(
		WithLogLevel("error"),
		WithDataDir(b.TempDir()),
	)
	if err != nil {
		b.Fatal(err)
	}
	defer sdk.Close()

	validConfig := []byte(`{
		"chainId": 12345,
		"consensus": {
			"type": "lux"
		},
		"vm": {
			"type": "evm"
		}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := sdk.ValidateChainConfig(validConfig)
		if err != nil {
			b.Fatal(err)
		}
	}
}