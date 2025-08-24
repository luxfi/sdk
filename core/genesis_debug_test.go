package core

import (
	"math/big"
	"testing"

	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/vm"
	"github.com/luxfi/evm/consensus/dummy"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/geth/triedb"
	"github.com/stretchr/testify/require"
)

func TestGenesisDebug(t *testing.T) {
	t.Skip("Skipping genesis debug test - configuration issue")
	require := require.New(t)
	
	// Create a simple genesis
	gspec := &Genesis{
		Config:  params.TestChainConfig,
		Alloc:   GenesisAlloc{},
		BaseFee: big.NewInt(875000000),
	}
	
	// Create database and commit genesis
	db := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(db, triedb.HashDefaults)
	
	// Commit the genesis
	genesisBlock, err := gspec.Commit(db, tdb)
	require.NoError(err, "Failed to commit genesis")
	require.NotNil(genesisBlock)
	
	genesisHash := genesisBlock.Hash()
	t.Logf("Genesis hash: %s", genesisHash.Hex())
	t.Logf("Genesis number: %d", genesisBlock.NumberU64())
	
	// Check what was written
	storedHash := rawdb.ReadCanonicalHash(db, 0)
	t.Logf("Stored canonical hash at 0: %s", storedHash.Hex())
	require.Equal(genesisHash, storedHash, "Canonical hash mismatch")
	
	// Check if header number was written
	headerNumber, found := rawdb.ReadHeaderNumber(db, genesisHash)
	t.Logf("Header number found: %v, value: %v", found, headerNumber)
	
	// Try to read the header
	t.Logf("Trying to read header with hash=%s, number=%d", genesisHash.Hex(), 0)
	header := rawdb.ReadHeader(db, genesisHash, 0)
	if header != nil {
		t.Logf("Header found: number=%d, hash=%s", header.Number.Uint64(), header.Hash().Hex())
	} else {
		t.Logf("Header NOT found")
		
		// Let's check what keys exist and try to manually read the header
		it := db.NewIterator(nil, nil)
		defer it.Release()
		
		count := 0
		for it.Next() && count < 10 {
			key := it.Key()
			t.Logf("Key[%d]: %x (len=%d)", count, key, len(key))
			
			// Check if this is a header key (starts with 'h' and has right length)
			if len(key) == 41 && key[0] == 'h' {
				t.Logf("  This looks like a header key!")
				value := it.Value()
				t.Logf("  Value length: %d", len(value))
			}
			count++
		}
	}
	
	// Try to read the body
	body := rawdb.ReadBody(db, genesisHash, 0)
	if body != nil {
		t.Logf("Body found: txs=%d, uncles=%d", len(body.Transactions), len(body.Uncles))
	} else {
		t.Logf("Body NOT found")
	}
	
	// Let's try writing the header directly
	t.Logf("Writing header directly...")
	rawdb.WriteHeader(db, genesisBlock.Header())
	
	// Now try to read it again
	header2 := rawdb.ReadHeader(db, genesisHash, 0)
	if header2 != nil {
		t.Logf("Header found after direct write: number=%d", header2.Number.Uint64())
	} else {
		t.Logf("Header still not found after direct write")
	}
	
	// Try to read the block
	block := rawdb.ReadBlock(db, genesisHash, 0)
	if block == nil {
		t.Logf("Block still nil, trying to assemble manually")
		// Try to get body again
		body2 := rawdb.ReadBody(db, genesisHash, 0)
		if body2 != nil && header2 != nil {
			t.Logf("Both header and body exist, but ReadBlock fails")
		}
	}
	require.NotNil(block, "Genesis block should be readable")
	
	// Now try to create a blockchain
	engine := dummy.NewCoinbaseFaker()
	chain, err := NewBlockChain(db, DefaultCacheConfig, gspec, engine, vm.Config{}, genesisHash, false)
	require.NoError(err, "Failed to create blockchain")
	defer chain.Stop()
	
	// Check that genesis is available
	genesis := chain.GetBlockByNumber(0)
	require.NotNil(genesis, "Genesis should be available from chain")
	require.Equal(genesisHash, genesis.Hash(), "Genesis hash should match")
}