# OP Stack L2 on Lux

Deploy an Optimism-compatible L2 using Lux for data availability.

## Overview

This configuration deploys an OP Stack L2 that:
- Uses Lux C-Chain for data availability (cheaper than Ethereum)
- Maintains full OP Stack compatibility
- Supports all Optimism tooling
- Enables fast withdrawals via Lux

## Components

1. **L1 Contracts** (deployed on Lux C-Chain)
   - `SystemConfig`
   - `L1CrossDomainMessenger`
   - `L1StandardBridge`
   - `OptimismPortal`

2. **L2 Components**
   - `op-node`: Rollup node
   - `op-geth`: Execution client
   - `op-batcher`: Transaction batcher
   - `op-proposer`: State root proposer

## Deployment

### Prerequisites

```bash
# Clone OP Stack
git clone https://github.com/ethereum-optimism/optimism.git
cd optimism

# Install dependencies
pnpm install
make install-geth
```

### Deploy L1 Contracts on Lux

```bash
# Set Lux RPC
export L1_RPC_URL=https://api.lux.network/rpc
export PRIVATE_KEY=<your-deployment-key>

# Deploy contracts
cd packages/contracts-bedrock
forge script scripts/Deploy.s.sol:Deploy \
  --rpc-url $L1_RPC_URL \
  --private-key $PRIVATE_KEY \
  --broadcast \
  --verify
```

### Configure L2

Create `config.json`:
```json
{
  "l1_chain_id": 96369,
  "l2_chain_id": 42069,
  "l1_rpc": "https://api.lux.network/rpc",
  "l2_genesis": {
    "baseFeePerGas": "0x3b9aca00",
    "gasLimit": "0x1c9c380"
  }
}
```

### Start L2 Services

```bash
# Start op-geth
./op-geth \
  --datadir ./datadir \
  --http \
  --http.port 8545 \
  --http.addr 0.0.0.0 \
  --authrpc.vhosts="*" \
  --authrpc.addr 0.0.0.0 \
  --authrpc.port 8551

# Start op-node
./op-node \
  --l1=https://api.lux.network/rpc \
  --l2=http://localhost:8551 \
  --network=custom \
  --rpc.addr=0.0.0.0 \
  --rpc.port=8547
```

## Bridge Assets

```javascript
// Bridge LUX to OP L2
const bridge = new ethers.Contract(L1_BRIDGE_ADDRESS, L1_BRIDGE_ABI, signer);
await bridge.depositETH(0, "0x", { value: ethers.parseEther("1.0") });
```

## Monitoring

- L2 RPC: `http://localhost:8545`
- Rollup Node: `http://localhost:8547`
- Metrics: `http://localhost:7300/metrics`

## Cost Analysis

Using Lux instead of Ethereum for data availability:
- ~95% reduction in data costs
- 2s L1 finality (vs 12s on Ethereum)
- Same security model with fraud proofs