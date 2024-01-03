#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Lux root directory
SUBNET_EVM_PATH=$(
  cd "$(dirname "${BASH_SOURCE[0]}")"
  cd .. && pwd
)

# Load the versions
source "$SUBNET_EVM_PATH"/scripts/versions.sh

# Load the constants
source "$SUBNET_EVM_PATH"/scripts/constants.sh

BUILD_IMAGE_ID=${BUILD_IMAGE_ID:-"${LUX_VERSION}-EVM-${CURRENT_BRANCH}"}

echo "Building Docker Image: $DOCKERHUB_REPO:$BUILD_IMAGE_ID based of $LUX_VERSION"
docker build -t "$DOCKERHUB_REPO:$BUILD_IMAGE_ID" "$SUBNET_EVM_PATH" -f "$SUBNET_EVM_PATH/Dockerfile" \
  --build-arg LUX_VERSION="$LUX_VERSION" \
  --build-arg SUBNET_EVM_COMMIT="$SUBNET_EVM_COMMIT" \
  --build-arg CURRENT_BRANCH="$CURRENT_BRANCH"

if [[ ${PUSH_DOCKER_IMAGE:-""} == "true" ]]; then
  docker push $DOCKERHUB_REPO:$BUILD_IMAGE_ID
fi
