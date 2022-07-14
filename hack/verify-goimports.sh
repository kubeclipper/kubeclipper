#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

. "$(dirname "${BASH_SOURCE[0]}")/lib/init.sh"

cd "${KUBE_ROOT}/hack" || exit 1

if ! command -v goimports &> /dev/null
then
    echo "goimports could not be found on your machine, please install it first"
    exit 1
fi

cd "${KUBE_ROOT}" || exit 1

IFS=$'\n' read -r -d '' -a files < <( find . -type f -name '*.go' ! -name 'mock_*.go' -not -path "./vendor/*" -not -path "./pkg/scheme/*" && printf '\0' )

output=$(goimports -local github.com/kubeclipper/kubeclipper -l "${files[@]}")

if [ "${output}" != "" ]; then
    echo "The following files are not import formatted "
    printf '%s\n' "${output[@]}"
    echo "Please run the following command:"
    echo "  make goimports"
    exit 1
fi
