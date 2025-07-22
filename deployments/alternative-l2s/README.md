# Alternative L2 Deployment Options

This directory contains configurations for deploying different types of L2s beyond standard Lux subnets.

## Deployment Types

### 1. Sovereign L1 on Lux
- **Type**: Independent L1 blockchain
- **Security**: Own validator set
- **Finality**: Customizable
- **Benefits**: Full sovereignty

### 2. Standard Lux L2 (Subnet)
- **Type**: Lux Subnet on Lux
- **Security**: Lux validator set
- **Finality**: ~2 seconds
- **Example**: ZOO, SPC, Hanzo

### 3. OP Stack Compatible L2
- **Type**: Optimistic Rollup using OP Stack
- **Security**: Fraud proofs + Lux for data availability
- **Finality**: 7 days (challenge period)
- **Benefits**: Ethereum tooling compatibility

### 4. Based Rollup
- **Type**: Based sequencing rollup
- **Security**: Ethereum L1 for sequencing
- **Finality**: Ethereum block time
- **Benefits**: Maximum decentralization

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

### Lux Subnet

```bash
cd /path/to/lux-cli
./lux subnet create mysubnet --evm
./lux subnet deploy mysubnet --testnet
```

## Comparison Matrix

| Feature | Lux L2 | OP Stack | Based Rollup | L1 on Lux |
|---------|---------|----------|--------------|-----------|
| Finality | 2s | 7 days | 12s | Custom |
| Security | Lux validators | Fraud proofs | Ethereum | Own validators |
| Decentralization | High | Medium | Very High | Variable |
| Cost | Low | Very Low | Medium | Medium |
| Complexity | Low | High | Medium | High |
| Ethereum Compat | Yes | Yes | Yes | Optional |

## Architecture Decisions

### When to Use Each Type

**Sovereign L1**
- Need complete control
- Custom consensus required
- Independent token economics

**Lux L2**
- Need fast finality
- Want Lux ecosystem integration
- Multi-chain strategy
- Simple deployment

**OP Stack**
- Need maximum Ethereum compatibility
- Can accept longer finality
- Want proven rollup technology

**Based Rollup**
- Maximum decentralization required
- No trusted sequencer acceptable
- Ethereum security model preferred
