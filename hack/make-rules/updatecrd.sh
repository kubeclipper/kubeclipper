#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail


KUBE_ROOT=$(dirname "${BASH_SOURCE[0]}")/../..
KUBE_VERBOSE="${KUBE_VERBOSE:-1}"
source "${KUBE_ROOT}/hack/lib/logging.sh"

pwd=$(pwd)

V=3 kube::log::info "pwd is ${pwd}"

update_file_in_darwin(){
  for file in "${pwd}"/configs/crds/*.yaml
  do
    V=2 kube::log::info "update file ${file}..."
    sed -i '' '/ metadata:/,+2d' "${file}"
  done
}


update_file_in_linux(){
  for file in "${pwd}"/configs/crds/*.yaml
  do
    V=2 kube::log::info "update file ${file}..."
    sed -i '/ metadata:/,+2d' "${file}"
  done
}


if [ "$(uname)" == "Darwin" ];then
  update_file_in_darwin
else
  update_file_in_linux
fi