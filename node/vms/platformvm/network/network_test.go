// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package network

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luxfi/metric"
	"github.com/luxfi/mock/gomock"
	"github.com/stretchr/testify/require"

	consensustest "github.com/luxfi/consensus/test/helpers"
	validators "github.com/luxfi/consensus/validator"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/math/set"
	"github.com/luxfi/warp"
	"github.com/luxfi/sdk/node/vms/components/lux"
	"github.com/luxfi/sdk/node/vms/platformvm/config"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/txs/mempool"

	pmempool "github.com/luxfi/sdk/node/vms/platformvm/txs/mempool"
)

// testSender implements warp.Sender for testing with optional call tracking
type testSender struct {
	sendGossipCalled bool
}

var _ warp.Sender = (*testSender)(nil)

func (t *testSender) SendRequest(ctx context.Context, nodeIDs set.Set[ids.NodeID], requestID uint32, requestBytes []byte) error {
	return nil
}

func (t *testSender) SendResponse(ctx context.Context, nodeID ids.NodeID, requestID uint32, responseBytes []byte) error {
	return nil
}

func (t *testSender) SendError(ctx context.Context, nodeID ids.NodeID, requestID uint32, errorCode int32, errorMessage string) error {
	return nil
}

func (t *testSender) SendGossip(ctx context.Context, config warp.SendConfig, gossipBytes []byte) error {
	t.sendGossipCalled = true
	return nil
}

var (
	errTest = errors.New("test error")

	testConfig = config.Network{
		MaxValidatorSetStaleness:                    time.Second,
		TargetGossipSize:                            1,
		PushGossipNumValidators:                     1,
		PushGossipNumPeers:                          0,
		PushRegossipNumValidators:                   1,
		PushRegossipNumPeers:                        0,
		PushGossipDiscardedCacheSize:                1,
		PushGossipMaxRegossipFrequency:              time.Second,
		PushGossipFrequency:                         time.Second,
		PullGossipPollSize:                          1,
		PullGossipFrequency:                         time.Second,
		PullGossipThrottlingPeriod:                  time.Second,
		PullGossipThrottlingLimit:                   1,
		ExpectedBloomFilterElements:                 10,
		ExpectedBloomFilterFalsePositiveProbability: .1,
		MaxBloomFilterFalsePositiveProbability:      .5,
	}
)

// mockValidatorState implements validators.State for testing
type mockValidatorState struct {
	height     uint64
	validators map[ids.NodeID]*validators.GetValidatorOutput
}

func (m *mockValidatorState) GetMinimumHeight(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (m *mockValidatorState) GetCurrentHeight(ctx context.Context) (uint64, error) {
	return m.height, nil
}

func (m *mockValidatorState) GetNetID(ctx context.Context, chainID ids.ID) (ids.ID, error) {
	return ids.Empty, nil
}

func (m *mockValidatorState) GetValidatorSet(
	ctx context.Context,
	height uint64,
	netID ids.ID,
) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	return m.validators, nil
}

func (m *mockValidatorState) GetCurrentValidators(
	ctx context.Context,
	height uint64,
	netID ids.ID,
) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	return m.validators, nil
}

// GetCurrentValidatorOutput represents a validator output
type GetCurrentValidatorOutput struct {
	NodeID    ids.NodeID
	PublicKey *bls.PublicKey
	Weight    uint64
}

func (m *mockValidatorState) GetCurrentValidatorSet(
	ctx context.Context,
	netID ids.ID,
) (map[ids.ID]*GetCurrentValidatorOutput, uint64, error) {
	// Not used in this test
	return nil, m.height, nil
}

func (m *mockValidatorState) GetWarpValidatorSet(
	ctx context.Context,
	height uint64,
	netID ids.ID,
) (*validators.WarpSet, error) {
	// Not used in this test
	return nil, nil
}

func (m *mockValidatorState) GetWarpValidatorSets(
	ctx context.Context,
	heights []uint64,
	netIDs []ids.ID,
) (map[ids.ID]map[uint64]*validators.WarpSet, error) {
	// Not used in this test
	return nil, nil
}

var _ TxVerifier = (*testTxVerifier)(nil)

type testTxVerifier struct {
	err error
}

func (t testTxVerifier) VerifyTx(*txs.Tx) error {
	return t.err
}

func TestNetworkIssueTxFromRPC(t *testing.T) {
	type test struct {
		name          string
		mempool       *pmempool.Mempool
		txVerifier    testTxVerifier
		appSenderFunc func(*gomock.Controller) warp.Sender
		tx            *txs.Tx
		expectedErr   error
	}

	tests := []test{
		{
			name: "mempool has transaction",
			mempool: func() *pmempool.Mempool {
				mempool, err := pmempool.New("", metric.NewRegistry())
				require.NoError(t, err)
				require.NoError(t, mempool.Add(&txs.Tx{Unsigned: &txs.BaseTx{}}))
				return mempool
			}(),
			appSenderFunc: func(ctrl *gomock.Controller) warp.Sender {
				return &testSender{}
			},
			tx:          &txs.Tx{Unsigned: &txs.BaseTx{}},
			expectedErr: mempool.ErrDuplicateTx,
		},
		{
			name: "transaction marked as dropped in mempool",
			mempool: func() *pmempool.Mempool {
				mempool, err := pmempool.New("", metric.NewRegistry())
				require.NoError(t, err)
				mempool.MarkDropped(ids.Empty, errTest)
				return mempool
			}(),
			appSenderFunc: func(ctrl *gomock.Controller) warp.Sender {
				// Shouldn't gossip the tx
				return &testSender{}
			},
			tx:          &txs.Tx{Unsigned: &txs.BaseTx{}},
			expectedErr: errTest,
		},
		{
			name: "tx dropped",
			mempool: func() *pmempool.Mempool {
				mempool, err := pmempool.New("", metric.NewRegistry())
				require.NoError(t, err)
				return mempool
			}(),
			txVerifier: testTxVerifier{err: errTest},
			appSenderFunc: func(ctrl *gomock.Controller) warp.Sender {
				// Shouldn't gossip the tx
				return &testSender{}
			},
			tx:          &txs.Tx{Unsigned: &txs.BaseTx{}},
			expectedErr: errTest,
		},
		{
			name: "tx too big",
			mempool: func() *pmempool.Mempool {
				mempool, err := pmempool.New("", metric.NewRegistry())
				require.NoError(t, err)
				return mempool
			}(),
			appSenderFunc: func(ctrl *gomock.Controller) warp.Sender {
				// Shouldn't gossip the tx
				return &testSender{}
			},
			tx: func() *txs.Tx {
				tx := &txs.Tx{Unsigned: &txs.BaseTx{}}
				bytes := make([]byte, mempool.MaxTxSize+1)
				tx.SetBytes(bytes, bytes)
				return tx
			}(),
			expectedErr: mempool.ErrTxTooLarge,
		},
		{
			name: "tx conflicts",
			mempool: func() *pmempool.Mempool {
				mempool, err := pmempool.New("", metric.NewRegistry())
				require.NoError(t, err)

				tx := &txs.Tx{
					Unsigned: &txs.BaseTx{
						BaseTx: lux.BaseTx{
							Ins: []*lux.TransferableInput{
								{
									UTXOID: lux.UTXOID{},
								},
							},
						},
					},
				}

				require.NoError(t, mempool.Add(tx))
				return mempool
			}(),
			appSenderFunc: func(ctrl *gomock.Controller) warp.Sender {
				// Shouldn't gossip the tx
				return &testSender{}
			},
			tx: func() *txs.Tx {
				tx := &txs.Tx{
					Unsigned: &txs.BaseTx{
						BaseTx: lux.BaseTx{
							Ins: []*lux.TransferableInput{
								{
									UTXOID: lux.UTXOID{},
								},
							},
						},
					},
					TxID: ids.ID{1},
				}
				return tx
			}(),
			expectedErr: mempool.ErrConflictsWithOtherTx,
		},
		{
			name: "mempool full",
			mempool: func() *pmempool.Mempool {
				m, err := pmempool.New("", metric.NewRegistry())
				require.NoError(t, err)

				// Fill the mempool to capacity (64 MiB / 2 MiB per tx = 32 txs)
				for i := 0; i < 32; i++ {
					tx := &txs.Tx{Unsigned: &txs.BaseTx{}}
					bytes := make([]byte, mempool.MaxTxSize)
					tx.SetBytes(bytes, bytes)
					tx.TxID = ids.GenerateTestID()
					require.NoError(t, m.Add(tx))
				}

				return m
			}(),
			appSenderFunc: func(ctrl *gomock.Controller) warp.Sender {
				// Shouldn't gossip the tx
				return &testSender{}
			},
			tx: func() *txs.Tx {
				tx := &txs.Tx{Unsigned: &txs.BaseTx{BaseTx: lux.BaseTx{}}}
				tx.SetBytes([]byte{1, 2, 3}, []byte{1, 2, 3})
				return tx
			}(),
			expectedErr: mempool.ErrMempoolFull,
		},
		{
			name: "happy path",
			mempool: func() *pmempool.Mempool {
				mempool, err := pmempool.New("", metric.NewRegistry())
				require.NoError(t, err)
				return mempool
			}(),
			appSenderFunc: func(ctrl *gomock.Controller) warp.Sender {
				// testSender tracks if SendGossip was called
				return &testSender{}
			},
			tx:          &txs.Tx{Unsigned: &txs.BaseTx{}},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)

			consensusCtx := consensustest.Context(t, ids.Empty)
			// Extract values directly from consensus context
			nodeID := consensusCtx.NodeID
			netID := consensusCtx.ChainID
			// Use a simple test logger for now
			logger := log.NoLog{}
			// Create a mock validator state that returns sensible defaults
			validatorState := &mockValidatorState{
				height: 100,
				validators: map[ids.NodeID]*validators.GetValidatorOutput{
					nodeID: {
						NodeID:    nodeID,
						PublicKey: nil,
						Weight:    100,
					},
				},
			}
			n, err := New(
				logger,
				nodeID,
				netID,
				validatorState,
				tt.txVerifier,
				tt.mempool,
				false,
				tt.appSenderFunc(ctrl),
				nil,
				nil,
				nil,
				metric.NewRegistry(),
				testConfig,
			)
			require.NoError(err)

			err = n.IssueTxFromRPC(tt.tx)
			require.ErrorIs(err, tt.expectedErr)

			require.NoError(n.txPushGossiper.Gossip(context.Background()))
		})
	}
}
