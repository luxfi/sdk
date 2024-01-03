#!/usr/bin/env bash
set -e

# This script starts N nodes (TODO N instead of 5) and waits for ctrl-c to shutdown the process group of Luxd processes
# Uses data directory to store all Luxd data neatly in one location with minimal config overhead
if ! [[ "$0" =~ scripts/run.sh ]]; then
  echo "must be run from repository root, but got $0"
  exit 255
fi

# Load the versions
SUBNET_EVM_PATH=$(
  cd "$(dirname "${BASH_SOURCE[0]}")"
  cd .. && pwd
)
source "$SUBNET_EVM_PATH"/scripts/versions.sh

# Load the constants
source "$SUBNET_EVM_PATH"/scripts/constants.sh

# Set up lux binary path and assume build directory is set
<<<<<<< HEAD
LUXGO_BUILD_PATH=${LUXGO_BUILD_PATH:-"$GOPATH/src/github.com/luxdefi/node/build"}
LUXGO_PATH=${LUXGO_PATH:-"$LUXGO_BUILD_PATH/node"}
LUXGO_PLUGIN_DIR=${LUXGO_PLUGIN_DIR:-"$LUXGO_BUILD_PATH/plugins"}
DATA_DIR=${DATA_DIR:-/tmp/evm-start-node/$(date "+%Y-%m-%d%:%H:%M:%S")}
=======
LUXD_BUILD_PATH=${LUXD_BUILD_PATH:-"$GOPATH/src/github.com/luxdefi/node/build"}
LUXD_PATH=${LUXD_PATH:-"$LUXD_BUILD_PATH/node"}
LUXD_PLUGIN_DIR=${LUXD_PLUGIN_DIR:-"$LUXD_BUILD_PATH/plugins"}
DATA_DIR=${DATA_DIR:-/tmp/subnet-evm-start-node/$(date "+%Y-%m-%d%:%H:%M:%S")}
>>>>>>> b36c20f (Update executable to luxd)

mkdir -p $DATA_DIR

# Set the config file contents for the path passed in as the first argument
function _set_config(){
  cat <<EOF >$1
  {
    "network-id": "local",
    "sybil-protection-enabled": false,
    "health-check-frequency": "5s",
    "plugin-dir": "$LUXD_PLUGIN_DIR"
  }
EOF
}

function execute_cmd() {
  echo "Executing command: $@"
  $@
}

NODE_NAME="node1"
NODE_DATA_DIR="$DATA_DIR/$NODE_NAME"
echo "Creating data directory: $NODE_DATA_DIR"
mkdir -p $NODE_DATA_DIR
NODE_CONFIG_FILE_PATH="$NODE_DATA_DIR/config.json"
_set_config $NODE_CONFIG_FILE_PATH

CMD="$LUXD_PATH --data-dir=$NODE_DATA_DIR --config-file=$NODE_CONFIG_FILE_PATH"

execute_cmd $CMD
