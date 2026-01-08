// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	metric "github.com/luxfi/metric"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	consensustest "github.com/luxfi/consensus/test/helpers"
	"github.com/luxfi/consensus/validator/uptime"
	validators "github.com/luxfi/consensus/validator"
	"github.com/luxfi/crypto/secp256k1"
	"github.com/luxfi/database/memdb"
	"github.com/luxfi/database/prefixdb"
	"github.com/luxfi/database/versiondb"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/sdk/node/chains"
	"github.com/luxfi/sdk/node/chains/atomic"
	"github.com/luxfi/sdk/node/codec"
	"github.com/luxfi/sdk/node/codec/linearcodec"
	"github.com/luxfi/upgrade/upgradetest"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/utils/timer/mockable"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/sdk/node/vms/platformvm/config"
	"github.com/luxfi/sdk/node/vms/platformvm/fx"
	"github.com/luxfi/sdk/node/vms/platformvm/genesis/genesistest"
	"github.com/luxfi/sdk/node/vms/platformvm/metrics"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/state/statetest"
	"github.com/luxfi/sdk/node/vms/platformvm/status"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/platformvm/txs/executor"
	"github.com/luxfi/sdk/node/vms/platformvm/txs/mempool"

	"github.com/luxfi/sdk/node/vms/platformvm/txs/txstest"
	"github.com/luxfi/sdk/node/vms/platformvm/utxo"
	"github.com/luxfi/sdk/node/vms/platformvm/validators/validatorstest"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/node/wallet/chain/p/wallet"

	txmempool "github.com/luxfi/sdk/node/vms/txs/mempool"
)

const (
	pending stakerStatus = iota
	current

	defaultMinStakingDuration = 24 * time.Hour
	defaultMaxStakingDuration = 365 * 24 * time.Hour

	defaultTxFee = 100 * units.NanoLux
)

var testNet1 *txs.Tx

type stakerStatus uint

type staker struct {
	nodeID             ids.NodeID
	rewardAddress      ids.ShortID
	startTime, endTime time.Time
}

type test struct {
	description           string
	stakers               []staker
	subnetStakers         []staker
	advanceTimeTo         []time.Time
	expectedStakers       map[ids.NodeID]stakerStatus
	expectedNetStakers map[ids.NodeID]stakerStatus
}

// testContext provides a mock context for testing
type testContext struct {
	context.Context // embed stdlib context
	NetworkID    uint32
	ChainID      ids.ID
	NodeID       ids.NodeID
	XChainID     ids.ID
	CChainID     ids.ID
	XAssetID     ids.ID
	Log          log.Logger
	Lock         *sync.RWMutex
	SharedMemory atomic.SharedMemory
}

type environment struct {
	blkManager Manager
	mempool    txmempool.Mempool[*txs.Tx]

	isBootstrapped *utils.Atomic[bool]
	config         *config.Internal
	clk            *mockable.Clock
	baseDB         *versiondb.Database
	ctx            *testContext
	fx             fx.Fx
	state          state.State
	mockedState    *state.MockState
	uptimes        uptime.Calculator
	utxosVerifier  utxo.Verifier
	backend        *executor.Backend
}

func newEnvironment(t *testing.T, ctrl *gomock.Controller, f upgradetest.Fork) *environment {
	res := &environment{
		isBootstrapped: &utils.Atomic[bool]{},
		config:         defaultConfig(f),
		clk:            defaultClock(),
	}
	res.isBootstrapped.Set(true)

	res.baseDB = versiondb.New(memdb.New())
	atomicDB := prefixdb.New([]byte{1}, res.baseDB)
	m := atomic.NewMemory(atomicDB)

	// Create consensus context from consensustest
	consensusCtx := consensustest.Context(t, consensustest.PChainID)
	
	// Build our testContext from the consensus context
	res.ctx = &testContext{
		Context:      context.Background(),
		NetworkID:    consensusCtx.NetworkID,
		ChainID:      consensusCtx.ChainID,
		NodeID:       consensusCtx.NodeID,
		XChainID:     consensusCtx.XChainID,
		CChainID:     consensusCtx.CChainID,
		XAssetID:     consensusCtx.XAssetID,
		Log:          consensusCtx.Log.(log.Logger),
		Lock:         &consensusCtx.Lock,
		SharedMemory: m.NewSharedMemory(consensusCtx.ChainID),
	}

	res.fx = defaultFx(res.clk, res.ctx.Log, res.isBootstrapped.Get())

	rewardsCalc := reward.NewCalculator(res.config.RewardConfig)

	// Create a node mockable clock for utxo handler
	nodeClock := &mockable.Clock{}
	nodeClock.Set(genesistest.DefaultValidatorStartTime)

	if ctrl == nil {
		res.state = statetest.New(t, statetest.Config{
			DB:         res.baseDB,
			Genesis:    genesistest.NewBytes(t, genesistest.Config{}),
			Validators: res.config.Validators,
			Context:    consensusCtx,
			Rewards:    rewardsCalc,
		})

		res.uptimes = &uptime.NoOpCalculator{}
		res.utxosVerifier = utxo.NewVerifier(res.clk, res.fx)
	} else {
		res.mockedState = state.NewMockState(ctrl)
		res.uptimes = &uptime.NoOpCalculator{}
		res.utxosVerifier = utxo.NewVerifier(res.clk, res.fx)

		// setup expectations strictly needed for environment creation
		res.mockedState.EXPECT().GetLastAccepted().Return(ids.GenerateTestID()).Times(1)
	}

	res.backend = &executor.Backend{
		Config:       res.config,
		Ctx:          consensusCtx,
		Clk:          res.clk,
		Bootstrapped: res.isBootstrapped,
		Fx:           res.fx,
		FlowChecker:  res.utxosVerifier,
		Uptimes:      &uptime.NoOpCalculator{},
		Rewards:      rewardsCalc,

	}

	registerer := metric.NewRegistry()

	platformMetrics := metrics.Noop

	var err error
	res.mempool, err = mempool.New("mempool", registerer)
	if err != nil {
		panic(fmt.Errorf("failed to create mempool: %w", err))
	}

	if ctrl == nil {
		res.blkManager = NewManager(
			res.mempool,
			platformMetrics,
			res.state,
			res.backend,
			validatorstest.Manager,
		)
		addNet(t, res)
	} else {
		res.blkManager = NewManager(
			res.mempool,
			platformMetrics,
			res.mockedState,
			res.backend,
			validatorstest.Manager,
		)
		// we do not add any subnet to state, since we can mock
		// whatever we need
	}

	t.Cleanup(func() {
		res.ctx.Lock.Lock()
		defer res.ctx.Lock.Unlock()

		if res.mockedState != nil {
			// state is mocked, nothing to do here
			return
		}

		require := require.New(t)

		// NoOpCalculator doesn't track validators, so no cleanup needed

		if res.state != nil {
			require.NoError(res.state.Close())
		}

		require.NoError(res.baseDB.Close())
	})

	return res
}

type walletConfig struct {
	keys      []*secp256k1.PrivateKey
	subnetIDs []ids.ID
}

func newWallet(t testing.TB, e *environment, c walletConfig) wallet.Wallet {
	if len(c.keys) == 0 {
		c.keys = genesistest.DefaultFundedKeys
	}
	
	// Get consensus context
	consensusCtx := consensustest.Context(t, consensustest.PChainID)
	


	
	return txstest.NewWallet(
		t,
		consensusCtx,
		&config.Config{
			TrackedChains:            e.config.TrackedChains,
			SybilProtectionEnabled: e.config.SybilProtectionEnabled,
			Chains:                 e.config.Chains,
		},
		e.state,
		secp256k1fx.NewKeychain(c.keys...),
		c.subnetIDs,
		nil, // validationIDs
		[]ids.ID{e.ctx.CChainID, e.ctx.XChainID},
	)
}

func addNet(t testing.TB, env *environment) {
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
	_, _, _, err = executor.StandardTx(
		env.backend,
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
		TrackedChains:            set.Of(constants.PrimaryNetworkID),
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

func defaultFx(clk *mockable.Clock, log log.Logger, isBootstrapped bool) fx.Fx {
	fxVMInt := &fxVMInt{
		registry: linearcodec.NewDefault(),
		clk:      clk,
		log:      log,
	}
	res := &secp256k1fx.Fx{}
	if err := res.Initialize(fxVMInt); err != nil {
		panic(err)
	}
	if isBootstrapped {
		if err := res.Bootstrapped(); err != nil {
			panic(err)
		}
	}
	return res
}

func addPendingValidator(
	t testing.TB,
	env *environment,
	startTime time.Time,
	endTime time.Time,
	nodeID ids.NodeID,
	rewardAddress ids.ShortID,
	keys []*secp256k1.PrivateKey,
) *txs.Tx {
	require := require.New(t)

	wallet := newWallet(t, env, walletConfig{
		keys: keys,
	})

	addValidatorTx, err := wallet.IssueAddValidatorTx(
		&txs.Validator{
			NodeID: nodeID,
			Start:  uint64(startTime.Unix()),
			End:    uint64(endTime.Unix()),
			Wght:   env.config.MinValidatorStake,
		},
		&secp256k1fx.OutputOwners{
			Threshold: 1,
			Addrs:     []ids.ShortID{rewardAddress},
		},
		reward.PercentDenominator,
	)
	require.NoError(err)

	staker, err := state.NewPendingStaker(
		addValidatorTx.ID(),
		addValidatorTx.Unsigned.(*txs.AddValidatorTx),
	)
	require.NoError(err)

	require.NoError(env.state.PutPendingValidator(staker))
	env.state.AddTx(addValidatorTx, status.Committed)
	env.state.SetHeight(1)
	require.NoError(env.state.Commit())
	return addValidatorTx
}
