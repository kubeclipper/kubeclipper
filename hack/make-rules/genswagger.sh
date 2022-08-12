#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# 暂时 workaround
# 生成 swagger 需要在 go-restful 里注册所有的路由，这部分代码在 k8s 的 sdk 里,它会去连 etcd
# TODO: 更新 hack/gen-swagger/main.go, 复写注册路由的逻辑, 避免连接 etcd

KUBE_ROOT=$(dirname "${BASH_SOURCE[0]}")/../..
KUBE_VERBOSE="${KUBE_VERBOSE:-1}"

source "${KUBE_ROOT}/hack/lib/logging.sh"

ETCD_IMAGE=k8s.gcr.io/etcd:3.4.3-0
NAME="swagger"
# TODO: 根据系统架构和当前用户，判断是否需要提权
CMD_PRIVILEGE_PREFIX="sudo"

start_etcd(){
  local -a cmd=()
  cmd+="${CMD_PRIVILEGE_PREFIX} docker run -d --name ${NAME} --net host ${ETCD_IMAGE} \
  etcd --name=${NAME} --snapshot-count=10000 \
  --advertise-client-urls http://127.0.0.1:2379 \
  --initial-advertise-peer-urls http://127.0.0.1:2380 \
  --listen-client-urls http://127.0.0.1:2379 \
  --listen-peer-urls http://127.0.0.1:2380 \
  --listen-metrics-urls http://127.0.0.1:2381 \
  --initial-cluster=${NAME}=http://127.0.0.1:2380"
  V=2 kube::log::info "start etcd cmd is: ${cmd}"
  ${cmd}
}

destroy_etcd(){
    "${CMD_PRIVILEGE_PREFIX}" docker stop "${NAME}"
    sleep 2
    "${CMD_PRIVILEGE_PREFIX}" docker rm "${NAME}"
}

V=2 kube::log::info "start container ${NAME}"

start_etcd

V=2 kube::log::info "generate swagger.json..."

go run ${KUBE_ROOT}/hack/gen-swagger/main.go --output=${KUBE_ROOT}/docs/swagger.json

V=2 kube::log::info "destroy container ${NAME}"

destroy_etcd
