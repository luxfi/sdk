# Lux SDK

The official Go SDK for building and managing Lux-compatible networks and blockchains. This SDK provides a comprehensive, easy-to-use interface for all Lux blockchain operations.

## Features

- 🚀 **Network Management**: Create and manage Lux networks using netrunner
- 🔗 **Blockchain Building**: Build L1/L2/L3 chains with custom VMs
- 💰 **Staking & Validation**: Programmatic staking, delegation, and validation
- 🪙 **Asset Management**: Create and manage assets on X-Chain
- 📜 **Smart Contracts**: Deploy and interact with smart contracts on C-Chain
- 🌉 **Cross-Chain Operations**: Seamless asset transfers between chains
- 👛 **Wallet Integration**: Built-in wallet management via M-Chain
- 🛡️ **Quantum-Resistant**: Q-Chain integration for post-quantum security
- 🌐 **WASM Support**: Multi-language support via WebAssembly

## Installation

```bash
go get github.com/luxfi/sdk
```

## Quick Start

```go
import (
    "github.com/luxfi/sdk"
)

// Initialize the SDK
luxSDK, err := sdk.New(
    sdk.WithLogLevel("info"),
    sdk.WithDataDir("~/.lux"),
)
if err != nil {
    log.Fatal(err)
}
defer luxSDK.Close()

// Create a network
network, err := luxSDK.CreateNetwork(ctx, &network.NetworkParams{
    Name:     "my-network",
    Type:     network.NetworkTypeLocal,
    NumNodes: 5,
})

// Create an L1 blockchain
blockchain, err := luxSDK.CreateL1(ctx, "my-chain", &blockchain.L1Params{
    VMType: blockchain.VMTypeEVM,
})
```

## Core Components

### Network Management (via netrunner)

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

```
┌─────────────────────────────────────────────────────┐
│                    Lux SDK                          │
├─────────────────────────────────────────────────────┤
│                  CLI Wrapper                        │
├─────────────────────────────────────────────────────┤
│   Network Manager  │  Blockchain Builder  │  VM Mgr │
├─────────────────────────────────────────────────────┤
│              Chain Manager                          │
│  ┌───────┬───────┬───────┬───────┬───────┐        │
│  │   P   │   X   │   C   │   M   │   Q   │        │
│  │ Chain │ Chain │ Chain │ Chain │ Chain │        │
│  └───────┴───────┴───────┴───────┴───────┘        │
├─────────────────────────────────────────────────────┤
│            Core Lux Components                      │
│  netrunner, node, consensus, database, etc.        │
└─────────────────────────────────────────────────────┘
```

## Requirements

- Go 1.22 or higher
- Access to a Lux node endpoint
- For local development: Docker (for netrunner)

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This SDK is licensed under the [BSD 3-Clause License](LICENSE).

## Support

- Documentation: https://docs.lux.network/sdk
- GitHub Issues: https://github.com/luxfi/sdk/issues
- Discord: https://discord.gg/lux

## Roadmap

- [ ] Enhanced WASM support with more runtimes
- [ ] Advanced DeFi protocol integrations
- [ ] Improved cross-chain messaging
- [ ] Hardware wallet support
- [ ] Mobile SDK variants
- [ ] GraphQL API support