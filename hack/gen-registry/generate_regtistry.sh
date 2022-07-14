#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail


if [ "$#" -lt 2 ] || [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename "$0") <package-name> <resource-kind>

  <package-name>      the package name for golang package.
  <resource-kind>     the resource kind name, for example: Token, User


Examples:
  $(basename "$0") token Token
EOF
  exit 0
fi

GENERIC_PACKAGE_NAME="$1"
GENERIC_KIND="$2"
shift 2

ROOT_PATH=$(pwd)
CURRENT_PATH=$(dirname "${BASH_SOURCE[0]}")
DEST_PATH="${ROOT_PATH}/pkg/server/registry/${GENERIC_PACKAGE_NAME}"

mkdir -pv "${DEST_PATH}"
cp "${CURRENT_PATH}/strategy.tmpl" "${DEST_PATH}/strategy.go"
cp "${CURRENT_PATH}/rest.tmpl" "${DEST_PATH}/rest.go"

update_file_in_darwin(){
  sed -i '' 's/GENERIC_PACKAGE_NAME/'${GENERIC_PACKAGE_NAME}'/g' "${DEST_PATH}/strategy.go"
  sed -i '' 's/GENERIC_PACKAGE_NAME/'${GENERIC_PACKAGE_NAME}'/g' "${DEST_PATH}/rest.go"
  sed -i '' 's/GENERIC_KIND/'${GENERIC_KIND}'/g' "${DEST_PATH}/strategy.go"
  sed -i '' 's/GENERIC_KIND/'${GENERIC_KIND}'/g' "${DEST_PATH}/rest.go"
}


update_file_in_linux(){
  sed -i 's/GENERIC_PACKAGE_NAME/'${GENERIC_PACKAGE_NAME}'/g' "${DEST_PATH}/strategy.go"
  sed -i 's/GENERIC_PACKAGE_NAME/'${GENERIC_PACKAGE_NAME}'/g' "${DEST_PATH}/rest.go"
  sed -i 's/GENERIC_KIND/'${GENERIC_KIND}'/g' "${DEST_PATH}/strategy.go"
  sed -i 's/GENERIC_KIND/'${GENERIC_KIND}'/g' "${DEST_PATH}/rest.go"
}

if [ "$(uname)" == "Darwin" ];then
  update_file_in_darwin
else
  update_file_in_linux
fi