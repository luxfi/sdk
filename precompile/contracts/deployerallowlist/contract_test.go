// (c) 2021-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package deployerallowlist

import (
	"testing"

	"github.com/luxdefi/evm/core/state"
	"github.com/luxdefi/evm/precompile/allowlist"
)

func TestContractDeployerAllowListRun(t *testing.T) {
	allowlist.RunPrecompileWithAllowListTests(t, Module, state.NewTestStateDB, nil)
}

func BenchmarkContractDeployerAllowList(b *testing.B) {
	allowlist.BenchPrecompileWithAllowList(b, Module, state.NewTestStateDB, nil)
}
