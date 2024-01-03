// (c) 2021-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package allowlist

import (
	"testing"

	"github.com/luxdefi/evm/precompile/modules"
)

var testModule = modules.Module{
	Address:      dummyAddr,
	Contract:     CreateAllowListPrecompile(dummyAddr),
	Configurator: &dummyConfigurator{},
	ConfigKey:    "dummy",
}

func TestVerifyAllowlist(t *testing.T) {
	VerifyPrecompileWithAllowListTests(t, testModule, nil)
}

func TestEqualAllowList(t *testing.T) {
	EqualPrecompileWithAllowListTests(t, testModule, nil)
}
