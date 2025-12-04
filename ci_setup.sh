#!/bin/bash
set -e

echo "Setting up CI environment..."

# Display Go version
echo "Go version:"
go version

# Configure GOPRIVATE for all luxfi packages
export GOPRIVATE=github.com/luxfi/*
export GONOSUMDB=github.com/luxfi/*

# Configure git to use GITHUB_TOKEN for authentication
if [ -n "$GITHUB_TOKEN" ]; then
    echo "Configuring git credentials..."
    git config --global url."https://x-access-token:${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"
fi

# Download dependencies from published packages
echo "Downloading dependencies..."
go mod download

echo "CI environment setup complete."
