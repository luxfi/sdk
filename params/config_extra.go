// (c) 2024 Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package params

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/luxfi/evm/params/extras"
	"github.com/luxfi/evm/precompile/precompileconfig"
	"github.com/luxfi/evm/utils"
)

const (
	// TODO: Value to pass to geth's Rules by default where the appropriate
	// context is not available in the lux code. (similar to context.TODO())
	IsMergeTODO = true
)

var (
	DefaultChainID = big.NewInt(43214)
	
	// Simple payloads system to store extra data per ChainConfig
	payloads = &chainConfigPayloads{
		extras: make(map[*ChainConfig]*extras.ChainConfig),
	}
)

type chainConfigPayloads struct {
	mu     sync.RWMutex
	extras map[*ChainConfig]*extras.ChainConfig
}

func (p *chainConfigPayloads) Get(c *ChainConfig) *extras.ChainConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.extras[c]
}

func (p *chainConfigPayloads) Set(c *ChainConfig, extra *extras.ChainConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.extras[c] = extra
}

// SetEthUpgrades enables Etheruem network upgrades using the same time as
// the Lux network upgrade that enables them.
//
// TODO: Prior to Cancun, Lux upgrades are referenced inline in the
// code in place of their Ethereum counterparts. The original Ethereum names
// should be restored for maintainability.
func SetEthUpgrades(c *ChainConfig, luxUpgrades extras.NetworkUpgrades) {
	if c.BerlinBlock == nil {
		c.BerlinBlock = big.NewInt(0)
	}
	if c.LondonBlock == nil {
		c.LondonBlock = big.NewInt(0)
	}
	if luxUpgrades.DurangoTimestamp != nil {
		c.ShanghaiTime = utils.NewUint64(*luxUpgrades.DurangoTimestamp)
	}
	if luxUpgrades.EtnaTimestamp != nil {
		c.CancunTime = utils.NewUint64(*luxUpgrades.EtnaTimestamp)
	}
}

func GetExtra(c *ChainConfig) *extras.ChainConfig {
	ex := payloads.Get(c)
	if ex == nil {
		ex = &extras.ChainConfig{}
		payloads.Set(c, ex)
	}
	return ex
}

func Copy(c *ChainConfig) ChainConfig {
	cpy := *c
	extraCpy := *GetExtra(c)
	return *WithExtra(&cpy, &extraCpy)
}

// WithExtra sets the extra payload on `c` and returns the modified argument.
func WithExtra(c *ChainConfig, extra *extras.ChainConfig) *ChainConfig {
	payloads.Set(c, extra)
	return c
}

func SetNetworkUpgradeDefaults(c *ChainConfig) {
	if c.HomesteadBlock == nil {
		c.HomesteadBlock = big.NewInt(0)
	}
	if c.EIP150Block == nil {
		c.EIP150Block = big.NewInt(0)
	}
	if c.EIP155Block == nil {
		c.EIP155Block = big.NewInt(0)
	}
	if c.EIP158Block == nil {
		c.EIP158Block = big.NewInt(0)
	}
	if c.ByzantiumBlock == nil {
		c.ByzantiumBlock = big.NewInt(0)
	}
	if c.ConstantinopleBlock == nil {
		c.ConstantinopleBlock = big.NewInt(0)
	}
	if c.PetersburgBlock == nil {
		c.PetersburgBlock = big.NewInt(0)
	}
	if c.IstanbulBlock == nil {
		c.IstanbulBlock = big.NewInt(0)
	}
	if c.MuirGlacierBlock == nil {
		c.MuirGlacierBlock = big.NewInt(0)
	}

	GetExtra(c).NetworkUpgrades.SetDefaults(GetExtra(c).ConsensusCtx.NetworkUpgrades)
}

// GetRulesExtra returns the Lux-specific rules for the given Ethereum rules.
func GetRulesExtra(rules Rules) *extras.Rules {
	// Create a ChainConfig from the Rules to get the extra data
	chainID := rules.ChainID
	if chainID == nil {
		chainID = DefaultChainID
	}
	
	// Create a minimal ChainConfig to get extras
	// This is a workaround since Rules doesn't have direct access to ChainConfig
	tempConfig := &ChainConfig{ChainID: chainID}
	extra := GetExtra(tempConfig)
	
	// Create rules based on the Lux upgrades
	return &extras.Rules{
		GenesisRules: extra.GetGenesisRules(0), // Using 0 as we don't have timestamp in Rules
		// Note: Precompiles, Predicaters, and AccepterPrecompiles are populated separately
		// by the caller since ethereum's Rules doesn't have these fields
		Precompiles: make(map[common.Address]precompileconfig.Config),
		Predicaters: make(map[common.Address]precompileconfig.Predicater),
		AccepterPrecompiles: make(map[common.Address]precompileconfig.Accepter),
	}
}
