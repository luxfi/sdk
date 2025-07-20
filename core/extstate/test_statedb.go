// (c) 2023, Hanzo Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package extstate

import (
	"testing"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/evm/core/state"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/evm/precompile/contract"
	"github.com/stretchr/testify/require"
)

func NewTestStateDB(t testing.TB) contract.StateDB {
	db := rawdb.NewMemoryDatabase()
	statedb, err := state.New(common.Hash{}, state.NewDatabase(db), nil)
	require.NoError(t, err)
	return New(statedb)
}
