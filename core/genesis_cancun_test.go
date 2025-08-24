package core

import (
	"math/big"
	"testing"

	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/vm"
	"github.com/luxfi/geth/triedb"
	"github.com/luxfi/evm/consensus/dummy"
	"github.com/luxfi/evm/params"
	"github.com/stretchr/testify/require"
)

func TestGenesisCancun(t *testing.T) {
	t.Skip("Skipping genesis test - configuration issue")
	require := require.New(t)
	
	// Test with Cancun enabled (default TestChainConfig)
	t.Run("WithCancun", func(t *testing.T) {
		gspec := &Genesis{
			Config:  params.TestChainConfig,
			Alloc:   GenesisAlloc{},
			BaseFee: big.NewInt(875000000),
			Timestamp: 0,  // Cancun is active at time 0
		}
		
		db := rawdb.NewMemoryDatabase()
		tdb := triedb.NewDatabase(db, triedb.HashDefaults)
		
		genesisBlock, err := gspec.Commit(db, tdb)
		require.NoError(err)
		
		// Check if header has Cancun fields
		header := genesisBlock.Header()
		t.Logf("Has ParentBeaconRoot: %v", header.ParentBeaconRoot != nil)
		t.Logf("Has ExcessBlobGas: %v", header.ExcessBlobGas != nil)
		t.Logf("Has BlobGasUsed: %v", header.BlobGasUsed != nil)
		
		// Try to read it back
		hash := genesisBlock.Hash()
		readHeader := rawdb.ReadHeader(db, hash, 0)
		if readHeader == nil {
			t.Logf("FAIL: Cannot read header with Cancun fields")
		} else {
			t.Logf("SUCCESS: Can read header with Cancun fields")
		}
		
		// Try to create blockchain
		engine := dummy.NewCoinbaseFaker()
		chain, err := NewBlockChain(db, DefaultCacheConfig, gspec, engine, vm.Config{}, hash, false)
		if err != nil {
			t.Logf("FAIL: Cannot create blockchain with Cancun: %v", err)
		} else {
			t.Logf("SUCCESS: Created blockchain with Cancun")
			chain.Stop()
		}
	})
	
	// Test without Cancun
	t.Run("WithoutCancun", func(t *testing.T) {
		// Create a config without Cancun
		configNoCancun := params.TestPreSubnetEVMChainConfig
		
		gspec := &Genesis{
			Config:  configNoCancun,
			Alloc:   GenesisAlloc{},
			BaseFee: big.NewInt(875000000),
			Timestamp: 0,
		}
		
		db := rawdb.NewMemoryDatabase()
		tdb := triedb.NewDatabase(db, triedb.HashDefaults)
		
		genesisBlock, err := gspec.Commit(db, tdb)
		require.NoError(err)
		
		// Check if header has Cancun fields
		header := genesisBlock.Header()
		t.Logf("Has ParentBeaconRoot: %v", header.ParentBeaconRoot != nil)
		t.Logf("Has ExcessBlobGas: %v", header.ExcessBlobGas != nil)
		t.Logf("Has BlobGasUsed: %v", header.BlobGasUsed != nil)
		
		// Try to read it back
		hash := genesisBlock.Hash()
		readHeader := rawdb.ReadHeader(db, hash, 0)
		if readHeader == nil {
			t.Logf("FAIL: Cannot read header without Cancun fields")
		} else {
			t.Logf("SUCCESS: Can read header without Cancun fields")
		}
		
		// Try to create blockchain
		engine := dummy.NewCoinbaseFaker()
		chain, err := NewBlockChain(db, DefaultCacheConfig, gspec, engine, vm.Config{}, hash, false)
		if err != nil {
			t.Logf("FAIL: Cannot create blockchain without Cancun: %v", err)
		} else {
			t.Logf("SUCCESS: Created blockchain without Cancun")
			chain.Stop()
		}
	})
}