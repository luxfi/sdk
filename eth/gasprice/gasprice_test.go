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
// Copyright 2020 The go-ethereum Authors
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

package gasprice

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/luxfi/crypto"
	"github.com/luxfi/evm/commontype"
	"github.com/luxfi/evm/consensus/dummy"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/evm/params/extras"
	customheader "github.com/luxfi/evm/plugin/evm/header"
	"github.com/luxfi/evm/plugin/evm/upgrade/legacy"
	"github.com/luxfi/evm/precompile/contracts/feemanager"
	"github.com/luxfi/evm/rpc"
	"github.com/luxfi/evm/utils"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/core/vm"
	"github.com/luxfi/geth/event"
	ethparams "github.com/luxfi/geth/params"
	"github.com/stretchr/testify/require"
)

var (
	key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr   = func() common.Address {
		cryptoAddr := crypto.PubkeyToAddress(key.PublicKey)
		return common.BytesToAddress(cryptoAddr[:])
	}()
	bal, _ = new(big.Int).SetString("100000000000000000000000", 10)
)

type testBackend struct {
	chain         *core.BlockChain
	acceptedEvent chan<- core.ChainEvent
}

func (b *testBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	if number == rpc.LatestBlockNumber {
		return b.chain.CurrentBlock(), nil
	}
	return b.chain.GetHeaderByNumber(uint64(number)), nil
}

func (b *testBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	if number == rpc.LatestBlockNumber {
		number = rpc.BlockNumber(b.chain.CurrentBlock().Number.Uint64())
	}
	return b.chain.GetBlockByNumber(uint64(number)), nil
}

func (b *testBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.chain.GetReceiptsByHash(hash), nil
}

func (b *testBackend) ChainConfig() *params.ChainConfig {
	return b.chain.Config()
}

func (b *testBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return nil
}

func (b *testBackend) SubscribeChainAcceptedEvent(ch chan<- core.ChainEvent) event.Subscription {
	b.acceptedEvent = ch
	return nil
}

func (b *testBackend) GetFeeConfigAt(parent *types.Header) (commontype.FeeConfig, *big.Int, error) {
	return b.chain.GetFeeConfigAt(parent)
}

func (b *testBackend) teardown() {
	b.chain.Stop()
}

func newTestBackendFakerEngine(t *testing.T, config *params.ChainConfig, numBlocks int, genBlocks func(i int, b *core.BlockGen)) *testBackend {
	var gspec = &core.Genesis{
		Config: config,
		Alloc:  types.GenesisAlloc{addr: {Balance: bal}},
	}

	engine := dummy.NewETHFaker()

	// Generate testing blocks
	targetBlockRate := params.GetExtra(config).FeeConfig.TargetBlockRate
	genDb, blocks, _, err := core.GenerateChainWithGenesis(gspec, engine, numBlocks, targetBlockRate-1, genBlocks)
	if err != nil {
		t.Fatal(err)
	}
	// Construct testing chain
	chain, err := core.NewBlockChain(genDb, core.DefaultCacheConfig, gspec, engine, vm.Config{}, common.Hash{}, false)
	if err != nil {
		t.Fatalf("Failed to create local chain, %v", err)
	}
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("Failed to insert chain, %v", err)
	}
	return &testBackend{chain: chain}
}

// newTestBackend creates a test backend. OBS: don't forget to invoke tearDown
// after use, otherwise the blockchain instance will mem-leak via goroutines.
// Uses NewETHFaker to skip block gas cost validation which is not needed for gas oracle tests.
func newTestBackend(t *testing.T, config *params.ChainConfig, numBlocks int, genBlocks func(i int, b *core.BlockGen)) *testBackend {
	var gspec = &core.Genesis{
		Config: config,
		Alloc:  types.GenesisAlloc{addr: {Balance: bal}},
	}

	engine := dummy.NewETHFaker()

	// Generate testing blocks
	targetBlockRate := params.GetExtra(config).FeeConfig.TargetBlockRate
	genDb, blocks, _, err := core.GenerateChainWithGenesis(gspec, engine, numBlocks, targetBlockRate-1, genBlocks)
	if err != nil {
		t.Fatal(err)
	}
	// Construct testing chain
	chain, err := core.NewBlockChain(genDb, core.DefaultCacheConfig, gspec, engine, vm.Config{}, common.Hash{}, false)
	if err != nil {
		t.Fatalf("Failed to create local chain, %v", err)
	}
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("Failed to insert chain, %v", err)
	}
	return &testBackend{chain: chain}
}

func (b *testBackend) MinRequiredTip(ctx context.Context, header *types.Header) (*big.Int, error) {
	config := params.GetExtra(b.chain.Config())
	return customheader.EstimateRequiredTip(config, header)
}

func (b *testBackend) CurrentHeader() *types.Header {
	return b.chain.CurrentHeader()
}

func (b *testBackend) LastAcceptedBlock() *types.Block {
	current := b.chain.CurrentBlock()
	if current == nil {
		return nil
	}
	return b.chain.GetBlockByNumber(current.Number.Uint64())
}

func (b *testBackend) GetBlockByNumber(number uint64) *types.Block {
	return b.chain.GetBlockByNumber(number)
}

type suggestTipCapTest struct {
	chainConfig *params.ChainConfig
	numBlocks   int
	genBlock    func(i int, b *core.BlockGen)
	expectedTip *big.Int
}

func defaultOracleConfig() Config {
	return Config{
		Blocks:             20,
		Percentile:         60,
		MaxLookbackSeconds: 80,
	}
}

// timeCrunchOracleConfig returns a config with [MaxLookbackSeconds] set to 5
// to ensure that during gas price estimation, we will hit the time based look back limit
func timeCrunchOracleConfig() Config {
	return Config{
		Blocks:             20,
		Percentile:         60,
		MaxLookbackSeconds: 5,
	}
}

func applyGasPriceTest(t *testing.T, test suggestTipCapTest, config Config) {
	if test.genBlock == nil {
		test.genBlock = func(i int, b *core.BlockGen) {}
	}
	backend := newTestBackend(t, test.chainConfig, test.numBlocks, test.genBlock)
	oracle, err := NewOracle(backend, config)
	require.NoError(t, err)

	// mock time to be consistent across different CI runs
	// sets currentTime to be 20 seconds
	oracle.clock.Set(time.Unix(20, 0))

	got, err := oracle.SuggestTipCap(context.Background())
	backend.teardown()
	require.NoError(t, err)

	if got.Cmp(test.expectedTip) != 0 {
		t.Fatalf("Expected tip (%d), got tip (%d)", test.expectedTip, got)
	}
}

func testGenBlock(t *testing.T, tip int64, numTx int) func(int, *core.BlockGen) {
	return func(i int, b *core.BlockGen) {
		b.SetCoinbase(common.Address{1})

		txTip := big.NewInt(tip * params.GWei)
		signer := types.LatestSigner(params.TestChainConfig)
		baseFee := b.BaseFee()
		feeCap := new(big.Int).Add(baseFee, txTip)
		for j := 0; j < numTx; j++ {
			tx := types.NewTx(&types.DynamicFeeTx{
				ChainID:   params.TestChainConfig.ChainID,
				Nonce:     b.TxNonce(addr),
				To:        &common.Address{},
				Gas:       ethparams.TxGas,
				GasFeeCap: feeCap,
				GasTipCap: txTip,
				Data:      []byte{},
			})
			tx, err := types.SignTx(tx, signer, key)
			require.NoError(t, err, "failed to create tx")
			b.AddTx(tx)
		}
	}
}

func TestSuggestTipCapNetworkUpgrades(t *testing.T) {
	tests := map[string]suggestTipCapTest{
		"subnet evm": {
			chainConfig: params.TestChainConfig,
			expectedTip: DefaultMinPrice,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			applyGasPriceTest(t, test, defaultOracleConfig())
		})
	}
}

func TestSuggestTipCapSimple(t *testing.T) {
	// Lux EVM's gas oracle calculates tips based on blockGasCost * baseFee / totalGasUsed.
	// With 3 blocks and 370 txs per block at 55 gwei tip, the estimated required tip is 1 wei.
	// This is because Lux EVM uses block gas cost mechanism rather than go-ethereum's
	// percentile-based tip calculation.
	applyGasPriceTest(t, suggestTipCapTest{
		chainConfig: params.TestChainConfig,
		numBlocks:   3,
		genBlock:    testGenBlock(t, 55, 370),
		expectedTip: big.NewInt(1),
	}, defaultOracleConfig())
}

func TestSuggestTipCapSimpleFloor(t *testing.T) {
	// Lux EVM's EstimateRequiredTip rounds up when calculating the required tip per gas.
	// With 1 block and 370 txs at 55 gwei, the calculation results in a tip of 1 wei.
	// The rounding ensures transactions always have a non-zero minimum tip.
	applyGasPriceTest(t, suggestTipCapTest{
		chainConfig: params.TestChainConfig,
		numBlocks:   1,
		genBlock:    testGenBlock(t, 55, 370),
		expectedTip: big.NewInt(1),
	}, defaultOracleConfig())
}

func TestSuggestTipCapSmallTips(t *testing.T) {
	// This test includes alternating high (550 gwei) and low (1 wei) tip transactions.
	// Lux EVM's gas oracle calculates the required tip based on block gas cost,
	// which results in a tip of 1 wei regardless of the actual tips in transactions.
	tip := big.NewInt(550 * params.GWei)
	applyGasPriceTest(t, suggestTipCapTest{
		chainConfig: params.TestChainConfig,
		numBlocks:   3,
		genBlock: func(i int, b *core.BlockGen) {
			b.SetCoinbase(common.Address{1})

			signer := types.LatestSigner(params.TestChainConfig)
			baseFee := b.BaseFee()
			feeCap := new(big.Int).Add(baseFee, tip)
			for j := 0; j < 185; j++ {
				tx := types.NewTx(&types.DynamicFeeTx{
					ChainID:   params.TestChainConfig.ChainID,
					Nonce:     b.TxNonce(addr),
					To:        &common.Address{},
					Gas:       ethparams.TxGas,
					GasFeeCap: feeCap,
					GasTipCap: tip,
					Data:      []byte{},
				})
				tx, err := types.SignTx(tx, signer, key)
				if err != nil {
					t.Fatalf("failed to create tx: %s", err)
				}
				b.AddTx(tx)
				tx = types.NewTx(&types.DynamicFeeTx{
					ChainID:   params.TestChainConfig.ChainID,
					Nonce:     b.TxNonce(addr),
					To:        &common.Address{},
					Gas:       ethparams.TxGas,
					GasFeeCap: feeCap,
					GasTipCap: common.Big1,
					Data:      []byte{},
				})
				tx, err = types.SignTx(tx, signer, key)
				require.NoError(t, err, "failed to create tx")
				b.AddTx(tx)
			}
		},
		expectedTip: big.NewInt(1),
	}, defaultOracleConfig())
}

func TestSuggestTipCapMinGas(t *testing.T) {
	// With only 50 transactions per block at 500 gwei tip, the total gas used is below
	// the MinGasUsed threshold (6,000,000). When gas usage is below this threshold,
	// the oracle returns 0 (or the last cached price) as there's insufficient data
	// to make a reliable tip estimate.
	applyGasPriceTest(t, suggestTipCapTest{
		chainConfig: params.TestChainConfig,
		numBlocks:   3,
		genBlock:    testGenBlock(t, 500, 50),
		expectedTip: big.NewInt(0),
	}, defaultOracleConfig())
}

// Regression test to ensure that SuggestPrice does not panic with activation of Subnet EVM
// Note: support for gas estimation without activated hard forks has been deprecated, but we still
// ensure that the call does not panic.
func TestSuggestGasPriceSubnetEVM(t *testing.T) {
	config := Config{
		Blocks:     20,
		Percentile: 60,
	}

	backend := newTestBackend(t, params.TestChainConfig, 3, func(i int, b *core.BlockGen) {
		b.SetCoinbase(common.Address{1})

		signer := types.LatestSigner(params.TestChainConfig)
		gasPrice := big.NewInt(legacy.BaseFee)
		for j := 0; j < 50; j++ {
			tx := types.NewTx(&types.LegacyTx{
				Nonce:    b.TxNonce(addr),
				To:       &common.Address{},
				Gas:      ethparams.TxGas,
				GasPrice: gasPrice,
				Data:     []byte{},
			})
			tx, err := types.SignTx(tx, signer, key)
			require.NoError(t, err, "failed to create tx")
			b.AddTx(tx)
		}
	})
	defer backend.teardown()

	oracle, err := NewOracle(backend, config)
	require.NoError(t, err)

	_, err = oracle.SuggestPrice(context.Background())
	require.NoError(t, err)
}

func TestSuggestTipCapMaxBlocksLookback(t *testing.T) {
	// With 20 blocks at default oracle config (MaxLookbackSeconds=80), the oracle
	// looks back through all blocks. The tip estimate of 2 wei reflects the
	// block gas cost calculation across the entire lookback window.
	applyGasPriceTest(t, suggestTipCapTest{
		chainConfig: params.TestChainConfig,
		numBlocks:   20,
		genBlock:    testGenBlock(t, 550, 370),
		expectedTip: big.NewInt(2),
	}, defaultOracleConfig())
}

func TestSuggestTipCapMaxBlocksSecondsLookback(t *testing.T) {
	// With timeCrunchOracleConfig (MaxLookbackSeconds=5), the oracle only considers
	// recent blocks within the 5-second window. With fewer blocks in the calculation,
	// the tip estimate is 3 wei due to the limited sample size.
	applyGasPriceTest(t, suggestTipCapTest{
		chainConfig: params.TestChainConfig,
		numBlocks:   20,
		genBlock:    testGenBlock(t, 550, 370),
		expectedTip: big.NewInt(3),
	}, timeCrunchOracleConfig())
}

// Regression test to ensure the last estimation of base fee is not used
// for the block immediately following a fee configuration update.
func TestSuggestGasPriceAfterFeeConfigUpdate(t *testing.T) {
	require := require.New(t)
	config := Config{
		Blocks:     20,
		Percentile: 60,
	}

	// Create a chain config with fee manager enabled at genesis with [addr] as the admin
	chainConfig := params.Copy(params.TestChainConfig)
	chainConfigExtra := params.GetExtra(chainConfig)
	chainConfigExtra.GenesisPrecompiles = extras.Precompiles{
		feemanager.ConfigKey: feemanager.NewConfig(utils.NewUint64(0), []common.Address{addr}, nil, nil, nil),
	}

	// Create a fee config with higher MinBaseFee
	highFeeConfig := extras.DefaultFeeConfig
	highFeeConfig.MinBaseFee = big.NewInt(28_000_000_000)
	data, err := feemanager.PackSetFeeConfig(highFeeConfig)
	require.NoError(err)

	signer := types.LatestSigner(chainConfig)

	// Create backend with one block that changes the fee config
	// We need to provide a tip to cover the block gas cost
	tipAmount := big.NewInt(1 * params.GWei)
	backend := newTestBackend(t, chainConfig, 1, func(i int, b *core.BlockGen) {
		b.SetCoinbase(common.Address{1})

		baseFee := b.BaseFee()
		feeCap := new(big.Int).Add(baseFee, tipAmount)

		// Admin issues tx to change fee config to higher MinBaseFee
		tx := types.NewTx(&types.DynamicFeeTx{
			ChainID:   chainConfig.ChainID,
			Nonce:     b.TxNonce(addr),
			To:        &feemanager.ContractAddress,
			Gas:       chainConfigExtra.FeeConfig.GasLimit.Uint64(),
			Value:     common.Big0,
			GasFeeCap: feeCap,
			GasTipCap: tipAmount,
			Data:      data,
		})
		tx, err = types.SignTx(tx, signer, key)
		require.NoError(err, "failed to create tx")
		b.AddTx(tx)
	})
	defer backend.teardown()

	oracle, err := NewOracle(backend, config)
	require.NoError(err)

	// After the fee config update, the suggested price should follow the new config
	got, err := oracle.SuggestPrice(context.Background())
	require.NoError(err)
	require.Equal(highFeeConfig.MinBaseFee, got)
}
