#!/usr/bin/env bash

set -euo pipefail

readonly GO_REQUIRED_VERSION="go1.26.10"
readonly MOBY_COMMIT="89c5e8fd66634b6128fc4c0e6f1236e2540e46e0"
readonly DOCKER_CLI_COMMIT="ecc694264de6b34e4b59d16245603382f22fa813"
# buildx v0.33.0 is the first release that vendors google.golang.org/grpc >= v1.79.3
# (CVE-2026-33186). It also pulls docker/cli v29.3.1 as a library dep (fixes
# CVE-2025-15558 without requiring Docker Engine 29).
readonly BUILDX_COMMIT="f7897eba028583e0071642db3c011e860444f8cf"
readonly COMPOSE_COMMIT="49b1c1e932aecd3a4843b392162d4bd56021935e"
readonly GRPC_MIN_VERSION="v1.79.3"

readonly WORK_DIR="/tmp/docker-28-patched-build"
readonly OUTPUT_DIR="/out"

fetch_source() {
  local repository="$1"
  local reference="$2"
  local commit="$3"
  local destination="$4"

  git init -q "${destination}"
  git -C "${destination}" remote add origin "${repository}"
  git -C "${destination}" fetch -q --depth 1 origin "${reference}"
  git -C "${destination}" checkout -q --detach FETCH_HEAD
  test "$(git -C "${destination}" rev-parse HEAD)" = "${commit}"
}

verify_go_binary() {
  local binary="$1"
  local actual_version

  actual_version="$(go version -m "${binary}" | awk 'NR == 1 { print $2 }')"
  if [[ "${actual_version}" != "${GO_REQUIRED_VERSION}" ]]; then
    echo "Expected ${binary} to use ${GO_REQUIRED_VERSION}, found ${actual_version}" >&2
    exit 1
  fi
}

# Force google.golang.org/grpc to the CVE-2026-33186 floor for module-based builds.
bump_grpc_module() {
  local module_dir="$1"

  (
    cd "${module_dir}"
    go get "google.golang.org/grpc@${GRPC_MIN_VERSION}"
    go mod tidy
  )
}

# Moby / docker/cli keep deps in vendor.mod + vendor/. Promote vendor.mod to a
# temporary go.mod, bump grpc, re-vendor, then restore vendor.mod/vendor.sum.
bump_grpc_vendor() {
  local module_dir="$1"

  (
    cd "${module_dir}"
    cp -f vendor.mod go.mod
    cp -f vendor.sum go.sum
    go get "google.golang.org/grpc@${GRPC_MIN_VERSION}"
    go mod tidy
    go mod vendor
    cp -f go.mod vendor.mod
    cp -f go.sum vendor.sum
    rm -f go.mod go.sum
  )
}

mkdir -p "${WORK_DIR}" "${OUTPUT_DIR}"
test "$(go env GOVERSION)" = "${GO_REQUIRED_VERSION}"
export GOTOOLCHAIN=local

# Rebuild the Docker 28.5.2 daemon and its exact bundled runtime versions.
# Moby v28.5.2 pins containerd v1.7.28 and runc v1.3.3 in its Dockerfile.
# Additionally bump vendored grpc to clear CVE-2026-33186 while staying on 28.5.2.
fetch_source https://github.com/moby/moby.git refs/tags/v28.5.2 \
  "${MOBY_COMMIT}" "${WORK_DIR}/moby"
bump_grpc_vendor "${WORK_DIR}/moby"
(
  cd "${WORK_DIR}/moby"
  GO111MODULE=off \
    VERSION=28.5.2 \
    DOCKER_GITCOMMIT="${MOBY_COMMIT}" \
    ./hack/make.sh binary
)
install -m 0755 "${WORK_DIR}/moby/bundles/binary-daemon/dockerd" "${OUTPUT_DIR}/dockerd"
install -m 0755 "${WORK_DIR}/moby/bundles/binary-daemon/docker-proxy" "${OUTPUT_DIR}/docker-proxy"

# Use Moby's own pinned source stages and build settings for containerd and runc.
fetch_source https://github.com/containerd/containerd.git \
  refs/tags/v1.7.28 b98a3aace656320842a23f4a392a33f46af97866 \
  "${WORK_DIR}/containerd"
(
  cd "${WORK_DIR}/containerd"
  VERSION=v1.7.28 make STATIC=1 binaries
)
install -m 0755 "${WORK_DIR}/containerd/bin/containerd" "${OUTPUT_DIR}/containerd"
install -m 0755 "${WORK_DIR}/containerd/bin/containerd-shim-runc-v2" \
  "${OUTPUT_DIR}/containerd-shim-runc-v2"
install -m 0755 "${WORK_DIR}/containerd/bin/ctr" "${OUTPUT_DIR}/ctr"

fetch_source https://github.com/opencontainers/runc.git \
  refs/tags/v1.3.3 d842d7719497cc3b774fd71620278ac9e17710e0 \
  "${WORK_DIR}/runc"
(
  cd "${WORK_DIR}/runc"
  CGO_ENABLED=1 make static
)
install -m 0755 "${WORK_DIR}/runc/runc" "${OUTPUT_DIR}/runc"

# Preserve the Docker 28 CLI while rebuilding with the patched Go toolchain and
# a grpc floor that clears CVE-2026-33186.
readonly CLI_GOPATH="${WORK_DIR}/docker-cli-gopath"
readonly CLI_SOURCE="${CLI_GOPATH}/src/github.com/docker/cli"
fetch_source https://github.com/docker/cli.git refs/tags/v28.5.2 \
  "${DOCKER_CLI_COMMIT}" "${CLI_SOURCE}"
if [[ -f "${CLI_SOURCE}/vendor.mod" ]]; then
  bump_grpc_vendor "${CLI_SOURCE}"
fi
(
  cd "${CLI_SOURCE}"
  GOPATH="${CLI_GOPATH}" DISABLE_WARN_OUTSIDE_CONTAINER=1 VERSION=28.5.2 make binary
)
install -m 0755 "${CLI_SOURCE}/build/docker" "${OUTPUT_DIR}/docker"

fetch_source https://github.com/docker/buildx.git refs/tags/v0.33.0 \
  "${BUILDX_COMMIT}" "${WORK_DIR}/buildx"
(
  cd "${WORK_DIR}/buildx"
  CGO_ENABLED=0 \
    DESTDIR="${OUTPUT_DIR}" \
    VERSION=v0.33.0 \
    REVISION="${BUILDX_COMMIT}" \
    GO_EXTRA_LDFLAGS="-s -w" \
    ./hack/build
)

# Compose v2.40.3 still vendors grpc < 1.79.3; bump in-tree before build rather
# than jumping to Compose v5 (major). Official Compose >= v5.2.0 also clears this.
fetch_source https://github.com/docker/compose.git refs/tags/v2.40.3 \
  "${COMPOSE_COMMIT}" "${WORK_DIR}/compose"
bump_grpc_module "${WORK_DIR}/compose"
(
  cd "${WORK_DIR}/compose"
  CGO_ENABLED=0 VERSION=v2.40.3 DESTDIR="${OUTPUT_DIR}" make build
)

for binary in \
  containerd \
  containerd-shim-runc-v2 \
  ctr \
  docker \
  docker-buildx \
  docker-compose \
  docker-proxy \
  dockerd \
  runc; do
  verify_go_binary "${OUTPUT_DIR}/${binary}"
done

# Drop source trees so they are not retained in the Dockerfile layer.
rm -rf "${WORK_DIR}"
