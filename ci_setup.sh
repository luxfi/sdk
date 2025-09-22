#!/bin/bash
set -e

echo "Setting up CI environment..."

# Ensure we're in the right directory
cd "$(dirname "$0")"

# Install required tools
echo "Installing build tools..."
go version

# Download dependencies
echo "Downloading dependencies..."
go mod download || true

echo "CI environment setup complete."