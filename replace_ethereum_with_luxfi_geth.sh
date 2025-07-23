#!/bin/bash
# Replace all github.com/ethereum/go-ethereum imports with github.com/luxfi/geth

find . -name "*.go" -type f -not -path "./vendor/*" -not -path "./geth/*" -exec sed -i '' \
    's|"github.com/ethereum/go-ethereum|"github.com/luxfi/geth|g' {} \;

echo "Replaced all go-ethereum imports with luxfi/geth"