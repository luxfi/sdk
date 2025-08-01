#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Root directory
EVM_PATH=$(
    cd "$(dirname "${BASH_SOURCE[0]}")"
    cd .. && pwd
)

# Load the constants
source "$EVM_PATH"/scripts/constants.sh

go test ./precompile/contracts/... -bench=./... -timeout="10m" "$@"
