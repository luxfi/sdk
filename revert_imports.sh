#!/bin/bash

# Revert imports back to luxfi/node/<package> since they're not separate repos

echo "Reverting imports back to luxfi/node/<package>..."

# List of packages to revert
PACKAGES=(
    "cache"
    "codec"
    "upgrade"
    "utils"
    "version"
    "vms"
    "network"
    "proto"
    "api"
    "message"
    "wallet"
)

# Revert imports in all Go files
for pkg in "${PACKAGES[@]}"; do
    echo "Reverting imports for package: $pkg"
    find . -name "*.go" -type f -exec sed -i "s|github.com/luxfi/${pkg}|github.com/luxfi/node/${pkg}|g" {} \;
done

echo "Import reversion completed!"