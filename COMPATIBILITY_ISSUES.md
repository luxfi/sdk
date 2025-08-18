# EVM Module Compatibility Issues

## Current Status
Using **node v1.13.7-lux.3** as requested (v1.16.15 was "outdated and broken" per user)

## Core Problem
The EVM module has fundamental type incompatibilities between:
- `github.com/luxfi/consensus` (standalone consensus package)
- `github.com/luxfi/node` (v1.13.7-lux.3)

These packages define the same interfaces with incompatible types.

## Specific Issues

### 1. Block Interface Mismatch
**Problem**: Different Block types
- Node expects: `github.com/luxfi/node/vms/components/chain.Block`
- Consensus uses: `github.com/luxfi/consensus/chain.Block`

**Files affected**: 
- `plugin/evm/vm.go` (lines 768-771)
- `plugin/evm/block.go`

### 2. Warp Message Types
**Problem**: Different UnsignedMessage types
- Node expects: `github.com/luxfi/node/vms/platformvm/warp.UnsignedMessage`
- We have: `github.com/luxfi/warp.UnsignedMessage`

**Files affected**:
- `plugin/evm/vm.go` (line 541)

### 3. BlockContext Types
**Problem**: Different Context structs
- Node has: `github.com/luxfi/node/consensus/engine/chain/block.Context`
- Consensus has: `github.com/luxfi/consensus/engine/chain/block.Context`

**Files affected**:
- `plugin/evm/block.go` (line 195)

### 4. AppError Types
**Problem**: Different AppError types
- Node expects: `github.com/luxfi/node/consensus/engine/core.AppError`
- Consensus has: `github.com/luxfi/consensus/core.AppError`

**Files affected**:
- `plugin/evm/vm.go` (line 541)

### 5. NetworkUpgrades Context
**Problem**: `ctx.NetworkUpgrades` doesn't exist
- The context is now `context.Context` not a struct with fields

**Files affected**:
- `plugin/evm/vm.go` (line 615)

### 6. StateSyncing Undefined
**Problem**: `consensus.StateSyncing` doesn't exist in the consensus package

**Files affected**:
- `plugin/evm/vm.go` (line 800)

## Root Cause
The codebase appears to be in transition between two architectures:
1. **Old**: Consensus was part of node (`node/consensus/...`)
2. **New**: Consensus is standalone (`github.com/luxfi/consensus`)

The node v1.13.x series expects the old architecture, but the code is being updated to use the new standalone consensus package.

## Possible Solutions

### Option 1: Use Node's Consensus (Not allowed)
The user explicitly said "NO WE DO NOT USE node/consensus it's luxfi/consensus"

### Option 2: Update Node to Compatible Version
Need a node version that works with the standalone consensus package. This may not exist yet.

### Option 3: Create Adapter Layer
Create comprehensive adapters between all incompatible types. This is complex and fragile.

### Option 4: Fork Node
Create a custom node version that uses the new consensus package.

## Next Steps Needed
1. Clarify which node version is compatible with the standalone consensus package
2. Or confirm if adapter layers should be created
3. Or confirm if a custom node fork is needed

## Files Modified So Far
- `plugin/evm/block.go` - Updated imports, SetStatus
- `plugin/evm/block_builder.go` - Fixed MessageType
- `plugin/evm/factory.go` - Fixed Logger interface
- `plugin/evm/gossip.go` - Updated imports
- `plugin/evm/vm.go` - Multiple partial fixes
- `plugin/evm/engine_adapter.go` - Created for AppError adaptation
- `plugin/evm/logger_adapter.go` - Created for Logger adaptation
- `go.mod` - Updated to node v1.13.7-lux.3