#!/usr/bin/env python3
import re
import sys

def fix_syntax_errors(filepath):
    """Fix common syntax errors introduced by incorrect lint fixes."""
    with open(filepath, 'r') as f:
        lines = f.readlines()

    modified = False
    for i, line in enumerate(lines):
        # Fix "_, _ = if" pattern
        if "_, _ = if" in line:
            lines[i] = line.replace("_, _ = if", "if")
            modified = True
        # Fix "_, _ = return" pattern
        if "_, _ = return" in line:
            lines[i] = line.replace("_, _ = return", "return")
            modified = True
        # Fix "_ = defer" pattern
        if "_ = defer" in line:
            lines[i] = re.sub(r'_ = (defer.*)', r'\1', line)
            modified = True
        # Fix double assignment like "_, _ = n, err ="
        if re.search(r'_, _ = \w+, \w+ =', line):
            lines[i] = re.sub(r'_, _ = (\w+, \w+ =)', r'\1', line)
            modified = True

    if modified:
        with open(filepath, 'w') as f:
            f.writelines(lines)
        print(f"Fixed: {filepath}")

    return modified

if __name__ == "__main__":
    import glob
    import os

    os.chdir("/home/z/work/lux/evm")

    # Find all Go files
    go_files = glob.glob("**/*.go", recursive=True)

    fixed_count = 0
    for filepath in go_files:
        if fix_syntax_errors(filepath):
            fixed_count += 1

    print(f"Fixed {fixed_count} files")