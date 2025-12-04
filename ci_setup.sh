#!/bin/bash
set -e

echo "Setting up CI environment..."

# Display Go version
echo "Go version:"
go version

# Download dependencies from published packages
echo "Downloading dependencies..."
go mod download

echo "CI environment setup complete."
