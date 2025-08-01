// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package allowlisttest

import (
	"testing"

	"github.com/luxfi/evm/precompile/allowlist"
	"github.com/luxfi/evm/precompile/modules"
)

var testModule = modules.Module{
	Address:      dummyAddr,
	Contract:     allowlist.CreateAllowListPrecompile(dummyAddr),
	Configurator: &dummyConfigurator{},
	ConfigKey:    "dummy",
}

func TestVerifyAllowlist(t *testing.T) {
	RunPrecompileWithAllowListTests(t, testModule, nil)
}

func TestEqualAllowList(t *testing.T) {
	EqualPrecompileWithAllowListTests(t, testModule, nil)
}
