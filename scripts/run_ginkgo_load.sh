#!/usr/bin/env bash
set -e

# This script assumes that an Lux and EVM binaries are available in the standard location
# within the $GOPATH
# The Lux and PluginDir paths can be specified via the environment variables used in ./scripts/run.sh.

SUBNET_EVM_PATH=$(
  cd "$(dirname "${BASH_SOURCE[0]}")"
  cd .. && pwd
)

source "$SUBNET_EVM_PATH"/scripts/constants.sh

EXTRA_ARGS=()
LUXD_BUILD_PATH="${LUXD_BUILD_PATH:-}"
if [[ -n "${LUXD_BUILD_PATH}" ]]; then
  EXTRA_ARGS=("--luxd-path=${LUXD_BUILD_PATH}/luxd")
  echo "Running with extra args:" "${EXTRA_ARGS[@]}"
fi

"${SUBNET_EVM_PATH}"/bin/ginkgo -vv --label-filter="${GINKGO_LABEL_FILTER:-}" ./tests/load -- "${EXTRA_ARGS[@]}"
