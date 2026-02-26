#!/bin/env bash
set -e

# Usage:
#   curl ... | ENV_VAR=... bash -
#       or
#   ENV_VAR=... ./get-kubeclipper.sh
#
# Example:
#   Installing the master branch kcctl:
#     curl ... | KC_VERSION=master bash -
#   Installing kcctl to the specified directory:
#     curl ... | KC_BIN_DIR=/usr/local/bin bash -
#
# Environment variables:
#   - KC_VERSION
#     Version of kcctl to download from AliCloud OSS.
#
#   - KC_BIN_DIR
#     Directory to install kcctl binary, or use /usr/local/bin as the default.
#
#   - KC_REGION
#     Region of installing kcctl, Will attempt to set image repo mirror.
#
#   - KC_IMAGE_REPO_MIRROR
#     Image repo mirror, or use registry.aliyuncs.com/google_containers as the default.

info() {
  echo '[INFO] ' "$@"
}
warn() {
  echo '[WARN] ' "$@" >&2
}
fatal() {
  echo '[ERROR] ' "$@" >&2
  exit 1
}

SUDO=sudo
if [ $(id -u) -eq 0 ]; then
  SUDO=
fi

OS_TYPE="linux"

# default github repo
DOWNLOAD_URL="https://oss.kubeclipper.io/kc"
# default directory for storing binary files
BIN_DIR="/usr/local/bin"

if [[ -z "${KC_VERSION}" ]]; then
  KC_VERSION="v1.5.0"
fi
info "The ${KC_VERSION} version will be installed"

verify_system() {
  verify_os
  verify_arch
  verify_downloader curl || verify_downloader wget || fatal 'Please install curl or wget first!'
}

verify_os() {
  os=$(uname)
  if [[ $os == 'Darwin' ]]; then
    OS_TYPE="darwin"
  fi
}

verify_arch() {
  if [[ -z "$ARCH" ]]; then
    ARCH=$(uname -m)
  fi
  case $ARCH in
  amd64)
    ARCH=amd64
    STRIP=0
    ;;
  x86_64)
    ARCH=amd64
    STRIP=0
    ;;
  arm64)
    ARCH=arm64
    STRIP=1
    ;;
  aarch64)
    ARCH=arm64
    STRIP=1
    ;;
  *)
    fatal "Unsupported architecture $ARCH"
    ;;
  esac
}

verify_downloader() {
  [ -x "$(command -v "$1")" ] || return 1
  DOWNLOADER="$1"
  return 0
}

set_env() {
  if [ "x${KC_REGION}" == "xcn" ]; then
    info "KC_REGION is assigned cn, kc.env file will be created"

    mirror='registry.aliyuncs.com/google_containers'
    if [[ -n ${KC_IMAGE_REPO_MIRROR} ]]; then
      mirror=${KC_IMAGE_REPO_MIRROR}
    fi
    create_env_file "$mirror"
    # set download url
    DOWNLOAD_URL="https://oss.kubeclipper.io/kc"
  fi

  if [ -n "${KC_BIN_DIR}" ]; then
    BIN_DIR=${KC_BIN_DIR}
  else
    IFS=":" read -ra dirs <<<"$(echo "${PATH}")"
    for dirx in "${dirs[@]}"; do
      if [ "$dirx" == "/usr/local/bin" ] || [ "$dirx" == "/usr/bin" ]; then
        BIN_DIR=$dirx
        break
      fi
    done
  fi
}

create_env_file() {
  file_path="/etc/kc/kc.env"
  info "env: Creating environment file $file_path}"
  $SUDO mkdir -pv "$(dirname $file_path)"
  $SUDO touch $file_path

  echo "KC_IMAGE_REPO_MIRROR=\"$1\"" >$file_path
}

install_pkg() {
  TMP_DIR=$(mktemp -d -t kcctl.XXXXXXXX)
  if [ $? -ne 0 ] || [ -z "$TMP_DIR" ]; then
    fatal 'Failed to create a temporary directory!'
  fi
  REMOTE=${DOWNLOAD_URL}/${KC_VERSION}/kc-${OS_TYPE}-${ARCH}.tar.gz
  TARGET=${TMP_DIR}"/kcctl.tar.gz"
  # download
  download "${TARGET}" "${REMOTE}"
  # decompress the file to the specified directory
  tar -zxvf "${TARGET}" -C "${TMP_DIR}" --strip-components "${STRIP}"
  info "Installing kcctl to ${BIN_DIR}/kcctl"
  # move binary file to BIN_DIR
  $SUDO mv "${TMP_DIR}"/kcctl "${BIN_DIR}"/
  rm -rf "${TMP_DIR}"
}

download() {
  [ $# -eq 2 ] || fatal 'download needs exactly 2 arguments'

  info "Downloading package $2"
  case $DOWNLOADER in
  curl)
    curl -o "$1" "$2"
    ;;
  wget)
    wget -O "$1" "$2"
    ;;
  *)
    fatal "Incorrect executable '$DOWNLOADER'"
    ;;
  esac

  [ $? -eq 0 ] || fatal 'Download package failed'
}

verify_install() {
  [ -x "$(command -v kcctl)" ] || fatal "Kcctl installation failed, please try again!"

  printf "%40s\n" "$(tput setaf 4)
  Kcctl has been installed successfully!
    Run 'kcctl version' to view the version.
    Run 'kcctl -h' for more help.

        __ ______________________
       / //_/ ____/ ____/_  __/ /
      / ,< / /   / /     / / / /
     / /| / /___/ /___  / / / /___
    /_/ |_\____/\____/ /_/ /_____/
    repository: github.com/kubeclipper
  $(tput sgr0)"
}

{
  verify_system
  set_env
  install_pkg
  verify_install
}

