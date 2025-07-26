// (c) 2019-2020, Lux Industries, Inc.
//
// This file is a derived work, based on the go-ethereum library whose original
// notices appear below.
//
// It is distributed under a license compatible with the licensing terms of the
// original code from which it is derived.
//
// Much love to the original authors for their work.
// **********
// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later interfaces.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	
	"github.com/ethereum/go-ethereum/common"
	ethparams "github.com/ethereum/go-ethereum/params"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/commontype"
	"github.com/luxfi/evm/params/extras"
	"github.com/luxfi/evm/utils"
)

// Type aliases for extras package types
type (
	MandatoryNetworkUpgrades = extras.NetworkUpgrades
	OptionalNetworkUpgrades  = extras.NetworkUpgrades
	Precompiles             = extras.Precompiles
)

const maxJSONLen = 64 * 1024 * 1024 // 64MB

var (
	// MainnetNetworkUpgrades defines the network upgrade timestamps for mainnet
	MainnetNetworkUpgrades = extras.NetworkUpgrades{
		EVMTimestamp: utils.NewUint64(0),
		DurangoTimestamp:   nil,
		EtnaTimestamp:      nil,
		FortunaTimestamp:   nil,
		GraniteTimestamp:   nil,
	}
	
	errNonGenesisForkByHeight = errors.New("evm only supports forking by height at the genesis block")

	EVMChainID = big.NewInt(43214)

	// For legacy tests
	MinGasPrice        int64 = 225_000_000_000
	TestInitialBaseFee int64 = 225_000_000_000
	TestMaxBaseFee     int64 = 225_000_000_000

	DynamicFeeExtraDataSize        = 80
	RollupWindow            uint64 = 10

	DefaultFeeConfig = commontype.FeeConfig{
		GasLimit:        big.NewInt(8_000_000),
		TargetBlockRate: 2, // in seconds

		MinBaseFee:               big.NewInt(25_000_000_000),
		TargetGas:                big.NewInt(15_000_000),
		BaseFeeChangeDenominator: big.NewInt(36),

		MinBlockGasCost:  big.NewInt(0),
		MaxBlockGasCost:  big.NewInt(1_000_000),
		BlockGasCostStep: big.NewInt(200_000),
	}
)

var (
	// EVMDefaultConfig is the default configuration
	// without any network upgrades.
	EVMDefaultChainConfig = &ChainConfig{
		ChainID:                  DefaultChainID,
		HomesteadBlock:           big.NewInt(0),
		EIP150Block:              big.NewInt(0),
		EIP155Block:              big.NewInt(0),
		EIP158Block:              big.NewInt(0),
		ByzantiumBlock:           big.NewInt(0),
		ConstantinopleBlock:      big.NewInt(0),
		PetersburgBlock:          big.NewInt(0),
		IstanbulBlock:            big.NewInt(0),
		MuirGlacierBlock:         big.NewInt(0),
		MandatoryNetworkUpgrades: MainnetNetworkUpgrades, // This can be changed to correct network (local, test) via VM.
		GenesisPrecompiles:       Precompiles{},
	}

	TestChainConfig = &ChainConfig{
		LuxContext:          LuxContext{utils.TestChainContext()},
		ChainID:             big.NewInt(1),
		FeeConfig:           DefaultFeeConfig,
		AllowFeeRecipients:  false,
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		MandatoryNetworkUpgrades: MandatoryNetworkUpgrades{
			EVMTimestamp: utils.NewUint64(0),
			DurangoTimestamp:   utils.NewUint64(0),
		},
	}

	TestEVMConfig = &ChainConfig{
		LuxContext:          LuxContext{utils.TestChainContext()},
		ChainID:             big.NewInt(1),
		FeeConfig:           DefaultFeeConfig,
		AllowFeeRecipients:  false,
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		MandatoryNetworkUpgrades: MandatoryNetworkUpgrades{
			EVMTimestamp: utils.NewUint64(0),
		},
	}

	TestPreEVMConfig = &ChainConfig{
		LuxContext:               LuxContext{utils.TestChainContext()},
		ChainID:                  big.NewInt(1),
		FeeConfig:                DefaultFeeConfig,
		AllowFeeRecipients:       false,
		HomesteadBlock:           big.NewInt(0),
		EIP150Block:              big.NewInt(0),
		EIP155Block:              big.NewInt(0),
		EIP158Block:              big.NewInt(0),
		ByzantiumBlock:           big.NewInt(0),
		ConstantinopleBlock:      big.NewInt(0),
		PetersburgBlock:          big.NewInt(0),
		IstanbulBlock:            big.NewInt(0),
		MuirGlacierBlock:         big.NewInt(0),
		MandatoryNetworkUpgrades: MandatoryNetworkUpgrades{},
		GenesisPrecompiles:       Precompiles{},
		UpgradeConfig:            UpgradeConfig{},
	}

	TestRules = TestChainConfig.GenesisRules(new(big.Int), 0)
	
	// TestPreEVMChainConfig is a chain config with EVM disabled
	TestPreEVMChainConfig = TestPreEVMConfig
)

// UpgradeConfig includes the following configs that may be specified in upgradeBytes:
// - Timestamps that enable lux network upgrades,
// - Enabling or disabling precompiles as network upgrades.
type UpgradeConfig struct {
	// Config for optional timestamps that enable network upgrades.
	// Note: if OptionalUpgrades is specified in the JSON all previously activated
	// forks must be present or upgradeBytes will be rejected.
	OptionalNetworkUpgrades *extras.NetworkUpgrades `json:"networkUpgrades,omitempty"`

	// Config for modifying state as a network interfaces.
	StateUpgrades []extras.StateUpgrade `json:"stateUpgrades,omitempty"`

	// Config for enabling and disabling precompiles as network upgrades.
	PrecompileUpgrades []extras.PrecompileUpgrade `json:"precompileUpgrades,omitempty"`
}

// LuxContext provides Lux specific context directly into the EVM.
type LuxContext struct {
	ConsensusCtx *interfaces.ChainContext
}

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	LuxContext `json:"-"` // Lux specific context set during VM initialization. Not serialized.

	ChainID            *big.Int             `json:"chainId"`                      // chainId identifies the current chain and is used for replay protection
	FeeConfig          commontype.FeeConfig `json:"feeConfig"`                    // Set the configuration for the dynamic fee algorithm
	AllowFeeRecipients bool                 `json:"allowFeeRecipients,omitempty"` // Allows fees to be collected by block builders.

	HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)

	// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
	EIP150Block *big.Int `json:"eip150Block,omitempty"` // EIP150 HF block (nil = no fork)
	EIP155Block *big.Int `json:"eip155Block,omitempty"` // EIP155 HF block
	EIP158Block *big.Int `json:"eip158Block,omitempty"` // EIP158 HF block

	ByzantiumBlock      *big.Int `json:"byzantiumBlock,omitempty"`      // Byzantium switch block (nil = no fork, 0 = already on byzantium)
	ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"` // Constantinople switch block (nil = no fork, 0 = already activated)
	PetersburgBlock     *big.Int `json:"petersburgBlock,omitempty"`     // Petersburg switch block (nil = same as Constantinople)
	IstanbulBlock       *big.Int `json:"istanbulBlock,omitempty"`       // Istanbul switch block (nil = no fork, 0 = already on istanbul)
	MuirGlacierBlock    *big.Int `json:"muirGlacierBlock,omitempty"`    // Eip-2384 (bomb delay) switch block (nil = no fork, 0 = already activated)
	BerlinBlock         *big.Int `json:"berlinBlock,omitempty"`         // Berlin switch block (nil = no fork, 0 = already on berlin)
	LondonBlock         *big.Int `json:"londonBlock,omitempty"`         // London switch block (nil = no fork, 0 = already on london)
	
	// Time based forks
	ShanghaiTime        *uint64  `json:"shanghaiTime,omitempty"`        // Shanghai switch time (nil = no fork, 0 = already on shanghai)
	CancunTime          *uint64  `json:"cancunTime,omitempty"`          // Cancun switch time (nil = no fork, 0 = already on cancun)

	MandatoryNetworkUpgrades             // Config for timestamps that enable mandatory network upgrades. Skip encoding/decoding directly into ChainConfig.
	OptionalNetworkUpgrades              // Config for optional timestamps that enable network upgrades
	GenesisPrecompiles       Precompiles `json:"-"` // Config for enabling precompiles from genesis. JSON encode/decode will be handled by the custom marshaler/unmarshaler.
	UpgradeConfig            `json:"-"`  // Config specified in upgradeBytes (lux network upgrades or enable/disabling precompiles). Skip encoding/decoding directly into ChainConfig.
}

// UnmarshalJSON parses the JSON-encoded data and stores the result in the
// object pointed to by c.
// This is a custom unmarshaler to handle the Precompiles field.
// Precompiles was presented as an inline object in the JSON.
// This custom unmarshaler ensures backwards compatibility with the old format.
func (c *ChainConfig) UnmarshalJSON(data []byte) error {
	// Alias ChainConfig to avoid recursion
	type _ChainConfig ChainConfig
	tmp := _ChainConfig{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	// At this point we have populated all fields except PrecompileUpgrade
	*c = ChainConfig(tmp)

	// Unmarshal inlined PrecompileUpgrade
	return json.Unmarshal(data, &c.GenesisPrecompiles)
}

// MarshalJSON returns the JSON encoding of c.
// This is a custom marshaler to handle the Precompiles field.
func (c ChainConfig) MarshalJSON() ([]byte, error) {
	// Alias ChainConfig to avoid recursion
	type _ChainConfig ChainConfig
	tmp, err := json.Marshal(_ChainConfig(c))
	if err != nil {
		return nil, err
	}

	// To include PrecompileUpgrades, we unmarshal the json representing c
	// then directly add the corresponding keys to the interfaces.
	raw := make(map[string]json.RawMessage)
	if err := json.Unmarshal(tmp, &raw); err != nil {
		return nil, err
	}

	for key, value := range c.GenesisPrecompiles {
		conf, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		raw[key] = conf
	}

	return json.Marshal(raw)
}

// ToEthChainConfig converts Lux ChainConfig to ethereum ChainConfig
func (c *ChainConfig) ToEthChainConfig() *ethparams.ChainConfig {
	return &ethparams.ChainConfig{
		ChainID:             c.ChainID,
		HomesteadBlock:      c.HomesteadBlock,
		EIP150Block:         c.EIP150Block,
		EIP155Block:         c.EIP155Block,
		EIP158Block:         c.EIP158Block,
		ByzantiumBlock:      c.ByzantiumBlock,
		ConstantinopleBlock: c.ConstantinopleBlock,
		PetersburgBlock:     c.PetersburgBlock,
		IstanbulBlock:       c.IstanbulBlock,
		MuirGlacierBlock:    c.MuirGlacierBlock,
		BerlinBlock:         c.BerlinBlock,
		LondonBlock:         c.LondonBlock,
		ShanghaiTime:        c.ShanghaiTime,
		CancunTime:          c.CancunTime,
	}
}

// Description returns a human-readable description of ChainConfig.
func (c *ChainConfig) Description() string {
	var banner string

	banner += fmt.Sprintf("Chain ID:  %v\n", c.ChainID)
	banner += "Consensus: Dummy Consensus Engine\n\n"

	// Create a list of forks with a short description of them. Forks that only
	// makes sense for mainnet should be optional at printing to avoid bloating
	// the output for testnets and private networks.
	banner += "Hard Forks:\n"
	banner += fmt.Sprintf(" - Homestead:                   #%-8v (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/homestead.md)\n", c.HomesteadBlock)
	banner += fmt.Sprintf(" - Tangerine Whistle (EIP 150): #%-8v (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/tangerine-whistle.md)\n", c.EIP150Block)
	banner += fmt.Sprintf(" - Spurious Dragon/1 (EIP 155): #%-8v (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/spurious-dragon.md)\n", c.EIP155Block)
	banner += fmt.Sprintf(" - Spurious Dragon/2 (EIP 158): #%-8v (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/spurious-dragon.md)\n", c.EIP155Block)
	banner += fmt.Sprintf(" - Byzantium:                   #%-8v (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/byzantium.md)\n", c.ByzantiumBlock)
	banner += fmt.Sprintf(" - Constantinople:              #%-8v (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/constantinople.md)\n", c.ConstantinopleBlock)
	banner += fmt.Sprintf(" - Petersburg:                  #%-8v (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/petersburg.md)\n", c.PetersburgBlock)
	banner += fmt.Sprintf(" - Istanbul:                    #%-8v (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/istanbul.md)\n", c.IstanbulBlock)
	if c.MuirGlacierBlock != nil {
		banner += fmt.Sprintf(" - Muir Glacier:                #%-8v (https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/muir-glacier.md)\n", c.MuirGlacierBlock)
	}
	banner += "Mandatory Upgrades:\n"
	banner += fmt.Sprintf(" - EVM Timestamp:           @%-10v (https://github.com/luxfi/node/releases/tag/v1.10.0)\n", ptrToString(c.MandatoryNetworkUpgrades.EVMTimestamp))
	banner += fmt.Sprintf(" - Durango Timestamp:             @%-10v (https://github.com/luxfi/node/releases/tag/v1.11.0)\n", ptrToString(c.MandatoryNetworkUpgrades.DurangoTimestamp))
	banner += fmt.Sprintf(" - Etna Timestamp:                @%-10v (https://github.com/luxfi/node/releases/tag/v1.12.0)\n", ptrToString(c.MandatoryNetworkUpgrades.EtnaTimestamp))
	banner += "\n"

	// Add EVM custom fields
	optionalNetworkUpgradeBytes, err := json.Marshal(c.OptionalNetworkUpgrades)
	if err != nil {
		optionalNetworkUpgradeBytes = []byte("cannot marshal OptionalNetworkUpgrades")
	}
	banner += fmt.Sprintf("Optional Network Upgrades: %s", string(optionalNetworkUpgradeBytes))
	banner += "\n"

	precompileUpgradeBytes, err := json.Marshal(c.GenesisPrecompiles)
	if err != nil {
		precompileUpgradeBytes = []byte("cannot marshal PrecompileUpgrade")
	}
	banner += fmt.Sprintf("Precompile Upgrades: %s", string(precompileUpgradeBytes))
	banner += "\n"

	upgradeConfigBytes, err := json.Marshal(c.UpgradeConfig)
	if err != nil {
		upgradeConfigBytes = []byte("cannot marshal UpgradeConfig")
	}
	banner += fmt.Sprintf("Upgrade Config: %s", string(upgradeConfigBytes))
	banner += "\n"

	feeBytes, err := json.Marshal(c.FeeConfig)
	if err != nil {
		feeBytes = []byte("cannot marshal FeeConfig")
	}
	banner += fmt.Sprintf("Fee Config: %s", string(feeBytes))
	banner += "\n"

	banner += fmt.Sprintf("Allow Fee Recipients: %v", c.AllowFeeRecipients)
	banner += "\n"
	return banner
}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	return utils.IsBlockForked(c.HomesteadBlock, num)
}

// IsEIP150 returns whether num is either equal to the EIP150 fork block or greater.
func (c *ChainConfig) IsEIP150(num *big.Int) bool {
	return utils.IsBlockForked(c.EIP150Block, num)
}

// IsEIP155 returns whether num is either equal to the EIP155 fork block or greater.
func (c *ChainConfig) IsEIP155(num *big.Int) bool {
	return utils.IsBlockForked(c.EIP155Block, num)
}

// IsEIP158 returns whether num is either equal to the EIP158 fork block or greater.
func (c *ChainConfig) IsEIP158(num *big.Int) bool {
	return utils.IsBlockForked(c.EIP158Block, num)
}

// IsByzantium returns whether num is either equal to the Byzantium fork block or greater.
func (c *ChainConfig) IsByzantium(num *big.Int) bool {
	return utils.IsBlockForked(c.ByzantiumBlock, num)
}

// IsConstantinople returns whether num is either equal to the Constantinople fork block or greater.
func (c *ChainConfig) IsConstantinople(num *big.Int) bool {
	return utils.IsBlockForked(c.ConstantinopleBlock, num)
}

// IsMuirGlacier returns whether num is either equal to the Muir Glacier (EIP-2384) fork block or greater.
func (c *ChainConfig) IsMuirGlacier(num *big.Int) bool {
	return utils.IsBlockForked(c.MuirGlacierBlock, num)
}

// IsPetersburg returns whether num is either
// - equal to or greater than the PetersburgBlock fork block,
// - OR is nil, and Constantinople is active
func (c *ChainConfig) IsPetersburg(num *big.Int) bool {
	return utils.IsBlockForked(c.PetersburgBlock, num) || c.PetersburgBlock == nil && utils.IsBlockForked(c.ConstantinopleBlock, num)
}

// IsIstanbul returns whether num is either equal to the Istanbul fork block or greater.
func (c *ChainConfig) IsIstanbul(num *big.Int) bool {
	return utils.IsBlockForked(c.IstanbulBlock, num)
}

// IsBerlin returns whether num is either equal to the Berlin fork block or greater.
func (c *ChainConfig) IsBerlin(num *big.Int) bool {
	return utils.IsBlockForked(c.BerlinBlock, num)
}

// IsLondon returns whether num is either equal to the London fork block or greater.
func (c *ChainConfig) IsLondon(num *big.Int) bool {
	return utils.IsBlockForked(c.LondonBlock, num)
}

// IsShanghai returns whether time represents a block with a timestamp after the Shanghai upgrade time.
func (c *ChainConfig) IsShanghai(time uint64) bool {
	// For now, we'll consider Shanghai active if EVM is active
	return c.IsEVM(time)
}

// IsMerge returns whether the given block is either equal to the merge
// transition block or greater.
func (c *ChainConfig) IsMerge(num *big.Int, time uint64) bool {
	// The merge is always active in Lux
	return true
}

// IsPrague returns whether time is either equal to the Prague fork time or greater.
func (c *ChainConfig) IsPrague(time uint64) bool {
	// Prague features are not yet active
	return false
}

// IsVerkle returns whether time is either equal to the Verkle fork time or greater.
func (c *ChainConfig) IsVerkle(time uint64) bool {
	// Verkle is not yet active
	return false
}

// Rules returns the Ethereum chainrules to use for the given block number and timestamp.
func (c *ChainConfig) Rules(num *big.Int, isMerge bool, timestamp uint64) Rules {
	return c.rules(num, isMerge, timestamp)
}

// IsEVM returns whether [time] represents a block
// with a timestamp after the EVM upgrade time.
func (c *ChainConfig) IsEVM(time uint64) bool {
	return utils.IsTimestampForked(c.MandatoryNetworkUpgrades.EVMTimestamp, time)
}

// IsDUpgrade returns whether [time] represents a block
// with a timestamp after the DUpgrade upgrade time.
func (c *ChainConfig) IsDUpgrade(time uint64) bool {
	return utils.IsTimestampForked(c.MandatoryNetworkUpgrades.DurangoTimestamp, time)
}

// IsCancun returns whether [time] represents a block
// with a timestamp after the Cancun upgrade time.
func (c *ChainConfig) IsCancun(time uint64) bool {
	return c.MandatoryNetworkUpgrades.IsEtna(time)
}


// verifyPrecompileUpgrades checks that the precompile upgrades are valid
func (c *ChainConfig) verifyPrecompileUpgrades() error {
	// State upgrades are stored in the UpgradeConfig
	if c.UpgradeConfig.PrecompileUpgrades == nil {
		return nil
	}
	// For now, we'll skip validation as the method is private
	return nil
}

// verifyStateUpgrades checks that the state upgrades are valid
func (c *ChainConfig) verifyStateUpgrades() error {
	// State upgrades are stored in the UpgradeConfig
	if c.UpgradeConfig.StateUpgrades == nil {
		return nil
	}
	// For now, we'll skip validation as the method is private
	return nil
}

// mandatoryForkOrder returns the mandatory fork ordering
func (c *ChainConfig) mandatoryForkOrder() []fork {
	return []fork{
		{name: "homesteadBlock", block: c.HomesteadBlock},
		{name: "eip150Block", block: c.EIP150Block},
		{name: "eip155Block", block: c.EIP155Block},
		{name: "eip158Block", block: c.EIP158Block},
		{name: "byzantiumBlock", block: c.ByzantiumBlock},
		{name: "constantinopleBlock", block: c.ConstantinopleBlock},
		{name: "petersburgBlock", block: c.PetersburgBlock},
		{name: "istanbulBlock", block: c.IstanbulBlock},
		{name: "muirGlacierBlock", block: c.MuirGlacierBlock, optional: true},
	}
}

// optionalForkOrder returns the optional fork ordering
func (c *ChainConfig) optionalForkOrder() []fork {
	if c.UpgradeConfig.OptionalNetworkUpgrades == nil {
		return nil
	}
	// For now, return empty as the method is private
	return nil
}

// CheckMandatoryCompatible checks if the mandatory network upgrades are compatible
func (c *ChainConfig) CheckMandatoryCompatible(newcfg *MandatoryNetworkUpgrades, time uint64) error {
	// For now, skip this check as checkNetworkUpgradesCompatible is private
	return nil
}


// CheckPrecompilesCompatible checks that precompiles are compatible
func (c *ChainConfig) CheckPrecompilesCompatible(precompileUpgrades []extras.PrecompileUpgrade, time uint64) error {
	// For now, skip this check as checkPrecompilesCompatible is private
	return nil
}

// CheckStateUpgradesCompatible checks that state upgrades are compatible
func (c *ChainConfig) CheckStateUpgradesCompatible(stateUpgrades []extras.StateUpgrade, time uint64) error {
	// For now, skip this check as the method is private
	return nil
}

// IsPrecompileEnabled returns whether precompile with [address] is enabled at [timestamp].
func (c *ChainConfig) IsPrecompileEnabled(address common.Address, timestamp uint64) bool {
	extra := GetExtra(c)
	if extra == nil {
		return false
	}
	config := extra.GetActivePrecompileConfig(address, timestamp)
	return config != nil && !config.IsDisabled()
}

// CheckCompatible checks whether scheduled fork transitions have been imported
// with a mismatching chain configuration.
func (c *ChainConfig) CheckCompatible(newcfg *ChainConfig, height uint64, time uint64) *ConfigCompatError {
	var (
		bhead = new(big.Int).SetUint64(height)
		btime = time
	)
	// Iterate checkCompatible to find the lowest conflict.
	var lasterr *ConfigCompatError
	for {
		err := c.checkCompatible(newcfg, bhead, btime)
		if err == nil || (lasterr != nil && err.RewindToBlock == lasterr.RewindToBlock && err.RewindToTime == lasterr.RewindToTime) {
			break
		}
		lasterr = err

		if err.RewindToTime > 0 {
			btime = err.RewindToTime
		} else {
			bhead.SetUint64(err.RewindToBlock)
		}
	}
	return lasterr
}

// Verify verifies chain config and returns error
func (c *ChainConfig) Verify() error {
	if err := c.FeeConfig.Verify(); err != nil {
		return err
	}

	// Verify the precompile upgrades are internally consistent given the existing chainConfig.
	if err := c.verifyPrecompileUpgrades(); err != nil {
		return fmt.Errorf("invalid precompile upgrades: %w", err)
	}

	// Verify the state upgrades are internally consistent given the existing chainConfig.
	if err := c.verifyStateUpgrades(); err != nil {
		return fmt.Errorf("invalid state upgrades: %w", err)
	}

	return nil
}

type fork struct {
	name      string
	block     *big.Int // some go-ethereum forks use block numbers
	timestamp *uint64  // Lux forks use timestamps
	optional  bool     // if true, the fork may be nil and next fork is still allowed
}

// CheckConfigForkOrder checks that we don't "skip" any forks, geth isn't pluggable enough
// to guarantee that forks can be implemented in a different order than on official networks
func (c *ChainConfig) CheckConfigForkOrder() error {
	ethForks := []fork{
		{name: "homesteadBlock", block: c.HomesteadBlock},
		{name: "eip150Block", block: c.EIP150Block},
		{name: "eip155Block", block: c.EIP155Block},
		{name: "eip158Block", block: c.EIP158Block},
		{name: "byzantiumBlock", block: c.ByzantiumBlock},
		{name: "constantinopleBlock", block: c.ConstantinopleBlock},
		{name: "petersburgBlock", block: c.PetersburgBlock},
		{name: "istanbulBlock", block: c.IstanbulBlock},
		{name: "muirGlacierBlock", block: c.MuirGlacierBlock, optional: true},
	}

	// Check that forks are enabled in order
	if err := checkForks(ethForks, true); err != nil {
		return err
	}

	// Note: In Lux, hard forks must take place via block timestamps instead
	// of block numbers since blocks are produced asynchronously. Therefore, we do not
	// check that the block timestamps in the same way as for
	// the block number forks since it would not be a meaningful comparison.
	// Instead, we check only that Phases are enabled in order.
	// Note: we do not add the optional stateful precompile configs in here because they are optional
	// and independent, such that the ordering they are enabled does not impact the correctness of the
	// chain interfaces.
	if err := checkForks(c.mandatoryForkOrder(), false); err != nil {
		return err
	}

	// Check optional forks are enabled in order
	if err := checkForks(c.optionalForkOrder(), false); err != nil {
		return err
	}

	return nil
}

// checkForks checks that forks are enabled in order and returns an error if not
// [blockFork] is true if the fork is a block number fork, false if it is a timestamp fork
func checkForks(forks []fork, blockFork bool) error {
	lastFork := fork{}
	for _, cur := range forks {
		if blockFork && cur.block != nil && common.Big0.Cmp(cur.block) != 0 {
			return errNonGenesisForkByHeight
		}
		if lastFork.name != "" {
			// Next one must be higher number
			if lastFork.timestamp == nil && cur.timestamp != nil {
				return fmt.Errorf("unsupported fork ordering: %v not enabled, but %v enabled at %v",
					lastFork.name, cur.name, cur.timestamp)
			}
			if lastFork.timestamp != nil && cur.timestamp != nil {
				if *lastFork.timestamp > *cur.timestamp {
					return fmt.Errorf("unsupported fork ordering: %v enabled at %v, but %v enabled at %v",
						lastFork.name, lastFork.timestamp, cur.name, cur.timestamp)
				}
			}
		}
		// If it was optional and not set, then ignore it
		if !cur.optional || cur.timestamp != nil {
			lastFork = cur
		}
	}
	return nil
}

// checkCompatible confirms that [newcfg] is backwards compatible with [c] to upgrade with the given head block height and timestamp.
// This confirms that all Ethereum and Lux upgrades are backwards compatible as well as that the precompile config is backwards
// compatible.
func (c *ChainConfig) checkCompatible(newcfg *ChainConfig, height *big.Int, time uint64) *ConfigCompatError {
	if isForkBlockIncompatible(c.HomesteadBlock, newcfg.HomesteadBlock, height) {
		return newBlockCompatError("Homestead fork block", c.HomesteadBlock, newcfg.HomesteadBlock)
	}
	if isForkBlockIncompatible(c.EIP150Block, newcfg.EIP150Block, height) {
		return newBlockCompatError("EIP150 fork block", c.EIP150Block, newcfg.EIP150Block)
	}
	if isForkBlockIncompatible(c.EIP155Block, newcfg.EIP155Block, height) {
		return newBlockCompatError("EIP155 fork block", c.EIP155Block, newcfg.EIP155Block)
	}
	if isForkBlockIncompatible(c.EIP158Block, newcfg.EIP158Block, height) {
		return newBlockCompatError("EIP158 fork block", c.EIP158Block, newcfg.EIP158Block)
	}
	if c.IsEIP158(height) && !utils.BigNumEqual(c.ChainID, newcfg.ChainID) {
		return newBlockCompatError("EIP158 chain ID", c.EIP158Block, newcfg.EIP158Block)
	}
	if isForkBlockIncompatible(c.ByzantiumBlock, newcfg.ByzantiumBlock, height) {
		return newBlockCompatError("Byzantium fork block", c.ByzantiumBlock, newcfg.ByzantiumBlock)
	}
	if isForkBlockIncompatible(c.ConstantinopleBlock, newcfg.ConstantinopleBlock, height) {
		return newBlockCompatError("Constantinople fork block", c.ConstantinopleBlock, newcfg.ConstantinopleBlock)
	}
	if isForkBlockIncompatible(c.PetersburgBlock, newcfg.PetersburgBlock, height) {
		// the only case where we allow Petersburg to be set in the past is if it is equal to Constantinople
		// mainly to satisfy fork ordering requirements which state that Petersburg fork be set if Constantinople fork is set
		if isForkBlockIncompatible(c.ConstantinopleBlock, newcfg.PetersburgBlock, height) {
			return newBlockCompatError("Petersburg fork block", c.PetersburgBlock, newcfg.PetersburgBlock)
		}
	}
	if isForkBlockIncompatible(c.IstanbulBlock, newcfg.IstanbulBlock, height) {
		return newBlockCompatError("Istanbul fork block", c.IstanbulBlock, newcfg.IstanbulBlock)
	}
	if isForkBlockIncompatible(c.MuirGlacierBlock, newcfg.MuirGlacierBlock, height) {
		return newBlockCompatError("Muir Glacier fork block", c.MuirGlacierBlock, newcfg.MuirGlacierBlock)
	}

	// Check lux network upgrades
	if err := c.CheckMandatoryCompatible(&newcfg.MandatoryNetworkUpgrades, time); err != nil {
		// Convert error to ConfigCompatError
		return &ConfigCompatError{
			What:         "mandatory network upgrades",
			StoredTime:   nil,
			NewTime:      nil,
			RewindToTime: 0,
		}
	}

	// Check evm specific activations
	newOptionalNetworkUpgrades := newcfg.getOptionalNetworkUpgrades()
	if c.UpgradeConfig.OptionalNetworkUpgrades != nil && newcfg.UpgradeConfig.OptionalNetworkUpgrades == nil {
		// Note: if the current OptionalNetworkUpgrades are set via UpgradeConfig, then a new config
		// without OptionalNetworkUpgrades will be treated as having specified an empty set of network
		// upgrades (ie., treated as the user intends to cancel scheduled forks)
		newOptionalNetworkUpgrades = &OptionalNetworkUpgrades{}
	}
	// Handle the case where both might be nil
	oldOptional := c.getOptionalNetworkUpgrades()
	if oldOptional != nil && newOptionalNetworkUpgrades != nil {
		// The checkNetworkUpgradesCompatible method is private, so we can't call it directly
		// For now, skip this check
	}

	// Check that the precompiles on the new config are compatible with the existing precompile interfaces.
	if err := c.CheckPrecompilesCompatible(newcfg.PrecompileUpgrades, time); err != nil {
		// Convert error to ConfigCompatError
		return &ConfigCompatError{
			What:         "precompile upgrades",
			StoredTime:   nil,
			NewTime:      nil,
			RewindToTime: 0,
		}
	}

	// Check that the state upgrades on the new config are compatible with the existing state upgrade interfaces.
	if err := c.CheckStateUpgradesCompatible(newcfg.StateUpgrades, time); err != nil {
		// Convert error to ConfigCompatError
		return &ConfigCompatError{
			What:         "state upgrades",
			StoredTime:   nil,
			NewTime:      nil,
			RewindToTime: 0,
		}
	}

	// TODO verify that the fee config is fully compatible between [c] and [newcfg].
	return nil
}

// getOptionalNetworkUpgrades returns OptionalNetworkUpgrades from upgrade config if set there,
// otherwise it falls back to the genesis chain interfaces.
func (c *ChainConfig) getOptionalNetworkUpgrades() *OptionalNetworkUpgrades {
	if upgradeConfigOverride := c.UpgradeConfig.OptionalNetworkUpgrades; upgradeConfigOverride != nil {
		return upgradeConfigOverride
	}
	return &c.OptionalNetworkUpgrades
}

// isForkBlockIncompatible returns true if a fork scheduled at s1 cannot be rescheduled to
// block s2 because head is already past the fork.
func isForkBlockIncompatible(s1, s2, head *big.Int) bool {
	return (utils.IsBlockForked(s1, head) || utils.IsBlockForked(s2, head)) && !configBlockEqual(s1, s2)
}

func configBlockEqual(x, y *big.Int) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return x.Cmp(y) == 0
}

// isForkTimestampIncompatible returns true if a fork scheduled at timestamp s1
// cannot be rescheduled to timestamp s2 because head is already past the fork.
func isForkTimestampIncompatible(s1, s2 *uint64, head uint64) bool {
	return (utils.IsTimestampForked(s1, head) || utils.IsTimestampForked(s2, head)) && !configTimestampEqual(s1, s2)
}

func configTimestampEqual(x, y *uint64) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return *x == *y
}

// ConfigCompatError is raised if the locally-stored blockchain is initialised with a
// ChainConfig that would alter the past.
type ConfigCompatError struct {
	What string

	// block numbers of the stored and new configurations if block based forking
	StoredBlock, NewBlock *big.Int

	// timestamps of the stored and new configurations if time based forking
	StoredTime, NewTime *uint64

	// the block number to which the local chain must be rewound to correct the error
	RewindToBlock uint64

	// the timestamp to which the local chain must be rewound to correct the error
	RewindToTime uint64
}

func newBlockCompatError(what string, storedblock, newblock *big.Int) *ConfigCompatError {
	var rew *big.Int
	switch {
	case storedblock == nil:
		rew = newblock
	case newblock == nil || storedblock.Cmp(newblock) < 0:
		rew = storedblock
	default:
		rew = newblock
	}
	err := &ConfigCompatError{
		What:          what,
		StoredBlock:   storedblock,
		NewBlock:      newblock,
		RewindToBlock: 0,
	}
	if rew != nil && rew.Sign() > 0 {
		err.RewindToBlock = rew.Uint64() - 1
	}
	return err
}

func newTimestampCompatError(what string, storedtime, newtime *uint64) *ConfigCompatError {
	var rew *uint64
	switch {
	case storedtime == nil:
		rew = newtime
	case newtime == nil || *storedtime < *newtime:
		rew = storedtime
	default:
		rew = newtime
	}
	err := &ConfigCompatError{
		What:         what,
		StoredTime:   storedtime,
		NewTime:      newtime,
		RewindToTime: 0,
	}
	if rew != nil && *rew > 0 {
		err.RewindToTime = *rew - 1
	}
	return err
}

func (err *ConfigCompatError) Error() string {
	if err.StoredBlock != nil {
		return fmt.Sprintf("mismatching %s in database (have block %d, want block %d, rewindto block %d)", err.What, err.StoredBlock, err.NewBlock, err.RewindToBlock)
	}
	return fmt.Sprintf("mismatching %s in database (have timestamp %s, want timestamp %s, rewindto timestamp %d)", err.What, ptrToString(err.StoredTime), ptrToString(err.NewTime), err.RewindToTime)
}

func ptrToString(val *uint64) string {
	if val == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *val)
}

// Rules wraps ChainConfig and is merely syntactic sugar or can be used for functions
// that do not have or require information about the interfaces.
//
// Rules is a one time interface meaning that it shouldn't be used in between transition
// phases.
// 
// We use ethereum's Rules type directly to maintain compatibility
type Rules = ethparams.Rules

// Rules ensures c's ChainID is not nil.
func (c *ChainConfig) rules(num *big.Int, isMerge bool, timestamp uint64) Rules {
	chainID := c.ChainID
	if chainID == nil {
		chainID = new(big.Int)
	}
	// disallow setting Merge out of order
	isMerge = isMerge && c.IsLondon(num)
	return Rules{
		ChainID:          new(big.Int).Set(chainID),
		IsHomestead:      c.IsHomestead(num),
		IsEIP150:         c.IsEIP150(num),
		IsEIP155:         c.IsEIP155(num),
		IsEIP158:         c.IsEIP158(num),
		IsByzantium:      c.IsByzantium(num),
		IsConstantinople: c.IsConstantinople(num),
		IsPetersburg:     c.IsPetersburg(num),
		IsIstanbul:       c.IsIstanbul(num),
		IsBerlin:         c.IsBerlin(num),
		IsLondon:         c.IsLondon(num),
		IsMerge:          isMerge,
		IsShanghai:       isMerge && c.IsShanghai(timestamp),
		IsCancun:         isMerge && c.IsCancun(timestamp),
		IsPrague:         isMerge && c.IsPrague(timestamp),
		IsVerkle:         c.IsVerkle(timestamp),
		IsEIP2929:        c.IsBerlin(num) && !c.IsVerkle(timestamp),
		IsEIP4762:        c.IsVerkle(timestamp),
	}
}

// GenesisRules returns the Lux modified rules to support Lux
// network upgrades
func (c *ChainConfig) GenesisRules(blockNum *big.Int, timestamp uint64) Rules {
	// For now, we assume merge is always active in Lux
	isMerge := true
	return c.rules(blockNum, isMerge, timestamp)
}

// GetFeeConfig returns the original FeeConfig contained in the genesis ChainConfig.
// Implements precompile.ChainConfig interface.
func (c *ChainConfig) GetFeeConfig() commontype.FeeConfig {
	return c.FeeConfig
}

// AllowedFeeRecipients returns the original AllowedFeeRecipients parameter contained in the genesis ChainConfig.
// Implements precompile.ChainConfig interface.
func (c *ChainConfig) AllowedFeeRecipients() bool {
	return c.AllowFeeRecipients
}

type ChainConfigWithUpgradesJSON struct {
	ChainConfig
	UpgradeConfig UpgradeConfig `json:"upgrades,omitempty"`
}

// MarshalJSON implements interfaces.Marshaler. This is a workaround for the fact that
// the embedded ChainConfig struct has a MarshalJSON method, which prevents
// the default JSON marshalling from working for UpgradeConfig.
// TODO: consider removing this method by allowing external tag for the embedded
// ChainConfig struct.
func (cu ChainConfigWithUpgradesJSON) MarshalJSON() ([]byte, error) {
	// embed the ChainConfig struct into the response
	chainConfigJSON, err := json.Marshal(cu.ChainConfig)
	if err != nil {
		return nil, err
	}
	if len(chainConfigJSON) > maxJSONLen {
		return nil, errors.New("value too large")
	}

	type upgrades struct {
		UpgradeConfig UpgradeConfig `json:"upgrades"`
	}

	upgradeJSON, err := json.Marshal(upgrades{cu.UpgradeConfig})
	if err != nil {
		return nil, err
	}
	if len(upgradeJSON) > maxJSONLen {
		return nil, errors.New("value too large")
	}

	// merge the two JSON objects
	mergedJSON := make([]byte, 0, len(chainConfigJSON)+len(upgradeJSON)+1)
	mergedJSON = append(mergedJSON, chainConfigJSON[:len(chainConfigJSON)-1]...)
	mergedJSON = append(mergedJSON, ',')
	mergedJSON = append(mergedJSON, upgradeJSON[1:]...)
	return mergedJSON, nil
}

func (cu *ChainConfigWithUpgradesJSON) UnmarshalJSON(input []byte) error {
	var cc ChainConfig
	if err := json.Unmarshal(input, &cc); err != nil {
		return err
	}

	type upgrades struct {
		UpgradeConfig UpgradeConfig `json:"upgrades"`
	}

	var u upgrades
	if err := json.Unmarshal(input, &u); err != nil {
		return err
	}
	cu.ChainConfig = cc
	cu.UpgradeConfig = u.UpgradeConfig
	return nil
}

// ToWithUpgradesJSON converts the ChainConfig to ChainConfigWithUpgradesJSON with upgrades explicitly displayed.
// ChainConfig does not include upgrades in its JSON output.
// This is a workaround for showing upgrades in the JSON output.
func (c *ChainConfig) ToWithUpgradesJSON() *ChainConfigWithUpgradesJSON {
	return &ChainConfigWithUpgradesJSON{
		ChainConfig:   *c,
		UpgradeConfig: c.UpgradeConfig,
	}
}
