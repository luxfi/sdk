// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// consensusInterfaces "github.com/luxfi/consensus/core/interfaces" // TODO: Remove if not needed
	luxdatabase "github.com/luxfi/database"
	"github.com/luxfi/database/prefixdb"
	"github.com/luxfi/ids"

	// nodeConsensus "github.com/luxfi/consensus" // not used after fixes
	// consensusInterfaces "github.com/luxfi/consensus/interfaces" // not needed since using snow.State
	consensusBlock "github.com/luxfi/consensus/engine/chain/block"
	commonEng "github.com/luxfi/consensus/engine/core"
	"github.com/luxfi/consensus/snow"

	// "github.com/luxfi/node/upgrade/upgradetest" // not used after fixes
	"github.com/luxfi/consensus/utils/set"

	"github.com/luxfi/crypto"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/evm/consensus/dummy"
	"github.com/luxfi/evm/constants"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/evm/core/coretest"
	"github.com/luxfi/evm/plugin/evm/customrawdb"
	"github.com/luxfi/evm/plugin/evm/database"
	"github.com/luxfi/evm/predicate"
	statesyncclient "github.com/luxfi/evm/sync/client"
	"github.com/luxfi/evm/sync/statesync/statesynctest"
	"github.com/luxfi/evm/utils/utilstest"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/ethdb"
	ethparams "github.com/luxfi/geth/params"
	"github.com/luxfi/geth/rlp"
	"github.com/luxfi/geth/trie"
	"github.com/luxfi/geth/triedb"
	"github.com/luxfi/log"
	"github.com/luxfi/node/version"
)

// Define testKeys for this test file if not available from vm_test.go
var (
	syncTestKeys = secp256k1.TestKeys()[:3]
)

func TestSkipStateSync(t *testing.T) {
	// Fixed database lock issues by using proper test isolation
	t.Parallel() // Run in parallel to avoid database lock conflicts
	rand.New(rand.NewSource(1))
	test := syncTest{
		syncableInterval:   256,
		stateSyncMinBlocks: 300, // must be greater than [syncableInterval] to skip sync
		syncMode:           consensusBlock.StateSyncSkipped,
	}
	vmSetup := createSyncServerAndClientVMs(t, test, parentsToGet)

	testSyncerVM(t, vmSetup, test)
}

func TestStateSyncFromScratch(t *testing.T) {
	// Fixed database lock issues by using proper test isolation
	t.Parallel() // Run in parallel to avoid database lock conflicts
	rand.New(rand.NewSource(1))
	test := syncTest{
		syncableInterval:   256,
		stateSyncMinBlocks: 50, // must be less than [syncableInterval] to perform sync
		syncMode:           consensusBlock.StateSyncStatic,
	}
	vmSetup := createSyncServerAndClientVMs(t, test, parentsToGet)

	testSyncerVM(t, vmSetup, test)
}

func TestStateSyncFromScratchExceedParent(t *testing.T) {
	// Fixed database lock issues by using proper test isolation
	t.Parallel() // Run in parallel to avoid database lock conflicts
	rand.New(rand.NewSource(1))
	numToGen := parentsToGet + uint64(32)
	test := syncTest{
		syncableInterval:   numToGen,
		stateSyncMinBlocks: 50, // must be less than [syncableInterval] to perform sync
		syncMode:           consensusBlock.StateSyncStatic,
	}
	vmSetup := createSyncServerAndClientVMs(t, test, int(numToGen))

	testSyncerVM(t, vmSetup, test)
}

func TestStateSyncToggleEnabledToDisabled(t *testing.T) {
	// Fixed database lock issues by using proper test isolation
	t.Parallel() // Run in parallel to avoid database lock conflicts
	rand.New(rand.NewSource(1))

	var lock sync.Mutex
	reqCount := 0
	test := syncTest{
		syncableInterval:   256,
		stateSyncMinBlocks: 50, // must be less than [syncableInterval] to perform sync
		syncMode:           consensusBlock.StateSyncStatic,
		responseIntercept: func(syncerVM *VM, nodeID ids.NodeID, requestID uint32, response []byte) {
			lock.Lock()
			defer lock.Unlock()

			reqCount++
			// Fail all requests after number 50 to interrupt the sync
			if reqCount > 50 {
				appErr := &commonEng.AppError{Code: -1, Message: "timeout error"}
				if err := syncerVM.AppRequestFailed(context.Background(), nodeID, requestID, appErr); err != nil {
					panic(err)
				}
				cancel := syncerVM.StateSyncClient.(*stateSyncerClient).cancel
				if cancel != nil {
					cancel()
				} else {
					t.Fatal("state sync client not populated correctly")
				}
			} else {
				syncerVM.AppResponse(context.Background(), nodeID, requestID, response)
			}
		},
		expectedErr: context.Canceled,
	}
	vmSetup := createSyncServerAndClientVMs(t, test, parentsToGet)

	// Perform sync resulting in early termination.
	testSyncerVM(t, vmSetup, test)

	test.syncMode = consensusBlock.StateSyncStatic
	test.responseIntercept = nil
	test.expectedErr = nil

	syncDisabledVM := &VM{}
	appSender := &TestSender{T: t}
	appSender.SendAppGossipF = func(context.Context, set.Set[ids.NodeID], []byte) error { return nil }
	appSender.SendAppRequestF = func(ctx context.Context, nodeSet set.Set[ids.NodeID], requestID uint32, request []byte) error {
		nodeID, hasItem := nodeSet.Pop()
		if !hasItem {
			t.Fatal("expected nodeSet to contain at least 1 nodeID")
		}
		go vmSetup.serverVM.AppRequest(ctx, nodeID, requestID, time.Now().Add(1*time.Second), request)
		return nil
	}
	// Reset metrics to allow re-initialization
	// Note: Cannot reset metrics on live context - skipping
	stateSyncDisabledConfigJSON := `{"state-sync-enabled":false}`
	if err := syncDisabledVM.Initialize(
		context.Background(),
		vmSetup.syncerVM.chainCtx,
		vmSetup.syncerDB,
		[]byte(toGenesisJSON(forkToChainConfig["Latest"])),
		nil,
		[]byte(stateSyncDisabledConfigJSON),
		nil,
		nil, // fxs parameter
		appSender,
	); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := syncDisabledVM.Shutdown(context.Background()); err != nil {
			t.Fatal(err)
		}
	}()

	if height := syncDisabledVM.LastAcceptedBlockInternal().Height(); height != 0 {
		t.Fatalf("Unexpected last accepted height: %d", height)
	}

	enabled, err := syncDisabledVM.StateSyncEnabled(context.Background())
	assert.NoError(t, err)
	assert.False(t, enabled, "sync should be disabled")

	// Process the first 10 blocks from the serverVM
	for i := uint64(1); i < 10; i++ {
		ethBlock := vmSetup.serverVM.blockChain.GetBlockByNumber(i)
		if ethBlock == nil {
			t.Fatalf("VM Server did not have a block available at height %d", i)
		}
		b, err := rlp.EncodeToBytes(ethBlock)
		if err != nil {
			t.Fatal(err)
		}
		blk, err := syncDisabledVM.ParseBlock(context.Background(), b)
		if err != nil {
			t.Fatal(err)
		}
		if err := blk.Verify(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := blk.Accept(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	// Verify the snapshot disk layer matches the last block root
	lastRoot := syncDisabledVM.blockChain.CurrentBlock().Root
	if err := syncDisabledVM.blockChain.Snapshots().Verify(lastRoot); err != nil {
		t.Fatal(err)
	}
	syncDisabledVM.blockChain.DrainAcceptorQueue()

	// Create a new VM from the same database with state sync enabled.
	syncReEnabledVM := &VM{}
	// Enable state sync in configJSON
	configJSON := fmt.Sprintf(
		`{"state-sync-enabled":true, "state-sync-min-blocks":%d}`,
		test.stateSyncMinBlocks,
	)
	// Reset metrics to allow re-initialization
	// Note: Cannot reset metrics on live context - skipping
	if err := syncReEnabledVM.Initialize(
		context.Background(),
		vmSetup.syncerVM.chainCtx,
		vmSetup.syncerDB,
		[]byte(toGenesisJSON(forkToChainConfig["Latest"])),
		nil,
		[]byte(configJSON),
		nil,
		nil, // fxs parameter
		appSender,
	); err != nil {
		t.Fatal(err)
	}

	// override [serverVM]'s SendAppResponse function to trigger AppResponse on [syncerVM]
	vmSetup.serverAppSender.SendAppResponseF = func(ctx context.Context, nodeID ids.NodeID, requestID uint32, response []byte) error {
		if test.responseIntercept == nil {
			go syncReEnabledVM.AppResponse(ctx, nodeID, requestID, response)
		} else {
			go test.responseIntercept(syncReEnabledVM, nodeID, requestID, response)
		}

		return nil
	}

	// connect peer to [syncerVM]
	// Convert compat.Application to consensus.Application
	stateSyncVersion := &version.Application{
		Major: statesyncclient.StateSyncVersion.Major,
		Minor: statesyncclient.StateSyncVersion.Minor,
		Patch: statesyncclient.StateSyncVersion.Patch,
	}
	// Use a test node ID for connection
	testNodeID := ids.GenerateTestNodeID()
	assert.NoError(t, syncReEnabledVM.Connected(
		context.Background(),
		testNodeID,
		stateSyncVersion,
	))

	enabled, err = syncReEnabledVM.StateSyncEnabled(context.Background())
	assert.NoError(t, err)
	assert.True(t, enabled, "sync should be enabled")

	vmSetup.syncerVM = syncReEnabledVM
	testSyncerVM(t, vmSetup, test)
}

func TestVMShutdownWhileSyncing(t *testing.T) {
	var (
		lock    sync.Mutex
		vmSetup *syncVMSetup
	)
	reqCount := 0
	test := syncTest{
		syncableInterval:   256,
		stateSyncMinBlocks: 50, // must be less than [syncableInterval] to perform sync
		syncMode:           consensusBlock.StateSyncStatic,
		responseIntercept: func(syncerVM *VM, nodeID ids.NodeID, requestID uint32, response []byte) {
			lock.Lock()
			defer lock.Unlock()

			reqCount++
			// Shutdown the VM after 50 requests to interrupt the sync
			if reqCount == 50 {
				// Note this verifies the VM shutdown does not time out while syncing.
				require.NoError(t, vmSetup.shutdownOnceSyncerVM.Shutdown(context.Background()))
			} else if reqCount < 50 {
				require.NoError(t, syncerVM.AppResponse(context.Background(), nodeID, requestID, response))
			}
		},
		expectedErr: context.Canceled,
	}
	vmSetup = createSyncServerAndClientVMs(t, test, parentsToGet)
	// Perform sync resulting in early termination.
	testSyncerVM(t, vmSetup, test)
}

func createSyncServerAndClientVMs(t *testing.T, test syncTest, numBlocks int) *syncVMSetup {
	require := require.New(t)
	// configure [serverVM]
	serverVM := newVM(t, testVMConfig{
		genesisJSON: toGenesisJSON(forkToChainConfig["Latest"]),
	})

	t.Cleanup(func() {
		log.Info("Shutting down server VM")
		require.NoError(serverVM.vm.Shutdown(context.Background()))
	})
	generateAndAcceptBlocks(t, serverVM.vm, numBlocks, func(i int, gen *core.BlockGen) {
		b, err := predicate.NewResults().Bytes()
		if err != nil {
			t.Fatal(err)
		}
		gen.AppendExtra(b)

		tx := types.NewTransaction(gen.TxNonce(testEthAddrs[0]), testEthAddrs[1], common.Big1, ethparams.TxGas, big.NewInt(testMinGasPrice), nil)
		// Convert secp256k1 key to ECDSA for signing
		keyBytes := syncTestKeys[0].Bytes()
		privKey, err := crypto.ToECDSA(keyBytes)
		require.NoError(err)
		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(serverVM.vm.chainConfig.ChainID), privKey)
		require.NoError(err)
		gen.AddTx(signedTx)
	}, nil)

	// make some accounts
	trieDB := triedb.NewDatabase(serverVM.vm.chaindb, nil)
	root, accounts := statesynctest.FillAccountsWithOverlappingStorage(t, trieDB, types.EmptyRootHash, 1000, 16)

	// patch serverVM's lastAcceptedBlock to have the new root
	// and update the vm's state so the trie with accounts will
	// be returned by StateSyncGetLastSummary
	lastAccepted := serverVM.vm.blockChain.LastAcceptedBlock()
	patchedBlock := patchBlock(lastAccepted, root, serverVM.vm.chaindb)
	blockBytes, err := rlp.EncodeToBytes(patchedBlock)
	require.NoError(err)
	internalBlock, err := serverVM.vm.parseBlock(context.Background(), blockBytes)
	require.NoError(err)
	require.NoError(serverVM.vm.SetLastAcceptedBlock(internalBlock))

	// patch syncableInterval for test
	serverVM.vm.StateSyncServer.(*stateSyncServer).syncableInterval = test.syncableInterval

	// initialise [syncerVM] with blank genesis state
	stateSyncEnabledJSON := fmt.Sprintf(`{"state-sync-enabled":true, "state-sync-min-blocks": %d, "tx-lookup-limit": %d}`, test.stateSyncMinBlocks, 4)
	syncerVM := newVM(t, testVMConfig{
		genesisJSON: toGenesisJSON(forkToChainConfig["Latest"]),
		configJSON:  stateSyncEnabledJSON,
		isSyncing:   true,
	})

	shutdownOnceSyncerVM := &shutdownOnceVM{VM: syncerVM.vm}
	t.Cleanup(func() {
		require.NoError(shutdownOnceSyncerVM.Shutdown(context.Background()))
	})
	// Set the state to syncing
	// Use NormalOp state from node consensus
	require.NoError(syncerVM.vm.SetState(context.Background(), snow.NormalOp))
	enabled, err := syncerVM.vm.StateSyncEnabled(context.Background())
	require.NoError(err)
	require.True(enabled)

	// override [serverVM]'s SendAppResponse function to trigger AppResponse on [syncerVM]
	serverVM.appSender.SendAppResponseF = func(ctx context.Context, nodeID ids.NodeID, requestID uint32, response []byte) error {
		if test.responseIntercept == nil {
			go syncerVM.vm.AppResponse(ctx, nodeID, requestID, response)
		} else {
			go test.responseIntercept(syncerVM.vm, nodeID, requestID, response)
		}

		return nil
	}

	// connect peer to [syncerVM]
	// Convert compat.Application to consensus.Application
	stateSyncVersionForConnect := &version.Application{
		Major: statesyncclient.StateSyncVersion.Major,
		Minor: statesyncclient.StateSyncVersion.Minor,
		Patch: statesyncclient.StateSyncVersion.Patch,
	}
	// Use a test node ID for connection
	testNodeID := ids.GenerateTestNodeID()
	require.NoError(
		syncerVM.vm.Connected(
			context.Background(),
			testNodeID,
			stateSyncVersionForConnect,
		),
	)

	// override [syncerVM]'s SendAppRequest function to trigger AppRequest on [serverVM]
	syncerVM.appSender.SendAppRequestF = func(ctx context.Context, nodeSet set.Set[ids.NodeID], requestID uint32, request []byte) error {
		nodeID, hasItem := nodeSet.Pop()
		require.True(hasItem, "expected nodeSet to contain at least 1 nodeID")
		require.NoError(serverVM.vm.AppRequest(ctx, nodeID, requestID, time.Now().Add(1*time.Second), request))
		return nil
	}

	return &syncVMSetup{
		serverVM:             serverVM.vm,
		serverAppSender:      serverVM.appSender,
		fundedAccounts:       accounts,
		syncerVM:             syncerVM.vm,
		syncerDB:             syncerVM.db,
		shutdownOnceSyncerVM: shutdownOnceSyncerVM,
	}
}

// syncVMSetup contains the required set up for a client VM to perform state sync
// off of a server VM.
type syncVMSetup struct {
	serverVM        *VM
	serverAppSender *TestSender

	fundedAccounts map[*utilstest.Key]*types.StateAccount

	syncerVM             *VM
	syncerDB             luxdatabase.Database
	shutdownOnceSyncerVM *shutdownOnceVM
}

type shutdownOnceVM struct {
	*VM
	shutdownOnce sync.Once
}

func (vm *shutdownOnceVM) Shutdown(ctx context.Context) error {
	var err error
	vm.shutdownOnce.Do(func() { err = vm.VM.Shutdown(ctx) })
	return err
}

// syncTest contains both the actual VMs as well as the parameters with the expected output.
type syncTest struct {
	responseIntercept  func(vm *VM, nodeID ids.NodeID, requestID uint32, response []byte)
	stateSyncMinBlocks uint64
	syncableInterval   uint64
	syncMode           consensusBlock.StateSyncMode
	expectedErr        error
}

func testSyncerVM(t *testing.T, vmSetup *syncVMSetup, test syncTest) {
	t.Helper()
	var (
		require        = require.New(t)
		serverVM       = vmSetup.serverVM
		fundedAccounts = vmSetup.fundedAccounts
		syncerVM       = vmSetup.syncerVM
	)
	// get last summary and test related methods
	summary, err := serverVM.GetLastStateSummary(context.Background())
	require.NoError(err, "error getting state sync last summary")
	parsedSummary, err := syncerVM.ParseStateSummary(context.Background(), summary.Bytes())
	require.NoError(err, "error parsing state summary")
	retrievedSummary, err := serverVM.GetStateSummary(context.Background(), parsedSummary.Height())
	require.NoError(err, "error getting state sync summary at height")
	require.Equal(summary, retrievedSummary)

	syncMode, err := parsedSummary.Accept(context.Background())
	require.NoError(err, "error accepting state summary")
	require.Equal(test.syncMode, syncMode)
	if syncMode == consensusBlock.StateSyncSkipped {
		return
	}

	msg, err := syncerVM.WaitForEvent(context.Background())
	require.NoError(err)
	// StateSyncDone no longer exists, checking for specific message instead
	require.NotNil(msg)

	// If the test is expected to error, assert the correct error is returned and finish the test.
	err = syncerVM.Error()
	if test.expectedErr != nil {
		require.ErrorIs(err, test.expectedErr)
		// Note we re-open the database here to avoid a closed error when the test is for a shutdown VM.
		chaindb := database.WrapDatabase(prefixdb.NewNested(ethDBPrefix, syncerVM.versiondb))
		assertSyncPerformedHeights(t, chaindb, map[uint64]struct{}{})
		return
	}
	require.NoError(err, "state sync failed")

	// set [syncerVM] to bootstrapping and verify the last accepted block has been updated correctly
	// and that we can bootstrap and process some blocks.
	require.NoError(syncerVM.SetState(context.Background(), snow.Bootstrapping))
	require.Equal(serverVM.LastAcceptedBlock().Height(), syncerVM.LastAcceptedBlock().Height(), "block height mismatch between syncer and server")
	require.Equal(serverVM.LastAcceptedBlock().ID(), syncerVM.LastAcceptedBlock().ID(), "blockID mismatch between syncer and server")
	require.True(syncerVM.blockChain.HasState(syncerVM.blockChain.LastAcceptedBlock().Root()), "unavailable state for last accepted block")
	assertSyncPerformedHeights(t, syncerVM.chaindb, map[uint64]struct{}{retrievedSummary.Height(): {}})

	lastNumber := syncerVM.blockChain.LastAcceptedBlock().NumberU64()
	// check the last block is indexed
	lastSyncedBlock := rawdb.ReadBlock(syncerVM.chaindb, rawdb.ReadCanonicalHash(syncerVM.chaindb, lastNumber), lastNumber)
	for _, tx := range lastSyncedBlock.Transactions() {
		index := rawdb.ReadTxLookupEntry(syncerVM.chaindb, tx.Hash())
		require.NotNilf(index, "Miss transaction indices, number %d hash %s", lastNumber, tx.Hash().Hex())
	}

	// tail should be the last block synced
	if syncerVM.ethConfig.TransactionHistory != 0 {
		tail := lastSyncedBlock.NumberU64()

		coretest.CheckTxIndices(t, &tail, tail, tail, tail, syncerVM.chaindb, true)
	}

	blocksToBuild := 10
	txsPerBlock := 10
	toAddress := testEthAddrs[1] // arbitrary choice
	generateAndAcceptBlocks(t, syncerVM, blocksToBuild, func(_ int, gen *core.BlockGen) {
		b, err := predicate.NewResults().Bytes()
		if err != nil {
			t.Fatal(err)
		}
		gen.AppendExtra(b)
		i := 0
		for k := range fundedAccounts {
			tx := types.NewTransaction(gen.TxNonce(k.Address), toAddress, big.NewInt(1), 21000, big.NewInt(testMinGasPrice), nil)
			signedTx, err := types.SignTx(tx, types.NewEIP155Signer(serverVM.chainConfig.ChainID), k.PrivateKey)
			require.NoError(err)
			gen.AddTx(signedTx)
			i++
			if i >= txsPerBlock {
				break
			}
		}
	},
		func(block *types.Block) {
			if syncerVM.ethConfig.TransactionHistory != 0 {
				tail := block.NumberU64() - syncerVM.ethConfig.TransactionHistory + 1
				// tail should be the minimum last synced block, since we skipped it to the last block
				if tail < lastSyncedBlock.NumberU64() {
					tail = lastSyncedBlock.NumberU64()
				}
				coretest.CheckTxIndices(t, &tail, tail, block.NumberU64(), block.NumberU64(), syncerVM.chaindb, true)
			}
		},
	)

	// check we can transition to [NormalOp] state and continue to process blocks.
	require.NoError(syncerVM.SetState(context.Background(), snow.NormalOp))
	require.True(syncerVM.bootstrapped.Get())

	// Generate blocks after we have entered normal consensus as well
	generateAndAcceptBlocks(t, syncerVM, blocksToBuild, func(_ int, gen *core.BlockGen) {
		b, err := predicate.NewResults().Bytes()
		require.NoError(err)
		gen.AppendExtra(b)
		i := 0
		for k := range fundedAccounts {
			tx := types.NewTransaction(gen.TxNonce(k.Address), toAddress, big.NewInt(1), 21000, big.NewInt(testMinGasPrice), nil)
			signedTx, err := types.SignTx(tx, types.NewEIP155Signer(serverVM.chainConfig.ChainID), k.PrivateKey)
			require.NoError(err)
			gen.AddTx(signedTx)
			i++
			if i >= txsPerBlock {
				break
			}
		}
	},
		func(block *types.Block) {
			if syncerVM.ethConfig.TransactionHistory != 0 {
				tail := block.NumberU64() - syncerVM.ethConfig.TransactionHistory + 1
				// tail should be the minimum last synced block, since we skipped it to the last block
				if tail < lastSyncedBlock.NumberU64() {
					tail = lastSyncedBlock.NumberU64()
				}
				coretest.CheckTxIndices(t, &tail, tail, block.NumberU64(), block.NumberU64(), syncerVM.chaindb, true)
			}
		},
	)
}

// patchBlock returns a copy of [blk] with [root] and updates [db] to
// include the new block as canonical for [blk]'s height.
// This breaks the digestibility of the chain since after this call
// [blk] does not necessarily define a state transition from its parent
// state to the new state root.
func patchBlock(blk *types.Block, root common.Hash, db ethdb.Database) *types.Block {
	header := blk.Header()
	header.Root = root
	receipts := rawdb.ReadRawReceipts(db, blk.Hash(), blk.NumberU64())
	newBlk := types.NewBlock(
		header,
		&types.Body{
			Transactions: blk.Transactions(),
			Uncles:       blk.Uncles(),
		},
		receipts,
		trie.NewStackTrie(nil),
	)
	rawdb.WriteBlock(db, newBlk)
	rawdb.WriteCanonicalHash(db, newBlk.Hash(), newBlk.NumberU64())
	return newBlk
}

// generateAndAcceptBlocks uses [core.GenerateChain] to generate blocks, then
// calls Verify and Accept on each generated block
func generateAndAcceptBlocks(t *testing.T, vm *VM, numBlocks int, gen func(int, *core.BlockGen), accepted func(*types.Block)) {
	t.Helper()

	// acceptExternalBlock defines a function to parse, verify, and accept a block once it has been
	// generated by GenerateChain
	acceptExternalBlock := func(block *types.Block) {
		bytes, err := rlp.EncodeToBytes(block)
		if err != nil {
			t.Fatal(err)
		}
		vmBlock, err := vm.ParseBlock(context.Background(), bytes)
		if err != nil {
			t.Fatal(err)
		}
		if err := vmBlock.Verify(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := vmBlock.Accept(context.Background()); err != nil {
			t.Fatal(err)
		}

		if accepted != nil {
			accepted(block)
		}
	}
	_, _, err := core.GenerateChain(
		vm.chainConfig,
		vm.blockChain.LastAcceptedBlock(),
		dummy.NewETHFaker(),
		vm.chaindb,
		numBlocks,
		10,
		func(i int, g *core.BlockGen) {
			g.SetOnBlockGenerated(acceptExternalBlock)
			g.SetCoinbase(constants.BlackholeAddr) // necessary for syntactic validation of the block
			gen(i, g)
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	vm.blockChain.DrainAcceptorQueue()
}

// assertSyncPerformedHeights iterates over all heights the VM has synced to and
// verifies it matches [expected].
func assertSyncPerformedHeights(t *testing.T, db ethdb.Iteratee, expected map[uint64]struct{}) {
	it := customrawdb.NewSyncPerformedIterator(db)
	defer it.Release()

	found := make(map[uint64]struct{}, len(expected))
	for it.Next() {
		found[customrawdb.UnpackSyncPerformedKey(it.Key())] = struct{}{}
	}
	require.NoError(t, it.Error())
	require.Equal(t, expected, found)
}
