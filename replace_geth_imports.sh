#!/bin/bash

# Replace go-ethereum imports with local geth imports

echo "Replacing go-ethereum imports with local geth imports..."

# Common packages that should be replaced
find . -name "*.go" -type f -not -path "./vendor/*" -not -path "./geth/*" -exec sed -i '' \
    -e 's|"github.com/ethereum/go-ethereum/common"|"github.com/luxfi/evm/geth/common"|g' \
    -e 's|"github.com/ethereum/go-ethereum/core/types"|"github.com/luxfi/evm/core/types"|g' \
    -e 's|"github.com/ethereum/go-ethereum/crypto"|"github.com/luxfi/evm/geth/crypto"|g' \
    -e 's|"github.com/ethereum/go-ethereum/log"|"github.com/luxfi/evm/geth/log"|g' \
    -e 's|"github.com/ethereum/go-ethereum/metrics"|"github.com/luxfi/evm/geth/metrics"|g' \
    -e 's|"github.com/ethereum/go-ethereum/event"|"github.com/luxfi/evm/geth/event"|g' \
    -e 's|"github.com/ethereum/go-ethereum/rlp"|"github.com/luxfi/evm/geth/rlp"|g' \
    -e 's|"github.com/ethereum/go-ethereum/trie"|"github.com/luxfi/evm/geth/trie"|g' \
    -e 's|"github.com/ethereum/go-ethereum/params"|"github.com/luxfi/evm/geth/params"|g' \
    -e 's|"github.com/ethereum/go-ethereum/accounts"|"github.com/luxfi/evm/geth/accounts"|g' \
    -e 's|"github.com/ethereum/go-ethereum/accounts/abi"|"github.com/luxfi/evm/accounts/abi"|g' \
    -e 's|"github.com/ethereum/go-ethereum/accounts/keystore"|"github.com/luxfi/evm/geth/accounts/keystore"|g' \
    -e 's|"github.com/ethereum/go-ethereum/ethclient"|"github.com/luxfi/evm/geth/ethclient"|g' \
    -e 's|"github.com/ethereum/go-ethereum/eth"|"github.com/luxfi/evm/geth/eth"|g' \
    -e 's|"github.com/ethereum/go-ethereum/ethdb"|"github.com/luxfi/evm/geth/ethdb"|g' \
    -e 's|"github.com/ethereum/go-ethereum/core"|"github.com/luxfi/evm/core"|g' \
    -e 's|"github.com/ethereum/go-ethereum/core/vm"|"github.com/luxfi/evm/core/vm"|g' \
    -e 's|"github.com/ethereum/go-ethereum/core/state"|"github.com/luxfi/evm/core/state"|g' \
    -e 's|"github.com/ethereum/go-ethereum/core/rawdb"|"github.com/luxfi/evm/core/rawdb"|g' \
    -e 's|"github.com/ethereum/go-ethereum/common/hexutil"|"github.com/luxfi/evm/geth/common/hexutil"|g' \
    -e 's|"github.com/ethereum/go-ethereum/common/math"|"github.com/luxfi/evm/common/math"|g' \
    -e 's|"github.com/ethereum/go-ethereum/rpc"|"github.com/luxfi/evm/rpc"|g' \
    {} \;

echo "Import replacement complete!"
echo "Checking remaining go-ethereum imports..."
grep -r "github.com/ethereum/go-ethereum" --include="*.go" | grep -v "vendor" | grep -v "geth" | wc -l