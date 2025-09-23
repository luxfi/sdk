package params

import (
	"math/big"
	"testing"
)

func TestLocalNetworkChainID(t *testing.T) {
	// Test that all test configurations use chain ID 1337 for local networks
	testConfigs := map[string]*ChainConfig{
		"TestChainConfig":             TestChainConfig,
		"TestPreSubnetEVMChainConfig": TestPreSubnetEVMChainConfig,
		"TestSubnetEVMChainConfig":    TestSubnetEVMChainConfig,
		"TestDurangoChainConfig":      TestDurangoChainConfig,
		"TestEtnaChainConfig":         TestEtnaChainConfig,
		"TestFortunaChainConfig":      TestFortunaChainConfig,
		"TestGraniteChainConfig":      TestGraniteChainConfig,
	}

	expectedChainID := big.NewInt(1337)

	for name, config := range testConfigs {
		if config.ChainID.Cmp(expectedChainID) != 0 {
			t.Errorf("%s: expected chain ID %s, got %s", name, expectedChainID, config.ChainID)
		}
	}
}