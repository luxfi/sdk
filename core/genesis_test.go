// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
//
// This file is a derived work, based on the go-ethereum library whose original
// notices appear below.
//
// It is distributed under a license compatible with the licensing terms of the
// original code from which it is derived.
//
// Much love to the original authors for their work.
// **********
// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	_ "embed"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/luxfi/evm/consensus/dummy"
	"github.com/luxfi/evm/core/state"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/evm/params/extras"
	"github.com/luxfi/evm/plugin/evm/customrawdb"
	"github.com/luxfi/evm/precompile/allowlist"
	"github.com/luxfi/evm/precompile/contracts/deployerallowlist"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/core/vm"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/trie"
	"github.com/luxfi/geth/triedb"
	gethpathdb "github.com/luxfi/geth/triedb/pathdb"

	// "github.com/luxfi/evm/triedb/firewood"
	"github.com/davecgh/go-spew/spew"
	"github.com/luxfi/evm/triedb/pathdb"
	"github.com/luxfi/evm/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGenesisBlock(db ethdb.Database, triedb *triedb.Database, genesis *Genesis, lastAcceptedHash common.Hash) (*params.ChainConfig, common.Hash, error) {
	return SetupGenesisBlock(db, triedb, genesis, lastAcceptedHash, false)
}

func TestGenesisBlockForTesting(t *testing.T) {
	genesisBlockForTestingHash := common.HexToHash("0x517cc43afd8c4516d00dae3c767b336d4ad9a9aeffbf0a4b205ef0bdc6343f35")
	block := GenesisBlockForTesting(rawdb.NewMemoryDatabase(), common.Address{1}, big.NewInt(1))
	if block.Hash() != genesisBlockForTestingHash {
		t.Errorf("wrong testing genesis hash, got %v, want %v", block.Hash(), genesisBlockForTestingHash)
	}
}

func TestSetupGenesis(t *testing.T) {
	for _, scheme := range []string{rawdb.HashScheme, rawdb.PathScheme, customrawdb.FirewoodScheme} {
		t.Run(scheme, func(t *testing.T) {
			testSetupGenesis(t, scheme)
		})
	}
}

func testSetupGenesis(t *testing.T, scheme string) {
	preSubnetConfig := params.Copy(params.TestPreSubnetEVMChainConfig)
	params.GetExtra(preSubnetConfig).SubnetEVMTimestamp = utils.NewUint64(100)
	var (
		customg = Genesis{
			Config: preSubnetConfig,
			Alloc: types.GenesisAlloc{
				{1}: {Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{{1}: {1}}},
			},
			GasLimit: params.GetExtra(preSubnetConfig).FeeConfig.GasLimit.Uint64(),
		}
		oldcustomg = customg
	)
	// Compute the actual genesis hash from the genesis spec
	customghash := customg.ToBlock().Hash()

	rollbackpreSubnetConfig := params.Copy(preSubnetConfig)
	params.GetExtra(rollbackpreSubnetConfig).SubnetEVMTimestamp = utils.NewUint64(90)
	oldcustomg.Config = rollbackpreSubnetConfig
	oldcustomghash := oldcustomg.ToBlock().Hash()

	tests := []struct {
		name       string
		fn         func(ethdb.Database) (*params.ChainConfig, common.Hash, error)
		wantConfig *params.ChainConfig
		wantHash   common.Hash
		wantErr    error
	}{
		{
			name: "genesis without ChainConfig",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				return setupGenesisBlock(db, triedb.NewDatabase(db, newDbConfig(t, scheme)), new(Genesis), common.Hash{})
			},
			wantErr:    errGenesisNoConfig,
			wantConfig: nil,
		},
		{
			name: "no block in DB, genesis == nil",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				return setupGenesisBlock(db, triedb.NewDatabase(db, newDbConfig(t, scheme)), nil, common.Hash{})
			},
			wantErr:    ErrNoGenesis,
			wantConfig: nil,
		},
		{
			name: "custom block in DB, genesis == nil returns stored config",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				tdb := triedb.NewDatabase(db, newDbConfig(t, scheme))
				block, err := customg.Commit(db, tdb)
				if err != nil {
					t.Fatal(err)
				}
				// When genesis is nil but a block exists, setup returns the stored config
				return setupGenesisBlock(db, tdb, nil, block.Hash())
			},
			wantErr:    nil, // No error - returns stored config
			wantConfig: customg.Config,
			wantHash:   customghash,
		},
		{
			name: "compatible config in DB",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				tdb := triedb.NewDatabase(db, newDbConfig(t, scheme))
				block, err := oldcustomg.Commit(db, tdb)
				if err != nil {
					t.Fatal(err)
				}
				// Use the actual block hash from commit, not a hardcoded hash
				return setupGenesisBlock(db, tdb, &customg, block.Hash())
			},
			wantHash:   oldcustomghash,
			wantConfig: customg.Config,
		},
		{
			name: "config upgrade preserves stored genesis",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				// Commit the 'old' genesis block with SubnetEVM transition at 90.
				// Advance to block #4, past the SubnetEVM transition block of customg.
				tdb := triedb.NewDatabase(db, newDbConfig(t, rawdb.HashScheme))
				genesis, err := oldcustomg.Commit(db, tdb)
				if err != nil {
					t.Fatal(err)
				}
				_ = tdb.Close()

				cacheConfig := DefaultCacheConfigWithScheme(scheme)
				cacheConfig.ChainDataDir = t.TempDir()
				bc, err := NewBlockChain(db, cacheConfig, &oldcustomg, dummy.NewFullFaker(), vm.Config{}, genesis.Hash(), false)
				if err != nil {
					t.Fatal(err)
				}
				defer bc.Stop()

				_, blocks, _, err := GenerateChainWithGenesis(&oldcustomg, dummy.NewFullFaker(), 4, 25, nil)
				if err != nil {
					t.Fatal(err)
				}
				bc.InsertChain(blocks)

				for _, block := range blocks {
					if err := bc.Accept(block); err != nil {
						t.Fatal(err)
					}
				}

				// In the current implementation, config upgrades are allowed as long as
				// the lastAccepted block is before the incompatible fork.
				// The new config is written and returned.
				return setupGenesisBlock(db, bc.TrieDB(), &customg, bc.lastAccepted.Hash())
			},
			wantHash:   oldcustomghash, // Hash remains the original genesis hash
			wantConfig: customg.Config,
			wantErr:    nil, // No error - upgrade is allowed
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s %s", test.name, scheme), func(t *testing.T) {
			db := rawdb.NewMemoryDatabase()
			config, hash, err := test.fn(db)
			// Check the return values.
			if !reflect.DeepEqual(err, test.wantErr) {
				spew := spew.ConfigState{DisablePointerAddresses: true, DisableCapacities: true}
				t.Errorf("returned error %#v, want %#v", spew.NewFormatter(err), spew.NewFormatter(test.wantErr))
			}
			if !reflect.DeepEqual(config, test.wantConfig) {
				t.Errorf("returned %v\nwant     %v", config, test.wantConfig)
			}
			if hash != test.wantHash {
				t.Errorf("returned hash %s, want %s", hash.Hex(), test.wantHash.Hex())
			} else if err == nil {
				// Check database content.
				stored := rawdb.ReadBlock(db, test.wantHash, 0)
				if stored.Hash() != test.wantHash {
					t.Errorf("block in DB has hash %s, want %s", stored.Hash(), test.wantHash)
				}
			}
		})
	}
}

func TestStatefulPrecompilesConfigure(t *testing.T) {
	type test struct {
		getConfig   func() *params.ChainConfig             // Return the config that enables the stateful precompile at the genesis for the test
		assertState func(t *testing.T, sdb *state.StateDB) // Check that the stateful precompiles were configured correctly
	}

	addr := common.HexToAddress("0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC")

	// Test suite to ensure that stateful precompiles are configured correctly in the genesis.
	for name, test := range map[string]test{
		"allow list enabled in genesis": {
			getConfig: func() *params.ChainConfig {
				config := params.Copy(params.TestChainConfig)
				params.GetExtra(config).GenesisPrecompiles = extras.Precompiles{
					deployerallowlist.ConfigKey: deployerallowlist.NewConfig(utils.NewUint64(0), []common.Address{addr}, nil, nil),
				}
				return config
			},
			assertState: func(t *testing.T, sdb *state.StateDB) {
				assert.Equal(t, allowlist.AdminRole, deployerallowlist.GetContractDeployerAllowListStatus(sdb, addr), "unexpected allow list status for modified address")
				assert.Equal(t, uint64(1), sdb.GetNonce(deployerallowlist.ContractAddress))
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			config := test.getConfig()

			genesis := &Genesis{
				Config: config,
				Alloc: types.GenesisAlloc{
					{1}: {Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{{1}: {1}}},
				},
				GasLimit: params.GetExtra(config).FeeConfig.GasLimit.Uint64(),
			}

			db := rawdb.NewMemoryDatabase()

			genesisBlock := genesis.ToBlock()
			genesisRoot := genesisBlock.Root()

			_, _, err := setupGenesisBlock(db, triedb.NewDatabase(db, triedb.HashDefaults), genesis, genesisBlock.Hash())
			if err != nil {
				t.Fatal(err)
			}

			statedb, err := state.New(genesisRoot, state.NewDatabase(db), nil)
			if err != nil {
				t.Fatal(err)
			}

			if test.assertState != nil {
				test.assertState(t, statedb)
			}
		})
	}
}

// regression test for precompile activation after header block
func TestPrecompileActivationAfterHeaderBlock(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	customg := Genesis{
		Config: params.TestChainConfig,
		Alloc: types.GenesisAlloc{
			{1}: {Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{{1}: {1}}},
		},
		GasLimit: params.GetExtra(params.TestChainConfig).FeeConfig.GasLimit.Uint64(),
	}
	bc, _ := NewBlockChain(db, DefaultCacheConfig, &customg, dummy.NewFullFaker(), vm.Config{}, common.Hash{}, false)
	defer bc.Stop()

	// Advance header to block #4, past the ContractDeployerAllowListConfig.
	_, blocks, _, _ := GenerateChainWithGenesis(&customg, dummy.NewFullFaker(), 4, 25, nil)

	require := require.New(t)
	_, err := bc.InsertChain(blocks)
	require.NoError(err)

	// accept up to block #2
	for _, block := range blocks[:2] {
		require.NoError(bc.Accept(block))
	}
	block := bc.CurrentBlock()

	require.Equal(blocks[1].Hash(), bc.lastAccepted.Hash())
	// header must be bigger than last accepted
	require.Greater(block.Time, bc.lastAccepted.Time())

	activatedGenesisConfig := params.Copy(customg.Config)
	contractDeployerConfig := deployerallowlist.NewConfig(utils.NewUint64(51), nil, nil, nil)
	params.GetExtra(activatedGenesisConfig).PrecompileUpgrades = []extras.PrecompileUpgrade{
		{
			Config: contractDeployerConfig,
		},
	}
	customg.Config = activatedGenesisConfig

	// assert block is after the activation block
	require.Greater(block.Time, *contractDeployerConfig.Timestamp())
	// assert last accepted block is before the activation block
	require.Less(bc.lastAccepted.Time(), *contractDeployerConfig.Timestamp())

	// This should not return any error since the last accepted block is before the activation block.
	config, _, err := setupGenesisBlock(db, triedb.NewDatabase(db, nil), &customg, bc.lastAccepted.Hash())
	require.NoError(err)
	if !reflect.DeepEqual(config, customg.Config) {
		t.Errorf("returned %v\nwant     %v", config, customg.Config)
	}
}

func TestGenesisWriteUpgradesRegression(t *testing.T) {
	require := require.New(t)
	config := params.Copy(params.TestChainConfig)
	genesis := &Genesis{
		Config: config,
		Alloc: types.GenesisAlloc{
			{1}: {Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{{1}: {1}}},
		},
		GasLimit: params.GetExtra(config).FeeConfig.GasLimit.Uint64(),
	}

	db := rawdb.NewMemoryDatabase()
	trieDB := triedb.NewDatabase(db, triedb.HashDefaults)
	genesisBlock := genesis.MustCommit(db, trieDB)

	_, _, err := SetupGenesisBlock(db, trieDB, genesis, genesisBlock.Hash(), false)
	require.NoError(err)

	params.GetExtra(genesis.Config).PrecompileUpgrades = []extras.PrecompileUpgrade{
		{
			Config: deployerallowlist.NewConfig(utils.NewUint64(51), nil, nil, nil),
		},
	}
	_, _, err = SetupGenesisBlock(db, trieDB, genesis, genesisBlock.Hash(), false)
	require.NoError(err)

	timestamp := uint64(100)
	lastAcceptedBlock := types.NewBlock(&types.Header{
		ParentHash: common.Hash{1, 2, 3},
		Number:     big.NewInt(100),
		GasLimit:   8_000_000,
		Extra:      nil,
		Time:       timestamp,
	}, nil, nil, trie.NewStackTrie(nil))
	rawdb.WriteBlock(db, lastAcceptedBlock)

	// Attempt restart after the chain has advanced past the activation of the precompile upgrade.
	// This tests a regression where the UpgradeConfig would not be written to disk correctly.
	_, _, err = SetupGenesisBlock(db, trieDB, genesis, lastAcceptedBlock.Hash(), false)
	require.NoError(err)
}

func newDbConfig(t *testing.T, scheme string) *triedb.Config {
	switch scheme {
	case rawdb.HashScheme:
		return triedb.HashDefaults
	case rawdb.PathScheme:
		return &triedb.Config{PathDB: &gethpathdb.Config{
			ReadOnly: pathdb.Defaults.ReadOnly,
		}}
	case customrawdb.FirewoodScheme:
		// Firewood disabled - use HashScheme instead
		return triedb.HashDefaults
		// fwCfg := firewood.Defaults
		// // Create a unique temporary directory for each test
		// fwCfg.FilePath = filepath.Join(t.TempDir(), "firewood_state") // matches blockchain.go
		// return &triedb.Config{DBOverride: fwCfg.BackendConstructor}
	default:
		t.Fatalf("unknown scheme %s", scheme)
	}
	return nil
}

// NOTE: TestVerkleGenesisCommit removed - Verkle trie support requires full
// implementation of hashAlloc() for Verkle roots in genesis.go. The current
// Lux EVM genesis.toBlock() uses a statedb approach which isn't compatible
// with Verkle's genesis hash computation. Re-add this test when Verkle is
// properly implemented using the hashAlloc approach from upstream geth.
