# Running Avalanche L1s alongside Lux

Deploy Avalanche L1s (formerly Subnets) that can interoperate with Lux networks.

## Overview

Since Lux is built on the same consensus family as Avalanche, you can run Lux
blockchais in parallel with Avalanche Network.

## Architecture

```
Lux Network (Primary)
    ├── LUX C-Chain (96369)
    └── Custom L1s
         ├── DeFi L1
         └── Gaming L1

Avalanche Network (Parallel)
    ├── AVAX C-Chain
    └── Custom L1s
         ├── DeFi L1
         └── Gaming L1
```

## Setup

### 1. Run Lux Node

Add to `compose.lux.yml`:

```yaml
services:
  lux-node:
    image: luxfi/node:latest
    container_name: lux-node
    ports:
      - "9650:9650"  # Different port from Lux
      - "9651:9651"  # Staking port
    volumes:
      - ./volumes/lux:/data
    environment:
      - LUX_NETWORK_ID=testnet  # or mainnet
      - LUX_HTTP_HOST=0.0.0.0
      - LUX_HTTP_PORT=9650
      - LUX_STAKING_PORT=9651
      - LUX_PUBLIC_IP=127.0.0.1
      - LUX_DB_DIR=/data/db
      - LUX_LOG_DIR=/data/logs
    networks:
      - lux-network
    command: [
      "--network-id=testnet",
      "--http-host=0.0.0.0",
      "--http-port=9650",
      "--staking-port=9651",
      "--public-ip=127.0.0.1",
      "--db-dir=/data/db",
      "--log-dir=/data/logs"
    ]
```

### 2. Deploy Lux L1

Using Lux-CLI:

```bash
# Install Lux-CLI
curl -sSfL https://raw.githubusercontent.com/luxfi/cli/main/scripts/install.sh | sh -s

# Create L1
lux subnet create myL1 --evm

# Deploy to local network
lux subnet deploy myL1 --local

# Deploy to Testnet testnet
lux subnet deploy myL1 --testnet
```

### 3. Configure Cross-Chain Bridge

Set up Avalanche Warp Messaging (AWM) for Avalanche-Lux communication:

```solidity
// Deploy on both Avalanche and Lux
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
make up  # Lux Network
docker-compose -f docker-compose.lux.yml up -d  # Lux

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
// Send from Avalanche to Lux L1
const bridge = new ethers.Contract(BRIDGE_ADDRESS, BRIDGE_ABI, luxSigner);
await bridge.sendMessage(
    LUX_L1_CHAIN_ID,
    ethers.utils.defaultAbiCoder.encode(
        ["address", "uint256"],
        [recipient, amount]
    )
);
```

## Benefits

1. **Ecosystem Access**: Tap into both Lux and Lux ecosystems
2. **Flexibility**: Choose consensus and validator sets per L1
3. **Interoperability**: Native cross-chain messaging
4. **Shared Security**: Leverage both networks' security

## Considerations

- Port conflicts (use different ports)
- Separate validator management
- Bridge security and monitoring
- Gas costs on both networks
