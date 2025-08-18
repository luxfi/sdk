# LLM.md - Lux Subnet EVM Project Guide

## Project Overview
This is the Lux Subnet EVM implementation - a simplified version of Coreth VM (C-Chain) that defines Subnet Contract Chains. It implements the Ethereum Virtual Machine and supports Solidity smart contracts.

## Architecture

### Core Components
- **accounts/**: Account management and ABI handling
- **core/**: Blockchain core (state, transactions, VM)
- **eth/**: Ethereum protocol implementation
- **consensus/**: Consensus mechanisms (dummy, misc)
- **plugin/**: VM plugin implementation for Luxd integration
- **warp/**: Warp messaging for cross-chain communication
- **precompile/**: Precompiled contracts (feemanager, rewardmanager, etc.)

### Key Dependencies
- `github.com/luxfi/node`: Main Lux node implementation
- `github.com/luxfi/consensus`: Consensus protocol implementation
- `github.com/luxfi/geth`: Modified go-ethereum fork for Lux
- `github.com/luxfi/database`: Database abstraction layer
- `github.com/luxfi/ids`: ID types and utilities

## Implementation Details

### Context Management
The project uses `context.Context` with helper functions from `consensus` package:
- `consensus.GetNetworkID(ctx)` - Get network ID
- `consensus.GetChainID(ctx)` - Get chain ID
- `consensus.GetSubnetID(ctx)` - Get subnet ID
- `consensus.GetValidatorState(ctx)` - Get validator state
- `consensus.GetChainDataDir(ctx)` - Get chain data directory

### Type Conversions
Frequent conversions between Lux and Ethereum types:
```go
// crypto.Address to common.Address
luxAddr := crypto.PubkeyToAddress(key.PublicKey)
addr := common.BytesToAddress(luxAddr[:])

// crypto.Hash to common.Hash
luxHash := crypto.Keccak256Hash(data)
hash := common.BytesToHash(luxHash[:])
```

### VM Integration
The VM struct uses `context.Context` for consensus context, not `snow.Context`:
```go
type VM struct {
    ctx context.Context  // Consensus context
    vmLock sync.RWMutex
    // ...
}
```

## Current Issues and Fixes Applied

### Fixed Issues
1. **Metrics Timer vs ResettingTimer**: Changed Timer to ResettingTimer in pathdb metrics
2. **Type conversions**: Fixed crypto.Address to common.Address conversions throughout
3. **API changes**: Updated ReadHeaderNumber, SetBalance, Commit signatures
4. **VM initialization**: Fixed vm.NewEVM to use 4 parameters instead of 5
5. **Import paths**: Changed imports from luxfi/node/consensus to luxfi/consensus

### Remaining Issues
1. **ValidatorState interface**: Mismatch between consensus.ValidatorState and expected methods
2. **Database metrics**: PrometheusRegistry missing in luxfi/database/meterdb
3. **Network interfaces**: Connected method signature mismatches
4. **Plugin/evm**: Some validator manager context access issues

## Build Status
- **Total packages**: 124
- **Building successfully**: 111/124 (89.5%)
- **Passing tests**: 31/124 (25%)
- **Build failures**: 13 packages (mostly in plugin/, warp/, tests/)

## Common Patterns

### Error Handling
```go
if err != nil {
    return fmt.Errorf("failed to %s: %w", action, err)
}
```

### Context Usage
```go
// Get chain properties from context
networkID := consensus.GetNetworkID(ctx)
chainID := consensus.GetChainID(ctx)
subnetID := consensus.GetSubnetID(ctx)
validatorState := consensus.GetValidatorState(ctx)
```

### State Management
```go
// Snapshot and revert pattern
snap := stateDB.Snapshot()
// ... perform operations ...
if err != nil {
    stateDB.RevertToSnapshot(snap)
}
```

## Testing
Run tests with:
```bash
go test ./...           # Run all tests
go test -short ./...    # Run short tests only
go build ./...          # Build all packages
```

## Key Files and Locations
- **VM Implementation**: `plugin/evm/vm.go`
- **Network Layer**: `network/network.go`
- **Consensus Integration**: `consensus/`
- **Warp Messaging**: `warp/service.go`, `warp/backend.go`
- **Precompiled Contracts**: `precompile/contracts/`
- **Core Blockchain**: `core/blockchain.go`, `core/state_processor.go`

## Development Notes
1. Always use luxfi/ packages instead of external equivalents
2. Use consensus helper functions for context field access
3. Convert between Lux and Ethereum types explicitly
4. Check for nil contexts before accessing fields
5. Use proper error wrapping with %w for error chains

## Next Steps for Full Compatibility
1. Fix ValidatorState interface to match expected signatures
2. Resolve database metrics PrometheusRegistry issues
3. Complete network interface implementations
4. Fix remaining test failures through proper mocking
5. Ensure all packages use luxfi/ dependencies consistently