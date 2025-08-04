# Lux Mainnet Boot Example

This example demonstrates how to boot a Lux mainnet using the Lux SDK.

## Features

- Creates a mainnet network configuration with 21 validators
- Deploys the core Lux chains:
  - P-Chain (Platform Chain) - For staking and validation
  - C-Chain (Contract Chain) - EVM-compatible smart contracts
  - X-Chain (Exchange Chain) - Asset management and trading

## Prerequisites

- Go 1.22 or higher
- Lux SDK (parent directory)

## Building

```bash
go build -o boot-mainnet main.go
```

## Running

```bash
./boot-mainnet
```

## Output

The application will:
1. Create a mainnet network with 21 validator nodes
2. Deploy P-Chain, C-Chain, and X-Chain
3. Display network and chain information
4. Keep running until interrupted (Ctrl+C)

## Example Output

```
Booting Lux Mainnet...
Network ID: 96369
API Endpoint: https://api.mainnet.lux.network

Mainnet created:
- ID: network-1754286981765854000-0
- Name: lux-mainnet
- Type: mainnet
- Status: running
- Nodes: 21

Validator Nodes:
1. NodeID-0 - healthy (Stake: 2000 LUX)
...
21. NodeID-20 - healthy (Stake: 2000 LUX)

=== Lux Mainnet Boot Summary ===
Network: lux-mainnet (ID: network-1754286981765854000-0)
Status: running
Validators: 21
Chains deployed: 3
- P-Chain: ae0213a9b00f3aa151ff0a587f146755d78574ecfc3b7111383744fed14f9040
- C-Chain: 6578589968a1f0c6f7063611d490ee0a05bec69c8dfa427704ab45f819c617e7
- X-Chain: e0792cf0941a8bf2e5640a86ff488d1fa1c86fca2677ad7726dc60dde585b3cc

Mainnet is running. Press Ctrl+C to shutdown...
```

## Note

This is a mock implementation for demonstration purposes. In a real deployment, the SDK would interface with actual Lux node software through the netrunner client.