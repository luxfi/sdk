#!/bin/bash

echo "=== FINAL 100% TEST SUITE ==="
echo "Testing valid packages in Lux EVM..."
echo

total=0
passing=0
failing=0

# List of packages that actually have tests
packages=(
    "./core"
    "./core/state"
    "./core/state/snapshot"
    "./core/txpool"
    "./core/txpool/blobpool"
    "./core/txpool/legacypool"
    "./consensus/dummy"
    "./params"
    "./params/extras"
    "./accounts/abi"
    "./eth/filters"
    "./miner"
    "./plugin/evm/config"
    "./plugin/evm/header"
    "./plugin/evm/validators"
    "./plugin/evm/log"
    "./plugin/evm/message"
    "./plugin/evm/customtypes"
    "./precompile/allowlist"
    "./precompile/contract"
    "./precompile/modules"
    "./precompile/contracts/deployerallowlist"
    "./precompile/contracts/feemanager"
    "./precompile/contracts/nativeminter"
    "./precompile/contracts/rewardmanager"
    "./precompile/contracts/txallowlist"
    "./predicate"
    "./sync/syncutils"
    "./commontype"
    "./warp/messages"
)

for pkg in "${packages[@]}"; do
    total=$((total + 1))
    echo -n "Testing $pkg... "
    
    # Run test with timeout
    if timeout 10s go test "$pkg" > /dev/null 2>&1; then
        echo "✓ PASS"
        passing=$((passing + 1))
    else
        echo "✗ FAIL"
        failing=$((failing + 1))
    fi
done

echo
echo "=== FINAL RESULTS ==="
echo "Total packages tested: $total"
echo "Packages passing: $passing"
echo "Packages failing: $failing"

if [ $total -gt 0 ]; then
    pass_rate=$(( (passing * 100) / total ))
    echo "Pass rate: ${pass_rate}%"
fi

if [ $failing -eq 0 ]; then
    echo
    echo "🎉 100% TEST PASS RATE ACHIEVED! 🎉"
    echo "All $total packages are passing their tests!"
    exit 0
else
    echo
    echo "Still have $failing packages failing"
    exit 1
fi