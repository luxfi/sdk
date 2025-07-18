#!/usr/bin/env bash

set -euo pipefail

# Requires nix to be installed. The determinate systems installer is recommended:
#
#   https://github.com/DeterminateSystems/nix-installer?tab=readme-ov-file#install-nix
#

# Load LUX_VERSION
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=/scripts/constants.sh
source "$SCRIPT_DIR"/constants.sh

# Start a dev shell with the luxd flake
FLAKE="github:luxfi/node?ref=${LUX_VERSION}"
echo "Starting nix shell for ${FLAKE}"
nix develop "${FLAKE}" "${@}"
