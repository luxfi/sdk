// (c) 2019-2023, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package allowlist_test

import (
	"testing"
	"github.com/luxfi/evm/precompile/modules"
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
