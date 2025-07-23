#!/bin/bash

# Script to migrate from ethereum/go-ethereum to luxfi/geth imports

echo "Starting migration from ethereum/go-ethereum to luxfi/geth..."

# Find all Go files and update imports
find . -name "*.go" -type f -not -path "./vendor/*" -not -path "./.git/*" | while read -r file; do
    # Skip if file doesn't contain ethereum/go-ethereum imports
    if ! grep -q "github.com/ethereum/go-ethereum" "$file"; then
        continue
    fi
    
    echo "Updating imports in: $file"
    
    # Replace the imports
    sed -i.bak \
        -e 's|"github.com/ethereum/go-ethereum/common"|"github.com/luxfi/geth/common"|g' \
        -e 's|"github.com/ethereum/go-ethereum/common/hexutil"|"github.com/luxfi/geth/common/hexutil"|g' \
        -e 's|"github.com/ethereum/go-ethereum/core/types"|"github.com/luxfi/geth/core/types"|g' \
        -e 's|"github.com/ethereum/go-ethereum/crypto"|"github.com/luxfi/geth/crypto"|g' \
        -e 's|"github.com/ethereum/go-ethereum/ethdb"|"github.com/luxfi/geth/ethdb"|g' \
        -e 's|"github.com/ethereum/go-ethereum/event"|"github.com/luxfi/geth/event"|g' \
        -e 's|"github.com/ethereum/go-ethereum/log"|"github.com/luxfi/geth/log"|g' \
        -e 's|eparams "github.com/ethereum/go-ethereum/params"|eparams "github.com/luxfi/geth/params"|g' \
        -e 's|ethparams "github.com/ethereum/go-ethereum/params"|ethparams "github.com/luxfi/geth/params"|g' \
        -e 's|ethtypes "github.com/ethereum/go-ethereum/core/types"|ethtypes "github.com/luxfi/geth/core/types"|g' \
        -e 's|gethevent "github.com/ethereum/go-ethereum/event"|gethevent "github.com/luxfi/geth/event"|g' \
        "$file"
    
    # Remove backup file if successful
    if [ $? -eq 0 ]; then
        rm -f "${file}.bak"
    else
        echo "Error updating $file"
        mv "${file}.bak" "$file"
    fi
done

echo "Migration complete!"
echo ""
echo "Next steps:"
echo "1. Run 'go mod tidy' to update dependencies"
echo "2. Run tests to ensure everything works correctly"
echo "3. Review any remaining references to ethereum/go-ethereum"