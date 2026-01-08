// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	consensustest "github.com/luxfi/consensus/test/helpers"
	consensuscontext "github.com/luxfi/consensus/context"
	"github.com/luxfi/database"
	"github.com/luxfi/database/memdb"
	"github.com/luxfi/database/prefixdb"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/metric"
	"github.com/luxfi/math/set"
	"github.com/luxfi/sdk/node/chains/atomic"
	"github.com/luxfi/sdk/node/genesis/builder"
	"github.com/luxfi/upgrade"
	"github.com/luxfi/upgrade/upgradetest"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/const"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/crypto/bls/signer/localsigner"
	"github.com/luxfi/sdk/utils/iterator"
	"github.com/luxfi/sdk/utils/timer/mockable"
	"github.com/luxfi/sdk/utils/units"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/components/gas"
	"github.com/luxfi/sdk/node/vms/components/verify"
	"github.com/luxfi/sdk/node/vms/platformvm/block"
	"github.com/luxfi/sdk/node/vms/platformvm/config"
	"github.com/luxfi/sdk/node/vms/platformvm/genesis/genesistest"
	"github.com/luxfi/sdk/node/vms/platformvm/metrics"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/state/statetest"
	"github.com/luxfi/sdk/node/vms/platformvm/status"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/platformvm/txs/executor"
	"github.com/luxfi/sdk/node/vms/platformvm/txs/mempool"
	"github.com/luxfi/sdk/node/vms/platformvm/txs/txstest"
	"github.com/luxfi/sdk/node/vms/platformvm/utxo"
	"github.com/luxfi/sdk/node/vms/secp256k1fx"

	txfee "github.com/luxfi/sdk/node/vms/platformvm/txs/fee"
	validatorfee "github.com/luxfi/sdk/node/vms/platformvm/validators/fee"
)

type testVerifierConfig struct {
	DB                 database.Database
	Upgrades           upgrade.Config
	Context            *consensuscontext.Context
	ValidatorFeeConfig validatorfee.Config
}

func newTestVerifier(t testing.TB, c testVerifierConfig) *verifier {
	require := require.New(t)

	if c.DB == nil {
		c.DB = memdb.New()
	}
	if c.Upgrades == (upgrade.Config{}) {
		c.Upgrades = upgradetest.GetConfig(upgradetest.Latest)
	}
	if c.Context == nil {
		c.Context = consensustest.Context(t, constants.PlatformChainID)
	}
	if c.ValidatorFeeConfig == (validatorfee.Config{}) {
		c.ValidatorFeeConfig = builder.LocalValidatorFeeConfig
	}

	mempool, err := mempool.New("", metric.NewRegistry())
	require.NoError(err)

	var (
		state = statetest.New(t, statetest.Config{
			DB:       c.DB,
			Upgrades: c.Upgrades,
			Context:  c.Context,
		})
		clock = &mockable.Clock{}
		fx    = &secp256k1fx.Fx{}
	)
	require.NoError(fx.InitializeVM(&secp256k1fx.TestVM{
		Clk: *clock,
		Log: log.NoLog{},
	}))

	return &verifier{
		backend: &backend{
			Mempool:      mempool,
			lastAccepted: state.GetLastAccepted(),
			blkIDToState: make(map[ids.ID]*blockState),
			state:        state,
			ctx:          c.Context,
		},
		txExecutorBackend: &executor.Backend{
			Config: &config.Internal{
				DynamicFeeConfig:       builder.LocalDynamicFeeConfig,
				ValidatorFeeConfig:     c.ValidatorFeeConfig,
				SybilProtectionEnabled: true,
				UpgradeConfig:          c.Upgrades,
			},
			Ctx: c.Context,
			Clk: clock,
			Fx:  fx,
			FlowChecker: utxo.NewVerifier(
				clock,
				fx,
			),
			Bootstrapped: utils.NewAtomic(true),
		},
	}
}

func TestVerifierVisitProposalBlock(t *testing.T) {
	var (
		require  = require.New(t)
		verifier = newTestVerifier(t, testVerifierConfig{
			Upgrades: upgradetest.GetConfig(upgradetest.ApricotPhasePost6),
		})
		initialTimestamp = verifier.state.GetTimestamp()
		newTimestamp     = initialTimestamp.Add(time.Second)
		proposalTx       = &txs.Tx{
			Unsigned: &txs.AdvanceTimeTx{
				Time: uint64(newTimestamp.Unix()),
			},
		}
	)
	require.NoError(proposalTx.Initialize(txs.Codec))

	// Build the block that will be executed on top of the last accepted block.
	lastAcceptedID := verifier.state.GetLastAccepted()
	lastAccepted, err := verifier.state.GetStatelessBlock(lastAcceptedID)
	require.NoError(err)

	proposalBlock, err := block.NewApricotProposalBlock(
		lastAcceptedID,
		lastAccepted.Height()+1,
		proposalTx,
	)
	require.NoError(err)

	// Execute the block.
	require.NoError(proposalBlock.Visit(verifier))

	// Verify that the block's execution was recorded as expected.
	blkID := proposalBlock.ID()
	require.Contains(verifier.blkIDToState, blkID)
	executedBlockState := verifier.blkIDToState[blkID]

	txID := proposalTx.ID()

	onCommit := executedBlockState.onCommitState
	require.NotNil(onCommit)
	acceptedTx, acceptedStatus, err := onCommit.GetTx(txID)
	require.NoError(err)
	require.Equal(proposalTx, acceptedTx)
	require.Equal(status.Committed, acceptedStatus)

	onAbort := executedBlockState.onAbortState
	require.NotNil(onAbort)
	acceptedTx, acceptedStatus, err = onAbort.GetTx(txID)
	require.NoError(err)
	require.Equal(proposalTx, acceptedTx)
	require.Equal(status.Aborted, acceptedStatus)

	require.Equal(
		&blockState{
			proposalBlockState: proposalBlockState{
				onCommitState: onCommit,
				onAbortState:  onAbort,
			},
			statelessBlock: proposalBlock,

			timestamp:       initialTimestamp,
			verifiedHeights: set.Of[uint64](0),
			metrics: metrics.Block{
				Block:          proposalBlock,
				GasPrice:       verifier.txExecutorBackend.Config.DynamicFeeConfig.MinPrice,
				ValidatorPrice: verifier.txExecutorBackend.Config.ValidatorFeeConfig.MinPrice,
			},
		},
		executedBlockState,
	)
}

func TestVerifierVisitAtomicBlock(t *testing.T) {
	var (
		require  = require.New(t)
		verifier = newTestVerifier(t, testVerifierConfig{
			Upgrades: upgradetest.GetConfig(upgradetest.ApricotPhase4),
		})
		wallet = txstest.NewWallet(
			t,
			verifier.ctx,
			&config.Config{
				TrackedChains:            verifier.txExecutorBackend.Config.TrackedChains,
				SybilProtectionEnabled: verifier.txExecutorBackend.Config.SybilProtectionEnabled,
				Chains:                 verifier.txExecutorBackend.Config.Chains,
			},
			verifier.state,
			secp256k1fx.NewKeychain(genesistest.DefaultFundedKeys[0]),
			nil, // subnetIDs
			nil, // validationIDs
			nil, // chainIDs
		)
		exportedOutput = &lux.TransferableOutput{
			Asset: lux.Asset{ID: verifier.ctx.XAssetID},
			Out: &secp256k1fx.TransferOutput{
				Amt:          units.NanoLux,
				OutputOwners: secp256k1fx.OutputOwners{},
			},
		}
		initialTimestamp = verifier.state.GetTimestamp()
	)

	// Build the transaction that will be executed.
	atomicTx, err := wallet.IssueExportTx(
		verifier.ctx.XChainID,
		[]*lux.TransferableOutput{
			exportedOutput,
		},
	)
	require.NoError(err)

	// Build the block that will be executed on top of the last accepted block.
	lastAcceptedID := verifier.state.GetLastAccepted()
	lastAccepted, err := verifier.state.GetStatelessBlock(lastAcceptedID)
	require.NoError(err)

	atomicBlock, err := block.NewApricotAtomicBlock(
		lastAcceptedID,
		lastAccepted.Height()+1,
		atomicTx,
	)
	require.NoError(err)

	// Execute the block.
	require.NoError(atomicBlock.Visit(verifier))

	// Verify that the block's execution was recorded as expected.
	blkID := atomicBlock.ID()
	require.Contains(verifier.blkIDToState, blkID)
	atomicBlockState := verifier.blkIDToState[blkID]
	onAccept := atomicBlockState.onAcceptState
	require.NotNil(onAccept)

	txID := atomicTx.ID()
	acceptedTx, acceptedStatus, err := onAccept.GetTx(txID)
	require.NoError(err)
	require.Equal(atomicTx, acceptedTx)
	require.Equal(status.Committed, acceptedStatus)

	exportedUTXO := &lux.UTXO{
		UTXOID: lux.UTXOID{
			TxID:        txID,
			OutputIndex: uint32(len(atomicTx.UTXOs())),
		},
		Asset: exportedOutput.Asset,
		Out:   exportedOutput.Out,
	}
	exportedUTXOID := exportedUTXO.InputID()
	exportedUTXOBytes, err := txs.Codec.Marshal(txs.CodecVersion, exportedUTXO)
	require.NoError(err)

	require.Equal(
		&blockState{
			statelessBlock: atomicBlock,

			onAcceptState: onAccept,

			timestamp: initialTimestamp,
			atomicRequests: map[ids.ID]*atomic.Requests{
				verifier.ctx.XChainID: {
					PutRequests: []*atomic.Element{
						{
							Key:    exportedUTXOID[:],
							Value:  exportedUTXOBytes,
							Traits: [][]byte{},
						},
					},
				},
			},
			verifiedHeights: set.Of[uint64](0),
			metrics: metrics.Block{
				Block:          atomicBlock,
				GasPrice:       verifier.txExecutorBackend.Config.DynamicFeeConfig.MinPrice,
				ValidatorPrice: verifier.txExecutorBackend.Config.ValidatorFeeConfig.MinPrice,
			},
		},
		atomicBlockState,
	)
}

func TestVerifierVisitStandardBlock(t *testing.T) {
	require := require.New(t)

	var (
		ctx = consensustest.Context(t, constants.PlatformChainID)

		baseDB  = memdb.New()
		stateDB = prefixdb.New([]byte{0}, baseDB)
		amDB    = prefixdb.New([]byte{1}, baseDB)

		am       = atomic.NewMemory(amDB)
		sm       = am.NewSharedMemory(ctx.ChainID)
		xChainSM = am.NewSharedMemory(ctx.XChainID)

		owner = secp256k1fx.OutputOwners{
			Threshold: 1,
			Addrs:     []ids.ShortID{genesistest.DefaultFundedKeys[0].Address()},
		}
		utxo = &lux.UTXO{
			UTXOID: lux.UTXOID{
				TxID:        ids.GenerateTestID(),
				OutputIndex: 1,
			},
			Asset: lux.Asset{ID: ctx.XAssetID},
			Out: &secp256k1fx.TransferOutput{
				Amt:          units.Lux,
				OutputOwners: owner,
			},
		}
	)

	inputID := utxo.InputID()
	utxoBytes, err := txs.Codec.Marshal(txs.CodecVersion, utxo)
	require.NoError(err)

	require.NoError(xChainSM.Apply(map[ids.ID]*atomic.Requests{
		ctx.ChainID: {
			PutRequests: []*atomic.Element{
				{
					Key:   inputID[:],
					Value: utxoBytes,
					Traits: [][]byte{
						genesistest.DefaultFundedKeys[0].Address().Bytes(),
					},
				},
			},
		},
	}))

	ctx.SharedMemory = sm

	var (
		verifier = newTestVerifier(t, testVerifierConfig{
			DB:       stateDB,
			Upgrades: upgradetest.GetConfig(upgradetest.ApricotPhase5),
			Context:  ctx,
		})
		wallet = txstest.NewWallet(
			t,
			verifier.ctx,
			&config.Config{
				TrackedChains:            verifier.txExecutorBackend.Config.TrackedChains,
				SybilProtectionEnabled: verifier.txExecutorBackend.Config.SybilProtectionEnabled,
				Chains:                 verifier.txExecutorBackend.Config.Chains,
			},
			verifier.state,
			secp256k1fx.NewKeychain(genesistest.DefaultFundedKeys[0]),
			nil,                    // subnetIDs
			nil,                    // validationIDs
			[]ids.ID{ctx.XChainID}, // Read the UTXO to import
		)
		initialTimestamp = verifier.state.GetTimestamp()
	)

	// Build the transaction that will be executed.
	tx, err := wallet.IssueImportTx(
		verifier.ctx.XChainID,
		&owner,
	)
	require.NoError(err)

	// Verify that the transaction consumes the imported UTXO.
	// Since the imported amount is sufficient to cover fees, only the imported UTXO is consumed.
	require.Len(tx.InputIDs(), 1)

	// Build the block that will be executed on top of the last accepted block.
	lastAcceptedID := verifier.state.GetLastAccepted()
	lastAccepted, err := verifier.state.GetStatelessBlock(lastAcceptedID)
	require.NoError(err)

	firstBlock, err := block.NewApricotStandardBlock(
		lastAcceptedID,
		lastAccepted.Height()+1,
		[]*txs.Tx{tx},
	)
	require.NoError(err)

	// Execute the block.
	require.NoError(firstBlock.Visit(verifier))

	// Verify that the block's execution was recorded as expected.
	firstBlockID := firstBlock.ID()
	{
		require.Contains(verifier.blkIDToState, firstBlockID)
		atomicBlockState := verifier.blkIDToState[firstBlockID]
		onAccept := atomicBlockState.onAcceptState
		require.NotNil(onAccept)

		txID := tx.ID()
		acceptedTx, acceptedStatus, err := onAccept.GetTx(txID)
		require.NoError(err)
		require.Equal(tx, acceptedTx)
		require.Equal(status.Committed, acceptedStatus)

		// Compare fields individually to avoid spew panic on unexported set fields
		require.Equal(firstBlock, atomicBlockState.statelessBlock)
		require.Equal(onAccept, atomicBlockState.onAcceptState)
		// Compare inputs: atomicBlockState.inputs only tracks atomic/imported inputs (not BaseTx inputs)
		// so we compare against InputUTXOs() (atomic inputs only), not InputIDs() (all inputs)
		importTx := tx.Unsigned.(*txs.ImportTx)
		expectedInputs := importTx.InputUTXOs()
		require.Equal(expectedInputs.Len(), atomicBlockState.inputs.Len(), "inputs should have same length")
		for id := range expectedInputs {
			require.True(atomicBlockState.inputs.Contains(id), "inputs should contain %s", id)
		}
		require.Equal(initialTimestamp, atomicBlockState.timestamp)
		require.Equal(map[ids.ID]*atomic.Requests{
			verifier.ctx.XChainID: {
				RemoveRequests: [][]byte{
					inputID[:],
				},
			},
		}, atomicBlockState.atomicRequests)
		// Compare verifiedHeights: expect same size and all expected heights present
		expectedHeights := set.Of[uint64](0)
		require.Equal(expectedHeights.Len(), atomicBlockState.verifiedHeights.Len(), "verifiedHeights should have same length")
		for h := range expectedHeights {
			require.True(atomicBlockState.verifiedHeights.Contains(h), "verifiedHeights should contain %d", h)
		}
		require.Equal(firstBlock, atomicBlockState.metrics.Block)
		require.Equal(verifier.txExecutorBackend.Config.DynamicFeeConfig.MinPrice, atomicBlockState.metrics.GasPrice)
		require.Equal(verifier.txExecutorBackend.Config.ValidatorFeeConfig.MinPrice, atomicBlockState.metrics.ValidatorPrice)
	}

	// Verify that the import transaction can not be replayed.
	// The replay fails because the BaseTx UTXO was consumed in the first block's
	// onAcceptState diff, so when the second block tries to execute the same
	// transaction, the UTXO lookup fails with "not found".
	{
		secondBlock, err := block.NewApricotStandardBlock(
			firstBlockID,
			firstBlock.Height()+1,
			[]*txs.Tx{tx}, // Replay the prior transaction
		)
		require.NoError(err)

		err = secondBlock.Visit(verifier)
		// Replaying a transaction should fail due to conflict detection
		require.Error(err)

		// Verify that the block's execution was not recorded.
		require.NotContains(verifier.blkIDToState, secondBlock.ID())
	}
}

func TestVerifierVisitCommitBlock(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	// Create mocked dependencies.
	s := state.NewMockState(ctrl)
	mempool, err := mempool.New("", metric.NewRegistry())
	require.NoError(err)
	parentID := ids.GenerateTestID()
	parentStatelessBlk := block.NewMockBlock(ctrl)
	parentOnDecisionState := state.NewMockDiff(ctrl)
	parentOnCommitState := state.NewMockDiff(ctrl)
	parentOnAbortState := state.NewMockDiff(ctrl)

	backend := &backend{
		blkIDToState: map[ids.ID]*blockState{
			parentID: {
				statelessBlock: parentStatelessBlk,
				proposalBlockState: proposalBlockState{
					onDecisionState: parentOnDecisionState,
					onCommitState:   parentOnCommitState,
					onAbortState:    parentOnAbortState,
				},
			},
		},
		Mempool: mempool,
		state:   s,
		ctx: &consensuscontext.Context{
			NetworkID: constants.UnitTestID,
			ChainID:   constants.PlatformChainID,
		},
	}
	manager := &manager{
		txExecutorBackend: &executor.Backend{
			Config: &config.Internal{
				UpgradeConfig: upgradetest.GetConfig(upgradetest.ApricotPhasePost6),
			},
			Clk:          &mockable.Clock{},
			Bootstrapped: utils.NewAtomic(true),
		},
		backend: backend,
	}

	apricotBlk, err := block.NewApricotCommitBlock(
		parentID,
		2,
	)
	require.NoError(err)

	// Set expectations for dependencies.
	timestamp := time.Now()
	gomock.InOrder(
		parentStatelessBlk.EXPECT().Height().Return(uint64(1)).Times(1),
		parentOnCommitState.EXPECT().GetTimestamp().Return(timestamp).Times(1),
		// Allow metrics to be calculated.
		parentOnCommitState.EXPECT().GetFeeState().Return(gas.State{}).Times(1),
		parentOnCommitState.EXPECT().GetL1ValidatorExcess().Return(gas.Gas(0)).Times(1),
		parentOnCommitState.EXPECT().NumActiveL1Validators().Return(0).Times(1),
		parentOnCommitState.EXPECT().GetAccruedFees().Return(uint64(0)).Times(1),
	)

	// Verify the block.
	blk := manager.NewBlock(apricotBlk)
	require.NoError(blk.Verify(context.Background()))

	// Assert expected state.
	require.Contains(manager.backend.blkIDToState, apricotBlk.ID())
	gotBlkState := manager.backend.blkIDToState[apricotBlk.ID()]
	require.Equal(parentOnAbortState, gotBlkState.onAcceptState)
	require.Equal(timestamp, gotBlkState.timestamp)

	// Visiting again should return nil without using dependencies.
	require.NoError(blk.Verify(context.Background()))
}

func TestVerifierVisitAbortBlock(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	// Create mocked dependencies.
	s := state.NewMockState(ctrl)
	mempool, err := mempool.New("", metric.NewRegistry())
	require.NoError(err)
	parentID := ids.GenerateTestID()
	parentStatelessBlk := block.NewMockBlock(ctrl)
	parentOnDecisionState := state.NewMockDiff(ctrl)
	parentOnCommitState := state.NewMockDiff(ctrl)
	parentOnAbortState := state.NewMockDiff(ctrl)

	backend := &backend{
		blkIDToState: map[ids.ID]*blockState{
			parentID: {
				statelessBlock: parentStatelessBlk,
				proposalBlockState: proposalBlockState{
					onDecisionState: parentOnDecisionState,
					onCommitState:   parentOnCommitState,
					onAbortState:    parentOnAbortState,
				},
			},
		},
		Mempool: mempool,
		state:   s,
		ctx: &consensuscontext.Context{
			NetworkID: constants.UnitTestID,
			ChainID:   constants.PlatformChainID,
		},
	}
	manager := &manager{
		txExecutorBackend: &executor.Backend{
			Config: &config.Internal{
				UpgradeConfig: upgradetest.GetConfig(upgradetest.ApricotPhasePost6),
			},
			Clk:          &mockable.Clock{},
			Bootstrapped: utils.NewAtomic(true),
		},
		backend: backend,
	}

	apricotBlk, err := block.NewApricotAbortBlock(
		parentID,
		2,
	)
	require.NoError(err)

	// Set expectations for dependencies.
	timestamp := time.Now()
	gomock.InOrder(
		parentStatelessBlk.EXPECT().Height().Return(uint64(1)).Times(1),
		parentOnAbortState.EXPECT().GetTimestamp().Return(timestamp).Times(1),
		// Allow metrics to be calculated.
		parentOnAbortState.EXPECT().GetFeeState().Return(gas.State{}).Times(1),
		parentOnAbortState.EXPECT().GetL1ValidatorExcess().Return(gas.Gas(0)).Times(1),
		parentOnAbortState.EXPECT().NumActiveL1Validators().Return(0).Times(1),
		parentOnAbortState.EXPECT().GetAccruedFees().Return(uint64(0)).Times(1),
	)

	// Verify the block.
	blk := manager.NewBlock(apricotBlk)
	require.NoError(blk.Verify(context.Background()))

	// Assert expected state.
	require.Contains(manager.backend.blkIDToState, apricotBlk.ID())
	gotBlkState := manager.backend.blkIDToState[apricotBlk.ID()]
	require.Equal(parentOnAbortState, gotBlkState.onAcceptState)
	require.Equal(timestamp, gotBlkState.timestamp)

	// Visiting again should return nil without using dependencies.
	require.NoError(blk.Verify(context.Background()))
}

// Assert that a block with an unverified parent fails verification.
func TestVerifyUnverifiedParent(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	// Create mocked dependencies.
	s := state.NewMockState(ctrl)
	mempool, err := mempool.New("", metric.NewRegistry())
	require.NoError(err)
	parentID := ids.GenerateTestID()

	backend := &backend{
		blkIDToState: map[ids.ID]*blockState{},
		Mempool:      mempool,
		state:        s,
		ctx: &consensuscontext.Context{
			NetworkID: constants.UnitTestID,
			ChainID:   constants.PlatformChainID,
		},
	}
	verifier := &verifier{
		txExecutorBackend: &executor.Backend{
			Config: &config.Internal{
				UpgradeConfig: upgradetest.GetConfig(upgradetest.ApricotPhasePost6),
			},
			Clk: &mockable.Clock{},
		},
		backend: backend,
	}

	blk, err := block.NewApricotAbortBlock(parentID /*not in memory or persisted state*/, 2 /*height*/)
	require.NoError(err)

	// Set expectations for dependencies.
	s.EXPECT().GetTimestamp().Return(time.Now()).Times(1)
	s.EXPECT().GetStatelessBlock(parentID).Return(nil, database.ErrNotFound).Times(1)

	// Verify the block.
	err = blk.Visit(verifier)
	require.ErrorIs(err, database.ErrNotFound)
}

func TestBanffAbortBlockTimestampChecks(t *testing.T) {
	ctrl := gomock.NewController(t)

	now := genesistest.DefaultValidatorStartTime.Add(time.Hour)

	tests := []struct {
		description string
		parentTime  time.Time
		childTime   time.Time
		result      error
	}{
		{
			description: "abort block timestamp matching parent's one",
			parentTime:  now,
			childTime:   now,
			result:      nil,
		},
		{
			description: "abort block timestamp before parent's one",
			childTime:   now.Add(-1 * time.Second),
			parentTime:  now,
			result:      errOptionBlockTimestampNotMatchingParent,
		},
		{
			description: "abort block timestamp after parent's one",
			parentTime:  now,
			childTime:   now.Add(time.Second),
			result:      errOptionBlockTimestampNotMatchingParent,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			require := require.New(t)

			// Create mocked dependencies.
			s := state.NewMockState(ctrl)
			mempool, err := mempool.New("", metric.NewRegistry())
			require.NoError(err)
			parentID := ids.GenerateTestID()
			parentStatelessBlk := block.NewMockBlock(ctrl)
			parentHeight := uint64(1)

			backend := &backend{
				blkIDToState: make(map[ids.ID]*blockState),
				Mempool:      mempool,
				state:        s,
				ctx: &consensuscontext.Context{
			NetworkID: constants.UnitTestID,
			ChainID:   constants.PlatformChainID,
		},
			}
			verifier := &verifier{
				txExecutorBackend: &executor.Backend{
					Config: &config.Internal{
						UpgradeConfig: upgradetest.GetConfig(upgradetest.Banff),
					},
					Clk: &mockable.Clock{},
				},
				backend: backend,
			}

			// build and verify child block
			childHeight := parentHeight + 1
			statelessAbortBlk, err := block.NewBanffAbortBlock(test.childTime, parentID, childHeight)
			require.NoError(err)

			// setup parent state
			parentTime := genesistest.DefaultValidatorStartTime
			s.EXPECT().GetLastAccepted().Return(parentID).Times(3)
			s.EXPECT().GetTimestamp().Return(parentTime).Times(3)
			s.EXPECT().GetFeeState().Return(gas.State{}).Times(3)
			s.EXPECT().GetL1ValidatorExcess().Return(gas.Gas(0)).Times(3)
			s.EXPECT().GetAccruedFees().Return(uint64(0)).Times(3)
			s.EXPECT().NumActiveL1Validators().Return(0).Times(3)

			onDecisionState, err := state.NewDiff(parentID, backend)
			require.NoError(err)
			onCommitState, err := state.NewDiff(parentID, backend)
			require.NoError(err)
			onAbortState, err := state.NewDiff(parentID, backend)
			require.NoError(err)
			backend.blkIDToState[parentID] = &blockState{
				timestamp:      test.parentTime,
				statelessBlock: parentStatelessBlk,
				proposalBlockState: proposalBlockState{
					onDecisionState: onDecisionState,
					onCommitState:   onCommitState,
					onAbortState:    onAbortState,
				},
			}

			// Set expectations for dependencies.
			parentStatelessBlk.EXPECT().Height().Return(uint64(1)).Times(1)

			err = statelessAbortBlk.Visit(verifier)
			require.ErrorIs(err, test.result)
		})
	}
}

// Note: Consider combining with TestApricotCommitBlockTimestampChecks for better test organization
func TestBanffCommitBlockTimestampChecks(t *testing.T) {
	ctrl := gomock.NewController(t)

	now := genesistest.DefaultValidatorStartTime.Add(time.Hour)

	tests := []struct {
		description string
		parentTime  time.Time
		childTime   time.Time
		result      error
	}{
		{
			description: "commit block timestamp matching parent's one",
			parentTime:  now,
			childTime:   now,
			result:      nil,
		},
		{
			description: "commit block timestamp before parent's one",
			childTime:   now.Add(-1 * time.Second),
			parentTime:  now,
			result:      errOptionBlockTimestampNotMatchingParent,
		},
		{
			description: "commit block timestamp after parent's one",
			parentTime:  now,
			childTime:   now.Add(time.Second),
			result:      errOptionBlockTimestampNotMatchingParent,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			require := require.New(t)

			// Create mocked dependencies.
			s := state.NewMockState(ctrl)
			mempool, err := mempool.New("", metric.NewRegistry())
			require.NoError(err)
			parentID := ids.GenerateTestID()
			parentStatelessBlk := block.NewMockBlock(ctrl)
			parentHeight := uint64(1)

			backend := &backend{
				blkIDToState: make(map[ids.ID]*blockState),
				Mempool:      mempool,
				state:        s,
				ctx: &consensuscontext.Context{
			NetworkID: constants.UnitTestID,
			ChainID:   constants.PlatformChainID,
		},
			}
			verifier := &verifier{
				txExecutorBackend: &executor.Backend{
					Config: &config.Internal{
						UpgradeConfig: upgradetest.GetConfig(upgradetest.Banff),
					},
					Clk: &mockable.Clock{},
				},
				backend: backend,
			}

			// build and verify child block
			childHeight := parentHeight + 1
			statelessCommitBlk, err := block.NewBanffCommitBlock(test.childTime, parentID, childHeight)
			require.NoError(err)

			// setup parent state
			parentTime := genesistest.DefaultValidatorStartTime
			s.EXPECT().GetLastAccepted().Return(parentID).Times(3)
			s.EXPECT().GetTimestamp().Return(parentTime).Times(3)
			s.EXPECT().GetFeeState().Return(gas.State{}).Times(3)
			s.EXPECT().GetL1ValidatorExcess().Return(gas.Gas(0)).Times(3)
			s.EXPECT().GetAccruedFees().Return(uint64(0)).Times(3)
			s.EXPECT().NumActiveL1Validators().Return(0).Times(3)

			onDecisionState, err := state.NewDiff(parentID, backend)
			require.NoError(err)
			onCommitState, err := state.NewDiff(parentID, backend)
			require.NoError(err)
			onAbortState, err := state.NewDiff(parentID, backend)
			require.NoError(err)
			backend.blkIDToState[parentID] = &blockState{
				timestamp:      test.parentTime,
				statelessBlock: parentStatelessBlk,
				proposalBlockState: proposalBlockState{
					onDecisionState: onDecisionState,
					onCommitState:   onCommitState,
					onAbortState:    onAbortState,
				},
			}

			// Set expectations for dependencies.
			parentStatelessBlk.EXPECT().Height().Return(uint64(1)).Times(1)

			err = statelessCommitBlk.Visit(verifier)
			require.ErrorIs(err, test.result)
		})
	}
}

func TestVerifierVisitApricotStandardBlockWithProposalBlockParent(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	// Create mocked dependencies.
	s := state.NewMockState(ctrl)
	mempool, err := mempool.New("", metric.NewRegistry())
	require.NoError(err)
	parentID := ids.GenerateTestID()
	parentStatelessBlk := block.NewMockBlock(ctrl)
	parentOnCommitState := state.NewMockDiff(ctrl)
	parentOnAbortState := state.NewMockDiff(ctrl)

	backend := &backend{
		blkIDToState: map[ids.ID]*blockState{
			parentID: {
				statelessBlock: parentStatelessBlk,
				proposalBlockState: proposalBlockState{
					onCommitState: parentOnCommitState,
					onAbortState:  parentOnAbortState,
				},
			},
		},
		Mempool: mempool,
		state:   s,
		ctx: &consensuscontext.Context{
			NetworkID: constants.UnitTestID,
			ChainID:   constants.PlatformChainID,
		},
	}
	verifier := &verifier{
		txExecutorBackend: &executor.Backend{
			Config: &config.Internal{
				UpgradeConfig: upgradetest.GetConfig(upgradetest.ApricotPhasePost6),
			},
			Clk: &mockable.Clock{},
		},
		backend: backend,
	}

	blk, err := block.NewApricotStandardBlock(
		parentID,
		2,
		[]*txs.Tx{
			{
				Unsigned: &txs.AdvanceTimeTx{},
				Creds:    []verify.Verifiable{},
			},
		},
	)
	require.NoError(err)

	parentStatelessBlk.EXPECT().Height().Return(uint64(1)).Times(1)

	err = verifier.ApricotStandardBlock(blk)
	require.ErrorIs(err, state.ErrMissingParentState)
}

func TestVerifierVisitBanffStandardBlockWithProposalBlockParent(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	// Create mocked dependencies.
	s := state.NewMockState(ctrl)
	mempool, err := mempool.New("", metric.NewRegistry())
	require.NoError(err)
	parentID := ids.GenerateTestID()
	parentStatelessBlk := block.NewMockBlock(ctrl)
	parentTime := time.Now()
	parentOnCommitState := state.NewMockDiff(ctrl)
	parentOnAbortState := state.NewMockDiff(ctrl)

	backend := &backend{
		blkIDToState: map[ids.ID]*blockState{
			parentID: {
				statelessBlock: parentStatelessBlk,
				proposalBlockState: proposalBlockState{
					onCommitState: parentOnCommitState,
					onAbortState:  parentOnAbortState,
				},
			},
		},
		Mempool: mempool,
		state:   s,
		ctx: &consensuscontext.Context{
			NetworkID: constants.UnitTestID,
			ChainID:   constants.PlatformChainID,
		},
	}
	verifier := &verifier{
		txExecutorBackend: &executor.Backend{
			Config: &config.Internal{
				UpgradeConfig: upgradetest.GetConfig(upgradetest.Banff),
			},
			Clk: &mockable.Clock{},
		},
		backend: backend,
	}

	blk, err := block.NewBanffStandardBlock(
		parentTime.Add(time.Second),
		parentID,
		2,
		[]*txs.Tx{
			{
				Unsigned: &txs.AdvanceTimeTx{},
				Creds:    []verify.Verifiable{},
			},
		},
	)
	require.NoError(err)

	parentStatelessBlk.EXPECT().Height().Return(uint64(1)).Times(1)

	err = verifier.BanffStandardBlock(blk)
	require.ErrorIs(err, state.ErrMissingParentState)
}

func TestVerifierVisitApricotCommitBlockUnexpectedParentState(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	// Create mocked dependencies.
	s := state.NewMockState(ctrl)
	parentID := ids.GenerateTestID()
	parentStatelessBlk := block.NewMockBlock(ctrl)
	verifier := &verifier{
		txExecutorBackend: &executor.Backend{
			Config: &config.Internal{
				UpgradeConfig: upgradetest.GetConfig(upgradetest.ApricotPhasePost6),
			},
			Clk: &mockable.Clock{},
		},
		backend: &backend{
			blkIDToState: map[ids.ID]*blockState{
				parentID: {
					statelessBlock: parentStatelessBlk,
				},
			},
			state: s,
			ctx: &consensuscontext.Context{
				NetworkID: constants.UnitTestID,
				ChainID:   constants.PlatformChainID,
			},
		},
	}

	blk, err := block.NewApricotCommitBlock(
		parentID,
		2,
	)
	require.NoError(err)

	// Set expectations for dependencies.
	parentStatelessBlk.EXPECT().Height().Return(uint64(1)).Times(1)

	// Verify the block.
	err = verifier.ApricotCommitBlock(blk)
	require.ErrorIs(err, state.ErrMissingParentState)
}

func TestVerifierVisitBanffCommitBlockUnexpectedParentState(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	// Create mocked dependencies.
	s := state.NewMockState(ctrl)
	parentID := ids.GenerateTestID()
	parentStatelessBlk := block.NewMockBlock(ctrl)
	timestamp := time.Unix(12345, 0)
	verifier := &verifier{
		txExecutorBackend: &executor.Backend{
			Config: &config.Internal{
				UpgradeConfig: upgradetest.GetConfig(upgradetest.Banff),
			},
			Clk: &mockable.Clock{},
		},
		backend: &backend{
			blkIDToState: map[ids.ID]*blockState{
				parentID: {
					statelessBlock: parentStatelessBlk,
					timestamp:      timestamp,
				},
			},
			state: s,
			ctx: &consensuscontext.Context{
				NetworkID: constants.UnitTestID,
				ChainID:   constants.PlatformChainID,
			},
		},
	}

	blk, err := block.NewBanffCommitBlock(
		timestamp,
		parentID,
		2,
	)
	require.NoError(err)

	// Set expectations for dependencies.
	parentStatelessBlk.EXPECT().Height().Return(uint64(1)).Times(1)

	// Verify the block.
	err = verifier.BanffCommitBlock(blk)
	require.ErrorIs(err, state.ErrMissingParentState)
}

func TestVerifierVisitApricotAbortBlockUnexpectedParentState(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	// Create mocked dependencies.
	s := state.NewMockState(ctrl)
	parentID := ids.GenerateTestID()
	parentStatelessBlk := block.NewMockBlock(ctrl)
	verifier := &verifier{
		txExecutorBackend: &executor.Backend{
			Config: &config.Internal{
				UpgradeConfig: upgradetest.GetConfig(upgradetest.ApricotPhasePost6),
			},
			Clk: &mockable.Clock{},
		},
		backend: &backend{
			blkIDToState: map[ids.ID]*blockState{
				parentID: {
					statelessBlock: parentStatelessBlk,
				},
			},
			state: s,
			ctx: &consensuscontext.Context{
				NetworkID: constants.UnitTestID,
				ChainID:   constants.PlatformChainID,
			},
		},
	}

	blk, err := block.NewApricotAbortBlock(
		parentID,
		2,
	)
	require.NoError(err)

	// Set expectations for dependencies.
	parentStatelessBlk.EXPECT().Height().Return(uint64(1)).Times(1)

	// Verify the block.
	err = verifier.ApricotAbortBlock(blk)
	require.ErrorIs(err, state.ErrMissingParentState)
}

func TestVerifierVisitBanffAbortBlockUnexpectedParentState(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	// Create mocked dependencies.
	s := state.NewMockState(ctrl)
	parentID := ids.GenerateTestID()
	parentStatelessBlk := block.NewMockBlock(ctrl)
	timestamp := time.Unix(12345, 0)
	verifier := &verifier{
		txExecutorBackend: &executor.Backend{
			Config: &config.Internal{
				UpgradeConfig: upgradetest.GetConfig(upgradetest.Banff),
			},
			Clk: &mockable.Clock{},
		},
		backend: &backend{
			blkIDToState: map[ids.ID]*blockState{
				parentID: {
					statelessBlock: parentStatelessBlk,
					timestamp:      timestamp,
				},
			},
			state: s,
			ctx: &consensuscontext.Context{
				NetworkID: constants.UnitTestID,
				ChainID:   constants.PlatformChainID,
			},
		},
	}

	blk, err := block.NewBanffAbortBlock(
		timestamp,
		parentID,
		2,
	)
	require.NoError(err)

	// Set expectations for dependencies.
	parentStatelessBlk.EXPECT().Height().Return(uint64(1)).Times(1)

	// Verify the block.
	err = verifier.BanffAbortBlock(blk)
	require.ErrorIs(err, state.ErrMissingParentState)
}

func TestBlockExecutionWithComplexity(t *testing.T) {
	verifier := newTestVerifier(t, testVerifierConfig{})
	wallet := txstest.NewWalletWithOptions(
		t,
		verifier.ctx,
		txstest.WalletConfig{
			Config: &config.Config{
				TrackedChains:            verifier.txExecutorBackend.Config.TrackedChains,
				SybilProtectionEnabled: verifier.txExecutorBackend.Config.SybilProtectionEnabled,
				Chains:                 verifier.txExecutorBackend.Config.Chains,
			},
			InternalCfg: verifier.txExecutorBackend.Config, // Pass internal config for dynamic fees
		},
		verifier.state,
		secp256k1fx.NewKeychain(genesistest.DefaultFundedKeys[0]),
		nil, // subnetIDs
		nil, // validationIDs
		nil, // chainIDs
	)

	baseTx0, err := wallet.IssueBaseTx([]*lux.TransferableOutput{})
	require.NoError(t, err)
	baseTx1, err := wallet.IssueBaseTx([]*lux.TransferableOutput{})
	require.NoError(t, err)

	blockComplexity, err := txfee.TxComplexity(baseTx0.Unsigned, baseTx1.Unsigned)
	require.NoError(t, err)
	blockGas, err := blockComplexity.ToGas(verifier.txExecutorBackend.Config.DynamicFeeConfig.Weights)
	require.NoError(t, err)

	const secondsToAdvance = 10

	initialFeeState := gas.State{}
	feeStateAfterTimeAdvanced := initialFeeState.AdvanceTime(
		verifier.txExecutorBackend.Config.DynamicFeeConfig.MaxCapacity,
		verifier.txExecutorBackend.Config.DynamicFeeConfig.MaxPerSecond,
		verifier.txExecutorBackend.Config.DynamicFeeConfig.TargetPerSecond,
		secondsToAdvance,
	)
	feeStateAfterGasConsumed, err := feeStateAfterTimeAdvanced.ConsumeGas(blockGas)
	require.NoError(t, err)

	tests := []struct {
		name             string
		timestamp        time.Time
		expectedErr      error
		expectedFeeState gas.State
	}{
		{
			name:        "no capacity",
			timestamp:   genesistest.DefaultValidatorStartTime,
			expectedErr: gas.ErrInsufficientCapacity,
		},
		{
			name:             "updates fee state",
			timestamp:        genesistest.DefaultValidatorStartTime.Add(secondsToAdvance * time.Second),
			expectedFeeState: feeStateAfterGasConsumed,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)

			// Clear the state to prevent prior tests from impacting this test.
			clear(verifier.blkIDToState)

			verifier.txExecutorBackend.Clk.Set(test.timestamp)
			timestamp, _, err := state.NextBlockTime(
				verifier.txExecutorBackend.Config.ValidatorFeeConfig,
				verifier.state,
				verifier.txExecutorBackend.Clk,
			)
			require.NoError(err)

			lastAcceptedID := verifier.state.GetLastAccepted()
			lastAccepted, err := verifier.state.GetStatelessBlock(lastAcceptedID)
			require.NoError(err)

			blk, err := block.NewBanffStandardBlock(
				timestamp,
				lastAcceptedID,
				lastAccepted.Height()+1,
				[]*txs.Tx{
					baseTx0,
					baseTx1,
				},
			)
			require.NoError(err)

			blkID := blk.ID()
			err = blk.Visit(verifier)
			require.ErrorIs(err, test.expectedErr)
			if err != nil {
				require.NotContains(verifier.blkIDToState, blkID)
				return
			}

			require.Contains(verifier.blkIDToState, blkID)
			blockState := verifier.blkIDToState[blkID]
			require.Equal(blk, blockState.statelessBlock)
			require.Equal(test.expectedFeeState, blockState.onAcceptState.GetFeeState())
		})
	}
}

func TestDeactivateLowBalanceL1Validators(t *testing.T) {
	sk, err := localsigner.New()
	require.NoError(t, err)

	var (
		pk      = sk.PublicKey()
		pkBytes = bls.PublicKeyToUncompressedBytes(pk)

		newL1Validator = func(endAccumulatedFee uint64) state.L1Validator {
			return state.L1Validator{
				ValidationID:      ids.GenerateTestID(),
				ChainID:          ids.GenerateTestID(),
				NodeID:            ids.GenerateTestNodeID(),
				PublicKey:         pkBytes,
				Weight:            1,
				EndAccumulatedFee: endAccumulatedFee,
			}
		}
		fractionalTimeL1Validator0 = newL1Validator(1 * units.NanoLux) // lasts .5 seconds
		fractionalTimeL1Validator1 = newL1Validator(1 * units.NanoLux) // lasts .5 seconds
		wholeTimeL1Validator       = newL1Validator(2 * units.NanoLux) // lasts 1 second
	)

	tests := []struct {
		name                                  string
		initialL1Validators                   []state.L1Validator
		expectedL1Validators                  []state.L1Validator
		expectedLowBalanceL1ValidatorsEvicted bool
	}{
		{
			name: "no L1 validators",
		},
		{
			name: "fractional L1 validator is not undercharged",
			initialL1Validators: []state.L1Validator{
				fractionalTimeL1Validator0,
			},
			expectedLowBalanceL1ValidatorsEvicted: true,
		},
		{
			name: "fractional L1 validators are not undercharged",
			initialL1Validators: []state.L1Validator{
				fractionalTimeL1Validator0,
				fractionalTimeL1Validator1,
			},
			expectedLowBalanceL1ValidatorsEvicted: true,
		},
		{
			name: "whole L1 validators are not overcharged",
			initialL1Validators: []state.L1Validator{
				wholeTimeL1Validator,
			},
			expectedL1Validators: []state.L1Validator{
				wholeTimeL1Validator,
			},
		},
		{
			name: "partial eviction",
			initialL1Validators: []state.L1Validator{
				fractionalTimeL1Validator0,
				wholeTimeL1Validator,
			},
			expectedL1Validators: []state.L1Validator{
				wholeTimeL1Validator,
			},
			expectedLowBalanceL1ValidatorsEvicted: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)

			s := statetest.New(t, statetest.Config{})
			for _, l1Validator := range test.initialL1Validators {
				require.NoError(s.PutL1Validator(l1Validator))
			}

			diff, err := state.NewDiffOn(s)
			require.NoError(err)

			config := validatorfee.Config{
				Capacity:                 builder.LocalValidatorFeeConfig.Capacity,
				Target:                   builder.LocalValidatorFeeConfig.Target,
				MinPrice:                 gas.Price(2 * units.NanoLux), // Min price is increased to allow fractional fees
				ExcessConversionConstant: builder.LocalValidatorFeeConfig.ExcessConversionConstant,
			}
			lowBalanceL1ValidatorsEvicted, err := deactivateLowBalanceL1Validators(config, diff)
			require.NoError(err)
			require.Equal(test.expectedLowBalanceL1ValidatorsEvicted, lowBalanceL1ValidatorsEvicted)

			l1Validators, err := diff.GetActiveL1ValidatorsIterator()
			require.NoError(err)
			require.Equal(
				test.expectedL1Validators,
				iterator.ToSlice(l1Validators),
			)
		})
	}
}

func TestDeactivateLowBalanceL1ValidatorBlockChanges(t *testing.T) {
	signer, err := localsigner.New()
	require.NoError(t, err)

	fractionalTimeL1Validator := state.L1Validator{
		ValidationID:      ids.GenerateTestID(),
		ChainID:          ids.GenerateTestID(),
		NodeID:            ids.GenerateTestNodeID(),
		PublicKey:         bls.PublicKeyToUncompressedBytes(signer.PublicKey()),
		Weight:            1,
		EndAccumulatedFee: 3 * units.NanoLux, // lasts 1.5 seconds
	}

	tests := []struct {
		name              string
		currentFork       upgradetest.Fork
		durationToAdvance time.Duration
		networkID         uint32
		expectedErr       error
	}{
		{
			name:              "Before F Upgrade - no L1 validators evicted",
			currentFork:       upgradetest.Etna,
			durationToAdvance: 0,
			networkID:         constants.UnitTestID,
			expectedErr:       ErrStandardBlockWithoutChanges,
		},
		{
			name:              "After F Upgrade - no L1 validators evicted",
			currentFork:       upgradetest.Fortuna,
			durationToAdvance: 0,
			networkID:         constants.UnitTestID,
			expectedErr:       ErrStandardBlockWithoutChanges,
		},
		{
			name:              "Before F Upgrade - L1 validators evicted",
			currentFork:       upgradetest.Etna,
			durationToAdvance: time.Second,
			networkID:         constants.UnitTestID,
		},
		{
			name:              "After F Upgrade - L1 validators evicted",
			currentFork:       upgradetest.Fortuna,
			durationToAdvance: time.Second,
			networkID:         constants.UnitTestID,
		},
		{
			name:              "Before F Upgrade - L1 validators evicted - on Testnet",
			currentFork:       upgradetest.Etna,
			durationToAdvance: time.Second,
			networkID:         constants.TestnetID,
		},
		{
			name:              "After F Upgrade - L1 validators evicted - on Testnet",
			currentFork:       upgradetest.Fortuna,
			durationToAdvance: time.Second,
			networkID:         constants.TestnetID,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)

			ctx := consensustest.Context(t, constants.PlatformChainID)
			ctx.NetworkID = test.networkID
			verifier := newTestVerifier(t, testVerifierConfig{
				Upgrades: upgradetest.GetConfig(test.currentFork),
				Context:  ctx,
				ValidatorFeeConfig: validatorfee.Config{
					Capacity:                 builder.LocalValidatorFeeConfig.Capacity,
					Target:                   builder.LocalValidatorFeeConfig.Target,
					MinPrice:                 gas.Price(2 * units.NanoLux), // Min price is increased to allow fractional fees
					ExcessConversionConstant: builder.LocalValidatorFeeConfig.ExcessConversionConstant,
				},
			})

			require.NoError(verifier.state.PutL1Validator(fractionalTimeL1Validator))

			blk, err := block.NewBanffStandardBlock(
				genesistest.DefaultValidatorStartTime.Add(test.durationToAdvance),
				verifier.state.GetLastAccepted(),
				1,   // This block is built on top of the genesis
				nil, // There are no transactions in the block
			)
			require.NoError(err)

			err = verifier.BanffStandardBlock(blk)
			require.ErrorIs(err, test.expectedErr)
		})
	}
}
