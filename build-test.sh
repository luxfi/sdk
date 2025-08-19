#!/bin/bash
set -e

echo "=== Lux EVM Build Test ==="
echo "Version: v0.8.7-lux.16"
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test function
test_build() {
    local package=$1
    local name=$2
    
    if go build -o /dev/null $package 2>/dev/null; then
        echo -e "${GREEN}✅${NC} $name"
        return 0
    else
        echo -e "${RED}❌${NC} $name"
        return 1
    fi
}

echo "Testing package builds..."
echo "========================"

# Core packages
test_build "./accounts/..." "Accounts"
test_build "./common/..." "Common"
test_build "./consensus/..." "Consensus"
test_build "./core/..." "Core"
test_build "./eth/..." "Eth"
test_build "./ethclient/..." "EthClient"
test_build "./ethdb/..." "EthDB"
test_build "./event/..." "Event"
test_build "./log/..." "Log"
test_build "./metrics/..." "Metrics"
test_build "./miner/..." "Miner"
test_build "./node/..." "Node"
test_build "./p2p/..." "P2P"
test_build "./params/..." "Params"
test_build "./plugin/evm/..." "Plugin EVM"
test_build "./rpc/..." "RPC"
test_build "./sync/..." "Sync"
test_build "./tests/..." "Tests"
test_build "./triedb/..." "TrieDB"
test_build "./warp/..." "Warp"

echo ""
echo "Testing binaries..."
echo "==================="

# Binaries
test_build "./cmd/simulator/main/main.go" "Simulator"
test_build "./examples/sign-uptime-message/main.go" "Sign Uptime Message"

echo ""
echo "Dependencies Check..."
echo "===================="

# Check dependencies
echo -n "Consensus version: "
go list -m github.com/luxfi/consensus | awk '{print $2}'

echo -n "Node version: "
go list -m github.com/luxfi/node | awk '{print $2}'

echo -n "Geth version: "
go list -m github.com/luxfi/geth | awk '{print $2}'

echo ""
echo "=== Build Test Complete ==="