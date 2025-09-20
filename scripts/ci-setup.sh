#!/bin/bash
set -e

echo "Setting up CI environment..."

# Comment out local replace directives for CI
if [ "$CI" = "true" ]; then
    echo "CI environment detected, removing local replace directives..."
    cp go.mod go.mod.backup
    sed -i 's/^replace github\.com\/luxfi\/node/\/\/ replace github.com\/luxfi\/node/' go.mod
    sed -i 's/^replace github\.com\/luxfi\/consensus/\/\/ replace github.com\/luxfi\/consensus/' go.mod
    sed -i 's/^replace github\.com\/luxfi\/crypto/\/\/ replace github.com\/luxfi\/crypto/' go.mod
    sed -i 's/^replace github\.com\/luxfi\/database/\/\/ replace github.com\/luxfi\/database/' go.mod
    sed -i 's/^replace github\.com\/luxfi\/geth/\/\/ replace github.com\/luxfi\/geth/' go.mod
    sed -i 's/^replace github\.com\/luxfi\/warp/\/\/ replace github.com\/luxfi\/warp/' go.mod
    sed -i 's/^replace github\.com\/luxfi\/metric/\/\/ replace github.com\/luxfi\/metric/' go.mod
    sed -i 's/^replace github\.com\/luxfi\/ids/\/\/ replace github.com\/luxfi\/ids/' go.mod
    sed -i 's/^replace github\.com\/luxfi\/log/\/\/ replace github.com\/luxfi\/log/' go.mod
    echo "Local replace directives commented out for CI"
fi

# Run go mod tidy to ensure dependencies are correct
echo "Running go mod tidy..."
go mod tidy

echo "CI setup complete"