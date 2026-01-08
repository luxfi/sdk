// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"context"
	"testing"
	"time"

	"github.com/luxfi/mock/gomock"
	"github.com/stretchr/testify/require"

	consensustest "github.com/luxfi/consensus/test/helpers"
	"github.com/luxfi/consensus/validator/uptime"
	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/node/vms/platformvm/block"
	"github.com/luxfi/sdk/node/vms/platformvm/config"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/status"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	"github.com/luxfi/sdk/node/vms/platformvm/txs/executor"
)

func TestBlockOptions(t *testing.T) {
	type test struct {
		name                   string
		blkF                   func(*gomock.Controller) *Block
		expectedPreferenceType block.Block
	}

	tests := []test{
		{
			name: "apricot proposal block; commit preferred",
			blkF: func(ctrl *gomock.Controller) *Block {
				state := state.NewMockState(ctrl)

				manager := &manager{
					backend: &backend{
						state: state,
						ctx:   consensustest.Context(t, ids.GenerateTestID()),
					},
					txExecutorBackend: &executor.Backend{
						Config: &config.Internal{
							UptimePercentage: 0,
						},
						Uptimes: &uptime.NoOpCalculator{},
					},
				Log: log.NoLog{},
				}

				return &Block{
					Block:   &block.ApricotProposalBlock{},
					manager: manager,
				}
			},
			expectedPreferenceType: &block.ApricotCommitBlock{},
		},
		{
			name: "banff proposal block; invalid proposal tx",
			blkF: func(ctrl *gomock.Controller) *Block {
				state := state.NewMockState(ctrl)

				manager := &manager{
					backend: &backend{
						state: state,
						ctx:   consensustest.Context(t, ids.GenerateTestID()),
					},
					txExecutorBackend: &executor.Backend{
						Config: &config.Internal{
							UptimePercentage: 0,
						},
						Uptimes: &uptime.NoOpCalculator{},
					},
				Log: log.NoLog{},
				}

				return &Block{
					Block: &block.BanffProposalBlock{
						ApricotProposalBlock: block.ApricotProposalBlock{
							Tx: &txs.Tx{
								Unsigned: &txs.CreateChainTx{},
							},
						},
					},
					manager: manager,
				}
			},
			expectedPreferenceType: &block.BanffCommitBlock{},
		},
		{
			name: "banff proposal block; missing tx",
			blkF: func(ctrl *gomock.Controller) *Block {
				stakerTxID := ids.GenerateTestID()

				state := state.NewMockState(ctrl)
				state.EXPECT().GetTx(stakerTxID).Return(nil, status.Unknown, database.ErrNotFound)

				manager := &manager{
					backend: &backend{
						state: state,
						ctx:   consensustest.Context(t, ids.GenerateTestID()),
					},
					txExecutorBackend: &executor.Backend{
						Config: &config.Internal{
							UptimePercentage: 0,
						},
						Uptimes: &uptime.NoOpCalculator{},
					},
				Log: log.NoLog{},
				}

				return &Block{
					Block: &block.BanffProposalBlock{
						ApricotProposalBlock: block.ApricotProposalBlock{
							Tx: &txs.Tx{
								Unsigned: &txs.RewardValidatorTx{
									TxID: stakerTxID,
								},
							},
						},
					},
					manager: manager,
				}
			},
			expectedPreferenceType: &block.BanffCommitBlock{},
		},
		{
			name: "banff proposal block; error fetching staker tx",
			blkF: func(ctrl *gomock.Controller) *Block {
				stakerTxID := ids.GenerateTestID()

				state := state.NewMockState(ctrl)
				state.EXPECT().GetTx(stakerTxID).Return(nil, status.Unknown, database.ErrClosed)

				manager := &manager{
					backend: &backend{
						state: state,
						ctx:   consensustest.Context(t, ids.GenerateTestID()),
					},
					txExecutorBackend: &executor.Backend{
						Config: &config.Internal{
							UptimePercentage: 0,
						},
						Uptimes: &uptime.NoOpCalculator{},
					},
				Log: log.NoLog{},
				}

				return &Block{
					Block: &block.BanffProposalBlock{
						ApricotProposalBlock: block.ApricotProposalBlock{
							Tx: &txs.Tx{
								Unsigned: &txs.RewardValidatorTx{
									TxID: stakerTxID,
								},
							},
						},
					},
					manager: manager,
				}
			},
			expectedPreferenceType: &block.BanffCommitBlock{},
		},
		{
			name: "banff proposal block; unexpected staker tx type",
			blkF: func(ctrl *gomock.Controller) *Block {
				stakerTxID := ids.GenerateTestID()
				stakerTx := &txs.Tx{
					Unsigned: &txs.CreateChainTx{},
				}

				state := state.NewMockState(ctrl)
				state.EXPECT().GetTx(stakerTxID).Return(stakerTx, status.Committed, nil)

				manager := &manager{
					backend: &backend{
						state: state,
						ctx:   consensustest.Context(t, ids.GenerateTestID()),
					},
					txExecutorBackend: &executor.Backend{
						Config: &config.Internal{
							UptimePercentage: 0,
						},
						Uptimes: &uptime.NoOpCalculator{},
					},
				Log: log.NoLog{},
				}

				return &Block{
					Block: &block.BanffProposalBlock{
						ApricotProposalBlock: block.ApricotProposalBlock{
							Tx: &txs.Tx{
								Unsigned: &txs.RewardValidatorTx{
									TxID: stakerTxID,
								},
							},
						},
					},
					manager: manager,
				}
			},
			expectedPreferenceType: &block.BanffCommitBlock{},
		},
		{
			name: "banff proposal block; missing primary network validator",
			blkF: func(ctrl *gomock.Controller) *Block {
				var (
					stakerTxID = ids.GenerateTestID()
					nodeID     = ids.GenerateTestNodeID()
					netID      = ids.GenerateTestID()
					stakerTx   = &txs.Tx{
						Unsigned: &txs.AddPermissionlessValidatorTx{
							Validator: txs.Validator{
								NodeID: nodeID,
							},
							Chain: netID,
						},
					}
				)

				state := state.NewMockState(ctrl)
				state.EXPECT().GetTx(stakerTxID).Return(stakerTx, status.Committed, nil)
				state.EXPECT().GetCurrentValidator(constants.PrimaryNetworkID, nodeID).Return(nil, database.ErrNotFound)

				manager := &manager{
					backend: &backend{
						state: state,
						ctx:   consensustest.Context(t, ids.GenerateTestID()),
					},
					txExecutorBackend: &executor.Backend{
						Config: &config.Internal{
							UptimePercentage: 0,
						},
						Uptimes: &uptime.NoOpCalculator{},
					},
				Log: log.NoLog{},
				}

				return &Block{
					Block: &block.BanffProposalBlock{
						ApricotProposalBlock: block.ApricotProposalBlock{
							Tx: &txs.Tx{
								Unsigned: &txs.RewardValidatorTx{
									TxID: stakerTxID,
								},
							},
						},
					},
					manager: manager,
				}
			},
			expectedPreferenceType: &block.BanffCommitBlock{},
		},
		{
			name: "banff proposal block; failed calculating primary network uptime",
			blkF: func(ctrl *gomock.Controller) *Block {
				var (
					stakerTxID = ids.GenerateTestID()
					nodeID     = ids.GenerateTestNodeID()
					netID      = constants.PrimaryNetworkID
					stakerTx   = &txs.Tx{
						Unsigned: &txs.AddPermissionlessValidatorTx{
							Validator: txs.Validator{
								NodeID: nodeID,
							},
							Chain: netID,
						},
					}
					primaryNetworkValidatorStartTime = time.Now()
					staker                           = &state.Staker{
						StartTime: primaryNetworkValidatorStartTime,
					}
				)

				state := state.NewMockState(ctrl)
				state.EXPECT().GetTx(stakerTxID).Return(stakerTx, status.Committed, nil)
				state.EXPECT().GetCurrentValidator(constants.PrimaryNetworkID, nodeID).Return(staker, nil)

				// Note: NoOpCalculator doesn't need mocking, it always returns 100% uptime

				manager := &manager{
					backend: &backend{
						state: state,
						ctx:   consensustest.Context(t, ids.GenerateTestID()),
					},
					txExecutorBackend: &executor.Backend{
						Config: &config.Internal{
							UptimePercentage: 0,
						},
						Uptimes: &uptime.NoOpCalculator{},
					},
				Log: log.NoLog{},
				}

				return &Block{
					Block: &block.BanffProposalBlock{
						ApricotProposalBlock: block.ApricotProposalBlock{
							Tx: &txs.Tx{
								Unsigned: &txs.RewardValidatorTx{
									TxID: stakerTxID,
								},
							},
						},
					},
					manager: manager,
				}
			},
			expectedPreferenceType: &block.BanffCommitBlock{},
		},
		{
			name: "banff proposal block; failed fetching net transformation",
			blkF: func(ctrl *gomock.Controller) *Block {
				var (
					stakerTxID = ids.GenerateTestID()
					nodeID     = ids.GenerateTestNodeID()
					netID      = ids.GenerateTestID()
					stakerTx   = &txs.Tx{
						Unsigned: &txs.AddPermissionlessValidatorTx{
							Validator: txs.Validator{
								NodeID: nodeID,
							},
							Chain: netID,
						},
					}
					primaryNetworkValidatorStartTime = time.Now()
					staker                           = &state.Staker{
						StartTime: primaryNetworkValidatorStartTime,
					}
				)

				state := state.NewMockState(ctrl)
				state.EXPECT().GetTx(stakerTxID).Return(stakerTx, status.Committed, nil)
				state.EXPECT().GetCurrentValidator(constants.PrimaryNetworkID, nodeID).Return(staker, nil)
				state.EXPECT().GetNetTransformation(netID).Return(nil, database.ErrNotFound)

				manager := &manager{
					backend: &backend{
						state: state,
						ctx:   consensustest.Context(t, ids.GenerateTestID()),
					},
					txExecutorBackend: &executor.Backend{
						Config: &config.Internal{
							UptimePercentage: 0,
						},
						Uptimes: &uptime.NoOpCalculator{},
					},
				Log: log.NoLog{},
				}

				return &Block{
					Block: &block.BanffProposalBlock{
						ApricotProposalBlock: block.ApricotProposalBlock{
							Tx: &txs.Tx{
								Unsigned: &txs.RewardValidatorTx{
									TxID: stakerTxID,
								},
							},
						},
					},
					manager: manager,
				}
			},
			expectedPreferenceType: &block.BanffCommitBlock{},
		},
		{
			name: "banff proposal block; prefers commit",
			blkF: func(ctrl *gomock.Controller) *Block {
				var (
					stakerTxID = ids.GenerateTestID()
					nodeID     = ids.GenerateTestNodeID()
					netID      = ids.GenerateTestID()
					stakerTx   = &txs.Tx{
						Unsigned: &txs.AddPermissionlessValidatorTx{
							Validator: txs.Validator{
								NodeID: nodeID,
							},
							Chain: netID,
						},
					}
					primaryNetworkValidatorStartTime = time.Now()
					staker                           = &state.Staker{
						StartTime: primaryNetworkValidatorStartTime,
					}
					transformNetTx = &txs.Tx{
						Unsigned: &txs.TransformChainTx{
							UptimeRequirement: .2 * reward.PercentDenominator,
						},
					}
				)

				state := state.NewMockState(ctrl)
				state.EXPECT().GetTx(stakerTxID).Return(stakerTx, status.Committed, nil)
				state.EXPECT().GetCurrentValidator(constants.PrimaryNetworkID, nodeID).Return(staker, nil)
				state.EXPECT().GetNetTransformation(netID).Return(transformNetTx, nil)

				// Note: NoOpCalculator doesn't need mocking, it always returns 100% uptime

				manager := &manager{
					backend: &backend{
						state: state,
						ctx:   consensustest.Context(t, ids.GenerateTestID()),
					},
					txExecutorBackend: &executor.Backend{
						Config: &config.Internal{
							UptimePercentage: .8,
						},
						Uptimes: &uptime.NoOpCalculator{},
					},
				Log: log.NoLog{},
				}

				return &Block{
					Block: &block.BanffProposalBlock{
						ApricotProposalBlock: block.ApricotProposalBlock{
							Tx: &txs.Tx{
								Unsigned: &txs.RewardValidatorTx{
									TxID: stakerTxID,
								},
							},
						},
					},
					manager: manager,
				}
			},
			expectedPreferenceType: &block.BanffCommitBlock{},
		},
		{
			name: "banff proposal block; prefers abort",
			blkF: func(ctrl *gomock.Controller) *Block {
				var (
					stakerTxID = ids.GenerateTestID()
					nodeID     = ids.GenerateTestNodeID()
					netID      = ids.GenerateTestID()
					stakerTx   = &txs.Tx{
						Unsigned: &txs.AddPermissionlessValidatorTx{
							Validator: txs.Validator{
								NodeID: nodeID,
							},
							Chain: netID,
						},
					}
					primaryNetworkValidatorStartTime = time.Now()
					staker                           = &state.Staker{
						StartTime: primaryNetworkValidatorStartTime,
					}
					transformNetTx = &txs.Tx{
						Unsigned: &txs.TransformChainTx{
							UptimeRequirement: 1.01 * reward.PercentDenominator,
						},
					}
				)

				state := state.NewMockState(ctrl)
				state.EXPECT().GetTx(stakerTxID).Return(stakerTx, status.Committed, nil)
				state.EXPECT().GetCurrentValidator(constants.PrimaryNetworkID, nodeID).Return(staker, nil)
				state.EXPECT().GetNetTransformation(netID).Return(transformNetTx, nil)

				// Note: NoOpCalculator doesn't need mocking, it always returns 100% uptime

				manager := &manager{
					backend: &backend{
						state: state,
						ctx:   consensustest.Context(t, ids.GenerateTestID()),
					},
					txExecutorBackend: &executor.Backend{
						Config: &config.Internal{
							UptimePercentage: .8,
						},
						Uptimes: &uptime.NoOpCalculator{},
					},
				Log: log.NoLog{},
				}

				return &Block{
					Block: &block.BanffProposalBlock{
						ApricotProposalBlock: block.ApricotProposalBlock{
							Tx: &txs.Tx{
								Unsigned: &txs.RewardValidatorTx{
									TxID: stakerTxID,
								},
							},
						},
					},
					manager: manager,
				}
			},
			expectedPreferenceType: &block.BanffAbortBlock{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			require := require.New(t)

			blk := tt.blkF(ctrl)
			options, err := blk.Options(context.Background())
			require.NoError(err)
			require.IsType(tt.expectedPreferenceType, options[0].(*Block).Block)
		})
	}
}
