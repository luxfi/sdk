// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package fee

import (
	"github.com/luxfi/const"
	"github.com/luxfi/sdk/node/vms/platformvm/txs"
)

var (
	_ Calculator  = (*staticCalculator)(nil)
	_ txs.Visitor = (*staticVisitor)(nil)
)

func NewSimpleStaticCalculator(config StaticConfig) Calculator {
	return &staticCalculator{
		config: config,
	}
}

type staticCalculator struct {
	config StaticConfig
}

func (c *staticCalculator) CalculateFee(tx txs.UnsignedTx) (uint64, error) {
	v := staticVisitor{
		config: c.config,
	}
	err := tx.Visit(&v)
	return v.fee, err
}

type staticVisitor struct {
	// inputs
	config StaticConfig

	// outputs
	fee uint64
}

func (*staticVisitor) AdvanceTimeTx(*txs.AdvanceTimeTx) error {
	return ErrUnsupportedTx
}

func (*staticVisitor) RewardValidatorTx(*txs.RewardValidatorTx) error {
	return ErrUnsupportedTx
}

func (c *staticVisitor) AddValidatorTx(*txs.AddValidatorTx) error {
	c.fee = c.config.AddPrimaryNetworkValidatorFee
	return nil
}

// Removed in regenesis
// func (c *staticVisitor) AddChainValidatorTx(*txs.AddChainValidatorTx) error {
// 	c.fee = c.config.AddNetValidatorFee
// 	return nil
// }

func (c *staticVisitor) AddDelegatorTx(*txs.AddDelegatorTx) error {
	c.fee = c.config.AddPrimaryNetworkDelegatorFee
	return nil
}

func (c *staticVisitor) CreateChainTx(*txs.CreateChainTx) error {
	c.fee = c.config.CreateBlockchainTxFee
	return nil
}

// Removed in regenesis
// func (c *staticVisitor) CreateNetTx(*txs.CreateNetTx) error {
// 	c.fee = c.config.CreateNetTxFee
// 	return nil
// }

// Removed in regenesis
// func (c *staticVisitor) RemoveChainValidatorTx(*txs.RemoveChainValidatorTx) error {
// 	c.fee = c.config.TxFee
// 	return nil
// }

// Removed in regenesis
// func (c *staticVisitor) TransformChainTx(*txs.TransformChainTx) error {
// 	c.fee = c.config.TransformChainTxFee
// 	return nil
// }

// Removed in regenesis
// func (c *staticVisitor) TransferChainOwnershipTx(*txs.TransferChainOwnershipTx) error {
// 	c.fee = c.config.TxFee
// 	return nil
// }

func (c *staticVisitor) AddPermissionlessValidatorTx(tx *txs.AddPermissionlessValidatorTx) error {
	if tx.Chain != constants.PrimaryNetworkID {
		c.fee = c.config.TxFee // Use TxFee since AddChainValidatorFee was removed in regenesis
	} else {
		c.fee = c.config.AddPrimaryNetworkValidatorFee
	}
	return nil
}

func (c *staticVisitor) AddPermissionlessDelegatorTx(tx *txs.AddPermissionlessDelegatorTx) error {
	if tx.Chain != constants.PrimaryNetworkID {
		c.fee = c.config.TxFee // Use TxFee since AddChainDelegatorFee was removed in regenesis
	} else {
		c.fee = c.config.AddPrimaryNetworkDelegatorFee
	}
	return nil
}

func (c *staticVisitor) BaseTx(*txs.BaseTx) error {
	c.fee = c.config.TxFee
	return nil
}

func (c *staticVisitor) ImportTx(*txs.ImportTx) error {
	c.fee = c.config.TxFee
	return nil
}

func (c *staticVisitor) ExportTx(*txs.ExportTx) error {
	c.fee = c.config.TxFee
	return nil
}

func (*staticVisitor) ConvertChainToL1Tx(*txs.ConvertChainToL1Tx) error {
	return ErrUnsupportedTx
}

func (*staticVisitor) DisableL1ValidatorTx(*txs.DisableL1ValidatorTx) error {
	return ErrUnsupportedTx
}

func (*staticVisitor) IncreaseL1ValidatorBalanceTx(*txs.IncreaseL1ValidatorBalanceTx) error {
	return ErrUnsupportedTx
}

func (*staticVisitor) RegisterL1ValidatorTx(*txs.RegisterL1ValidatorTx) error {
	return ErrUnsupportedTx
}

func (*staticVisitor) SetL1ValidatorWeightTx(*txs.SetL1ValidatorWeightTx) error {
	return ErrUnsupportedTx
}

func (v *staticVisitor) AddChainValidatorTx(*txs.AddChainValidatorTx) error {
	v.fee = v.config.AddSubnetValidatorFee
	return nil
}

func (v *staticVisitor) CreateSubnetTx(*txs.CreateSubnetTx) error {
	v.fee = v.config.CreateSubnetTxFee
	return nil
}

func (v *staticVisitor) RemoveChainValidatorTx(*txs.RemoveChainValidatorTx) error {
	v.fee = v.config.TxFee
	return nil
}

func (v *staticVisitor) TransformChainTx(*txs.TransformChainTx) error {
	v.fee = v.config.TransformChainTxFee
	return nil
}

func (v *staticVisitor) TransferChainOwnershipTx(*txs.TransferChainOwnershipTx) error {
	v.fee = v.config.TxFee
	return nil
}
