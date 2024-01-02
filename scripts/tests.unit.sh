#!/usr/bin/env bash
# Copyright (C) 2023-2024, Lux Partners Limited. All rights reserved.
# See the file LICENSE for licensing terms.

set -e

if ! [[ "$0" =~ scripts/tests.unit.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

VMSDK_PATH=$(
  cd "$(dirname "${BASH_SOURCE[0]}")"
  cd .. && pwd
)
source "$VMSDK_PATH"/scripts/constants.sh

go test -race -timeout="3m" -coverprofile="coverage.out" -covermode="atomic" $(go list ./... | grep -v tests)
