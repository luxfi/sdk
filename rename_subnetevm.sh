#!/bin/bash

# Rename SubnetEVM to EVM throughout the codebase
# This script updates variable names, function names, and comments

echo "Renaming SubnetEVM to EVM..."

# Update Go files
find . -name "*.go" -type f | while read file; do
    # Skip vendor and .git directories
    if [[ "$file" == *"/vendor/"* ]] || [[ "$file" == *"/.git/"* ]]; then
        continue
    fi
    
    # Create a temporary file
    temp_file="${file}.tmp"
    
    # Perform replacements
    sed -e 's/SubnetEVM/EVM/g' \
        -e 's/SubnetEvm/Evm/g' \
        -e 's/subnetEVM/evm/g' \
        -e 's/subnet-evm/evm/g' \
        -e 's/Subnet EVM/EVM/g' \
        "$file" > "$temp_file"
    
    # Only replace if changes were made
    if ! cmp -s "$file" "$temp_file"; then
        mv "$temp_file" "$file"
        echo "Updated: $file"
    else
        rm "$temp_file"
    fi
done

# Update markdown files
find . -name "*.md" -type f | while read file; do
    if [[ "$file" == *"/vendor/"* ]] || [[ "$file" == *"/.git/"* ]]; then
        continue
    fi
    
    temp_file="${file}.tmp"
    
    sed -e 's/Subnet-EVM/EVM/g' \
        -e 's/Subnet EVM/EVM/g' \
        -e 's/subnet-evm/evm/g' \
        -e 's/SubnetEVM/EVM/g' \
        "$file" > "$temp_file"
    
    if ! cmp -s "$file" "$temp_file"; then
        mv "$temp_file" "$file"
        echo "Updated: $file"
    else
        rm "$temp_file"
    fi
done

# Update JSON files
find . -name "*.json" -type f | while read file; do
    if [[ "$file" == *"/vendor/"* ]] || [[ "$file" == *"/.git/"* ]]; then
        continue
    fi
    
    temp_file="${file}.tmp"
    
    sed -e 's/SubnetEVM/EVM/g' \
        -e 's/subnet-evm/evm/g' \
        "$file" > "$temp_file"
    
    if ! cmp -s "$file" "$temp_file"; then
        mv "$temp_file" "$file"
        echo "Updated: $file"
    else
        rm "$temp_file"
    fi
done

echo "Renaming complete!"