#!/usr/bin/env python3
import os
import re

def fix_go_file(filepath):
    """Fix Go imports in a single file."""
    with open(filepath, 'r') as f:
        content = f.read()
    
    original_content = content
    
    # Replace luxdefi/node with luxfi/node
    content = content.replace('github.com/luxdefi/node', 'github.com/luxfi/node')
    
    # Write back if changed
    if content != original_content:
        with open(filepath, 'w') as f:
            f.write(content)
        return True
    return False

def process_directory(directory):
    """Process all Go files in a directory."""
    fixed_files = []
    
    for root, dirs, files in os.walk(directory):
        # Skip vendor and .git directories
        if 'vendor' in root or '.git' in root:
            continue
            
        for file in files:
            if file.endswith('.go'):
                filepath = os.path.join(root, file)
                if fix_go_file(filepath):
                    fixed_files.append(filepath)
    
    return fixed_files

if __name__ == '__main__':
    # Process the entire evm directory
    fixed_files = process_directory('/Users/z/work/lux/evm')
    
    print(f"Fixed {len(fixed_files)} files:")
    for file in sorted(fixed_files):
        print(f"  {file}")