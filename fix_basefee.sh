#!/bin/bash

# Find all test files that reference BaseFee and might have nil pointer issues
echo "Finding test files with potential BaseFee nil pointer issues..."

FILES=$(grep -r "\.BaseFee" --include="*_test.go" -l)

for file in $FILES; do
    echo "Checking $file..."
    
    # Check if file has nil checks for BaseFee
    if ! grep -q "BaseFee != nil" "$file"; then
        echo "  - File might need BaseFee nil checks: $file"
        
        # Count occurrences that might need fixing
        COUNT=$(grep -c "\.BaseFee" "$file")
        echo "    Found $COUNT references to BaseFee"
    fi
done

echo "Done analyzing. Please fix the files listed above."