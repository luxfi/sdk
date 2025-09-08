#!/bin/bash
# CI Verification Script for Lux EVM
# This script verifies that all packages compile successfully

set -e

echo "=========================================="
echo "    Lux EVM CI Verification Script       "
echo "=========================================="

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track failures
FAILED_PACKAGES=""
TOTAL_PACKAGES=0
PASSED_PACKAGES=0

# Function to check package compilation
check_package() {
    local package=$1
    TOTAL_PACKAGES=$((TOTAL_PACKAGES + 1))
    
    echo -n "Checking $package... "
    if go build -o /dev/null "$package" 2>/dev/null; then
        echo -e "${GREEN}✓ PASS${NC}"
        PASSED_PACKAGES=$((PASSED_PACKAGES + 1))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}"
        FAILED_PACKAGES="$FAILED_PACKAGES $package"
        return 1
    fi
}

echo ""
echo "1. Core Packages:"
echo "-----------------"
check_package "./core/..."
check_package "./core/vm"
check_package "./core/state"
check_package "./core/types"

echo ""
echo "2. Network & Plugin:"
echo "--------------------"
check_package "./network"
check_package "./plugin/evm"

echo ""
echo "3. Parameters & Config:"
echo "------------------------"
check_package "./params"

echo ""
echo "4. Precompiled Contracts:"
echo "--------------------------"
check_package "./precompile/contracts/pqcrypto"
check_package "./precompile/contract"
check_package "./precompile/allowlist"

echo ""
echo "5. Trie Database:"
echo "-----------------"
check_package "./triedb/pathdb"

echo ""
echo "6. Ethereum Components:"
echo "------------------------"
check_package "./eth/gasestimator"
check_package "./eth/gasprice"

echo ""
echo "=========================================="
echo "              SUMMARY                     "
echo "=========================================="
echo -e "Total Packages Checked: ${YELLOW}$TOTAL_PACKAGES${NC}"
echo -e "Passed: ${GREEN}$PASSED_PACKAGES${NC}"
echo -e "Failed: ${RED}$((TOTAL_PACKAGES - PASSED_PACKAGES))${NC}"

if [ "$PASSED_PACKAGES" -eq "$TOTAL_PACKAGES" ]; then
    echo ""
    echo -e "${GREEN}✓ All packages compile successfully!${NC}"
    echo -e "${GREEN}✓ CI Build Status: 100% PASS${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}Failed packages:${NC}"
    for pkg in $FAILED_PACKAGES; do
        echo -e "  ${RED}✗${NC} $pkg"
    done
    exit 1
fi