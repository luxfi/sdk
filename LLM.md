# LLM.md - Lux EVM Module

## Project Overview
Lux EVM (formerly Subnet-EVM) is the Ethereum Virtual Machine implementation for Lux subnets. This module provides EVM compatibility for the Lux network.

## CRITICAL VERSION REQUIREMENTS
**ALWAYS use these specific versions for backwards compatibility:**
- `github.com/luxfi/node v1.13.4` - NOT v1.16.x (maintains compatibility with avalanchego)
- `github.com/luxfi/geth v1.16.2-lux.4` - Our fork of go-ethereum
- Local packages from parent directory for: consensus, crypto, warp

### IMPORTANT: Version-specific changes
When using node v1.13.4:
- Use `acp118` package instead of `lp118` (renamed in later versions)
- Import: `github.com/luxfi/node/network/p2p/acp118`
- All handler IDs: `acp118.HandlerID`
- All functions: `acp118.NewCachedHandler`, `acp118.NewSignatureAggregator`

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
go build ./...
go test ./...
```

### Common Issues and Fixes

1. **Version Mismatch Errors**
   - Always use node v1.13.4, not latest
   - Check go.mod replace directives

2. **Interface Compatibility**
   - Block needs SetStatus method (even if no-op)
   - BuildBlock must return consensuschain.Block
   - Context is context.Context, not a struct

3. **Missing Metrics**
   - Metrics registration is currently disabled (TODO)
   - Will be re-enabled when consensus context supports it

## Testing
- Run with `-short` flag for quick tests
- 28 packages with tests, 14 without (expected)
- No test failures should occur

## Important Notes
- This module is actively being migrated from subnet-evm
- Maintains backwards compatibility with existing Lux subnets
- Uses single validator POA for development (k=1 consensus)