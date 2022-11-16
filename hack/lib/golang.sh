#!/usr/bin/env bash

readonly KUBE_GO_PACKAGE=github.com/kubeclipper/kubeclipper
readonly KUBE_GOPATH="${KUBE_OUTPUT}"

kube::golang::server_targets() {
  local targets=(
    cmd/kubeclipper-server
    cmd/kubeclipper-agent
    cmd/kubeclipper-proxy
    cmd/kcctl
    test/e2e
    vendor/github.com/onsi/ginkgo/ginkgo
  )
  echo "${targets[@]}"
}

IFS=" " read -ra KUBE_SERVER_TARGETS <<< "$(kube::golang::server_targets)"
readonly KUBE_SERVER_TARGETS
readonly KUBE_SERVER_BINARIES=("${KUBE_SERVER_TARGETS[@]##*/}")


readonly KUBE_ALL_TARGETS=(
  "${KUBE_SERVER_TARGETS[@]}"
)

readonly KUBE_ALL_BINARIES=("${KUBE_ALL_TARGETS[@]##*/}")

readonly KUBE_STATIC_LIBRARIES=(
  kubeclipper-server
  kubeclipper-agent
  kubeclipper-proxy
  kcctl
  e2e.test
)


kube::golang::host_platform() {
  echo "$(go env GOHOSTOS)/$(go env GOHOSTARCH)"
}

# kube::binaries_from_targets take a list of build targets and return the
# full go package to be built
kube::golang::binaries_from_targets() {
  local target
  for target; do
    # If the target starts with what looks like a domain name, assume it has a
    # fully-qualified package name rather than one that needs the Kubernetes
    # package prepended.
    if [[ "${target}" =~ ^([[:alnum:]]+".")+[[:alnum:]]+"/" ]]; then
      echo "${target}"
    else
      echo "${KUBE_GO_PACKAGE}/${target}"
    fi
  done
}

kube::golang::build_binaries() {
  # Create a sub-shell so that we don't pollute the outer environment
  (
    local host_platform
    host_platform=$(kube::golang::host_platform)

    local goflags goldflags
    # If GOLDFLAGS is unset, then set it to the a default of "-s -w".
    # Disable SC2153 for this, as it will throw a warning that the local
    # variable goldflags will exist, and it suggest changing it to this.
    # shellcheck disable=SC2153
    goldflags="${GOLDFLAGS=-s -w -buildid=} $(kube::version::ldflags)"

    local -a targets=()
    local arg

    for arg; do
      if [[ "${arg}" == -* ]]; then
        # Assume arguments starting with a dash are flags to pass to go.
        goflags+=("${arg}")
      else
        targets+=("${arg}")
      fi
    done

    if [[ ${#targets[@]} -eq 0 ]]; then
      targets=("${KUBE_ALL_TARGETS[@]}")
    fi

    local -a platforms
    IFS=" " read -ra platforms <<< "${KUBE_BUILD_PLATFORMS:-}"
    if [[ ${#platforms[@]} -eq 0 ]]; then
      platforms=("${host_platform}")
    fi

    local -a binaries
    while IFS="" read -r binary; do binaries+=("$binary"); done < <(kube::golang::binaries_from_targets "${targets[@]}")

    for platform in "${platforms[@]}"; do
      (
        kube::golang::set_platform_envs "${platform}"
        kube::golang::build_binaries_for_platform "${platform}"
      )
    done
  )
}


# Takes the platform name ($1) and sets the appropriate golang env variables
# for that platform.
kube::golang::set_platform_envs() {
  [[ -n ${1-} ]] || {
    kube::log::error_exit "!!! Internal error. No platform set in kube::golang::set_platform_envs"
  }

  export GOOS=${platform%/*}
  export GOARCH=${platform##*/}

  # Do not set CC when building natively on a platform, only if cross-compiling from linux/amd64
  if [[ $(kube::golang::host_platform) == "linux/amd64" ]]; then
    # Dynamic CGO linking for other server architectures than linux/amd64 goes here
    # If you want to include support for more server platforms than these, add arch-specific gcc names here
    case "${platform}" in
      "linux/arm")
        export CGO_ENABLED=1
        export CC=arm-linux-gnueabihf-gcc
        ;;
      "linux/arm64")
        export CGO_ENABLED=1
        export CC=aarch64-linux-gnu-gcc
        ;;
      "linux/ppc64le")
        export CGO_ENABLED=1
        export CC=powerpc64le-linux-gnu-gcc
        ;;
      "linux/s390x")
        export CGO_ENABLED=1
        export CC=s390x-linux-gnu-gcc
        ;;
    esac
  fi
}

kube::golang::unset_platform_envs() {
  unset GOOS
  unset GOARCH
  unset GOROOT
  unset CGO_ENABLED
  unset CC
}


kube::golang::build_binaries_for_platform() {
  # This is for sanity.  Without it, user umasks can leak through.
  umask 0022

  local platform=$1

  local -a statics=()
  local -a nonstatics=()
  local -a tests=()

  V=2 kube::log::info "Env for ${platform}: GOOS=${GOOS-} GOARCH=${GOARCH-} GOROOT=${GOROOT-} CGO_ENABLED=${CGO_ENABLED-} CC=${CC-} TAGS=${GOTAGS-}"

  for binary in "${binaries[@]}"; do
    if [[ "${binary}" =~ ".test"$ ]]; then
      tests+=("${binary}")
    elif kube::golang::is_statically_linked_library "${binary}"; then
      statics+=("${binary}")
    else
      nonstatics+=("${binary}")
    fi
  done

  local -a build_args
  if [[ "${#statics[@]}" != 0 ]]; then
    build_args=(
      -installsuffix static
      ${goflags:+"${goflags[@]}"}
      #-gcflags "${gogcflags:-}"
      #-asmflags "${goasmflags:-}"
      -ldflags "${goldflags:-}"
      -tags "${GOTAGS:-}"
    )
    V=2 kube::log::info "build with disable cgo..."
    CGO_ENABLED=0 kube::golang::build_some_binaries "${statics[@]}"
  fi

  if [[ "${#nonstatics[@]}" != 0 ]]; then
    build_args=(
      ${goflags:+"${goflags[@]}"}
      #-gcflags "${gogcflags:-}"
      #-asmflags "${goasmflags:-}"
      -ldflags "${goldflags:-}"
      -tags "${GOTAGS:-}"
    )
    kube::golang::build_some_binaries "${nonstatics[@]}"
  fi

  for test in "${tests[@]:+${tests[@]}}"; do
    local outfile testpkg
    outfile=$(kube::golang::outfile_for_binary "${test}" "${platform}")
    testpkg=$(dirname "${test}")

    mkdir -p "$(dirname "${outfile}")"
    go test -c \
      ${goflags:+"${goflags[@]}"} \
      -gcflags "${gogcflags:-}" \
      -asmflags "${goasmflags:-}" \
      -ldflags "${goldflags:-}" \
      -tags "${gotags:-}" \
      -o "${outfile}" \
      "${testpkg}"
  done
}


# KUBE_CGO_OVERRIDES is a space-separated list of binaries which should be built
# with CGO enabled, assuming CGO is supported on the target platform.
# This overrides any entry in KUBE_STATIC_LIBRARIES.
IFS=" " read -ra KUBE_CGO_OVERRIDES <<< "${KUBE_CGO_OVERRIDES:-}"
readonly KUBE_CGO_OVERRIDES
# KUBE_STATIC_OVERRIDES is a space-separated list of binaries which should be
# built with CGO disabled. This is in addition to the list in
# KUBE_STATIC_LIBRARIES.
IFS=" " read -ra KUBE_STATIC_OVERRIDES <<< "${KUBE_STATIC_OVERRIDES:-}"
readonly KUBE_STATIC_OVERRIDES

kube::golang::is_statically_linked_library() {
  local e
  # Explicitly enable cgo when building kubectl for darwin from darwin.
#  [[ "$(go env GOHOSTOS)" == "darwin" && "$(go env GOOS)" == "darwin" &&
#    "$1" == *"/kubectl" ]] && return 1
  if [[ -n "${KUBE_CGO_OVERRIDES:+x}" ]]; then
    for e in "${KUBE_CGO_OVERRIDES[@]}"; do [[ "${1}" == *"/${e}" ]] && return 1; done;
  fi
  for e in "${KUBE_STATIC_LIBRARIES[@]}"; do [[ "${1}" == *"/${e}" ]] && return 0; done;
  if [[ -n "${KUBE_STATIC_OVERRIDES:+x}" ]]; then
    for e in "${KUBE_STATIC_OVERRIDES[@]}"; do [[ "${1}" == *"/${e}" ]] && return 0; done;
  fi
  return 1;
}


# Arguments: a list of kubernetes packages to build.
# Expected variables: ${build_args} should be set to an array of Go build arguments.
# In addition, ${package} and ${platform} should have been set earlier, and if
# ${KUBE_BUILD_WITH_COVERAGE} is set, coverage instrumentation will be enabled.
#
# Invokes Go to actually build some packages. If coverage is disabled, simply invokes
# go install. If coverage is enabled, builds covered binaries using go test, temporarily
# producing the required unit test files and then cleaning up after itself.
# Non-covered binaries are then built using go install as usual.
kube::golang::build_some_binaries() {
  V=3 kube::log::info "Building args is ${build_args[@]} ..."
  for package in "$@"; do
    # shellcheck disable=SC2145
    V=2 kube::log::info "Building ${package} ..."
    go build -o "$(kube::golang::outfile_for_binary "${package}" "${platform}")" \
      "${build_args[@]}" \
      "${package}"
  done
}


# Try and replicate the native binary placement of go install without
# calling go install.
kube::golang::outfile_for_binary() {
  local binary=$1
  local platform=$2
  local output_path="${KUBE_GOPATH}"
  local bin
  bin=$(basename "${binary}")
  if [[ "${platform}" != "${host_platform}" || ${KUBE_ALL_WITH_PREFIX:-"false"} == "true" ]]; then
    output_path="${output_path}/${platform//\//_}"
  fi
  if [[ ${GOOS} == "windows" ]]; then
    bin="${bin}.exe"
  fi
  echo "${output_path}/${bin}"
}