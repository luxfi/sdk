# Lux L2 (Subnet) Deployment Examples

This directory contains deployment configurations and examples for deploying L2s (subnets) on the Lux network.

## Overview

Each L2 runs as a subnet on the Lux primary network, providing:
- EVM compatibility
- Custom token economics
- Independent validator sets
- Cross-chain communication via Teleporter

## Network Configurations

### ZOO L2
- **Chain ID**: 200200
- **Token**: ZOO
- **Purpose**: Gaming and NFT ecosystem
- **Genesis**: `zoo/genesis.json`

### SPC L2
- **Chain ID**: 36911
- **Token**: SPC
- **Purpose**: DeFi and trading
- **Genesis**: `spc/genesis.json`

### Hanzo L2
- **Chain ID**: 36963
- **Token**: AI
- **Purpose**: AI services and compute
- **Genesis**: `hanzo/genesis.json`

## Deployment Steps

### 1. Local Development

```bash
# Start local Lux network
cd /home/z/work/lux/stack
make network-up

# Deploy L2
make deploy-zoo     # Deploy ZOO L2
make deploy-spc     # Deploy SPC L2
make deploy-hanzo   # Deploy Hanzo L2

# Or deploy all at once
make deploy-all-l2s
```

### 2. Testnet Deployment

```bash
# Create subnet configuration
cd /home/z/work/lux/cli
./lux subnet create zoo --evm --genesis ../evm/deployments/zoo/genesis.json

# Deploy to testnet
./lux subnet deploy zoo --testnet --key mykey

# Get subnet info
./lux subnet describe zoo
```

### 3. Mainnet Deployment

```bash
# Deploy to mainnet (requires AVAX for subnet creation)
./lux subnet deploy zoo --mainnet --ledger

# Add validators
./lux subnet join zoo --mainnet
```

## Configuration Files

Each L2 has its own directory with:
- `genesis.json` - Genesis configuration
- `subnet.json` - Subnet configuration
- `vm-config.json` - VM-specific settings
- `README.md` - Network-specific documentation

## Cross-Chain Communication

L2s can communicate with each other and the C-Chain using:
- Teleporter protocol for messages
- Bridge contracts for token transfers
- Shared security from Lux validators

## Monitoring

Track your L2s:
```bash
# Check subnet status
make l2-info name=zoo

# List all L2s
make l2-info

# Monitor logs
docker logs lux-node -f
```