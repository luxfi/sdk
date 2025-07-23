#!/bin/bash
# Fix imports that reference the old internal geth directory

# Replace references to internal geth with direct luxfi/geth references
find . -name "*.go" -type f -not -path "./vendor/*" -exec sed -i '' \
    's|"github.com/luxfi/evm/geth/|"github.com/luxfi/geth/|g' {} \;

echo "Fixed all internal geth imports to use luxfi/geth"