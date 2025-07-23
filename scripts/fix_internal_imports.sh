#!/bin/bash

# Script to fix internal imports in the EVM package
# These should remain as github.com/luxfi/evm/* not github.com/luxfi/geth/*

echo "Fixing internal imports in EVM package..."

# Find all Go files and fix imports that should be internal
find . -name "*.go" -type f -not -path "./vendor/*" -not -path "./.git/*" | while read -r file; do
    # Skip if file doesn't contain our imports
    if ! grep -q "github.com/luxfi/evm" "$file"; then
        continue
    fi
    
    # List of packages that are INTERNAL to EVM and should NOT use geth
    # These are implemented within the EVM package itself
    sed -i.bak \
        -e 's|"github.com/luxfi/geth/common"|"github.com/luxfi/evm/common"|g' \
        -e 's|"github.com/luxfi/geth/common/hexutil"|"github.com/luxfi/evm/common/hexutil"|g' \
        -e 's|"github.com/luxfi/geth/common/math"|"github.com/luxfi/evm/common/math"|g' \
        -e 's|"github.com/luxfi/geth/core/types"|"github.com/luxfi/evm/core/types"|g' \
        -e 's|"github.com/luxfi/geth/core/rawdb"|"github.com/luxfi/evm/core/rawdb"|g' \
        -e 's|"github.com/luxfi/geth/core/state"|"github.com/luxfi/evm/core/state"|g' \
        -e 's|"github.com/luxfi/geth/core/vm"|"github.com/luxfi/evm/core/vm"|g' \
        -e 's|"github.com/luxfi/geth/crypto"|"github.com/luxfi/evm/crypto"|g' \
        -e 's|"github.com/luxfi/geth/ethdb"|"github.com/luxfi/evm/ethdb"|g' \
        -e 's|"github.com/luxfi/geth/event"|"github.com/luxfi/evm/event"|g' \
        -e 's|"github.com/luxfi/geth/log"|"github.com/luxfi/evm/log"|g' \
        -e 's|"github.com/luxfi/geth/params"|"github.com/luxfi/evm/params"|g' \
        -e 's|"github.com/luxfi/geth/accounts"|"github.com/luxfi/evm/accounts"|g' \
        -e 's|"github.com/luxfi/geth/rlp"|"github.com/luxfi/evm/rlp"|g' \
        -e 's|"github.com/luxfi/geth/trie"|"github.com/luxfi/evm/trie"|g' \
        -e 's|"github.com/luxfi/geth/metrics"|"github.com/luxfi/evm/metrics"|g' \
        "$file"
    
    # Remove backup file if successful
    if [ $? -eq 0 ]; then
        rm -f "${file}.bak"
    else
        echo "Error updating $file"
        mv "${file}.bak" "$file"
    fi
done

echo "Internal import fix complete!"
echo ""
echo "Note: The EVM package should use its own internal packages."
echo "Only use luxfi/geth for things that come from go-ethereum."