#!/usr/bin/env bash

set -euo pipefail

# Directory above this script
EVM_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd .. && pwd )
# Load the constants
source "$EVM_PATH"/scripts/constants.sh

echo "Building Workload..."
go build -o "$EVM_PATH/build/workload" "$EVM_PATH/tests/antithesis/"*.go
