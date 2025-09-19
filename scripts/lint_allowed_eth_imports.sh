#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Ensure that there are no eth imports that are not marked as explicitly allowed via ./scripts/eth-allowed-packages.txt
# 1. Recursively search through all go files for any lines that include a direct import from geth
# 2. Ignore lines that import geth with a named import starting with "eth" or by _ import.
# 3. Sort the unique results
# 4. Print out the difference between the search results and the list of specified allowed package imports from geth.
geth_regexp='"github.com/luxfi/geth/.*"'
extra_imports=$(find . -type f \( -name "*.go" \) ! -name "mocks.go" ! -path "simulator" ! -path "./core/main_test.go" ! -name "gen_*.go" -print0 |
  xargs -0 grep "${geth_regexp}" |
  grep -v 'eth\w\+ "' |
  grep -v '_ "' |
  grep -o "${geth_regexp}" |
  LC_ALL=C sort -u | LC_ALL=C comm -23 - ./scripts/eth-allowed-packages.txt)
if [ -n "${extra_imports}" ]; then
    echo "new ethereum imports should be added to ./scripts/eth-allowed-packages.txt to prevent accidental imports:"
    echo "${extra_imports}"
    exit 1
fi

extra_imports=$(grep -r --include='*.go' '"github.com/luxfi/coreth/.*"' -o -h || true | LC_ALL=C sort -u)
if [ -n "${extra_imports}" ]; then
    echo "evm should not import packages from coreth:"
    echo "${extra_imports}"
    exit 1
fi
