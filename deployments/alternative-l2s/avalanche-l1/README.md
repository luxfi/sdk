# Running Avalanche L1s alongside Lux

Deploy Avalanche L1s (formerly Subnets) that can interoperate with Lux networks.

## Overview

Since Lux is built on the same Snow consensus family as Avalanche, you can run Avalanche L1s in parallel with Lux networks. From Lux's perspective, these Avalanche L1s function as L2s.

## Architecture

```
Lux Network (Primary)
    ├── LUX C-Chain (96369)
    ├── ZOO L2 (200200)
    ├── SPC L2 (36911)
    └── Hanzo L2 (36963)

Avalanche Network (Parallel)
    ├── AVAX C-Chain
    └── Custom L1s
         ├── DeFi L1
         └── Gaming L1
```

## Setup

### 1. Run Avalanche Node

Add to `docker-compose.avalanche.yml`:

```yaml
version: '3.8'

services:
  avalanche-node:
    image: avaplatform/avalanchego:latest
    container_name: avalanche-node
    ports:
      - "9660:9650"  # Different port from Lux
      - "9661:9651"  # Staking port
    volumes:
      - ./volumes/avalanche:/data
    environment:
      - AVAX_NETWORK_ID=fuji  # or mainnet
      - AVAX_HTTP_HOST=0.0.0.0
      - AVAX_HTTP_PORT=9650
      - AVAX_STAKING_PORT=9651
      - AVAX_PUBLIC_IP=127.0.0.1
      - AVAX_DB_DIR=/data/db
      - AVAX_LOG_DIR=/data/logs
    networks:
      - lux-network
    command: [
      "--network-id=fuji",
      "--http-host=0.0.0.0",
      "--http-port=9650",
      "--staking-port=9651",
      "--public-ip=127.0.0.1",
      "--db-dir=/data/db",
      "--log-dir=/data/logs"
    ]
```

### 2. Deploy Avalanche L1

Using Avalanche-CLI:

```bash
# Install Avalanche-CLI
curl -sSfL https://raw.githubusercontent.com/ava-labs/avalanche-cli/main/scripts/install.sh | sh -s

# Create L1
avalanche subnet create myL1 --evm

# Deploy to local network
avalanche subnet deploy myL1 --local

# Deploy to Fuji testnet
avalanche subnet deploy myL1 --fuji
```

### 3. Configure Cross-Chain Bridge

Set up Avalanche Warp Messaging (AWM) for Lux-Avalanche communication:

```solidity
// Deploy on both Lux and Avalanche
contract CrossChainBridge {
    ITeleporterMessenger public teleporter;
    
    mapping(uint256 => address) public remoteContracts;
    
    function sendMessage(
        uint256 destinationChainId,
        bytes calldata message
    ) external {
        teleporter.sendCrossChainMessage(
            TeleporterMessageInput({
                destinationChainId: destinationChainId,
                destinationAddress: remoteContracts[destinationChainId],
                feeInfo: TeleporterFeeInfo({
                    contractAddress: address(0),
                    amount: 0
                }),
                requiredGasLimit: 100000,
                allowedRelayerAddresses: new address[](0),
                message: message
            })
        );
    }
}
```

### 4. Start Services

```bash
# Start both networks
make up  # Lux network
docker-compose -f docker-compose.avalanche.yml up -d  # Avalanche

# Verify both are running
curl -X POST --data '{"jsonrpc":"2.0","id":1,"method":"info.getNetworkID","params":[]}' \
  -H 'content-type:application/json' http://localhost:9650/ext/info
  
curl -X POST --data '{"jsonrpc":"2.0","id":1,"method":"info.getNetworkID","params":[]}' \
  -H 'content-type:application/json' http://localhost:9660/ext/info
```

## Interoperability

### Message Passing

Use Teleporter/AWM for cross-chain messages:

```javascript
// Send from Lux to Avalanche L1
const bridge = new ethers.Contract(BRIDGE_ADDRESS, BRIDGE_ABI, luxSigner);
await bridge.sendMessage(
    AVALANCHE_L1_CHAIN_ID,
    ethers.utils.defaultAbiCoder.encode(
        ["address", "uint256"],
        [recipient, amount]
    )
);
```

### Asset Bridging

Bridge assets between networks:

```javascript
// Lock tokens on Lux, mint on Avalanche L1
await luxBridge.lockTokens(tokenAddress, amount, avalancheL1ChainId);

// Burn on Avalanche L1, unlock on Lux
await avalancheBridge.burnTokens(tokenAddress, amount, luxChainId);
```

## Benefits

1. **Ecosystem Access**: Tap into both Lux and Avalanche ecosystems
2. **Flexibility**: Choose consensus and validator sets per L1
3. **Interoperability**: Native cross-chain messaging
4. **Shared Security**: Leverage both networks' security

## Considerations

- Port conflicts (use different ports)
- Separate validator management
- Bridge security and monitoring
- Gas costs on both networks