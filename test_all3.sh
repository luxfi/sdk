#!/bin/bash
PASS=0
FAIL=0
for dir in $(find . -name "*_test.go" -type f | xargs dirname | sort -u | tail -n +41); do
  if go test "$dir" 2>&1 | grep -q "^ok"; then
    echo "✓ $dir"
    ((PASS++))
  else
    echo "✗ $dir"
    ((FAIL++))
  fi
done
echo "Results: $PASS passing, $FAIL failing"
if [ $((PASS + FAIL)) -gt 0 ]; then
  echo "Pass rate: $(( PASS * 100 / (PASS + FAIL) ))%"
fi
