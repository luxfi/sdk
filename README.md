# Lux SDK

The official Go SDK for building and managing Lux-compatible networks and blockchains. This SDK provides a unified interface integrating the full Lux ecosystem - netrunner for network orchestration, the CLI for user-friendly operations, and direct node APIs for high-performance applications.

## 🎯 Key Capabilities

The Lux SDK is your complete toolkit for blockchain development, offering:

### Network Orchestration
- **Multi-Network Management**: Launch and manage mainnet, testnet, or custom local networks
- **Dynamic Scaling**: Add/remove nodes on-the-fly with automatic rebalancing
- **Network Simulation**: Test network behaviors and consensus under various conditions
- **Performance Testing**: Built-in benchmarking and stress testing capabilities

### Blockchain Development
- **Multi-Layer Architecture**: Build L1 sovereign chains, L2 rollups, or L3 app-specific chains
- **VM Flexibility**: Deploy EVM, WASM, or custom VMs with full language support
- **Rapid Prototyping**: Go from idea to deployed blockchain in minutes
- **Migration Tools**: Seamlessly migrate from subnets to independent L1s

### Developer Experience
- **Unified API**: Single SDK interface for all Lux operations
- **Smart Defaults**: Automatic selection of best method (CLI → netrunner → native)
- **Type Safety**: Full Go type safety with comprehensive error handling
- **Extensive Examples**: Production-ready code samples for common use cases

## Features

- 🚀 **Network Management**: Full netrunner integration for complex network orchestration
- 🔗 **Blockchain Building**: Build L1/L2/L3 chains with any VM type
- 💰 **Staking & Validation**: Complete P-Chain operations for network security
- 🪙 **Asset Management**: Create, trade, and manage assets on X-Chain
- 📜 **Smart Contracts**: Deploy and interact with contracts on C-Chain
- 🌉 **Cross-Chain Operations**: Atomic swaps and seamless asset transfers
- 👛 **Wallet Integration**: HD wallets, multisig, and hardware wallet support
- 🛡️ **Quantum-Resistant**: Q-Chain integration for post-quantum cryptography
- 🌐 **WASM Support**: Run any language via WebAssembly
- 🔧 **CLI Integration**: Programmatic access to all CLI commands
- 📊 **Monitoring & Telemetry**: Built-in metrics and observability
- 🏗️ **Infrastructure as Code**: Define entire networks declaratively

## Installation

```bash
go get github.com/luxfi/sdk
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    
    "github.com/luxfi/sdk"
    "github.com/luxfi/sdk/config"
)

func main() {
    // Initialize SDK with auto-detection of available tools
    cfg := config.Default()
    cfg.NodeEndpoint = "http://localhost:9650" // Optional: connect to existing node
    
    luxSDK, err := sdk.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    
    // Launch a network (uses CLI or netrunner automatically)
    network, err := luxSDK.LaunchNetwork(ctx, "local", 5)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create and deploy blockchain with best available method
    blockchain, err := luxSDK.CreateAndDeployBlockchain(ctx, &sdk.BlockchainParams{
        Name:    "my-chain",
        Type:    blockchain.BlockchainTypeL1,
        VMType:  blockchain.VMTypeEVM,
        Network: network,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Blockchain %s deployed on network %s", blockchain.Name, network.Name)
}
```

## 🔧 Integrated Tools

The SDK seamlessly integrates with the Lux ecosystem's core tools:

### Netrunner Integration
Full network orchestration capabilities:
```go
// Direct netrunner access for advanced scenarios
netrunner := luxSDK.Netrunner()
if netrunner != nil {
    // Start netrunner server
    err := netrunner.StartServer(ctx)
    
    // Create complex network topology
    network, err := netrunner.CreateNetwork(ctx, "testnet", 11)
    
    // Deploy blockchain with specific configuration
    err = netrunner.DeployBlockchain(ctx, networkID, &BlockchainSpec{
        Name:   "my-blockchain",
        VMType: "evm",
        Genesis: customGenesis,
    })
}
```

### CLI Wrapper
Programmatic access to all CLI commands:
```go
// Use CLI commands directly
cli := luxSDK.CLI()
if cli != nil {
    // Execute any CLI command
    output, err := cli.Execute(ctx, "network", "status")
    
    // Type-safe wrappers for common operations
    err = cli.CreateBlockchain(ctx, "my-chain", "evm")
    err = cli.DeployBlockchain(ctx, "my-chain", "local")
    
    // Key management
    keys, err := cli.ListKeys(ctx)
    key, err := cli.CreateKey(ctx, "validator-key")
}
```

### Node API Client
Direct node communication for high-performance operations:
```go
// Access node APIs directly
node := luxSDK.Node()
if node != nil {
    // Get node information
    info, err := node.GetNodeInfo(ctx)
    
    // Platform chain operations
    validators, err := node.GetCurrentValidators(ctx, constants.PrimaryNetworkID)
    
    // Keystore operations
    addresses, err := node.ListAddresses(ctx, "my-user")
}
```

## 🚀 General-Purpose Use Cases

### Infrastructure Automation
```go
// Define infrastructure as code
infrastructure := &NetworkDefinition{
    Networks: []NetworkSpec{
        {Name: "prod-mainnet", Type: "mainnet", Nodes: 21},
        {Name: "staging", Type: "testnet", Nodes: 11},
        {Name: "dev", Type: "local", Nodes: 5},
    },
    Blockchains: []BlockchainSpec{
        {Name: "defi-chain", VM: "evm", Network: "prod-mainnet"},
        {Name: "nft-chain", VM: "evm", Network: "prod-mainnet"},
        {Name: "game-chain", VM: "wasm", Network: "staging"},
    },
}

// Deploy entire infrastructure
deployer := sdk.NewInfrastructureDeployer(luxSDK)
err := deployer.Deploy(ctx, infrastructure)
```

### Multi-Chain DApp Development
```go
// Build cross-chain DApp
dapp := sdk.NewDApp(luxSDK)

// Deploy contracts across multiple chains
contracts := dapp.DeployContracts(ctx, map[string][]byte{
    "c-chain": dexContract,
    "nft-chain": nftContract,
    "game-chain": gameContract,
})

// Set up cross-chain messaging
bridge := dapp.CreateBridge(ctx, []string{"c-chain", "nft-chain", "game-chain"})

// Monitor all chains
monitor := dapp.Monitor(ctx)
for event := range monitor.Events() {
    log.Printf("Chain: %s, Event: %s", event.Chain, event.Type)
}
```

### Network Testing & Simulation
```go
// Create test scenarios
tester := sdk.NewNetworkTester(luxSDK)

// Simulate network partitions
tester.SimulatePartition(ctx, []string{"node1", "node2"}, []string{"node3", "node4", "node5"})

// Test consensus under load
results := tester.StressTest(ctx, &StressTestParams{
    TPS:      10000,
    Duration: 5 * time.Minute,
    Scenario: "high-value-transfers",
})

// Chaos testing
tester.ChaosTest(ctx, &ChaosParams{
    RandomlyKillNodes: true,
    NetworkLatency:    100 * time.Millisecond,
    PacketLoss:        0.1,
})
```

### Enterprise Integration
```go
// Enterprise-grade deployment
enterprise := sdk.NewEnterpriseDeployment(luxSDK)

// Deploy with compliance requirements
chain, err := enterprise.DeployCompliantChain(ctx, &ComplianceParams{
    DataResidency: "US",
    Encryption:    "AES-256",
    AuditLog:      true,
    GDPR:          true,
})

// Set up monitoring and alerting
enterprise.SetupMonitoring(ctx, &MonitoringConfig{
    Prometheus: true,
    Grafana:    true,
    Alerts: []Alert{
        {Type: "node-down", Threshold: 1, Action: "page-oncall"},
        {Type: "high-latency", Threshold: "100ms", Action: "email"},
    },
})

// Generate compliance reports
report := enterprise.GenerateComplianceReport(ctx, time.Now().AddDate(0, -1, 0), time.Now())
```

## Core Components

### Network Management

The SDK uses netrunner for comprehensive network management:

```go
// Create a network
network, err := sdk.CreateNetwork(ctx, &network.NetworkParams{
    Name:             "test-network",
    Type:             network.NetworkTypeLocal,
    NumNodes:         5,
    EnableStaking:    true,
    EnableMonitoring: true,
})

// Add nodes
node, err := sdk.AddNode(ctx, network.ID, &network.NodeParams{
    Name:        "validator-01",
    Type:        network.NodeTypeValidator,
    StakeAmount: 2000,
})

// Manage network lifecycle
err = sdk.StartNetwork(ctx, networkID)
err = sdk.StopNetwork(ctx, networkID)
```

### Blockchain Building

Build any type of blockchain on Lux:

```go
// Create L1 (Sovereign Chain)
l1, err := sdk.CreateL1(ctx, "my-l1", &blockchain.L1Params{
    VMType:      blockchain.VMTypeEVM,
    Genesis:     genesisBytes,
    ChainConfig: configBytes,
})

// Create L2 (Based Rollup)
l2, err := sdk.CreateL2(ctx, "my-rollup", &blockchain.L2Params{
    VMType:          blockchain.VMTypeEVM,
    SequencerType:   "centralized",
    DALayer:         "celestia",
    SettlementChain: l1.ID,
})

// Create L3 (App Chain)
l3, err := sdk.CreateL3(ctx, "my-game", &blockchain.L3Params{
    VMType:  blockchain.VMTypeWASM,
    L2Chain: l2.ID,
    AppType: "gaming",
})
```

### Chain Operations

#### P-Chain (Platform Chain)

```go
// Stake on primary network
txID, err := chainManager.Stake(ctx, 
    big.NewInt(2000), // 2000 LUX
    14 * 24 * time.Hour, // 14 days
)

// Delegate to validator
txID, err := chainManager.Delegate(ctx, nodeID, amount, duration)

// Create subnet
subnetID, err := chainManager.P().CreateSubnet(ctx, &CreateSubnetParams{
    ControlKeys: []ids.ShortID{key1, key2},
    Threshold:   2,
})

// Add subnet validator
txID, err := chainManager.P().AddSubnetValidator(ctx, &AddSubnetValidatorParams{
    NodeID:   nodeID,
    SubnetID: subnetID,
    Weight:   100,
})
```

#### X-Chain (Exchange Chain)

```go
// Create asset
assetID, err := chainManager.CreateAsset(ctx, "MyToken", "MTK", totalSupply)

// Send asset
txID, err := chainManager.SendAsset(ctx, assetID, amount, recipient)

// Create NFT collection
nftID, err := chainManager.X().CreateNFT(ctx, &CreateNFTParams{
    Name:   "LuxNFT",
    Symbol: "LNFT",
})

// Trade assets
orderID, err := chainManager.TradeAssets(ctx, 
    sellAsset, sellAmount,
    buyAsset, buyAmount,
)
```

#### C-Chain (Contract Chain)

```go
// Deploy contract
address, txHash, err := chainManager.C().DeployContract(ctx, &DeployContractParams{
    Bytecode: contractBytecode,
    GasLimit: 300000,
})

// Call contract
result, err := chainManager.C().CallContract(ctx, &CallContractParams{
    To:   contractAddress,
    Data: callData,
})

// DeFi operations (coming soon)
txHash, err := chainManager.C().SwapTokens(ctx, &SwapParams{
    TokenIn:  USDC,
    TokenOut: LUX,
    Amount:   amount,
})
```

### Cross-Chain Operations

```go
// Transfer between chains
txID, err := chainManager.TransferCrossChain(ctx, &CrossChainTransferParams{
    SourceChain: "C",
    TargetChain: "P",
    AssetID:     luxAssetID,
    Amount:      amount,
    To:          recipient,
})

// Get balances across all chains
balances, err := chainManager.GetBalance(ctx, address)
for chain, balance := range balances.Chains {
    fmt.Printf("%s-Chain: %s\n", chain, balance)
}
```

### Wallet Management

```go
// Create wallet
wallet, err := chainManager.CreateWallet(ctx, "my-wallet")

// Create multisig wallet
multisig, err := chainManager.CreateMultisigWallet(ctx, "treasury", 
    []ids.ShortID{owner1, owner2, owner3},
    2, // threshold
)

// List wallets
wallets, err := chainManager.ListWallets(ctx)
```

## Advanced Features

### Custom VM Development

```go
// Create custom VM
vm, err := sdk.CreateVM(ctx, &vm.CreateParams{
    Name:     "MyVM",
    Type:     vm.TypeWASM,
    Runtime:  wasmRuntime,
    Handlers: handlers,
})

// Register VM
err = sdk.RegisterVM(ctx, vm)
```

### High-Performance Features

- **Parallel Processing**: Leverage Go's concurrency for parallel operations
- **Batch Operations**: Execute multiple operations in a single transaction
- **Connection Pooling**: Efficient connection management for high throughput
- **Caching**: Built-in caching for frequently accessed data

### WASM Support

```go
// Deploy WASM contract
wasmID, err := sdk.DeployWASM(ctx, &WASMDeployParams{
    Code:     wasmBytes,
    InitArgs: initParams,
})

// Execute WASM function
result, err := sdk.ExecuteWASM(ctx, wasmID, "transfer", args)
```

## CLI Integration

The SDK wraps the Lux CLI for seamless command execution:

```go
// Execute any CLI command programmatically
output, err := sdk.ExecuteCommand(ctx, "network", "status")

// Use CLI commands with Go types
result, err := sdk.ExecuteCommand(ctx, "subnet", "create", 
    "--control-keys", strings.Join(keys, ","),
    "--threshold", "2",
)
```

## Complete Example

See [examples/complete/main.go](examples/complete/main.go) for a comprehensive example covering:

- Network creation and management
- Blockchain deployment (L1/L2/L3)
- Staking and delegation
- Asset creation and trading
- Smart contract deployment
- Cross-chain transfers

## Architecture

The SDK provides a unified interface to the entire Lux ecosystem:

```
┌─────────────────────────────────────────────────────┐
│                    Lux SDK                          │
│         Unified API with Smart Defaults             │
├─────────────────────────────────────────────────────┤
│              Integration Layer                      │
│  ┌─────────────┬─────────────┬─────────────┐      │
│  │    CLI      │  Netrunner  │    Node     │      │
│  │ Integration │ Integration │    APIs     │      │
│  └─────────────┴─────────────┴─────────────┘      │
├─────────────────────────────────────────────────────┤
│              Core SDK Components                    │
│  ┌─────────────┬─────────────┬─────────────┐      │
│  │   Network   │ Blockchain  │     VM      │      │
│  │   Manager   │   Builder   │   Manager   │      │
│  └─────────────┴─────────────┴─────────────┘      │
├─────────────────────────────────────────────────────┤
│              Chain Operations                       │
│  ┌───────┬───────┬───────┬───────┬───────┐        │
│  │   P   │   X   │   C   │   M   │   Q   │        │
│  │ Chain │ Chain │ Chain │ Chain │ Chain │        │
│  └───────┴───────┴───────┴───────┴───────┘        │
├─────────────────────────────────────────────────────┤
│         External Lux Components                     │
│  ┌─────────────┬─────────────┬─────────────┐      │
│  │   luxd/cli  │  netrunner  │ lux/node    │      │
│  │  (wrapper)  │  (network)  │   (APIs)    │      │
│  └─────────────┴─────────────┴─────────────┘      │
└─────────────────────────────────────────────────────┘
```

### Integration Strategy

The SDK intelligently selects the best available method for each operation:

1. **CLI First**: For user-friendly operations with good defaults
2. **Netrunner**: For complex network orchestration and testing
3. **Node APIs**: For high-performance direct node communication
4. **Built-in**: Fallback implementation when external tools unavailable

This ensures your code works in any environment - from development laptops to production clusters.

## 🏭 Production Deployment

### Mainnet Launch
```go
// Production deployment with monitoring
prod := sdk.NewProductionDeployment(luxSDK)

// Deploy mainnet with 21 validators
mainnet, err := prod.LaunchMainnet(ctx, &MainnetParams{
    Validators:      21,
    InitialStake:    big.NewInt(2000),
    ConsensusParams: mainnetConsensus,
})

// Set up production monitoring
prod.EnableMonitoring(ctx, &MonitoringParams{
    MetricsEndpoint: "prometheus:9090",
    LogAggregation:  "elasticsearch:9200",
    AlertManager:    "alertmanager:9093",
})

// Enable automatic backups
prod.EnableBackups(ctx, &BackupParams{
    Interval: 6 * time.Hour,
    Storage:  "s3://lux-backups",
    Retention: 30 * 24 * time.Hour,
})
```

### Migration from Other Platforms
```go
// Migrate from Ethereum/Polygon/BSC
migrator := sdk.NewMigrator(luxSDK)

// Analyze existing deployment
analysis := migrator.Analyze(ctx, &MigrationSource{
    Type:     "ethereum",
    Endpoint: "https://mainnet.infura.io/v3/YOUR_KEY",
    Contracts: []string{
        "0x...", // Your contract addresses
    },
})

// Generate migration plan
plan := migrator.Plan(ctx, analysis, &MigrationTarget{
    Network: "lux-mainnet",
    VMType:  "evm",
    OptimizeFor: []string{"lower-fees", "faster-finality"},
})

// Execute migration
result := migrator.Execute(ctx, plan)
log.Printf("Migration complete: %d contracts, %d users migrated", 
    result.ContractsMigrated, result.UsersMigrated)
```

## 🔬 Advanced Features

### Custom Consensus Parameters
```go
// Fine-tune consensus for your use case
consensus := &ConsensusParams{
    SnowballParameters: SnowballParameters{
        K:               21,  // Sample size
        AlphaPreference: 15,  // Quorum size
        AlphaConfidence: 19,  // Confidence threshold
        Beta:            8,   // Decision rounds
    },
    OptimalProcessing: 9630 * time.Millisecond,
}

network, err := luxSDK.LaunchNetworkWithConsensus(ctx, consensus)
```

### Hardware Security Module (HSM) Integration
```go
// Use HSM for validator keys
hsm := sdk.NewHSMIntegration(luxSDK, &HSMConfig{
    Provider: "thales",
    Endpoint: "hsm.internal:9999",
})

// Generate validator keys in HSM
validatorKey, err := hsm.GenerateValidatorKey(ctx, "validator-01")

// Sign transactions with HSM
signedTx, err := hsm.SignTransaction(ctx, tx, validatorKey)
```

### Performance Optimization
```go
// Configure for maximum throughput
perf := sdk.NewPerformanceOptimizer(luxSDK)

// Optimize for specific workload
perf.Optimize(ctx, &WorkloadProfile{
    TransactionType: "simple-transfer",
    ExpectedTPS:     50000,
    LatencyTarget:   100 * time.Millisecond,
})

// Enable performance monitoring
metrics := perf.StartMetrics(ctx)
go func() {
    for m := range metrics {
        log.Printf("TPS: %d, Latency: %v", m.TPS, m.Latency)
    }
}()
```

## Requirements

- Go 1.22 or higher
- Access to a Lux node endpoint (optional)
- For local development: Docker (optional, for netrunner)
- For production: Linux/macOS/Windows

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This SDK is licensed under the [BSD 3-Clause License](LICENSE).

## Support

- Documentation: https://docs.lux.network/sdk
- GitHub Issues: https://github.com/luxfi/sdk/issues
- Discord: https://discord.gg/lux

## 🗺️ Roadmap

### Near Term (Q1 2025)
- ✅ Full netrunner integration for network orchestration
- ✅ CLI wrapper for all operations
- ✅ Direct node API access
- [ ] Enhanced WASM support (Rust, C++, AssemblyScript)
- [ ] Advanced DeFi protocol templates
- [ ] GraphQL API support
- [ ] SDK plugins system

### Medium Term (Q2-Q3 2025)
- [ ] AI-powered network optimization
- [ ] Zero-knowledge proof integration
- [ ] Multi-cloud deployment automation
- [ ] Advanced cross-chain messaging protocol
- [ ] Built-in DEX and AMM templates
- [ ] Mobile SDK (iOS/Android)
- [ ] Browser SDK (WASM-based)

### Long Term (Q4 2025+)
- [ ] Quantum-resistant cryptography throughout
- [ ] Fully autonomous network management
- [ ] Inter-blockchain communication (IBC) support
- [ ] Native integration with major cloud providers
- [ ] Enterprise blockchain-as-a-service templates
- [ ] Advanced governance modules
- [ ] Regulatory compliance automation

## 💡 Why Choose Lux SDK?

1. **Unified Interface**: One SDK for all blockchain operations
2. **Production Ready**: Battle-tested components from mainnet
3. **Developer Friendly**: Intuitive APIs with excellent documentation
4. **Performance**: Optimized for high-throughput applications
5. **Flexibility**: Support for any VM type and consensus configuration
6. **Enterprise Grade**: Built for mission-critical deployments
7. **Future Proof**: Quantum-resistant and continuously evolving