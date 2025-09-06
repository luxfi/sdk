#!/bin/bash
echo "=== COMPREHENSIVE TEST RESULTS ==="
TOTAL=0
PASS=0
FAIL_LIST=""

test_package() {
  local pkg=$1
  TOTAL=$((TOTAL+1))
  if go test "$pkg" 2>&1 | grep -q "^ok"; then
    echo "✓ $pkg"
    PASS=$((PASS+1))
    return 0
  else
    echo "✗ $pkg"
    FAIL_LIST="$FAIL_LIST $pkg"
    return 1
  fi
}

# Core packages
for pkg in core consensus consensus/dummy params params/extras rpc; do
  test_package ./$pkg
done

# Plugin/evm packages
for pkg in plugin/evm/blockgascost plugin/evm/config plugin/evm/header plugin/evm/validators plugin/evm/log plugin/evm/message; do
  test_package ./$pkg
done

# Precompile packages
for pkg in precompile/allowlist precompile/contract precompile/modules; do
  test_package ./$pkg
done

# Other important packages
for pkg in accounts/abi eth/filters miner core/state core/txpool/legacypool; do
  test_package ./$pkg
done

echo ""
echo "=== FINAL RESULTS ==="
echo "Total packages tested: $TOTAL"
echo "Packages passing: $PASS"
echo "Packages failing: $((TOTAL - PASS))"
echo "Pass rate: $((PASS * 100 / TOTAL))%"
if [ -n "$FAIL_LIST" ]; then
  echo "Failed packages:$FAIL_LIST"
fi
