# EVM Readonly Deployment Guide

## Purpose
Deploy an EVM instance with read-only access to legacy PebbleDB for data export.

## Prerequisites
- Lux node built from `~/work/lux/node`
- EVM plugin built from `~/work/lux/evm` with database readonly support
- Legacy PebbleDB at known path (e.g. `~/.luxd-5node-rpc/node2/chains/.../db`)

## Configuration

### 1. Chain Config JSON
Create `chain-config-readonly.json`:

```json
{
  "chain-id": 96369,
  "database-type": "pebbledb",
  "database-read-only": true,
  "database-path": "/Users/z/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb",
  "log-level": "info",
  "warp-api-enabled": true,
  "local-txs-enabled": true,
  "api-enabled": true
}
```

### 2. Node Config
Create `node-config-readonly.json`:

```json
{
  "network-id": "local",
  "http-host": "0.0.0.0",
  "http-port": 9650,
  "api-admin-enabled": true,
  "api-keystore-enabled": false,
  "api-metrics-enabled": true,
  "log-level": "info",
  "log-display-level": "info"
}
```

### 3. Deploy Script
Create `deploy-readonly-evm.sh`:

```bash
#!/bin/bash
set -e

# Paths
NODE_BIN="$HOME/work/lux/node/build/luxd"
EVM_PLUGIN="$HOME/work/lux/evm/build/evmplugin"
DB_PATH="$HOME/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb"
CHAIN_ID="dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ"

# Verify prerequisites
echo "Checking prerequisites..."
[ -f "$NODE_BIN" ] || { echo "Node binary not found"; exit 1; }
[ -f "$EVM_PLUGIN" ] || { echo "EVM plugin not found"; exit 1; }
[ -d "$DB_PATH" ] || { echo "Database not found at $DB_PATH"; exit 1; }

# Create plugin directory
mkdir -p ~/.luxd/plugins

# Copy EVM plugin
cp "$EVM_PLUGIN" ~/.luxd/plugins/

# Start node with readonly EVM
echo "Starting node with readonly EVM..."
$NODE_BIN \
  --config-file=node-config-readonly.json \
  --chain-config-dir=. \
  --plugin-dir=$HOME/.luxd/plugins
```

## Usage

```bash
# 1. Build prerequisites
cd ~/work/lux/node && go build -o build/luxd ./cmd/luxd
cd ~/work/lux/evm && go build -o build/evmplugin ./plugin/evm

# 2. Deploy readonly EVM
chmod +x deploy-readonly-evm.sh
./deploy-readonly-evm.sh

# 3. Verify RPC accessible
curl -X POST -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:9650/ext/bc/$CHAIN_ID/rpc

# Expected: {"jsonrpc":"2.0","id":1,"result":"0x108abc"} (~1,082,780 blocks)

# 4. Export data
cd ~/work/lux/cli
./bin/lux export \
  --rpc http://localhost:9650/ext/bc/$CHAIN_ID/rpc \
  --start 0 \
  --end 1082780 \
  --output evm-full-export.json \
  --parallel 10
```

## Key Points

- **Database readonly mode**: Prevents any writes to legacy DB
- **No DB copying**: Points directly to existing PebbleDB
- **Same network**: Can run alongside C-Chain for migration
- **Export ready**: RPC endpoint ready for `lux export` command

## Troubleshooting

### EVM won't start
- Check plugin path: `ls ~/.luxd/plugins/evmplugin`
- Check DB permissions: DB must be readable
- Check logs: `tail -f ~/.luxd/logs/C.log`

### Height returns 0x0
- DB path incorrect
- DB corrupted or wrong format
- Readonly flag not working (database package needs update)

### Export fails
- Increase --parallel if slow
- Check disk space for export file (can be large)
- Use --skip-existing to resume interrupted exports
