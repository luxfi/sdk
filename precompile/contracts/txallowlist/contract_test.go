// (c) 2022-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package txallowlist

import (
	"testing"

	"github.com/luxdefi/evm/core/state"
	"github.com/luxdefi/evm/precompile/allowlist"
)

func TestTxAllowListRun(t *testing.T) {
	allowlist.RunPrecompileWithAllowListTests(t, Module, state.NewTestStateDB, nil)
}

func BenchmarkTxAllowList(b *testing.B) {
	allowlist.BenchPrecompileWithAllowList(b, Module, state.NewTestStateDB, nil)
}
