# Alternative L2 Deployment Options

This directory contains configurations for deploying different types of L2s beyond standard Avalanche subnets.

## Deployment Types

### 1. Standard Lux L2 (Subnet)
- **Type**: Avalanche Subnet on Lux
- **Security**: Lux validator set
- **Finality**: ~2 seconds
- **Example**: ZOO, SPC, Hanzo

### 2. OP Stack Compatible L2
- **Type**: Optimistic Rollup using OP Stack
- **Security**: Fraud proofs + Lux for data availability
- **Finality**: 7 days (challenge period)
- **Benefits**: Ethereum tooling compatibility

### 3. Based Rollup
- **Type**: Based sequencing rollup
- **Security**: Ethereum L1 for sequencing
- **Finality**: Ethereum block time
- **Benefits**: Maximum decentralization

### 4. Avalanche L1 (Running alongside Lux)
- **Type**: Avalanche L1 running in parallel
- **Security**: Avalanche validators
- **Finality**: ~2 seconds
- **Benefits**: Access to both ecosystems
- **Consensus**: Avalanche Snowman (separate from Lux)

### 5. Sovereign L1 on Lux
- **Type**: Independent L1 blockchain
- **Security**: Own validator set
- **Finality**: Customizable
- **Benefits**: Full sovereignty

## Quick Start

### OP Stack L2 on Lux

```bash
# Deploy OP Stack contracts on Lux
cd op-stack/
make deploy-contracts network=lux

# Start OP Stack sequencer
make start-sequencer

# Configure bridge
make setup-bridge
```

### Based Rollup

```bash
# Deploy based rollup contracts
cd based-rollup/
npm run deploy:mainnet

# Start prover
npm run start:prover
```

### Avalanche Subnet

```bash
# Use Avalanche CLI instead of Lux CLI
cd /path/to/avalanche-cli
./avalanche subnet create mysubnet --evm
./avalanche subnet deploy mysubnet --fuji
```

## Comparison Matrix

| Feature | Lux L2 | OP Stack | Based Rollup | AVAX Subnet | L1 on Lux |
|---------|---------|----------|--------------|-------------|-----------|
| Finality | 2s | 7 days | 12s | 2s | Custom |
| Security | Lux validators | Fraud proofs | Ethereum | AVAX validators | Own validators |
| Decentralization | High | Medium | Very High | High | Variable |
| Cost | Low | Very Low | Medium | Low | Medium |
| Complexity | Low | High | Medium | Low | High |
| Ethereum Compat | Yes | Yes | Yes | Yes | Optional |

## Architecture Decisions

### When to Use Each Type

**Lux L2 (Subnet)**
- Need fast finality
- Want Lux ecosystem integration
- Simple deployment

**OP Stack**
- Need maximum Ethereum compatibility
- Can accept longer finality
- Want proven rollup technology

**Based Rollup**
- Maximum decentralization required
- No trusted sequencer acceptable
- Ethereum security model preferred

**Avalanche Subnet**
- Want Avalanche ecosystem access
- Need specific Avalanche features
- Multi-chain strategy

**Sovereign L1**
- Need complete control
- Custom consensus required
- Independent token economics