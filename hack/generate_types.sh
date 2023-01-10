#!/bin/bash


SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

./hack/generate_internal_group.sh "deepcopy" \
  github.com/kubeclipper/kubeclipper/pkg/generated github.com/kubeclipper/kubeclipper/pkg/scheme github.com/kubeclipper/kubeclipper/pkg/scheme \
  "core:v1 iam:v1" \
  --output-base ./ \
  --go-header-file "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \

mv github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/*.go ./pkg/scheme/core/v1/
mv github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1/*.go ./pkg/scheme/iam/v1/


rm -rf github.com