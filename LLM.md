# LLM.md - Lux EVM Module

## Project Overview
Lux EVM (formerly Subnet-EVM) is the Ethereum Virtual Machine implementation for Lux subnets. This module provides EVM compatibility for the Lux network.

## CRITICAL VERSION REQUIREMENTS
**ALWAYS use these Lux-specific versions:**
- `github.com/luxfi/node v1.16.15` - Latest Lux node version
- `github.com/luxfi/geth v1.16.2-lux.4` - Our fork of go-ethereum
- Local packages from parent directory: consensus, crypto, warp (tagged v1.16.15-lux)

### IMPORTANT: Package Usage
- Use `lp118` package for p2p handlers
- Import: `github.com/luxfi/node/network/p2p/lp118`
- All handler IDs: `lp118.HandlerID`
- All functions: `lp118.NewCachedHandler`, `lp118.NewSignatureAggregator`
- NEVER import from ava-labs packages
- NEVER use go-ethereum directly, always use luxfi/geth

## Module Structure
```
/home/z/work/lux/evm/
├── plugin/evm/        # Main VM implementation
├── core/              # Core blockchain logic
├── consensus/         # Consensus engine (dummy for Lux)
├── eth/               # Ethereum protocol implementation
├── miner/             # Block building and mining
├── precompile/        # Precompiled contracts and warp
├── network/           # P2P networking
├── params/            # Chain configuration
├── scripts/           # Build and test scripts
└── warp/              # Cross-subnet messaging
```

## Current Build Status
**🔄 IN PROGRESS - Updating to Latest Lux Versions**

The module is being updated to use:
1. **node v1.16.15** - Latest Lux node version with lp118 support
2. **consensus v1.16.15-lux** - Tagged to match node version
3. **crypto v1.16.15-lux** - Tagged to match node version
4. **warp v1.16.15-lux** - Tagged to match node version

### Main Compatibility Issues
1. **ID Types**: Node expects `node/ids.NodeID` but consensus uses `luxfi/ids.NodeID`
2. **Block Interface**: Mismatch between `consensus/chain.Block` and `node/consensus/chain.Block`
3. **Logger Interface**: luxfi/log.Logger vs node/utils/logging.Logger incompatibility
4. **AppError Types**: consensus/core.AppError vs node/snow/engine/common.AppError

### Fixes Applied
- Updated to latest Lux node v1.16.15
- Using lp118 package for p2p handlers
- Context changed from struct to context.Context
- Network module uses ID conversion functions
- All local modules tagged with v1.16.15-lux for consistency

### Remaining Work
1. **Create adapter layer** between node and consensus ID types
2. **Implement logger wrapper** for luxfi/log to node/utils/logging
3. **Fix Block interface** to satisfy both consensus and node requirements
4. **Update all ID conversions** throughout the codebase

## Key Implementation Details

### Context Management
- VM uses `context.Context` instead of consensus.Context struct
- Access consensus data via helper functions:
  - `consensus.GetChainID(ctx)`
  - `consensus.GetNetworkID(ctx)`
  - `consensus.GetNodeID(ctx)`
  - `consensus.GetLogger(ctx)`
  - `consensus.GetWarpSigner(ctx)`

### Interface Compatibility
- Block implements both `chain.Block` and `consensuschain.Block`
- BuildBlock returns `consensuschain.Block` for compatibility
- AppSender uses `set.Set[ids.NodeID]` for node sets
- Version uses `consensus/version.Application` not node's version

### Package Dependencies
**NEVER use these packages:**
- ❌ `github.com/ethereum/go-ethereum` - Use `github.com/luxfi/geth`
- ❌ `github.com/ava-labs/*` - Use `github.com/luxfi/*`
- ❌ `github.com/luxfi/node v1.16.x` - Use v1.13.4 for compatibility

**Always use:**
- ✅ `github.com/luxfi/consensus` - Local consensus package
- ✅ `github.com/luxfi/crypto` - Local crypto package
- ✅ `github.com/luxfi/warp` - Local warp package
- ✅ `github.com/luxfi/geth` - Our Ethereum fork

### Build Commands
```bash
cd /home/z/work/lux/evm
go build ./...  # Currently fails due to interface issues
go test ./...   # Will work after build issues are resolved
```

### Common Issues and Fixes

1. **Version Requirements**
   - Always use node v1.16.15 or later Lux versions
   - Never use ava-labs packages
   - Check go.mod replace directives

2. **Interface Compatibility**
   - Block needs SetStatus method (even if no-op)
   - BuildBlock must return consensuschain.Block
   - Context is context.Context, not a struct

3. **Missing Metrics**
   - Metrics registration is currently disabled (TODO)
   - Will be re-enabled when consensus context supports it

4. **ID Type Conversions**
   ```go
   // Convert between node's IDs and consensus IDs
   func nodeIDToConsensus(id nodeids.NodeID) ids.NodeID {
       var consensusID ids.NodeID
       copy(consensusID[:], id[:])
       return consensusID
   }
   ```

## Testing
- Run with `-short` flag for quick tests
- 28 packages with tests, 14 without (expected)
- Tests will pass after build issues are resolved

## Important Notes
- This module is actively being migrated from subnet-evm
- Maintains backwards compatibility with existing Lux subnets
- Uses single validator POA for development (k=1 consensus)
- Major refactoring needed to reconcile ID type differences between packages