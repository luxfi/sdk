// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"github.com/luxfi/metric"

	"github.com/luxfi/sdk/node/vms/platformvm/txs"
)

const txLabel = "tx"

var (
	_ txs.Visitor = (*txMetrics)(nil)

	txLabels = []string{txLabel}
)

type txMetrics struct {
	addValidatorTxCounter metric.Counter
	addDelegatorTxCounter metric.Counter
	createChainTxCounter metric.Counter
	importTxCounter metric.Counter
	exportTxCounter metric.Counter
	advanceTimeTxCounter metric.Counter
	rewardValidatorTxCounter metric.Counter
	addPermissionlessValidatorTxCounter metric.Counter
	addPermissionlessDelegatorTxCounter metric.Counter
	baseTxCounter metric.Counter
	convertChainToL1TxCounter metric.Counter
	registerL1ValidatorTxCounter metric.Counter
	setL1ValidatorWeightTxCounter metric.Counter
	increaseL1ValidatorBalanceTxCounter metric.Counter
	disableL1ValidatorTxCounter metric.Counter
	addChainValidatorTxCounter metric.Counter
	createSubnetTxCounter metric.Counter
	removeChainValidatorTxCounter metric.Counter
	transformChainTxCounter metric.Counter
	transferChainOwnershipTxCounter metric.Counter
}

func newTxMetrics(registerer metric.Registerer) (*txMetrics, error) {
	m := &txMetrics{
		addValidatorTxCounter: metric.NewCounter("txs_accepted_add_validator", "number of add validator transactions accepted"),
		addDelegatorTxCounter: metric.NewCounter("txs_accepted_add_delegator", "number of add delegator transactions accepted"),
		createChainTxCounter: metric.NewCounter("txs_accepted_create_chain", "number of create chain transactions accepted"),
		importTxCounter: metric.NewCounter("txs_accepted_import", "number of import transactions accepted"),
		exportTxCounter: metric.NewCounter("txs_accepted_export", "number of export transactions accepted"),
		advanceTimeTxCounter: metric.NewCounter("txs_accepted_advance_time", "number of advance time transactions accepted"),
		rewardValidatorTxCounter: metric.NewCounter("txs_accepted_reward_validator", "number of reward validator transactions accepted"),
		addPermissionlessValidatorTxCounter: metric.NewCounter("txs_accepted_add_permissionless_validator", "number of add permissionless validator transactions accepted"),
		addPermissionlessDelegatorTxCounter: metric.NewCounter("txs_accepted_add_permissionless_delegator", "number of add permissionless delegator transactions accepted"),
		baseTxCounter: metric.NewCounter("txs_accepted_base", "number of base transactions accepted"),
		convertChainToL1TxCounter: metric.NewCounter("txs_accepted_convert_net_to_l1", "number of convert net to l1 transactions accepted"),
		registerL1ValidatorTxCounter: metric.NewCounter("txs_accepted_register_l1_validator", "number of register l1 validator transactions accepted"),
		setL1ValidatorWeightTxCounter: metric.NewCounter("txs_accepted_set_l1_validator_weight", "number of set l1 validator weight transactions accepted"),
		increaseL1ValidatorBalanceTxCounter: metric.NewCounter("txs_accepted_increase_l1_validator_balance", "number of increase l1 validator balance transactions accepted"),
		disableL1ValidatorTxCounter: metric.NewCounter("txs_accepted_disable_l1_validator", "number of disable l1 validator transactions accepted"),
		addChainValidatorTxCounter: metric.NewCounter("txs_accepted_add_net_validator", "number of add net validator transactions accepted"),
		createSubnetTxCounter: metric.NewCounter("txs_accepted_create_subnet", "number of create subnet transactions accepted"),
		removeChainValidatorTxCounter: metric.NewCounter("txs_accepted_remove_net_validator", "number of remove net validator transactions accepted"),
		transformChainTxCounter: metric.NewCounter("txs_accepted_transform_net", "number of transform net transactions accepted"),
		transferChainOwnershipTxCounter: metric.NewCounter("txs_accepted_transfer_net_ownership", "number of transfer net ownership transactions accepted"),
	}
	return m, nil
}

func (m *txMetrics) AddValidatorTx(*txs.AddValidatorTx) error {
	m.addValidatorTxCounter.Inc()
	return nil
}

// Removed in regenesis
// func (m *txMetrics) AddChainValidatorTx(*txs.AddChainValidatorTx) error {
// 	m.addSubnetValidatorTxCounter.Inc()
// 	return nil
// }

func (m *txMetrics) AddDelegatorTx(*txs.AddDelegatorTx) error {
	m.addDelegatorTxCounter.Inc()
	return nil
}

func (m *txMetrics) CreateChainTx(*txs.CreateChainTx) error {
	m.createChainTxCounter.Inc()
	return nil
}

// Removed in regenesis
// func (m *txMetrics) CreateNetTx(*txs.CreateNetTx) error {
// 	m.createSubnetTxCounter.Inc()
// 	return nil
// }

func (m *txMetrics) ImportTx(*txs.ImportTx) error {
	m.importTxCounter.Inc()
	return nil
}

func (m *txMetrics) ExportTx(*txs.ExportTx) error {
	m.exportTxCounter.Inc()
	return nil
}

func (m *txMetrics) AdvanceTimeTx(*txs.AdvanceTimeTx) error {
	m.advanceTimeTxCounter.Inc()
	return nil
}

func (m *txMetrics) RewardValidatorTx(*txs.RewardValidatorTx) error {
	m.rewardValidatorTxCounter.Inc()
	return nil
}

// Removed in regenesis
// func (m *txMetrics) RemoveChainValidatorTx(*txs.RemoveChainValidatorTx) error {
// 	m.removeSubnetValidatorTxCounter.Inc()
// 	return nil
// }

// Removed in regenesis
// func (m *txMetrics) TransformChainTx(*txs.TransformChainTx) error {
// 	m.transformSubnetTxCounter.Inc()
// 	return nil
// }

func (m *txMetrics) AddPermissionlessValidatorTx(*txs.AddPermissionlessValidatorTx) error {
	m.addPermissionlessValidatorTxCounter.Inc()
	return nil
}

func (m *txMetrics) AddPermissionlessDelegatorTx(*txs.AddPermissionlessDelegatorTx) error {
	m.addPermissionlessDelegatorTxCounter.Inc()
	return nil
}

// Removed in regenesis
// func (m *txMetrics) TransferChainOwnershipTx(*txs.TransferChainOwnershipTx) error {
// 	m.transferSubnetOwnershipTxCounter.Inc()
// 	return nil
// }

func (m *txMetrics) BaseTx(*txs.BaseTx) error {
	m.baseTxCounter.Inc()
	return nil
}

func (m *txMetrics) ConvertChainToL1Tx(*txs.ConvertChainToL1Tx) error {
	m.convertChainToL1TxCounter.Inc()
	return nil
}

func (m *txMetrics) RegisterL1ValidatorTx(*txs.RegisterL1ValidatorTx) error {
	m.registerL1ValidatorTxCounter.Inc()
	return nil
}

func (m *txMetrics) SetL1ValidatorWeightTx(*txs.SetL1ValidatorWeightTx) error {
	m.setL1ValidatorWeightTxCounter.Inc()
	return nil
}

func (m *txMetrics) IncreaseL1ValidatorBalanceTx(*txs.IncreaseL1ValidatorBalanceTx) error {
	m.increaseL1ValidatorBalanceTxCounter.Inc()
	return nil
}

func (m *txMetrics) DisableL1ValidatorTx(*txs.DisableL1ValidatorTx) error {
	m.disableL1ValidatorTxCounter.Inc()
	return nil
}

func (m *txMetrics) AddChainValidatorTx(*txs.AddChainValidatorTx) error {
	m.addChainValidatorTxCounter.Inc()
	return nil
}

func (m *txMetrics) CreateSubnetTx(*txs.CreateSubnetTx) error {
	m.createSubnetTxCounter.Inc()
	return nil
}

func (m *txMetrics) RemoveChainValidatorTx(*txs.RemoveChainValidatorTx) error {
	m.removeChainValidatorTxCounter.Inc()
	return nil
}

func (m *txMetrics) TransformChainTx(*txs.TransformChainTx) error {
	m.transformChainTxCounter.Inc()
	return nil
}

func (m *txMetrics) TransferChainOwnershipTx(*txs.TransferChainOwnershipTx) error {
	m.transferChainOwnershipTxCounter.Inc()
	return nil
}
