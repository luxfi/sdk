#!/usr/bin/env python3
import os
import re

def fix_go_file(filepath):
    """Fix Go imports in a single file."""
    with open(filepath, 'r') as f:
        content = f.read()
    
    original_content = content
    
    # Map of old imports to new imports based on go-ethereum structure
    import_mappings = {
        # Core types and interfaces
        'github.com/luxdefi/evm/core/types': 'github.com/ethereum/go-ethereum/core/types',
        'github.com/luxdefi/evm/core/rawdb': 'github.com/ethereum/go-ethereum/core/rawdb',
        'github.com/luxdefi/evm/core/vm': 'github.com/ethereum/go-ethereum/core/vm',
        'github.com/luxdefi/evm/accounts': 'github.com/ethereum/go-ethereum/accounts',
        'github.com/luxdefi/evm/accounts/external': 'github.com/ethereum/go-ethereum/accounts/external',
        'github.com/luxdefi/evm/accounts/keystore': 'github.com/ethereum/go-ethereum/accounts/keystore',
        'github.com/luxdefi/evm/ethdb': 'github.com/ethereum/go-ethereum/ethdb',
        'github.com/luxdefi/evm/trie': 'github.com/ethereum/go-ethereum/trie',
        'github.com/luxdefi/evm/metrics': 'github.com/ethereum/go-ethereum/metrics',
        
        # Interfaces that no longer exist - use direct imports
        'github.com/luxdefi/evm/interfaces/ethdb': 'github.com/ethereum/go-ethereum/ethdb',
        'github.com/luxdefi/evm/interfaces/core/vm': 'github.com/ethereum/go-ethereum/core/vm',
        'github.com/luxdefi/evm/interfaces/libevm': 'github.com/ethereum/go-ethereum/libevm',
        'github.com/luxdefi/evm/interfaces/libevm/legacy': 'github.com/ethereum/go-ethereum/libevm/legacy',
        'github.com/luxdefi/evm/interfaces/libevm/stateconf': 'github.com/ethereum/go-ethereum/libevm/stateconf',
        'github.com/luxdefi/evm/interfaces/params': 'github.com/ethereum/go-ethereum/params',
        'github.com/luxdefi/evm/interfaces/trie/trienode': 'github.com/ethereum/go-ethereum/trie/trienode',
        'github.com/luxdefi/evm/interfaces/trie/triestate': 'github.com/ethereum/go-ethereum/trie/triestate',
        'github.com/luxdefi/evm/interfaces/triedb': 'github.com/ethereum/go-ethereum/triedb',
        'github.com/luxdefi/evm/interfaces/triedb/database': 'github.com/ethereum/go-ethereum/triedb/database',
        'github.com/luxdefi/evm/interfaces/triedb/pathdb': 'github.com/ethereum/go-ethereum/triedb/pathdb',
        
        # golang.org/x/exp/slices is now in standard library
        'golang.org/x/exp/slices': 'slices',
        'golang.org/x/exp/slog': 'log/slog',
    }
    
    # Apply mappings
    for old_import, new_import in import_mappings.items():
        content = content.replace(f'"{old_import}"', f'"{new_import}"')
        content = content.replace(f"'{old_import}'", f"'{new_import}'")
    
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