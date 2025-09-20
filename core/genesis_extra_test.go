// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package core

import (
	"math/big"
	"testing"
	"time"

	"github.com/luxfi/evm/commontype"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/evm/params/extras"
	"github.com/luxfi/evm/utils"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/triedb"
	"github.com/stretchr/testify/require"
)

func TestGenesisEthUpgrades(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	preEthUpgrades := params.WithExtra(
		&params.ChainConfig{
			ChainID:        big.NewInt(43114), // Specifically refers to mainnet for this UT
			HomesteadBlock: big.NewInt(0),
			// For this test to be a proper regression test, DAOForkBlock and
			// DAOForkSupport should be set to match the values in
			// [params.SetEthUpgrades]. Otherwise, in case of a regression, the test
			// would pass as there would be a mismatch at genesis, which is
			// incorrectly considered a success.
			DAOForkBlock:        big.NewInt(0),
			DAOForkSupport:      true,
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			IstanbulBlock:       big.NewInt(0),
			MuirGlacierBlock:    big.NewInt(0),
		},
		&extras.ChainConfig{
			FeeConfig: commontype.FeeConfig{
				MinBaseFee: big.NewInt(1),
			},
			NetworkUpgrades: extras.NetworkUpgrades{
				SubnetEVMTimestamp: utils.NewUint64(0),
			},
		},
	)
	tdb := triedb.NewDatabase(db, triedb.HashDefaults)
	config := *preEthUpgrades
	// Set this up once, just to get the genesis hash
	_, genHash, err := SetupGenesisBlock(db, tdb, &Genesis{Config: &config}, common.Hash{}, false)
	require.NoError(t, err)
	// Write the configuration back to the db as it would be in prior versions
	rawdb.WriteChainConfig(db, genHash, preEthUpgrades)
	// Make some other block
	block := types.NewBlock(
		&types.Header{
			Number:     big.NewInt(1640340), // Berlin activation on mainnet
			Difficulty: big.NewInt(1),
			ParentHash: genHash,
			Time:       uint64(time.Now().Unix()),
		},
		nil, nil, nil,
	)
	rawdb.WriteBlock(db, block)
	// We should still be able to re-initialize
	config = *preEthUpgrades
	require.NoError(t, params.SetEthUpgrades(&config)) // New versions will set additional fields eg, LondonBlock
	_, _, err = SetupGenesisBlock(db, tdb, &Genesis{Config: &config}, block.Hash(), false)
	require.NoError(t, err)
}
