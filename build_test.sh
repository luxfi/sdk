#!/bin/bash
set -e

echo "Testing EVM compilation..."
echo "=========================="

# Build main plugin
echo -n "Building EVM plugin... "
go build -o /tmp/evm-plugin ./plugin/evm
echo "✓"

# Build available commands
for cmd in evm abigen precompilegen; do
  if [ -d "./cmd/$cmd" ]; then
    echo -n "Building $cmd... "
    go build -o /tmp/$cmd ./cmd/$cmd
    echo "✓"
  fi
done

# Build core packages
echo -n "Building core packages... "
go build ./core/...
echo "✓"

echo -n "Building eth packages... "
go build ./eth/...
echo "✓"

echo -n "Building miner packages... "
go build ./miner/...
echo "✓"

echo -n "Building consensus packages... "
go build ./consensus/...
echo "✓"

echo -n "Building params packages... "
go build ./params/...
echo "✓"

echo -n "Building accounts packages... "
go build ./accounts/...
echo "✓"

# Clean up
rm -f /tmp/evm-plugin /tmp/evm /tmp/abigen /tmp/precompilegen

echo ""
echo "All builds successful! ✓"
echo "=========================="
