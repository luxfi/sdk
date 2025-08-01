// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package allowlisttest

import (
	"testing"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/evm/precompile/allowlist"
	"github.com/luxfi/evm/precompile/contract"
	"github.com/luxfi/evm/precompile/modules"
	"github.com/luxfi/evm/precompile/precompileconfig"
)

var (
	_ precompileconfig.Config = (*dummyConfig)(nil)
	_ contract.Configurator   = (*dummyConfigurator)(nil)

	dummyAddr = common.Address{1}
)

type dummyConfig struct {
	precompileconfig.Upgrade
	allowlist.AllowListConfig
}

func (d *dummyConfig) Key() string      { return "dummy" }
func (d *dummyConfig) IsDisabled() bool { return false }
func (d *dummyConfig) Verify(chainConfig precompileconfig.ChainConfig) error {
	return d.AllowListConfig.Verify(chainConfig, d.Upgrade)
}

func (d *dummyConfig) Equal(cfg precompileconfig.Config) bool {
	other, ok := (cfg).(*dummyConfig)
	if !ok {
		return false
	}
	return d.AllowListConfig.Equal(&other.AllowListConfig)
}

type dummyConfigurator struct{}

func (d *dummyConfigurator) MakeConfig() precompileconfig.Config {
	return &dummyConfig{}
}

func (d *dummyConfigurator) Configure(
	chainConfig precompileconfig.ChainConfig,
	precompileConfig precompileconfig.Config,
	state contract.StateDB,
	blockContext contract.ConfigurationBlockContext,
) error {
	cfg := precompileConfig.(*dummyConfig)
	return cfg.AllowListConfig.Configure(chainConfig, dummyAddr, state, blockContext)
}

func TestAllowListRun(t *testing.T) {
	dummyModule := modules.Module{
		Address:      dummyAddr,
		Contract:     allowlist.CreateAllowListPrecompile(dummyAddr),
		Configurator: &dummyConfigurator{},
		ConfigKey:    "dummy",
	}
	RunPrecompileWithAllowListTests(t, dummyModule, nil)
}
