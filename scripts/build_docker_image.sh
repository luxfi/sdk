#!/usr/bin/env bash

set -euo pipefail

# If set to non-empty, prompts the building of a multi-arch image when the image
# name indicates use of a registry.
#
# A registry is required to build a multi-arch image since a multi-arch image is
# not really an image at all. A multi-arch image (also called a manifest) is
# basically a list of arch-specific images available from the same registry that
# hosts the manifest. Manifests are not supported for local images.
#
# Reference: https://docs.docker.com/build/building/multi-platform/
PLATFORMS="${PLATFORMS:-}"

# If set to non-empty, the image will be published to the registry.
PUBLISH="${PUBLISH:-}"

# Directory above this script
EVM_PATH=$(
  cd "$(dirname "${BASH_SOURCE[0]}")"
  cd .. && pwd
)

# Load the constants
source "$EVM_PATH"/scripts/constants.sh

# ALLOW_TAG_LATEST is used to tag the image as 'latest' if set to true.
# It only works if the image is built from the master branch. This is to avoid
# tagging images from a manual triggered build as 'latest' with older luxd versions.
ALLOW_TAG_LATEST="${ALLOW_TAG_LATEST:-}"

# buildx (BuildKit) improves the speed and UI of builds over the legacy builder and
# simplifies creation of multi-arch images.
#
# Reference: https://docs.docker.com/build/buildkit/
DOCKER_CMD="docker buildx build"
ispush=0
if [[ -n "${PUBLISH}" ]]; then
  echo "Pushing $IMAGE_NAME:$BUILD_IMAGE_ID"
  ispush=1
  # A populated DOCKER_USERNAME env var triggers login
  if [[ -n "${DOCKER_USERNAME:-}" ]]; then
    echo "$DOCKER_PASS" | docker login --username "$DOCKER_USERNAME" --password-stdin
  fi
fi

# Build a specified platform image if requested
if [[ -n "${PLATFORMS}" ]]; then
  DOCKER_CMD="${DOCKER_CMD} --platform=${PLATFORMS}"
  if [[ "$PLATFORMS" == *,* ]]; then ## Multi-arch
    if [[ "${IMAGE_NAME}" != *"/"* ]]; then
      echo "ERROR: Multi-arch images (multi-platform) must be pushed to a registry."
      exit 1
    fi
    ispush=1
  fi
fi

if [[ $ispush -eq 1 ]]; then
  DOCKER_CMD="${DOCKER_CMD} --push"
else
  ## Single arch
  #
  # Building a single-arch image with buildx and having the resulting image show up
  # in the local store of docker images (ala 'docker build') requires explicitly
  # loading it from the buildx store with '--load'.
  DOCKER_CMD="${DOCKER_CMD} --load"
fi

VM_ID=${VM_ID:-"${DEFAULT_VM_ID}"}

# Default to the release image. Will need to be overridden when testing against unreleased versions.
LUXD_NODE_IMAGE="${LUXD_NODE_IMAGE:-${LUXD_IMAGE_NAME}:${LUX_VERSION}}"

# Build the luxd image if it cannot be pulled. This will usually be due to
# LUX_VERSION being not yet merged since the image is published post-merge.
if ! docker pull "${LUXD_NODE_IMAGE}"; then
  # Build a multi-arch luxd image if the evm image build is multi-arch
  BUILD_MULTI_ARCH="$([[ "$PLATFORMS" =~ , ]] && echo 1 || echo "")"

  # - Use a image name without a repository (i.e. without 'avaplatform/' prefix ) to build a
  #   local single-arch image that will not be pushed.
  # - Use a image name with a repository to build a multi-arch image that will be pushed.
  LUXD_LOCAL_IMAGE_NAME="${LUXD_LOCAL_IMAGE_NAME:-luxd}"

  if [[ -n "${BUILD_MULTI_ARCH}" && "${LUXD_LOCAL_IMAGE_NAME}" != *"/"* ]]; then
    echo "ERROR: Multi-arch images must be pushed to a registry."
    exit 1
  fi

  LUXD_NODE_IMAGE="${LUXD_LOCAL_IMAGE_NAME}:${LUX_VERSION}"
  echo "Building ${LUXD_NODE_IMAGE} locally"

  source "${EVM_PATH}"/scripts/lib_luxd_clone.sh
  clone_luxd "${LUX_VERSION}"
  SKIP_BUILD_RACE=1 \
    DOCKER_IMAGE="${LUXD_LOCAL_IMAGE_NAME}" \
    BUILD_MULTI_ARCH="${BUILD_MULTI_ARCH}" \
    "${LUXD_CLONE_PATH}"/scripts/build_image.sh
fi

echo "Building Docker Image: $IMAGE_NAME:$BUILD_IMAGE_ID based of Luxd@$LUX_VERSION"
${DOCKER_CMD} -t "$IMAGE_NAME:$BUILD_IMAGE_ID" -t "$IMAGE_NAME:${DOCKERHUB_TAG}" \
  "$EVM_PATH" -f "$EVM_PATH/Dockerfile" \
  --build-arg LUXD_NODE_IMAGE="$LUXD_NODE_IMAGE" \
  --build-arg EVM_COMMIT="$EVM_COMMIT" \
  --build-arg CURRENT_BRANCH="$CURRENT_BRANCH" \
  --build-arg VM_ID="$VM_ID"

if [[ -n "${PUBLISH}" && $CURRENT_BRANCH == "master" && $ALLOW_TAG_LATEST == true ]]; then
  echo "Tagging current image as $IMAGE_NAME:latest"
  docker buildx imagetools create -t "$IMAGE_NAME:latest" "$IMAGE_NAME:$BUILD_IMAGE_ID"
fi
