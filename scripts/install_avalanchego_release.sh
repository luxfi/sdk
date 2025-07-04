#!/usr/bin/env bash
set -e

# Load the versions
SUBNET_EVM_PATH=$(
  cd "$(dirname "${BASH_SOURCE[0]}")"
  cd .. && pwd
)
source "$SUBNET_EVM_PATH"/scripts/versions.sh

# Load the constants
source "$SUBNET_EVM_PATH"/scripts/constants.sh

############################
# download node
# https://github.com/luxdefi/node/releases
GOARCH=$(go env GOARCH)
GOOS=$(go env GOOS)
BASEDIR=${BASEDIR:-"/tmp/node-release"}
LUXD_BUILD_PATH=${LUXD_BUILD_PATH:-${BASEDIR}/node}

mkdir -p ${BASEDIR}

LUXD_DOWNLOAD_URL=https://github.com/luxdefi/node/releases/download/${LUX_VERSION}/node-linux-${GOARCH}-${LUX_VERSION}.tar.gz
LUXD_DOWNLOAD_PATH=${BASEDIR}/node-linux-${GOARCH}-${LUX_VERSION}.tar.gz

if [[ ${GOOS} == "darwin" ]]; then
  LUXD_DOWNLOAD_URL=https://github.com/luxdefi/node/releases/download/${LUX_VERSION}/node-macos-${LUX_VERSION}.zip
  LUXD_DOWNLOAD_PATH=${BASEDIR}/node-macos-${LUX_VERSION}.zip
fi

BUILD_DIR=${LUXD_BUILD_PATH}-${LUX_VERSION}

extract_archive() {
  mkdir -p ${BUILD_DIR}

  if [[ ${LUXD_DOWNLOAD_PATH} == *.tar.gz ]]; then
    tar xzvf ${LUXD_DOWNLOAD_PATH} --directory ${BUILD_DIR} --strip-components 1
  elif [[ ${LUXD_DOWNLOAD_PATH} == *.zip ]]; then
    unzip ${LUXD_DOWNLOAD_PATH} -d ${BUILD_DIR}
    mv ${BUILD_DIR}/build/* ${BUILD_DIR}
    rm -rf ${BUILD_DIR}/build/
  fi
}

# first check if we already have the archive
if [[ -f ${LUXD_DOWNLOAD_PATH} ]]; then
  # if the download path already exists, extract and exit
  echo "found node ${LUX_VERSION} at ${LUXD_DOWNLOAD_PATH}"

  extract_archive
else
  # try to download the archive if it exists
  if curl -s --head --request GET ${LUXD_DOWNLOAD_URL} | grep "302" > /dev/null; then
    echo "${LUXD_DOWNLOAD_URL} found"
    echo "downloading to ${LUXD_DOWNLOAD_PATH}"
    curl -L ${LUXD_DOWNLOAD_URL} -o ${LUXD_DOWNLOAD_PATH}

    extract_archive
  else
    # else the version is a git commitish (or it's invalid)
    GIT_CLONE_URL=https://github.com/luxdefi/node.git
    GIT_CLONE_PATH=${BASEDIR}/node-repo/

    # check to see if the repo already exists, if not clone it
    if [[ ! -d ${GIT_CLONE_PATH} ]]; then
      echo "cloning ${GIT_CLONE_URL} to ${GIT_CLONE_PATH}"
      git clone --no-checkout ${GIT_CLONE_URL} ${GIT_CLONE_PATH}
    fi

    # check to see if the commitish exists in the repo
    WORKDIR=$(pwd)

    cd ${GIT_CLONE_PATH}

    git fetch

    echo "checking out ${LUX_VERSION}"

    set +e
    # try to checkout the branch
    git checkout origin/${LUX_VERSION} > /dev/null 2>&1
    CHECKOUT_STATUS=$?
    set -e

    # if it's not a branch, try to checkout the commit
    if [[ $CHECKOUT_STATUS -ne 0 ]]; then
      set +e
      git checkout ${LUX_VERSION} > /dev/null 2>&1
      CHECKOUT_STATUS=$?
      set -e

      if [[ $CHECKOUT_STATUS -ne 0 ]]; then
        echo
        echo "'${VERSION}' is not a valid release tag, commit hash, or branch name"
        exit 1
      fi
    fi

    COMMIT=$(git rev-parse HEAD)

    # use the commit hash instead of the branch name or tag
    BUILD_DIR=${LUXD_BUILD_PATH}-${COMMIT}

    # if the build-directory doesn't exist, build node
    if [[ ! -d ${BUILD_DIR} ]]; then
      echo "building node ${COMMIT} to ${BUILD_DIR}"
      ./scripts/build.sh
      mkdir -p ${BUILD_DIR}

      mv ${GIT_CLONE_PATH}/build/* ${BUILD_DIR}/
    fi

    cd $WORKDIR
  fi
fi

LUXD_PATH=${LUXD_BUILD_PATH}/node
LUXD_PLUGIN_DIR=${LUXD_BUILD_PATH}/plugins

mkdir -p ${LUXD_BUILD_PATH}

cp ${BUILD_DIR}/node ${LUXD_PATH}


echo "Installed Lux release ${LUX_VERSION}"
echo "Lux Path: ${LUXD_PATH}"
echo "Plugin Dir: ${LUXD_PLUGIN_DIR}"
