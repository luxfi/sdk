#!/bin/bash
# Update all imports in newly copied directories from coreth

echo "Updating imports in newly copied directories..."

# Directories to update
DIRS="eth ethclient trie core/vm accounts/keystore accounts/external"

for dir in $DIRS; do
    if [ -d "$dir" ]; then
        echo "Updating imports in $dir..."
        # Replace ethereum/go-ethereum with luxfi/geth
        find "$dir" -name "*.go" -type f -exec sed -i '' \
            's|"github.com/ethereum/go-ethereum|"github.com/luxfi/geth|g' {} \;
        
        # Replace ava-labs/coreth with luxfi/evm
        find "$dir" -name "*.go" -type f -exec sed -i '' \
            's|"github.com/ava-labs/coreth|"github.com/luxfi/evm|g' {} \;
        
        # Replace avalanchego imports with luxfi/node
        find "$dir" -name "*.go" -type f -exec sed -i '' \
            's|"github.com/ava-labs/avalanchego|"github.com/luxfi/node|g' {} \;
    fi
done

echo "Import updates complete!"