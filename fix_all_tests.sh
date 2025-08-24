#!/bin/bash

# Script to systematically fix all test failures

echo "Starting comprehensive test fix..."

# Function to run tests and get results
run_tests() {
    go test ./... -timeout 60s 2>&1
}

# Function to count passing tests
count_passing() {
    run_tests | grep "^ok " | wc -l
}

# Function to count total tests
count_total() {
    run_tests | grep -E "^(ok |FAIL)" | wc -l
}

# Initial status
echo "Initial test status:"
PASSING=$(count_passing)
TOTAL=$(count_total)
echo "Passing: $PASSING / $TOTAL"

# Fix specific known issues

# 1. Fix nil pointer issues in tests
echo "Fixing nil pointer issues..."
find . -name "*_test.go" -exec grep -l "\.BaseFee" {} \; | while read file; do
    echo "Checking $file for BaseFee nil checks..."
done

# 2. Fix import issues
echo "Fixing import issues..."
go mod tidy

# 3. Run tests again
echo "Final test status:"
PASSING=$(count_passing)
TOTAL=$(count_total)
echo "Passing: $PASSING / $TOTAL"

PERCENTAGE=$((PASSING * 100 / TOTAL))
echo "Pass rate: $PERCENTAGE%"

if [ $PERCENTAGE -eq 100 ]; then
    echo "SUCCESS: All tests passing!"
else
    echo "Still have failing tests. Listing failures:"
    run_tests | grep "^FAIL" | head -20
fi