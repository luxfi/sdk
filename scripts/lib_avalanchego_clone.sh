#!/usr/bin/env bash

set -euo pipefail

# Defines functions for interacting with git clones of the luxd repo.

if [[ -z "${SUBNET_EVM_PATH}" ]]; then
  echo "SUBNET_EVM_PATH must be set"
  exit 1
fi

export LUXD_CLONE_PATH=${LUXD_CLONE_PATH:-${SUBNET_EVM_PATH}/luxd}

# Clones the luxd repo to the configured path and checks out the specified version.
function clone_luxd {
  local lux_version="$1"

  echo "checking out target luxd version ${lux_version} to ${LUXD_CLONE_PATH}"
  if [[ -d "${LUXD_CLONE_PATH}" ]]; then
    echo "updating existing clone"
    cd "${LUXD_CLONE_PATH}"
    git fetch
  else
    echo "creating new clone"
    git clone https://github.com/luxfi/node.git "${LUXD_CLONE_PATH}"
    cd "${LUXD_CLONE_PATH}"
  fi
  # Branch will be reset to $lux_version if it already exists
  git checkout -B "test-${lux_version}" "${lux_version}"
  cd "${SUBNET_EVM_PATH}"
}

# Derives an image tag from the current state of the luxd clone
function luxd_image_tag_from_clone {
  local commit_hash
  commit_hash="$(git --git-dir="${LUXD_CLONE_PATH}/.git" rev-parse HEAD)"
  echo "${commit_hash::8}"
}
