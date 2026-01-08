// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	consensustest "github.com/luxfi/consensus/test/helpers"
	consensusctx "github.com/luxfi/consensus/context"
	"github.com/luxfi/consensus/core/coremock"
	"github.com/luxfi/consensus/validator/uptime"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/database/memdb"
	"github.com/luxfi/database/prefixdb"
	"github.com/luxfi/database/versiondb"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/p2p"

	"github.com/luxfi/sdk/node/chains"
	"github.com/luxfi/sdk/node/chains/atomic"
	"github.com/luxfi/sdk/node/codec"
	"github.com/luxfi/sdk/node/codec/linearcodec"
	"github.com/luxfi/upgrade/upgradetest"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/utils/timer/mockable"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/sdk/node/vms/platformvm/config"
	"github.com/luxfi/sdk/node/vms/platformvm/fx"
	"github.com/luxfi/sdk/node/vms/platformvm/genesis/genesistest"
	"github.com/luxfi/sdk/node/vms/platformvm/metrics"
	"github.com/luxfi/sdk/node/vms/platformvm/network"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/state/statetest"
	"github.com/luxfi/sdk/node/vms/platformvm/status"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/platformvm/txs/mempool"
	"github.com/luxfi/sdk/node/vms/platformvm/txs/txstest"
	"github.com/luxfi/sdk/node/vms/platformvm/utxo"
	"github.com/luxfi/sdk/node/vms/platformvm/validators/validatorstest"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/node/wallet/chain/p/wallet"

	blockexecutor "github.com/luxfi/sdk/node/vms/platformvm/block/executor"
	"github.com/luxfi/sdk/node/vms/platformvm/testcontext"
	"github.com/luxfi/sdk/node/vms/platformvm/warp"
	txexecutor "github.com/luxfi/sdk/node/vms/platformvm/txs/executor"
	txmempool "github.com/luxfi/sdk/node/vms/txs/mempool"

	validators "github.com/luxfi/consensus/validator"
)

const (
	defaultMinStakingDuration = 24 * time.Hour
	defaultMaxStakingDuration = 365 * 24 * time.Hour
)

var testNet1 *txs.Tx

// mockValidatorState implements consensusctx.ValidatorState for testing
type mockValidatorState struct{}

func (m *mockValidatorState) GetChainID(netID ids.ID) (ids.ID, error) {
	// Return the chain ID for the given net ID
	return ids.Empty, nil
}

func (m *mockValidatorState) GetNetworkID(chainID ids.ID) (ids.ID, error) {
	// Return Primary Network ID for all chains
	return constants.PrimaryNetworkID, nil
}

func (m *mockValidatorState) GetSubnetID(chainID ids.ID) (ids.ID, error) {
	// Return Primary Network ID for all chains
	return constants.PrimaryNetworkID, nil
}

func (m *mockValidatorState) GetValidatorSet(height uint64, netID ids.ID) (map[ids.NodeID]uint64, error) {
	// Return an empty validator set for tests
	return make(map[ids.NodeID]uint64), nil
}

func (m *mockValidatorState) GetCurrentHeight(ctx context.Context) (uint64, error) {
	// Return a default height for tests
	return 100, nil
}

func (m *mockValidatorState) GetMinimumHeight(ctx context.Context) (uint64, error) {
	// Return a minimum height for tests
	return 0, nil
}

type mutableSharedMemory struct {
	atomic.SharedMemory
}

type environment struct {
	Builder
	blkManager blockexecutor.Manager
	mempool    txmempool.Mempool[*txs.Tx]
	network    *network.Network
	sender     *coremock.MockAppSender

	isBootstrapped *utils.Atomic[bool]
	config         *config.Internal
	clk            *mockable.Clock
	baseDB         *versiondb.Database
	ctx            *testcontext.Context
	msm            *mutableSharedMemory
	fx             fx.Fx
	state          state.State
	uptimes        uptime.Calculator
	utxosVerifier  utxo.Verifier
	backend        txexecutor.Backend
}

func newEnvironment(t *testing.T, f upgradetest.Fork) *environment { //nolint:unparam
	require := require.New(t)

	res := &environment{
		isBootstrapped: &utils.Atomic[bool]{},
		config:         defaultConfig(f),
		clk:            defaultClock(),
	}
	res.isBootstrapped.Set(true)

	res.baseDB = versiondb.New(memdb.New())
	atomicDB := prefixdb.New([]byte{1}, res.baseDB)
	m := atomic.NewMemory(atomicDB)

	// Create test context with Lock
	// Use PlatformChainID to match genesis transactions
	consensusCtx := consensustest.Context(t, constants.PlatformChainID)
	res.ctx = testcontext.New(context.Background())
	res.ctx.NetworkID = consensusCtx.NetworkID
	res.ctx.ChainID = consensusCtx.ChainID
	res.ctx.NodeID = consensusCtx.NodeID
	res.ctx.ChainID = consensusCtx.ChainID
	res.ctx.XAssetID = consensusCtx.XAssetID
	res.ctx.XAssetID = consensusCtx.XAssetID
	res.ctx.XChainID = consensusCtx.XChainID
	res.ctx.CChainID = consensusCtx.CChainID
	res.msm = &mutableSharedMemory{
		SharedMemory: m.NewSharedMemory(res.ctx.ChainID),
	}
	res.ctx.SharedMemory = res.msm

	// Create a mock ValidatorState that implements consensusctx.ValidatorState
	res.ctx.ValidatorState = &mockValidatorState{}

	res.ctx.Lock.Lock()
	defer res.ctx.Lock.Unlock()

	res.fx = defaultFx(t, res.clk, res.ctx.Log, res.isBootstrapped.Get())

	rewardsCalc := reward.NewCalculator(res.config.RewardConfig)
	// Convert testcontext.Context to consensusctx.Context for state
	stateConsensusCtx := &consensusctx.Context{
		NetworkID: res.ctx.NetworkID,
		ChainID:   res.ctx.ChainID,
		NodeID:    res.ctx.NodeID,
		XAssetID:  res.ctx.XAssetID,
		Log:       res.ctx.Log,
	}
	res.state = statetest.New(t, statetest.Config{
		DB:         res.baseDB,
		Genesis:    genesistest.NewBytes(t, genesistest.Config{}),
		Validators: res.config.Validators,
		Context:    stateConsensusCtx,
		Rewards:    rewardsCalc,
	})

	// Uptime calculator is set to NoOp in backend
	res.utxosVerifier = utxo.NewVerifier(res.clk, res.fx)

	genesisID := res.state.GetLastAccepted()
	// Convert testcontext.Context to consensusctx.Context
	backendConsensusCtx := &consensusctx.Context{
		NetworkID:      res.ctx.NetworkID,
		ChainID:        res.ctx.ChainID,
		NodeID:         res.ctx.NodeID,
		XAssetID:       res.ctx.XAssetID,
		Log:            res.ctx.Log,
		ValidatorState: res.ctx.ValidatorState,
		SharedMemory:   res.ctx.SharedMemory,
	}

	res.backend = txexecutor.Backend{
		Config:       res.config,
		Ctx:          backendConsensusCtx,
		Clk:          res.clk,
		Bootstrapped: res.isBootstrapped,
		Fx:           res.fx,
		FlowChecker:  res.utxosVerifier,
		Uptimes:      &uptime.NoOpCalculator{},
		Rewards:      rewardsCalc,
	}

	registerer := prometheus.NewRegistry()
	res.sender = &coremock.MockAppSender{
		SendGossipF: func(context.Context, p2p.SendConfig, []byte) error {
			return nil
		},
	}

	platformMetrics, err := metrics.New(registerer)
	require.NoError(err)

	res.mempool, err = mempool.New("mempool", registerer)
	require.NoError(err)

	res.blkManager = blockexecutor.NewManager(
		res.mempool,
		platformMetrics,
		res.state,
		&res.backend,
		validatorstest.Manager,
	)

	// Use validatorstest.Manager for validator state
	txVerifier := network.NewLockedTxVerifier(res.ctx.Lock, res.blkManager)

	// Create a mock warp signer if needed
	var warpSigner warp.Signer
	if res.ctx.WarpSigner != nil {
		if ws, ok := res.ctx.WarpSigner.(warp.Signer); ok {
			warpSigner = ws
		}
	}

	res.network, err = network.New(
		res.ctx.Log,
		res.ctx.NodeID,
		res.ctx.ChainID,
		validatorstest.Manager,
		txVerifier,
		res.mempool,
		res.backend.Config.PartialSyncPrimaryNetwork,
		res.sender,
		res.ctx.Lock,
		res.state,
		warpSigner,
		registerer,
		config.DefaultNetwork,
	)
	require.NoError(err)

	res.Builder = New(
		res.mempool,
		&res.backend,
		res.blkManager,
	)

	res.blkManager.SetPreference(genesisID)
	addNet(t, res)

	t.Cleanup(func() {
		// Note: We need to be careful about the cleanup order.
		// The lock should already be released before cleanup runs.
		// State and DB should be closed only after all operations complete.
		if res.state != nil {
			_ = res.state.Close()
		}
		if res.baseDB != nil {
			_ = res.baseDB.Close()
		}
	})

	return res
}

type walletConfig struct {
	keys   []*secp256k1.PrivateKey
	netIDs []ids.ID
}

func newWallet(t testing.TB, e *environment, c walletConfig) wallet.Wallet {
	if len(c.keys) == 0 {
		c.keys = genesistest.DefaultFundedKeys
	}
	// Convert testcontext.Context to consensusctx.Context for wallet
	walletCtx := &consensusctx.Context{
		NetworkID:    e.ctx.NetworkID,
		ChainID:      e.ctx.ChainID,
		NodeID:       e.ctx.NodeID,
		XAssetID:     e.ctx.XAssetID,
		SharedMemory: e.ctx.SharedMemory,
	}
	// Create a minimal Config for the wallet
	walletCfg := &config.Config{
		TxFee:                         units.MilliLux,
		CreateAssetTxFee:              units.MilliLux,
		CreateNetTxFee:             units.Lux,
		CreateBlockchainTxFee:         units.Lux,
		AddPrimaryNetworkValidatorFee: 0,
		AddPrimaryNetworkDelegatorFee: 0,
	}
	return txstest.NewWallet(
		t,
		walletCtx,
		walletCfg,
		e.state,
		secp256k1fx.NewKeychain(c.keys...),
		c.netIDs,
		nil, // validationIDs
		[]ids.ID{e.ctx.CChainID, e.ctx.XChainID},
	)
}

func addNet(t *testing.T, env *environment) {
	require := require.New(t)

	wallet := newWallet(t, env, walletConfig{
		keys: genesistest.DefaultFundedKeys[:1],
	})

	var err error
	testNet1, err = wallet.IssueCreateSubnetTx(
		&secp256k1fx.OutputOwners{
			Threshold: 2,
			Addrs: []ids.ShortID{
				genesistest.DefaultFundedKeys[0].Address(),
				genesistest.DefaultFundedKeys[1].Address(),
				genesistest.DefaultFundedKeys[2].Address(),
			},
		},
	)
	require.NoError(err)

	genesisID := env.state.GetLastAccepted()
	stateDiff, err := state.NewDiff(genesisID, env.blkManager)
	require.NoError(err)

	feeCalculator := state.PickFeeCalculator(env.config, stateDiff)
	_, _, _, err = txexecutor.StandardTx(
		&env.backend,
		feeCalculator,
		testNet1,
		stateDiff,
	)
	require.NoError(err)

	stateDiff.AddTx(testNet1, status.Committed)
	require.NoError(stateDiff.Apply(env.state))
	require.NoError(env.state.Commit())
}

func defaultConfig(f upgradetest.Fork) *config.Internal {
	upgrades := upgradetest.GetConfigWithUpgradeTime(f, time.Time{})
	// This package neglects fork ordering
	upgradetest.SetTimesTo(
		&upgrades,
		min(f, upgradetest.ApricotPhase5),
		genesistest.DefaultValidatorEndTime,
	)

	return &config.Internal{
		Chains:                 chains.TestManager,
		UptimeLockedCalculator: uptime.NewLockedCalculator(),
		Validators:             validators.NewManager(),
		MinValidatorStake:      5 * units.MilliLux,
		MaxValidatorStake:      500 * units.MilliLux,
		MinDelegatorStake:      1 * units.MilliLux,
		MinStakeDuration:       defaultMinStakingDuration,
		MaxStakeDuration:       defaultMaxStakingDuration,
		RewardConfig: reward.Config{
			MaxConsumptionRate: .12 * reward.PercentDenominator,
			MinConsumptionRate: .10 * reward.PercentDenominator,
			MintingPeriod:      365 * 24 * time.Hour,
			SupplyCap:          720 * units.MegaLux,
		},
		UpgradeConfig: upgrades,
	}
}

func defaultClock() *mockable.Clock {
	// set time after Banff fork (and before default nextStakerTime)
	clk := &mockable.Clock{}
	clk.Set(genesistest.DefaultValidatorStartTime)
	return clk
}

type fxVMInt struct {
	registry codec.Registry
	clk      *mockable.Clock
	log      log.Logger
}

func (fvi *fxVMInt) CodecRegistry() codec.Registry {
	return fvi.registry
}

func (fvi *fxVMInt) Clock() *mockable.Clock {
	return fvi.clk
}

func (fvi *fxVMInt) Logger() log.Logger {
	return fvi.log
}

func defaultFx(t *testing.T, clk *mockable.Clock, log log.Logger, isBootstrapped bool) fx.Fx {
	require := require.New(t)

	fxVMInt := &fxVMInt{
		registry: linearcodec.NewDefault(),
		clk:      clk,
		log:      log,
	}
	res := &secp256k1fx.Fx{}
	require.NoError(res.Initialize(fxVMInt))
	if isBootstrapped {
		require.NoError(res.Bootstrapped())
	}
	return res
}
