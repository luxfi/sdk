// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package exchangevm

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	consensusctx "github.com/luxfi/consensus/context"
	core "github.com/luxfi/consensus/core"
	"github.com/luxfi/consensus/core/choices"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/database/memdb"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/warp"
	"github.com/luxfi/sdk/node/chains/atomic"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/utils/formatting/address"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/upgrade"
	"github.com/luxfi/upgrade/upgradetest"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/exchangevm/txs"
	"github.com/luxfi/sdk/node/vms/exchangevm/txs/txstest"
	"github.com/luxfi/sdk/node/vms/nftfx"
	"github.com/luxfi/sdk/node/vms/propertyfx"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
)

// Test keys for use in tests
var keys = secp256k1.TestKeys()

// durango is a shorthand for the Durango fork config
var durango = upgradetest.GetConfig(upgradetest.Durango)

// Test constants
const (
	// Increased from 50000 to 10*units.Lux to match larger genesis allocation
	// This ensures tests have enough funds for sequential transactions
	startBalance uint64 = 10 * units.Lux // 10 LUX = 10,000,000,000 nanoLux
	testTxFee    uint64 = 1000
)

var assetID = ids.GenerateTestID()

// envConfig configures the test environment
type envConfig struct {
	fork              upgrade.Config
	notLinearized     bool
	additionalFxs     []interface{}
	indexTransactions bool // Enable transaction indexing
}

// testEnv is the test environment
type testEnv struct {
	vm            *VM
	consensusCtx  *consensusctx.Context
	genesisBytes  []byte
	genesisTx     *txs.Tx
	testLock      *sync.Mutex
	txBuilder     *txstest.Builder
	sharedMemory  *atomic.Memory
}

// newGenesisBytesTest creates test genesis bytes
func newGenesisBytesTest(t *testing.T) []byte {
	require := require.New(t)

	// Format address properly as Bech32
	addr, err := address.FormatBech32(constants.GetHRP(constants.UnitTestID), keys[0].PublicKey().Address().Bytes())
	require.NoError(err)

	// Create a simple genesis with one asset (LUX)
	// Increased from 100 LUX to 1000 LUX to ensure sufficient funds for multiple transactions in tests
	genesisData := map[string]GenesisAssetDefinition{
		"LUX": {
			Name:         "Lux",
			Symbol:       "LUX",
			Denomination: 9,
			InitialState: AssetInitialState{
				FixedCap: []GenesisHolder{
					{
						Amount:  1000 * units.Lux,
						Address: addr,
					},
				},
			},
		},
	}

	genesis, err := NewGenesis(constants.UnitTestID, genesisData)
	require.NoError(err)

	genesisBytes, err := genesis.Bytes()
	require.NoError(err)

	return genesisBytes
}

// getCreateTxFromGenesisTest extracts a create asset tx from genesis
func getCreateTxFromGenesisTest(t *testing.T, genesisBytes []byte, assetAlias string) *txs.Tx {
	require := require.New(t)

	c, err := newGenesisCodec()
	require.NoError(err)

	genesis := &Genesis{}
	_, err = c.Unmarshal(genesisBytes, genesis)
	require.NoError(err)

	for _, asset := range genesis.Txs {
		if asset.Alias == assetAlias {
			tx := &txs.Tx{Unsigned: &asset.CreateAssetTx}
			require.NoError(tx.Initialize(c))
			return tx
		}
	}

	require.FailNow("asset not found in genesis", assetAlias)
	return nil
}

// mockValidatorState is a simple validator state for tests
type mockValidatorState struct {
	chainID ids.ID
}

func (m *mockValidatorState) GetChainID(ids.ID) (ids.ID, error) {
	return m.chainID, nil
}

func (m *mockValidatorState) GetNetworkID(ids.ID) (ids.ID, error) {
	return m.chainID, nil
}

func (m *mockValidatorState) GetSubnetID(chainID ids.ID) (ids.ID, error) {
	return ids.Empty, nil
}

func (m *mockValidatorState) GetMinimumHeight(context.Context) (uint64, error) {
	return 0, nil
}

func (m *mockValidatorState) GetCurrentHeight(context.Context) (uint64, error) {
	return 0, nil
}

func (m *mockValidatorState) GetValidatorSet(uint64, ids.ID) (map[ids.NodeID]uint64, error) {
	// Return a simple validator set with the test node
	return map[ids.NodeID]uint64{
		ids.GenerateTestNodeID(): 1000,
	}, nil
}

// testSharedMemory wraps atomic.SharedMemory to match VM's SharedMemory interface
type testSharedMemory struct {
	mem atomic.SharedMemory
}

func (t *testSharedMemory) Get(peerChainID ids.ID, keys [][]byte) ([][]byte, error) {
	return t.mem.Get(peerChainID, keys)
}

func (t *testSharedMemory) Apply(requests map[ids.ID]interface{}, _ ...interface{}) error {
	// Convert interface{} map to *atomic.Requests map
	atomicRequests := make(map[ids.ID]*atomic.Requests)
	for chainID, req := range requests {
		if atomicReq, ok := req.(*atomic.Requests); ok {
			atomicRequests[chainID] = atomicReq
		}
	}
	return t.mem.Apply(atomicRequests)
}

// setup creates a test environment
func setup(t testing.TB, config *envConfig) *testEnv {
	require := require.New(t)

	if config == nil {
		config = &envConfig{
			fork: upgradetest.GetConfig(upgradetest.Latest),
		}
	}

	chainID := ids.GenerateTestID()
	ctx := &consensusctx.Context{
		NetworkID:      constants.UnitTestID,
		ChainID:        chainID,
		XChainID:       ids.GenerateTestID(),
		CChainID:       ids.GenerateTestID(),
		NodeID:         ids.GenerateTestNodeID(),
		ValidatorState: &mockValidatorState{chainID: chainID},
	}

	baseDB := memdb.New()
	sharedMemory := atomic.NewMemory(memdb.New())

	vm := &VM{}
	genesisBytes := newGenesisBytesTest(t.(*testing.T))
	// Create shared memory wrapper that matches VM's interface
	atomicMem := sharedMemory.NewSharedMemory(ctx.ChainID)
	vm.SharedMemory = &testSharedMemory{mem: atomicMem}

	testLock := &sync.Mutex{}
	testLock.Lock()

	// Create a mock AppSender
	appSender := &noOpAppSender{}

	// ALWAYS include secp256k1fx first (required for genesis parsing)
	// Then add additional Fxs if provided, or default to nftfx and propertyfx
	fxs := []interface{}{
		&core.Fx{
			ID: secp256k1fx.ID,
			Fx: &secp256k1fx.Fx{},
		},
	}
	
	if len(config.additionalFxs) == 0 {
		// No additional Fxs specified - add default nftfx and propertyfx
		fxs = append(fxs,
			&core.Fx{
				ID: nftfx.ID,
				Fx: &nftfx.Fx{},
			},
			&core.Fx{
				ID: propertyfx.ID,
				Fx: &propertyfx.Fx{},
			},
		)
	} else {
		// Additional Fxs specified - append them after secp256k1fx
		fxs = append(fxs, config.additionalFxs...)
	}

	// Create config for VM with optional indexing
	vmConfig := DefaultConfig
	if config.indexTransactions {
		vmConfig.IndexTransactions = true
	}
	configBytes, err := json.Marshal(vmConfig)
	require.NoError(err)

	toEngine := make(chan interface{}, 1)
	require.NoError(vm.Initialize(
		context.Background(),
		ctx,
		baseDB,
		genesisBytes,
		nil,
		configBytes,
		toEngine,
		fxs,
		appSender,
	))

	// Get the genesis transaction
	genesisTx := getCreateTxFromGenesisTest(t.(*testing.T), genesisBytes, "LUX")

	// Create transaction builder with SharedMemory
	atomicMemForBuilder := sharedMemory.NewSharedMemory(ctx.ChainID)
	txBuilder := txstest.New(
		vm.parser.Codec(),
		context.Background(),
		&vm.Config,
		vm.feeAssetID,
		vm.state,
		atomicMemForBuilder,
	)

	// Set the context IDs from the consensus context
	txBuilder.SetContextIDs(ctx.NetworkID, ctx.ChainID)

	env := &testEnv{
		vm:            vm,
		consensusCtx:  ctx,
		genesisBytes:  genesisBytes,
		genesisTx:     genesisTx,
		testLock:      testLock,
		txBuilder:     txBuilder,
		sharedMemory:  sharedMemory,
	}

	// Register cleanup to prevent goroutine leaks
	// This ensures PushGossip and PullGossip goroutines are properly terminated
	t.Cleanup(func() {
		// Shutdown the VM to cancel onShutdownCtx and stop gossip goroutines
		_ = vm.Shutdown()
	})

	// Linearize the DAG to initialize the network
	// This simulates what happens during normal VM bootstrap
	if !config.notLinearized {
		// Use the genesis transaction ID as the stop vertex
		stopVertexID := genesisTx.ID()
		toEngineChan := make(chan core.Message, 1)
		require.NoError(vm.Linearize(context.Background(), stopVertexID, toEngineChan))

		// Mark the backend as bootstrapped so tests can issue transactions
		vm.txBackend.Bootstrapped = true
	}

	// Lock the VM so tests can unlock it when ready
	vm.Lock.Lock()

	return env
}

// issueAndAccept issues and accepts a transaction
func issueAndAccept(require *require.Assertions, vm *VM, tx *txs.Tx) {
	// Issue the transaction to the network
	require.NoError(vm.network.IssueTxFromRPC(tx))

	// Build a block containing the transaction
	blkIntf, err := vm.BuildBlock(context.Background())
	require.NoError(err)

	// Verify the block
	require.NoError(blkIntf.Verify(context.Background()))

	// Accept the block
	require.NoError(blkIntf.Accept(context.Background()))

	// Update the preferred block (normally done by consensus engine)
	require.NoError(vm.SetPreference(context.Background(), blkIntf.ID()))

	// Commit the versiondb so indexed data is visible
	require.NoError(vm.db.Commit())

	// Verify the block status is accepted
	require.Equal(uint8(choices.Accepted), uint8(blkIntf.Status()))
}



// newTx creates a simple test transaction
func newTx(tb testing.TB, genesisBytes []byte, chainID ids.ID, parser txs.Parser, assetName string) *txs.Tx {
	require := require.New(tb)

	createTx := getCreateTxFromGenesisTest(tb.(*testing.T), genesisBytes, assetName)
	// Genesis creates 1000 LUX for keys[0]
	// This tx spends the entire UTXO and creates a change output back to keys[0]
	// Must account for transaction fee (testTxFee = 1000 nanoLux)
	inputAmt := uint64(1000 * units.Lux)
	outputAmt := inputAmt - testTxFee // Deduct fee from output

	tx := &txs.Tx{Unsigned: &txs.BaseTx{
		BaseTx: lux.BaseTx{
			NetworkID:    constants.UnitTestID,
			BlockchainID: chainID,
			Ins: []*lux.TransferableInput{{
				UTXOID: lux.UTXOID{
					TxID:        createTx.ID(),
					OutputIndex: 0, // First output (fixed cap holder output)
				},
				Asset: lux.Asset{ID: createTx.ID()},
				In: &secp256k1fx.TransferInput{
					Amt: inputAmt, // Must match UTXO amount
					Input: secp256k1fx.Input{
						SigIndices: []uint32{0},
					},
				},
			}},
			Outs: []*lux.TransferableOutput{{
				Asset: lux.Asset{ID: createTx.ID()},
				Out: &secp256k1fx.TransferOutput{
					Amt: outputAmt, // Output amount after fee deduction
					OutputOwners: secp256k1fx.OutputOwners{
						Threshold: 1,
						Addrs:     []ids.ShortID{keys[0].PublicKey().Address()},
					},
				},
			}},
		},
	}}
	require.NoError(
		tx.SignSECP256K1Fx(parser.Codec(), [][]*secp256k1.PrivateKey{{keys[0]}}),
	)
	return tx
}

// noOpAppSender is a minimal implementation of warp.Sender for tests
type noOpAppSender struct{}

var _ warp.Sender = (*noOpAppSender)(nil)

func (n *noOpAppSender) SendRequest(ctx context.Context, nodeIDs set.Set[ids.NodeID], requestID uint32, requestBytes []byte) error {
	return nil
}

func (n *noOpAppSender) SendResponse(ctx context.Context, nodeID ids.NodeID, requestID uint32, responseBytes []byte) error {
	return nil
}

func (n *noOpAppSender) SendError(ctx context.Context, nodeID ids.NodeID, requestID uint32, errorCode int32, errorMessage string) error {
	return nil
}

func (n *noOpAppSender) SendGossip(ctx context.Context, config warp.SendConfig, gossipBytes []byte) error {
	return nil
}
