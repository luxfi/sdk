# CI Build Guide for LUX EVM Module

## Problem

The EVM module has local replace directives in go.mod that point to local directories. These directories don't exist in CI environments, causing build failures.

## Solution for CI

### Option 1: Use the CI Build Script (Recommended)

```bash
# In your CI workflow:
./ci_setup.sh
go build ./...
go test ./...
```

### Option 2: Manual Setup

1. **Remove local replace directives before building:**

```bash
# Save original go.mod
cp go.mod go.mod.backup

# Remove local replace directives (keep tablewriter replace)
sed -i '/replace.*\.\.\/.*$/d' go.mod
sed -i '/replace.*\/home\/.*$/d' go.mod

# Build
go build ./...
```

2. **Exclude problematic packages:**

Some packages depend on unpublished node packages. Exclude them from CI:

```bash
# Build excluding tests and examples
go build $(go list ./... | grep -v '/tests' | grep -v '/examples')

# Test excluding problematic packages
go test $(go list ./... | grep -v '/tests' | grep -v '/examples')
```

### Option 3: GitHub Actions Workflow

Add this to your `.github/workflows/build.yml`:

```yaml
- name: Prepare for CI Build
  run: |
    # Remove local replace directives
    cp go.mod go.mod.backup
    sed -i '/replace.*\.\.\/.*$/d' go.mod
    sed -i '/replace.*\/home\/.*$/d' go.mod

- name: Build
  run: |
    go build $(go list ./... | grep -v '/tests' | grep -v '/examples')

- name: Test
  run: |
    go test $(go list ./... | grep -v '/tests' | grep -v '/examples')
```

## Local Development

For local development with the monorepo structure, keep the replace directives uncommented:

```go
replace (
    github.com/luxfi/consensus => ../consensus
    github.com/luxfi/crypto => ../crypto
    github.com/luxfi/database => ../database
    github.com/luxfi/geth => /home/z/work/lux/geth
    github.com/luxfi/go-bip39 => ../go-bip39
    github.com/luxfi/ids => ../ids
    github.com/luxfi/log => ../log
    github.com/luxfi/metric => ../metric
    github.com/luxfi/node => ../node
    github.com/luxfi/warp => ../warp
)
```

## Files Requiring Special Handling

These files import packages not available in published modules:

1. `examples/sign-uptime-message/main.go` - Uses `github.com/luxfi/node/wallet/net/primary`
2. `tests/utils/subnet.go` - Uses `github.com/luxfi/node/wallet/net/primary`
3. `precompile/contracts/warp/signature_verification_test.go` - Uses `github.com/luxfi/node/utils/crypto/bls`

These should be excluded from CI builds as they're test/example files that require the full monorepo.

## Complete CI Setup Script

Create a `ci_setup.sh` file:

```bash
#!/bin/bash
set -e

echo "Setting up CI environment..."

# Remove local replace directives
cp go.mod go.mod.backup
sed -i '/replace.*\.\.\/.*$/d' go.mod
sed -i '/replace.*\/home\/.*$/d' go.mod

# Download dependencies
go mod download

echo "CI setup complete."
```

## Summary

The key issue is that the go.mod file has replace directives pointing to local directories that don't exist in CI. The solution is to:

1. Remove these replace directives for CI builds
2. Exclude test/example files that depend on unpublished packages
3. Use the provided scripts or GitHub Actions workflow configuration