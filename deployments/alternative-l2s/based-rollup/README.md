# Based Rollup on Ethereum

Deploy a based rollup that uses Ethereum L1 for sequencing while settling on Lux.

## Overview

Based rollups achieve maximum decentralization by:
- Using Ethereum L1 for transaction sequencing
- No centralized sequencer
- Settling state on Lux for cheaper execution
- MEV captured by Ethereum validators

## Architecture

```
Ethereum L1 (Sequencing) → Based Rollup → Lux C-Chain (Settlement)
```

## Components

1. **Ethereum Contracts**
   - `Inbox`: Accepts L2 transactions
   - `Outbox`: Processes withdrawals

2. **Lux Contracts**
   - `StateCommitment`: Stores L2 state roots
   - `FraudVerifier`: Handles fraud proofs
   - `Bridge`: Asset bridging

3. **Services**
   - `Prover`: Generates execution proofs
   - `Relayer`: Syncs between chains

## Deployment

### Deploy on Ethereum

```bash
cd contracts/ethereum
export ETH_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/<key>

# Deploy inbox
forge create --rpc-url $ETH_RPC_URL \
  --private-key $PRIVATE_KEY \
  src/Inbox.sol:Inbox \
  --constructor-args $LUX_COMMITMENT_CONTRACT
```

### Deploy on Lux

```bash
cd contracts/lux
export LUX_RPC_URL=https://api.lux.network/rpc

# Deploy state commitment
forge create --rpc-url $LUX_RPC_URL \
  --private-key $PRIVATE_KEY \
  src/StateCommitment.sol:StateCommitment \
  --constructor-args $ETH_INBOX_ADDRESS
```

### Start Services

```bash
# Start prover
npm run prover:start -- \
  --eth-rpc $ETH_RPC_URL \
  --lux-rpc $LUX_RPC_URL \
  --inbox $ETH_INBOX_ADDRESS \
  --commitment $LUX_COMMITMENT_ADDRESS

# Start relayer
npm run relayer:start -- \
  --source ethereum \
  --destination lux
```

## Usage

### Submit Transaction

```javascript
// Send L2 transaction via Ethereum
const inbox = new ethers.Contract(INBOX_ADDRESS, INBOX_ABI, ethSigner);

const l2Tx = {
  to: "0x...",
  value: 0,
  data: "0x...",
  gasLimit: 1000000
};

await inbox.submitTransaction(l2Tx, { value: estimatedCost });
```

### Read L2 State

```javascript
// Query state from Lux
const commitment = new ethers.Contract(
  COMMITMENT_ADDRESS, 
  COMMITMENT_ABI, 
  luxProvider
);

const stateRoot = await commitment.stateRoots(blockNumber);
```

## Benefits

1. **Maximum Decentralization**
   - No sequencer monopoly
   - Ethereum validators order transactions
   - Censorship resistance from Ethereum

2. **Cost Efficiency**
   - Settlement on Lux is cheaper
   - Users pay Ethereum gas + small Lux fee
   - Batch proofs amortize costs

3. **Security**
   - Ethereum ordering security
   - Lux economic security
   - Fraud proof protection

## Trade-offs

- Higher latency (Ethereum block time)
- More complex architecture
- Requires monitoring two chains