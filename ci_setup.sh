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

# Detect OS
OS=$(uname -s)
echo "Detected OS: $OS"

# macOS specific setup
if [ "$OS" = "Darwin" ]; then
    echo "Setting up macOS environment..."
    # Use Go from PATH, don't rely on Homebrew
    export PATH="/usr/local/go/bin:$PATH"
    export GOPATH="$HOME/go"
    export PATH="$GOPATH/bin:$PATH"
fi

# Validate Go installation
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    exit 1
fi

echo "Go version: $(go version)"

# Backup original go.mod
echo "Backing up go.mod..."
cp go.mod go.mod.backup

# Remove local replace directives (keep tablewriter replace)
echo "Removing local replace directives..."
sed -i.bak '/replace.*\.\.\/.*$/d' go.mod
sed -i.bak '/replace.*\/home\/.*$/d' go.mod

# Show what's left in replace directives
echo "Remaining replace directives:"
grep "^replace" go.mod || echo "  (none except commented ones)"

# Download dependencies
echo "Downloading dependencies..."
go mod download || echo "Warning: Some dependencies may not be available"

# Run go mod tidy
echo "Running go mod tidy..."
go mod tidy || true

echo ""
echo "=== CI Setup Complete ==="
echo ""
