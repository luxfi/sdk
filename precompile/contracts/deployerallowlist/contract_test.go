// (c) 2019-2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package deployerallowlist_test

import (
	"testing"
	"github.com/luxdefi/evm/core/state"
	"github.com/luxdefi/evm/precompile/allowlist"
)

func TestContractDeployerAllowListRun(t *testing.T) {
	allowlist.RunPrecompileWithAllowListTests(t, Module, extstate.NewTestStateDB, nil)
}

func BenchmarkContractDeployerAllowList(b *testing.B) {
	allowlist.BenchPrecompileWithAllowList(b, Module, extstate.NewTestStateDB, nil)
}
