set -o errexit
set -o nounset
set -o pipefail

KUBE_ROOT=$(dirname "${BASH_SOURCE[0]}")/../..
KUBE_VERBOSE="${KUBE_VERBOSE:-1}"

source "${KUBE_ROOT}/hack/lib/init.sh"

kube::golang::build_binaries "$@"
