// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package core

import (
	"errors"
	"fmt"

	"github.com/luxfi/node/utils/set"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/log"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/evm/precompile/precompileconfig"
	"github.com/luxfi/evm/predicate"
)

var ErrMissingPredicateContext = errors.New("missing predicate context")

// CheckPredicates verifies the predicates of [tx] and returns the result. Returning an error invalidates the block.
func CheckPredicates(rules params.Rules, predicateContext *precompileconfig.PredicateContext, tx *types.Transaction) (map[common.Address][]byte, error) {
	// Check that the transaction can cover its IntrinsicGas (including the gas required by the predicate) before
	// verifying the predicate.
	intrinsicGas, err := IntrinsicGas(tx.Data(), tx.AccessList(), tx.To() == nil, rules)
	if err != nil {
		return nil, err
	}
	if tx.Gas() < intrinsicGas {
		return nil, fmt.Errorf("%w for predicate verification (%d) < intrinsic gas (%d)", ErrIntrinsicGas, tx.Gas(), intrinsicGas)
	}

	// TODO: Properly integrate predicates with extras.Rules
	// For now, return empty predicate results
	predicateResults := make(map[common.Address][]byte)
	return predicateResults, nil
}
