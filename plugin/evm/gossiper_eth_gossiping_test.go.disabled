// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/luxfi/consensus/utils/set"
	"github.com/luxfi/ids"

	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/common"

	"github.com/stretchr/testify/assert"

	"github.com/luxfi/evm/core"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/geth/core/types"
)

func fundAddressByGenesis(addrs []common.Address) (string, error) {
	balance := big.NewInt(0xffffffffffffff)

	// Use params.TestChainConfig which has proper network upgrade timestamps
	// Use params.Copy to properly copy extras from the sync.Map
	cpyCfg := params.Copy(params.TestChainConfig)
	cpyCfg.ChainID = big.NewInt(43111)

	// Create genesis with proper config
	genesis := &core.Genesis{
		Difficulty: common.Big0,
		GasLimit:   params.GetExtra(cpyCfg).FeeConfig.GasLimit.Uint64(),
		Timestamp:  0,
	}
	funds := make(map[common.Address]types.Account)
	for _, addr := range addrs {
		funds[addr] = types.Account{
			Balance: balance,
		}
	}
	genesis.Alloc = funds
	genesis.Config = cpyCfg

	// Marshal the genesis normally
	b, err := json.Marshal(genesis)
	if err != nil {
		return "", err
	}

	// Now we need to add the network upgrades to the JSON
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(b, &jsonMap); err != nil {
		return "", err
	}

	// Add the network upgrades to the config
	if configMap, ok := jsonMap["config"].(map[string]interface{}); ok {
		if extra := params.GetExtra(cpyCfg); extra != nil {
			// Add the network upgrade timestamps
			if extra.SubnetEVMTimestamp != nil {
				configMap["subnetEVMTimestamp"] = *extra.SubnetEVMTimestamp
			}
			if extra.DurangoTimestamp != nil {
				configMap["durangoTimestamp"] = *extra.DurangoTimestamp
			}
			if extra.EtnaTimestamp != nil {
				configMap["etnaTimestamp"] = *extra.EtnaTimestamp
			}
			if extra.FortunaTimestamp != nil {
				configMap["fortunaTimestamp"] = *extra.FortunaTimestamp
			}
			if extra.GraniteTimestamp != nil {
				configMap["graniteTimestamp"] = *extra.GraniteTimestamp
			}
		}
	}

	// Marshal the modified map back to JSON
	bytes, err := json.Marshal(jsonMap)
	return string(bytes), err
}

func getValidEthTxs(key *ecdsa.PrivateKey, count int, gasPrice *big.Int) []*types.Transaction {
	res := make([]*types.Transaction, count)

	to := common.Address{}
	amount := big.NewInt(0)
	gasLimit := uint64(37000)

	for i := 0; i < count; i++ {
		tx, _ := types.SignTx(
			types.NewTransaction(
				uint64(i),
				to,
				amount,
				gasLimit,
				gasPrice,
				[]byte(strings.Repeat("aaaaaaaaaa", 100))),
			types.HomesteadSigner{}, key)
		tx.SetTime(time.Now().Add(-1 * time.Minute))
		res[i] = tx
	}
	return res
}

// show that a geth tx discovered from gossip is requested to the same node that
// gossiped it
func TestMempoolEthTxsAppGossipHandling(t *testing.T) {
	// Fixed database lock issues by using proper test isolation
	t.Parallel() // Run in parallel to avoid database lock conflicts
	assert := assert.New(t)

	key, err := crypto.GenerateKey()
	assert.NoError(err)

	addr := crypto.PubkeyToAddress(key.PublicKey)

	genesisJSON, err := fundAddressByGenesis([]common.Address{common.Address(addr)})
	assert.NoError(err)

	tvm := newVM(t, testVMConfig{
		genesisJSON: genesisJSON,
	})

	defer func() {
		err := tvm.vm.Shutdown(context.Background())
		assert.NoError(err)
	}()
	tvm.vm.txPool.SetGasTip(common.Big1)
	tvm.vm.txPool.SetMinFee(common.Big0)

	var (
		wg          sync.WaitGroup
		txRequested bool
	)
	tvm.appSender.CantSendAppGossip = false
	tvm.appSender.SendAppRequestF = func(context.Context, set.Set[ids.NodeID], uint32, []byte) error {
		txRequested = true
		return nil
	}
	wg.Add(1)
	tvm.appSender.SendAppGossipF = func(context.Context, set.Set[ids.NodeID], []byte) error {
		wg.Done()
		return nil
	}

	// Set up push gossiper with loop for tests that use newVM()
	// Note: Don't call setupGossipInfrastructure() since the handler is already registered
	// by onNormalOperationsStarted() during SetState(VMNormalOp) in newVM()
	cancelGossip := setupPushGossiperWithLoop(t, tvm.vm, tvm.appSender)
	defer cancelGossip()

	// prepare a tx
	tx := getValidEthTxs(key, 1, common.Big1)[0]

	// Txs must be submitted over the API to be included in push gossip.
	// (i.e., txs received via p2p are not included in push gossip)
	err = tvm.vm.eth.APIBackend.SendTx(context.Background(), tx)
	assert.NoError(err)
	assert.False(txRequested, "tx should not be requested")

	// wait for transaction to be re-gossiped
	attemptAwait(t, &wg, 5*time.Second)
}

func attemptAwait(t *testing.T, wg *sync.WaitGroup, delay time.Duration) {
	ticker := make(chan struct{})

	// Wait for [wg] and then close [ticket] to indicate that
	// the wait group has finished.
	go func() {
		wg.Wait()
		close(ticker)
	}()

	select {
	case <-time.After(delay):
		t.Fatal("Timed out waiting for wait group to complete")
	case <-ticker:
		// The wait group completed without issue
	}
}
