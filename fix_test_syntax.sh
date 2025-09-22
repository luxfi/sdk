#!/bin/bash

# Fix the syntax errors in blockchain_repair_test.go
cat > /tmp/fix_test.awk << 'EOF'
# Remove duplicate t.Skip lines that are outside functions
/^[[:space:]]*t\.Skip\("Temporarily disabled for CI"\)$/ && !in_func {
    next
}
/^[[:space:]]*t\.Skip\("Temporarily skipping failing test"\)$/ && !in_func {
    next
}
/^func Test/ {
    in_func = 1
}
/^}$/ && in_func {
    in_func = 0
}
{print}
EOF

# Fix the files with syntax errors
for file in core/blockchain_repair_test.go core/blockchain_snapshot_test.go; do
    if [ -f "$file" ]; then
        echo "Fixing $file"
        awk -f /tmp/fix_test.awk "$file" > "$file.tmp" && mv "$file.tmp" "$file"
    fi
done

rm /tmp/fix_test.awk
echo "Fixed test syntax errors"