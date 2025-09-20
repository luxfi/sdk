#!/bin/bash

# Fix all imports from luxfi/node/<package> to luxfi/<package> for externalized packages

echo "Fixing imports to use externalized packages..."

# List of externalized packages
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

# Fix imports in all Go files
for pkg in "${PACKAGES[@]}"; do
    echo "Fixing imports for package: $pkg"
    find . -name "*.go" -type f -exec sed -i "s|github.com/luxfi/node/${pkg}|github.com/luxfi/${pkg}|g" {} \;
done

# Special cases for subpackages that might be referenced
echo "Fixing special case imports..."
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/cache/lru|github.com/luxfi/cache/lru|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/utils/units|github.com/luxfi/utils/units|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/codec/linearcodec|github.com/luxfi/codec/linearcodec|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/network/p2p|github.com/luxfi/network/p2p|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/vms/platformvm/warp|github.com/luxfi/vms/platformvm/warp|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/utils/timer|github.com/luxfi/utils/timer|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/utils/constants|github.com/luxfi/utils/constants|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/utils/wrappers|github.com/luxfi/utils/wrappers|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/utils/math|github.com/luxfi/utils/math|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/utils/profiler|github.com/luxfi/utils/profiler|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/utils/json|github.com/luxfi/utils/json|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/utils/rpc|github.com/luxfi/utils/rpc|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/utils/perms|github.com/luxfi/utils/perms|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/utils/ulimit|github.com/luxfi/utils/ulimit|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/utils/compression|github.com/luxfi/utils/compression|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/vms/rpcchainvm|github.com/luxfi/vms/rpcchainvm|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/vms/components/chain|github.com/luxfi/vms/components/chain|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/cache/metercacher|github.com/luxfi/cache/metercacher|g' {} \;
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/node/network/peer|github.com/luxfi/network/peer|g' {} \;

echo "Import fixes completed!"