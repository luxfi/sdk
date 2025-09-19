// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package params

import (
	"math/big"
	"testing"
	"time"

	"github.com/luxfi/evm/params/extras"
	"github.com/luxfi/evm/utils"
	"github.com/luxfi/node/upgrade"
	"github.com/stretchr/testify/require"
)

// Test fork types for upgrade testing
type testFork int

const (
	NoUpgrades testFork = iota
	Durango
	Etna
)

func (f testFork) String() string {
	switch f {
	case NoUpgrades:
		return "NoUpgrades"
	case Durango:
		return "Durango"
	case Etna:
		return "Etna"
	default:
		return "Unknown"
	}
}

// testGetConfig returns a mock upgrade config for testing
func testGetConfig(fork testFork) upgrade.Config {
	var activationTime time.Time
	switch fork {
	case NoUpgrades:
		// Far future time to indicate no upgrades
		activationTime = time.Unix(1<<62, 0)
	case Durango, Etna:
		// Use genesis time for active upgrades
		activationTime = time.Unix(int64(initiallyActive), 0)
	default:
		activationTime = time.Unix(1<<62, 0)
	}
	return upgrade.Config{
		ActivationTime: activationTime,
	}
}

func TestSetEthUpgrades(t *testing.T) {
	genesisBlock := big.NewInt(0)
	genesisTimestamp := utils.NewUint64(initiallyActive)
	tests := []struct {
		fork     testFork
		expected *ChainConfig
	}{
		{
			fork: NoUpgrades,
			expected: &ChainConfig{
				HomesteadBlock:      genesisBlock,
				EIP150Block:         genesisBlock,
				EIP155Block:         genesisBlock,
				EIP158Block:         genesisBlock,
				ByzantiumBlock:      genesisBlock,
				ConstantinopleBlock: genesisBlock,
				PetersburgBlock:     genesisBlock,
				IstanbulBlock:       genesisBlock,
				MuirGlacierBlock:    genesisBlock,
				BerlinBlock:         genesisBlock,
				LondonBlock:         genesisBlock,
				ShanghaiTime:        nil,
				CancunTime:          nil,
			},
		},
		{
			fork: Durango,
			expected: &ChainConfig{
				HomesteadBlock:      genesisBlock,
				EIP150Block:         genesisBlock,
				EIP155Block:         genesisBlock,
				EIP158Block:         genesisBlock,
				ByzantiumBlock:      genesisBlock,
				ConstantinopleBlock: genesisBlock,
				PetersburgBlock:     genesisBlock,
				IstanbulBlock:       genesisBlock,
				MuirGlacierBlock:    genesisBlock,
				BerlinBlock:         genesisBlock,
				LondonBlock:         genesisBlock,
				ShanghaiTime:        genesisTimestamp,
				CancunTime:          nil,
			},
		},
		{
			fork: Etna,
			expected: &ChainConfig{
				HomesteadBlock:      genesisBlock,
				EIP150Block:         genesisBlock,
				EIP155Block:         genesisBlock,
				EIP158Block:         genesisBlock,
				ByzantiumBlock:      genesisBlock,
				ConstantinopleBlock: genesisBlock,
				PetersburgBlock:     genesisBlock,
				IstanbulBlock:       genesisBlock,
				MuirGlacierBlock:    genesisBlock,
				BerlinBlock:         genesisBlock,
				LondonBlock:         genesisBlock,
				ShanghaiTime:        genesisTimestamp,
				CancunTime:          genesisTimestamp,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.fork.String(), func(t *testing.T) {
			require := require.New(t)

			extraConfig := &extras.ChainConfig{
				NetworkUpgrades: extras.GetNetworkUpgrades(testGetConfig(test.fork)),
			}
			actual := WithExtra(
				&ChainConfig{},
				extraConfig,
			)
			require.NoError(SetEthUpgrades(actual))

			expected := WithExtra(
				test.expected,
				extraConfig,
			)
			require.Equal(expected, actual)
		})
	}
}
