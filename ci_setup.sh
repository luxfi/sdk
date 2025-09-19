#!/bin/bash
# CI Setup Script for LUX EVM Module
# This script prepares the module for CI builds by removing local replace directives

set -e

echo "=== CI Setup for LUX EVM Module ==="

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "Error: go.mod not found. Please run this script from the evm directory."
    exit 1
fi

# Backup original go.mod
echo "Backing up go.mod..."
cp go.mod go.mod.backup

# Remove local replace directives (keep tablewriter replace)
echo "Removing local replace directives..."
sed -i '/replace.*\.\.\/.*$/d' go.mod
sed -i '/replace.*\/home\/.*$/d' go.mod

# Show what's left in replace directives
echo "Remaining replace directives:"
grep "^replace" go.mod || echo "  (none except commented ones)"

# Download dependencies
echo "Downloading dependencies..."
go mod download

# Verify the module
echo "Verifying module..."
go mod verify || true

echo ""
echo "=== CI Setup Complete ==="
echo ""
echo "You can now build with:"
echo "  go build \$(go list ./... | grep -v '/tests' | grep -v '/examples')"
echo ""
echo "Or run tests with:"
echo "  go test \$(go list ./... | grep -v '/tests' | grep -v '/examples')"
echo ""
echo "To restore original go.mod:"
echo "  mv go.mod.backup go.mod"