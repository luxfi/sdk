#!/bin/bash
set -e

echo "Setting up CI environment..."

# Always remove local replace directives for CI (GitHub Actions always sets CI=true)
echo "Removing local replace directives for CI..."
cp go.mod go.mod.backup

# Use portable sed syntax that works on both Linux and macOS
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS requires -i ''
    sed -i '' '/^replace github\.com\/luxfi\/.*=>.*/d' go.mod
else
    # Linux sed
    sed -i '/^replace github\.com\/luxfi\/.*=>.*/d' go.mod
fi

echo "Local replace directives removed for CI"

# Fix the node version to v1.13.5 (the latest available release)
echo "Fixing luxfi/node version to v1.13.5..."
go get github.com/luxfi/node@v1.13.5

# Run go mod tidy to ensure dependencies are correct
echo "Running go mod tidy..."
go mod tidy

echo "CI setup complete"