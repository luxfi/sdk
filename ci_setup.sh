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

# Run go mod tidy to ensure dependencies are correct
echo "Running go mod tidy..."
go mod tidy

echo "CI setup complete"