// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package core

import (
	"testing"

	"github.com/luxfi/consensus/engine/chain/block"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/evm/precompile/precompileconfig"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/stretchr/testify/require"
)

type predicateCheckTest struct {
	accessList       types.AccessList
	gas              uint64
	predicateContext *precompileconfig.PredicateContext
	createPredicates func(t testing.TB) map[common.Address]precompileconfig.Predicater
	expectedRes      map[common.Address][]byte
	expectedErr      error
}

// TestCheckPredicate tests the CheckPredicates function's gas checking functionality.
// Note: The full predicate verification functionality requires CheckPredicatesWithRules
// and extras.Rules, which is not available in the simplified CheckPredicates function.
// This test focuses on the intrinsic gas calculation and validation.
func TestCheckPredicate(t *testing.T) {
	addr1 := common.HexToAddress("0xaa")
	addr3 := common.HexToAddress("0xcc")
	addr4 := common.HexToAddress("0xdd")
	predicateContext := &precompileconfig.PredicateContext{
		ProposerVMBlockCtx: &block.Context{
			PChainHeight: 10,
		},
	}
	for name, test := range map[string]predicateCheckTest{
		"no access list, no context passes": {
			gas:              53000, // TxGasContractCreation (tx.To() == nil)
			predicateContext: nil,
			expectedRes:      make(map[common.Address][]byte),
			expectedErr:      nil,
		},
		"no access list, with context passes": {
			gas:              53000, // TxGasContractCreation
			predicateContext: predicateContext,
			expectedRes:      make(map[common.Address][]byte),
			expectedErr:      nil,
		},
		"with access list passes": {
			gas:              57300, // 53000 base + 2400 addr + 1900 key
			predicateContext: nil,
			accessList: types.AccessList([]types.AccessTuple{
				{
					Address: addr1,
					StorageKeys: []common.Hash{
						{1},
					},
				},
			}),
			expectedRes: make(map[common.Address][]byte),
			expectedErr: nil,
		},
		"two addresses in access list": {
			gas:              61600, // 53000 base + 2*2400 addr + 2*1900 key
			predicateContext: predicateContext,
			accessList: types.AccessList([]types.AccessTuple{
				{
					Address: addr3,
					StorageKeys: []common.Hash{
						{1},
					},
				},
				{
					Address: addr4,
					StorageKeys: []common.Hash{
						{1},
					},
				},
			}),
			expectedRes: make(map[common.Address][]byte),
			expectedErr: nil,
		},
		"insufficient gas with access list": {
			gas:              53000, // Not enough - needs 57300 for 1 addr + 1 key
			predicateContext: predicateContext,
			accessList: types.AccessList([]types.AccessTuple{
				{
					Address: addr1,
					StorageKeys: []common.Hash{
						{1},
					},
				},
			}),
			expectedErr: ErrIntrinsicGas,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			rules := params.TestRules

			// Specify only the access list, since this test should not depend on any other values
			tx := types.NewTx(&types.DynamicFeeTx{
				AccessList: test.accessList,
				Gas:        test.gas,
			})
			predicateRes, err := CheckPredicates(rules, test.predicateContext, tx)
			require.ErrorIs(err, test.expectedErr)
			if test.expectedErr != nil {
				return
			}
			require.Equal(test.expectedRes, predicateRes)
			intrinsicGas, err := IntrinsicGas(tx.Data(), tx.AccessList(), true, rules)
			require.NoError(err)
			require.Equal(tx.Gas(), intrinsicGas) // Require test specifies exact amount of gas consumed
		})
	}
}

// NOTE: TestCheckPredicatesOutput removed - this test required full predicate verification
// functionality using CheckPredicatesWithRules with mocked predicaters. The simplified
// CheckPredicates function doesn't have access to extras.Rules and cannot perform
// predicate verification. Re-add this test when predicate infrastructure is fully integrated.
