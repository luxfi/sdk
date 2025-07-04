// (c) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package extstate

import (
	"testing"
<<<<<<< HEAD:core/state/test_statedb.go
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/luxdefi/evm/precompile/contract"
	"github.com/ethereum/go-ethereum/common"
=======
	"github.com/luxdefi/evm/interfaces/core/rawdb"
	"github.com/luxdefi/evm/core/state"
>>>>>>> v0.7.5:core/extstate/test_statedb.go
	"github.com/stretchr/testify/require"
)

func NewTestStateDB(t testing.TB) contract.StateDB {
	db := rawdb.NewMemoryDatabase()
	statedb, err := state.New(common.Hash{}, state.NewDatabase(db), nil)
	require.NoError(t, err)
	return New(statedb)
}
