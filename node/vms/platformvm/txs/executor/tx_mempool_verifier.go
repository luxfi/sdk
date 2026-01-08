// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package executor

import (
	"errors"
	"fmt"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/node/vms/platformvm/state"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
)

var (
	_ txs.Visitor = (*MempoolTxVerifier)(nil)

	ErrFutureStakeTime = errors.New("staker starts in the future")
)

type MempoolTxVerifier struct {
	*Backend
	ParentID      ids.ID
	StateVersions state.Versions
	Tx            *txs.Tx
}

func (*MempoolTxVerifier) AdvanceTimeTx(*txs.AdvanceTimeTx) error {
	return ErrWrongTxType
}

func (*MempoolTxVerifier) RewardValidatorTx(*txs.RewardValidatorTx) error {
	return ErrWrongTxType
}

func (v *MempoolTxVerifier) AddValidatorTx(tx *txs.AddValidatorTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) AddChainValidatorTx(tx *txs.AddChainValidatorTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) AddDelegatorTx(tx *txs.AddDelegatorTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) CreateChainTx(tx *txs.CreateChainTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) CreateSubnetTx(tx *txs.CreateSubnetTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) ImportTx(tx *txs.ImportTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) ExportTx(tx *txs.ExportTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) RemoveChainValidatorTx(tx *txs.RemoveChainValidatorTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) TransformChainTx(tx *txs.TransformChainTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) AddPermissionlessValidatorTx(tx *txs.AddPermissionlessValidatorTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) AddPermissionlessDelegatorTx(tx *txs.AddPermissionlessDelegatorTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) TransferChainOwnershipTx(tx *txs.TransferChainOwnershipTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) BaseTx(tx *txs.BaseTx) error {
	return v.standardTx(tx)
}

// Etna Transactions:
func (v *MempoolTxVerifier) ConvertChainToL1Tx(tx *txs.ConvertChainToL1Tx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) RegisterL1ValidatorTx(tx *txs.RegisterL1ValidatorTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) SetL1ValidatorWeightTx(tx *txs.SetL1ValidatorWeightTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) IncreaseL1ValidatorBalanceTx(tx *txs.IncreaseL1ValidatorBalanceTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) DisableL1ValidatorTx(tx *txs.DisableL1ValidatorTx) error {
	return v.standardTx(tx)
}

func (v *MempoolTxVerifier) standardTx(tx txs.UnsignedTx) error {
	baseState, err := v.standardBaseState()
	if err != nil {
		return err
	}

	executor := standardTxExecutor{
		backend: v.Backend,
		state:   baseState,
		tx:      v.Tx,
	}
	err = tx.Visit(&executor)
	// We ignore [ErrFutureStakeTime] here because the time will be advanced
	// when this transaction is issued.
	if errors.Is(err, ErrFutureStakeTime) {
		return nil
	}
	return err
}

func (v *MempoolTxVerifier) standardBaseState() (state.Diff, error) {
	state, err := state.NewDiff(v.ParentID, v.StateVersions)
	if err != nil {
		return nil, err
	}

	nextBlkTime, err := v.nextBlockTime(state)
	if err != nil {
		return nil, err
	}

	_, err = AdvanceTimeTo(v.Backend, state, nextBlkTime)
	if err != nil {
		return nil, err
	}
	state.SetTimestamp(nextBlkTime)

	return state, nil
}

func (v *MempoolTxVerifier) nextBlockTime(chainState state.Diff) (time.Time, error) {
	var (
		parentTime  = chainState.GetTimestamp()
		nextBlkTime = v.Clk.Time()
	)
	if parentTime.After(nextBlkTime) {
		nextBlkTime = parentTime
	}
	nextStakerChangeTime, err := state.GetNextStakerChangeTime(
		v.Backend.Config.ValidatorFeeConfig,
		chainState,
		nextBlkTime,
	)
	if err != nil {
		return time.Time{}, fmt.Errorf("could not calculate next staker change time: %w", err)
	}
	if !nextBlkTime.Before(nextStakerChangeTime) {
		nextBlkTime = nextStakerChangeTime
	}
	return nextBlkTime, nil
}
