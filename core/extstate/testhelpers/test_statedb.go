// (c) 2023, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package testhelpers

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/luxfi/evm/core/state"
	"github.com/luxfi/evm/core/rawdb"
	"github.com/luxfi/evm/core/extstate"
	"github.com/luxfi/evm/precompile/contract"
	"github.com/stretchr/testify/require"
)

func NewTestStateDB(t testing.TB) contract.StateDB {
	db := rawdb.NewMemoryDatabase()
	statedb, err := state.New(common.Hash{}, state.NewDatabase(db), nil)
	require.NoError(t, err)
	return extstate.New(statedb)
}