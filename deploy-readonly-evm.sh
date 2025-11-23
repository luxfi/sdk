#!/bin/bash
set -e

# Paths
NODE_BIN="$HOME/work/lux/node/build/luxd"
EVM_PLUGIN="$HOME/work/lux/evm/build/evmplugin"
DB_PATH="$HOME/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb"
CHAIN_ID="dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ"

# Verify prerequisites
echo "Checking prerequisites..."
[ -f "$NODE_BIN" ] || { echo "Node binary not found at $NODE_BIN"; exit 1; }
[ -f "$EVM_PLUGIN" ] || { echo "EVM plugin not found at $EVM_PLUGIN"; exit 1; }
[ -d "$DB_PATH" ] || { echo "Database not found at $DB_PATH"; exit 1; }

echo "✓ Node binary found"
echo "✓ EVM plugin found"
echo "✓ Database found ($(ls -1 $DB_PATH | wc -l | xargs) files, $(du -sh $DB_PATH | cut -f1))"

# Create plugin directory
mkdir -p ~/.luxd/plugins

# Copy EVM plugin
echo "Copying EVM plugin..."
cp "$EVM_PLUGIN" ~/.luxd/plugins/
chmod +x ~/.luxd/plugins/evmplugin
echo "✓ Plugin copied"

# Create chain config directory with blockchain ID
CHAIN_CONFIG_DIR="$HOME/.luxd/chains/$CHAIN_ID"
mkdir -p "$CHAIN_CONFIG_DIR"

# Copy chain config
echo "Setting up chain configuration..."
cp chain-config-readonly.json "$CHAIN_CONFIG_DIR/config.json"
echo "✓ Chain config created at $CHAIN_CONFIG_DIR/config.json"

# Start node with readonly EVM
echo ""
echo "====================================="
echo "Starting node with readonly EVM..."
echo "====================================="
echo "Database: $DB_PATH"
echo "Chain ID: $CHAIN_ID"
echo "RPC: http://localhost:9650/ext/bc/$CHAIN_ID/rpc"
echo ""

$NODE_BIN \
  --config-file=node-config-readonly.json \
  --plugin-dir=$HOME/.luxd/plugins
