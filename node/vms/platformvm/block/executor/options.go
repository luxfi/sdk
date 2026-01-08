// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"errors"
	"fmt"

	"github.com/luxfi/consensus/validator/uptime"
	"github.com/luxfi/const"
	"github.com/luxfi/log"
	"github.com/luxfi/sdk/node/vms/platformvm/block"
	"github.com/luxfi/sdk/node/vms/platformvm/reward"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
	txexecutor "github.com/luxfi/sdk/node/vms/platformvm/txs/executor"
)

var (
	_ block.Visitor = (*options)(nil)

	ErrNotOracle                          = errors.New("block doesn't have options")
	errUnexpectedProposalTxType           = errors.New("unexpected proposal transaction type")
	errFailedFetchingStakerTx             = errors.New("failed fetching staker transaction")
	errUnexpectedStakerTxType             = errors.New("unexpected staker transaction type")
	errFailedFetchingPrimaryStaker        = errors.New("failed fetching primary staker")
	errFailedFetchingNetTransformation = errors.New("failed fetching net transformation")
	errFailedCalculatingUptime            = errors.New("failed calculating uptime")
)

// options supports build new option blocks
type options struct {
	// inputs populated before calling this struct's methods:
	log                     log.Logger
	primaryUptimePercentage float64
	uptimes                 uptime.Calculator
	state                   state.Chain

	// outputs populated by this struct's methods:
	preferredBlock block.Block
	alternateBlock block.Block
}

func (*options) BanffAbortBlock(*block.BanffAbortBlock) error {
	return ErrNotOracle
}

func (*options) BanffCommitBlock(*block.BanffCommitBlock) error {
	return ErrNotOracle
}

func (o *options) BanffProposalBlock(b *block.BanffProposalBlock) error {
	timestamp := b.Timestamp()
	blkID := b.ID()
	nextHeight := b.Height() + 1

	commitBlock, err := block.NewBanffCommitBlock(timestamp, blkID, nextHeight)
	if err != nil {
		return fmt.Errorf(
			"failed to create commit block: %w",
			err,
		)
	}

	abortBlock, err := block.NewBanffAbortBlock(timestamp, blkID, nextHeight)
	if err != nil {
		return fmt.Errorf(
			"failed to create abort block: %w",
			err,
		)
	}

	prefersCommit, err := o.prefersCommit(b.Tx)
	if err != nil {
		o.log.Debug("falling back to prefer commit",
			"error", err,
		)
		// We fall back to commit here to err on the side of over-rewarding
		// rather than under-rewarding.
		//
		// Invariant: We must not return the error here, because the error would
		// be treated as fatal. Errors can occur here due to a malicious block
		// proposer or even in unusual virtuous cases.
		prefersCommit = true
	}

	if prefersCommit {
		o.preferredBlock = commitBlock
		o.alternateBlock = abortBlock
	} else {
		o.preferredBlock = abortBlock
		o.alternateBlock = commitBlock
	}
	return nil
}

func (*options) BanffStandardBlock(*block.BanffStandardBlock) error {
	return ErrNotOracle
}

func (*options) ApricotAbortBlock(*block.ApricotAbortBlock) error {
	return ErrNotOracle
}

func (*options) ApricotCommitBlock(*block.ApricotCommitBlock) error {
	return ErrNotOracle
}

func (o *options) ApricotProposalBlock(b *block.ApricotProposalBlock) error {
	blkID := b.ID()
	nextHeight := b.Height() + 1

	var err error
	o.preferredBlock, err = block.NewApricotCommitBlock(blkID, nextHeight)
	if err != nil {
		return fmt.Errorf(
			"failed to create commit block: %w",
			err,
		)
	}

	o.alternateBlock, err = block.NewApricotAbortBlock(blkID, nextHeight)
	if err != nil {
		return fmt.Errorf(
			"failed to create abort block: %w",
			err,
		)
	}
	return nil
}

func (*options) ApricotStandardBlock(*block.ApricotStandardBlock) error {
	return ErrNotOracle
}

func (*options) ApricotAtomicBlock(*block.ApricotAtomicBlock) error {
	return ErrNotOracle
}

func (o *options) prefersCommit(tx *txs.Tx) (bool, error) {
	unsignedTx, ok := tx.Unsigned.(*txs.RewardValidatorTx)
	if !ok {
		return false, fmt.Errorf("%w: %T", errUnexpectedProposalTxType, tx.Unsigned)
	}

	stakerTx, _, err := o.state.GetTx(unsignedTx.TxID)
	if err != nil {
		return false, fmt.Errorf("%w: %w", errFailedFetchingStakerTx, err)
	}

	staker, ok := stakerTx.Unsigned.(txs.Staker)
	if !ok {
		return false, fmt.Errorf("%w: %T", errUnexpectedStakerTxType, stakerTx.Unsigned)
	}

	nodeID := staker.NodeID()
	primaryNetworkValidator, err := o.state.GetCurrentValidator(
		constants.PrimaryNetworkID,
		nodeID,
	)
	if err != nil {
		return false, fmt.Errorf("%w: %w", errFailedFetchingPrimaryStaker, err)
	}

	netID := staker.ChainID()
	expectedUptimePercentage := o.primaryUptimePercentage
	if netID != constants.PrimaryNetworkID {
		transformNet, err := txexecutor.GetTransformChainTx(o.state, netID)
		if err != nil {
			return false, fmt.Errorf("%w: %w", errFailedFetchingNetTransformation, err)
		}

		expectedUptimePercentage = float64(transformNet.UptimeRequirement) / reward.PercentDenominator
	}

	uptime, err := o.uptimes.CalculateUptimePercentFrom(
		nodeID,
		netID,
		primaryNetworkValidator.StartTime,
	)
	if err != nil {
		return false, fmt.Errorf("%w: %w", errFailedCalculatingUptime, err)
	}

	return uptime >= expectedUptimePercentage, nil
}
