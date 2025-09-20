#!/usr/bin/env bash
set -e

echo "=== Installing Lux Node Release ==="

EVM_PATH=$(
  cd "$(dirname "${BASH_SOURCE[0]}")"
  cd .. && pwd
)

# Load the constants
source "$EVM_PATH"/scripts/constants.sh

# Set LUXD_VERSION from LUX_VERSION if not already set
# Remove leading 'v' if present to avoid double 'v' in URLs
LUXD_VERSION=${LUXD_VERSION:-${LUX_VERSION#v}}

############################
# download luxd/avalanchego
# https://github.com/ava-labs/avalanchego/releases
GOARCH=$(go env GOARCH)
GOOS=$(go env GOOS)
BASEDIR=${BASEDIR:-"/tmp/luxd-release"}
LUXD_BUILD_PATH=${LUXD_BUILD_PATH:-${BASEDIR}/luxd}

echo "Installing to: $LUXD_BUILD_PATH"
echo "OS: $GOOS, Arch: $GOARCH"

# Create base directory
mkdir -p "${BASEDIR}"

# Check if already installed
if [[ -f "${LUXD_BUILD_PATH}" ]]; then
  echo "luxd already installed at ${LUXD_BUILD_PATH}"
  INSTALLED_VERSION=$("${LUXD_BUILD_PATH}" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || echo "unknown")
  echo "Installed version: $INSTALLED_VERSION"
  
  if [[ "${LUXD_VERSION}" == "${INSTALLED_VERSION}" ]]; then
    echo "Version matches required ${LUXD_VERSION}, skipping download"
    exit 0
  fi
  
  echo "Version mismatch, reinstalling..."
  rm -f "${LUXD_BUILD_PATH}"
fi

############################
# download and install luxd
############################

# For now, we'll build from source if the binary isn't available
echo "Checking for pre-built binary..."
DOWNLOAD_URL="https://github.com/ava-labs/avalanchego/releases/download/v${LUXD_VERSION}/avalanchego-linux-${GOARCH}-v${LUXD_VERSION}.tar.gz"

if [[ "$GOOS" == "darwin" ]]; then
  DOWNLOAD_URL="https://github.com/ava-labs/avalanchego/releases/download/v${LUXD_VERSION}/avalanchego-macos-v${LUXD_VERSION}.zip"
elif [[ "$GOOS" == "windows" ]]; then
  DOWNLOAD_URL="https://github.com/ava-labs/avalanchego/releases/download/v${LUXD_VERSION}/avalanchego-windows-${GOARCH}-v${LUXD_VERSION}.zip"
fi

echo "Attempting to download from: $DOWNLOAD_URL"

# Try to download pre-built binary
if command -v curl &> /dev/null; then
  HTTP_CODE=$(curl -sL -w "%{http_code}" -o "${BASEDIR}/luxd.tar.gz" "$DOWNLOAD_URL")
  
  if [[ "$HTTP_CODE" == "200" ]]; then
    echo "Downloaded pre-built binary"
    if [[ "$GOOS" == "darwin" ]]; then
      unzip -q "${BASEDIR}/luxd.tar.gz" -d "${BASEDIR}"
    else
      tar -xzf "${BASEDIR}/luxd.tar.gz" -C "${BASEDIR}"
    fi
    
    # Find the luxd/avalanchego binary (try both names)
    LUXD_PATH=$(find "${BASEDIR}" -name "luxd" -type f | head -1)
    if [[ -z "$LUXD_PATH" ]]; then
      LUXD_PATH=$(find "${BASEDIR}" -name "avalanchego" -type f | head -1)
    fi
    if [[ -n "$LUXD_PATH" ]]; then
      mv "$LUXD_PATH" "${LUXD_BUILD_PATH}"
      chmod +x "${LUXD_BUILD_PATH}"
      echo "Installed luxd to ${LUXD_BUILD_PATH}"
    else
      echo "Error: luxd binary not found in archive"
      exit 1
    fi
  else
    echo "Pre-built binary not available (HTTP $HTTP_CODE)"
    
    # Fallback: build from source
    echo "Building from source..."
    TEMP_DIR="${BASEDIR}/build_tmp"
    rm -rf "$TEMP_DIR"
    git clone --depth 1 --branch "v${LUXD_VERSION}" https://github.com/ava-labs/avalanchego.git "$TEMP_DIR"
    
    cd "$TEMP_DIR"
    ./scripts/build.sh
    
    if [[ -f "build/luxd" ]]; then
      mv "build/luxd" "${LUXD_BUILD_PATH}"
      echo "Built and installed luxd to ${LUXD_BUILD_PATH}"
    else
      echo "Error: Failed to build luxd"
      exit 1
    fi
    
    cd "$EVM_PATH"
    rm -rf "$TEMP_DIR"
  fi
else
  echo "Error: curl not found"
  exit 1
fi

# Verify installation
if [[ ! -f "${LUXD_BUILD_PATH}" ]]; then
  echo "Error: Installation failed - luxd not found at ${LUXD_BUILD_PATH}"
  exit 1
fi

echo "Verifying installation..."
"${LUXD_BUILD_PATH}" --version || {
  echo "Error: luxd verification failed"
  exit 1
}

echo "=== Lux Node Installation Complete ==="
