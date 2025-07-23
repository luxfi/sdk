# Unified Lux EVM Architecture

## Vision
A single EVM package that serves all Lux Network EVM needs:
- **C-Chain**: Primary network EVM chain (like Avalanche C-Chain)
- **L2 Subnets**: Classic Lux Subnet EVM (like Zoo mainnet)
- **L3+**: Advanced configurations with custom sequencers, OP Stack, etc.

## Core Architecture

### 1. Mode-Based Configuration

```go
package evm

type Mode string

const (
    ModeCChain    Mode = "c-chain"     // Primary network C-Chain
    ModeSubnetL2  Mode = "subnet-l2"   // Classic subnet EVM
    ModeSequencer Mode = "sequencer"   // Sequencer-based L2/L3
    ModeOPStack   Mode = "op-stack"    // OP Stack compatible
    ModeHybrid    Mode = "hybrid"      // Multi-consensus omnichain
)

type Config struct {
    // Common EVM configuration
    ChainID        *big.Int
    Mode           Mode
    NetworkID      uint32
    
    // Mode-specific configurations
    CChainConfig   *CChainConfig   `json:",omitempty"`
    SubnetConfig   *SubnetConfig   `json:",omitempty"`
    SequencerConfig *SequencerConfig `json:",omitempty"`
    OPStackConfig  *OPStackConfig  `json:",omitempty"`
    
    // Consensus configuration
    ConsensusEngine string // "snowman", "sequencer", "op-stack", "hybrid"
    
    // State configuration
    StateDBPath    string
    StateMigration *StateMigrationConfig
}
```

### 2. Consensus Abstraction Layer

```go
package consensus

// ConsensusEngine abstracts different consensus mechanisms
type ConsensusEngine interface {
    // Block production
    BuildBlock(ctx context.Context) (Block, error)
    VerifyBlock(block Block) error
    
    // State transitions
    ProcessBlock(block Block) error
    Finalize(block Block) error
    
    // Network-specific
    GetValidators() ([]Validator, error)
    GetSequencer() (Sequencer, error) // For L2/L3
}

// Implementations
type SnowmanConsensus struct{} // For C-Chain and classic subnets
type SequencerConsensus struct{} // For centralized sequencer
type OPStackConsensus struct{} // For OP Stack
type HybridConsensus struct{} // For multi-consensus
```

### 3. Unified VM Structure

```go
package plugin

type VM struct {
    // Core components (shared across all modes)
    ctx           *snow.Context
    db            database.Database
    blockchain    *core.BlockChain
    txPool        *core.TxPool
    state         *StateManager
    
    // Mode-specific components
    mode          Mode
    consensus     ConsensusEngine
    
    // APIs
    publicAPI     []rpc.API
    adminAPI      []rpc.API
    
    // Configuration
    config        *Config
}

// Initialize configures the VM based on mode
func (vm *VM) Initialize(
    ctx *snow.Context,
    dbManager manager.Manager,
    genesisBytes []byte,
    upgradeBytes []byte,
    configBytes []byte,
    toEngine chan<- commonEng.Message,
    fxs []*commonEng.Fx,
    appSender commonEng.AppSender,
) error {
    var config Config
    if err := json.Unmarshal(configBytes, &config); err != nil {
        return err
    }
    
    vm.mode = config.Mode
    vm.config = &config
    
    // Initialize consensus based on mode
    switch config.Mode {
    case ModeCChain:
        vm.consensus = NewSnowmanConsensus(config.CChainConfig)
    case ModeSubnetL2:
        vm.consensus = NewSnowmanConsensus(config.SubnetConfig)
    case ModeSequencer:
        vm.consensus = NewSequencerConsensus(config.SequencerConfig)
    case ModeOPStack:
        vm.consensus = NewOPStackConsensus(config.OPStackConfig)
    case ModeHybrid:
        vm.consensus = NewHybridConsensus(config)
    }
    
    // Initialize blockchain with appropriate configuration
    vm.initBlockchain()
    
    // Handle state migration if needed (e.g., loading Zoo mainnet state)
    if config.StateMigration != nil {
        vm.migrateState(config.StateMigration)
    }
    
    return nil
}
```

### 4. State Migration Support

```go
package state

type StateMigrationConfig struct {
    SourceType   string // "pebbledb", "leveldb", "snapshot"
    SourcePath   string // Path to existing state
    TargetHeight uint64 // Height to migrate up to
    Verify       bool   // Verify state after migration
}

func MigrateState(
    source database.Database,
    target database.Database,
    config *StateMigrationConfig,
) error {
    // Migrate existing chain state (e.g., Zoo mainnet)
    // to new unified EVM instance
}
```

### 5. Package Structure (Avoiding Import Cycles)

```
evm/
├── go.mod
├── core/                    # Core EVM logic (extends go-ethereum)
│   ├── blockchain.go
│   ├── state/
│   ├── types/
│   └── vm/
├── consensus/              # Consensus abstraction
│   ├── interface.go
│   ├── snowman.go         # Avalanche consensus
│   ├── sequencer.go       # Centralized sequencer
│   ├── opstack.go         # OP Stack consensus
│   └── hybrid.go          # Multi-consensus
├── plugin/                 # VM implementation
│   ├── vm.go              # Main VM struct
│   ├── factory.go         # VM factory
│   └── config.go          # Configuration
├── api/                    # RPC APIs
│   ├── public.go
│   └── admin.go
├── sync/                   # State sync
├── warp/                   # Cross-chain messaging
└── utils/                  # Utilities

// Key: No imports from plugin/ to core/ or consensus/
// Only core/ and consensus/ can be imported by plugin/
```

### 6. Configuration Examples

#### C-Chain Configuration
```json
{
  "mode": "c-chain",
  "chainId": 43114,
  "consensusEngine": "snowman",
  "cChainConfig": {
    "allowFeeRecipients": true,
    "feeConfig": {
      "gasLimit": 15000000,
      "targetBlockRate": 2,
      "minBaseFee": 25000000000
    }
  }
}
```

#### L2 Subnet Configuration (like Zoo)
```json
{
  "mode": "subnet-l2",
  "chainId": 281123,
  "consensusEngine": "snowman",
  "subnetConfig": {
    "subnetID": "2PYmKXzSJCPcgKUhKgxPAHKfxeFbPFBH2Y7JqfmV3mJYLHaAZS",
    "validatorOnly": false
  },
  "stateMigration": {
    "sourceType": "pebbledb",
    "sourcePath": "/zoo-mainnet/chaindata",
    "targetHeight": 1000000
  }
}
```

#### L3 with Sequencer
```json
{
  "mode": "sequencer",
  "chainId": 999999,
  "consensusEngine": "sequencer",
  "sequencerConfig": {
    "sequencerAddress": "0x...",
    "blockTime": 2,
    "maxBlockSize": 30000000
  }
}
```

### 7. Implementation Priority

1. **Fix Import Cycles**: Restructure packages to have clear dependency flow
2. **Create Mode System**: Implement configuration-based mode selection
3. **Abstract Consensus**: Create consensus interface and implementations
4. **State Migration**: Add ability to load existing chain states
5. **Test Integration**: Verify with existing Zoo mainnet data

This architecture provides:
- Single codebase for all EVM needs
- Clean separation between modes
- No import cycles
- Flexibility for future consensus mechanisms
- Ability to migrate existing chains
- Support for L1/L2/L3 configurations