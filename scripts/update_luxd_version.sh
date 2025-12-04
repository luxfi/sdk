#!/usr/bin/env bash

set -euo pipefail

if ! [[ "$0" =~ scripts/update_luxd_version.sh ]]; then
  echo "must be run from repository root, but got $0"
  exit 255
fi

EVM_PATH=$(
  cd "$(dirname "${BASH_SOURCE[0]}")"
  cd .. && pwd
)

# If version is not provided, the existing version in go.mod is assumed
VERSION="${1:-}"

if [[ -n "${VERSION}" ]]; then
  echo "Ensuring Node version $VERSION in go.mod"
  go get "github.com/luxfi/node@${VERSION}"
  go mod tidy
fi

# Discover LUX_VERSION
. "$EVM_PATH"/scripts/constants.sh

# The full SHA is required for versioning custom actions.
CURL_ARGS=(curl -s)
if [[ -n "${GITHUB_TOKEN:-}" ]]; then
  # Using an auth token avoids being rate limited when run in CI
  CURL_ARGS+=(-H "Authorization: token ${GITHUB_TOKEN}")
else
  echo "No GITHUB_TOKEN found, using unauthenticated requests"
fi

GIT_COMMIT=$("${CURL_ARGS[@]}" "https://api.github.com/repos/luxfi/node/commits/${LUX_VERSION}")
FULL_LUX_VERSION="$(grep -m1 '"sha":' <<< "${GIT_COMMIT}" | cut -d'"' -f4)"

# Ensure the custom action version matches the lux version
WORKFLOW_PATH=".github/workflows/tests.yml"
CUSTOM_ACTION="luxfi/node/.github/actions/run-monitored-tmpnet-cmd"
echo "Ensuring Node version ${FULL_LUX_VERSION} for ${CUSTOM_ACTION} custom action in ${WORKFLOW_PATH} "
sed -i.bak "s|\(uses: ${CUSTOM_ACTION}\)@.*|\1@${FULL_LUX_VERSION}|g" "${WORKFLOW_PATH}" && rm -f "${WORKFLOW_PATH}.bak"
