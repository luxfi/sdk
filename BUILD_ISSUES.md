# EVM Build Issues and Solutions

## Current Problems

### 1. Import Cycles
The EVM package has several import cycles:
- `core` → `consensus` → `core/state` → `core` (cycle)
- `core` → `consensus` → `params` → `params/extras` → `precompile/modules` → `core/vm` → `core` (cycle)
- `plugin/evm` → `sync/client` → `sync/handlers` → `ethdb/memorydb` → cycle
- `triedb/pathdb` → `triedb/pathdb` (self-import cycle)

### 2. Missing Internal Packages
The build system is treating internal EVM packages as external modules:
- `github.com/luxfi/evm/common`
- `github.com/luxfi/evm/crypto`
- `github.com/luxfi/evm/ethdb`
- etc.

### 3. Architecture Issues
The current structure has too many interdependencies between packages.

## Solution Approach

### Step 1: Break Import Cycles
1. **Extract Interfaces**: Create interface packages to break circular dependencies
2. **Reorganize Package Structure**: Move shared types to common packages
3. **Use Dependency Injection**: Pass dependencies rather than importing directly

### Step 2: Fix Package Structure
```
evm/
├── interfaces/          # Shared interfaces (no imports from other evm packages)
│   ├── consensus.go
│   ├── state.go
│   └── vm.go
├── types/              # Shared types (minimal imports)
│   ├── block.go
│   ├── transaction.go
│   └── state.go
├── core/               # Core logic (imports interfaces and types)
│   ├── blockchain.go
│   ├── state/
│   └── vm/
├── consensus/          # Consensus engines (imports interfaces)
│   └── dummy/
├── plugin/             # VM plugin (imports everything)
│   └── evm/
└── utils/              # Utilities (no imports from other packages)
```

### Step 3: Migration Path
1. Create the new package structure
2. Move interfaces to break cycles
3. Update imports
4. Test compilation

## Immediate Fix

For now, to get things compiling, we need to:
1. Use go-ethereum imports directly where needed
2. Simplify the package structure
3. Remove circular dependencies