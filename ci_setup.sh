#!/bin/bash
set -e

echo "Setting up CI environment..."

# Ensure we're in the right directory
cd "$(dirname "$0")"
EVM_DIR=$(pwd)
PARENT_DIR=$(dirname "$EVM_DIR")

# Install required tools
echo "Installing build tools..."
go version

# Clone dependent repositories for local replace directives
echo "Cloning dependent repositories..."

clone_repo() {
    local repo=$1
    local branch=${2:-main}
    local dir="$PARENT_DIR/$repo"

    if [ ! -d "$dir" ]; then
        echo "Cloning $repo..."
        git clone --depth 1 --branch "$branch" "https://github.com/luxfi/$repo.git" "$dir" || \
        git clone --depth 1 "https://github.com/luxfi/$repo.git" "$dir"
    else
        echo "$repo already exists at $dir"
    fi
}

# Clone all required repositories
clone_repo "geth" "main"
clone_repo "node" "main"
clone_repo "consensus" "main"
clone_repo "warp" "main"
clone_repo "database" "main"
clone_repo "genesis" "main"

# Download dependencies
echo "Downloading dependencies..."
go mod download || true

echo "CI environment setup complete."
