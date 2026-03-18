#!/usr/bin/env sh
set -eu
ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"
METADATA_SCRIPT="${ROOT_DIR}/scripts/metadata.sh"

## Print usage information and exit.
##
## :returns: Does not return (exits with code 1).
## :rtype: None
usage() {
  echo "Usage: $0 <target> [buildx args...]"
  echo ""
  echo "Arguments:"
  echo "  target          Build target (e.g., nginx, haproxy, apache-httpd)"
  echo "  buildx args     Additional arguments passed to docker buildx build"
  echo ""
  echo "Environment variables:"
  echo "  BUILDKIT_PLATFORM     Target platform for cross-platform builds"
  echo "                      Default: linux/amd64"
  echo "                      Example: BUILDKIT_PLATFORM=linux/arm64"
  echo "  BUILD_OUTPUT_DEST     Output destination directory for build artifacts"
  echo "                      Default: .out/<target>/"
  echo "                      Example (CI): .out/<target>/"
  echo "                      Example: BUILD_OUTPUT_DEST=/custom/output/nginx"
  echo "  BUILDKIT_CACHE_BACKEND Cache backend type for Docker BuildKit"
  echo "                      Default: local"
  echo "                      Allowed: local, gha"
  echo "                      Example: BUILDKIT_CACHE_BACKEND=gha"
  echo "  BUILDKIT_NETWORK      Network mode for Docker BuildKit"
  echo "                      Default: default"
  echo "                      Allowed: default, host, none"
  echo "                      Example: BUILDKIT_NETWORK=default"
  echo "  CI                   CI environment flag"
  echo "                      When true or GITHUB_ACTIONS=true, uses CI defaults"
  echo "                      Example: CI=true"
  echo "  GITHUB_ACTIONS        GitHub Actions environment flag"
  echo "                      When true, uses CI defaults"
  echo "                      Example: GITHUB_ACTIONS=true"
  echo ""
  echo "Examples:"
  echo "  $0 nginx"
  echo "  $0 haproxy --progress=plain"
  echo "  BUILDKIT_PLATFORM=linux/arm64 $0 nginx"
  echo "  BUILD_OUTPUT_DEST=.out/custom/nginx $0 nginx"
  exit 1
}
if [ "$#" -lt 1 ]; then
  usage
fi
TARGET="$1"
shift
case "${TARGET}" in
  *[!A-Za-z0-9._-]*)
    echo "Error: Invalid target '${TARGET}'"
    echo "Allowed characters: letters, numbers, dot, underscore, hyphen"
    exit 1
    ;;
esac
if [ ! -d "${ROOT_DIR}/${TARGET}" ]; then
  echo "Error: Target directory '${TARGET}' not found"
  exit 1
fi
if [ ! -f "${ROOT_DIR}/${TARGET}/Dockerfile" ]; then
  echo "Error: Dockerfile not found in '${TARGET}'"
  exit 1
fi
BUILDKIT_PLATFORM="${BUILDKIT_PLATFORM:-linux/amd64}"
BUILDKIT_CACHE_BACKEND="${BUILDKIT_CACHE_BACKEND:-local}"
BUILDKIT_NETWORK="${BUILDKIT_NETWORK:-default}"
case "${BUILDKIT_CACHE_BACKEND}" in
  local)
    mkdir -p "${ROOT_DIR}/.cache/${TARGET}"
    CACHE_FROM="type=local,src=${ROOT_DIR}/.cache/${TARGET}"
    CACHE_TO="type=local,dest=${ROOT_DIR}/.cache/${TARGET},mode=max"
    ;;
  gha)
    CACHE_FROM="type=gha,scope=${TARGET}"
    CACHE_TO="type=gha,mode=max,scope=${TARGET}"
    ;;
  *)
    echo "Error: Invalid BUILDKIT_CACHE_BACKEND '${BUILDKIT_CACHE_BACKEND}'"
    echo "Allowed values: local, gha"
    exit 1
    ;;
esac
BUILD_OUTPUT_DEST="${BUILD_OUTPUT_DEST:-${ROOT_DIR}/.out/${TARGET}}"
mkdir -p "${BUILD_OUTPUT_DEST}"
while IFS='=' read -r key value || [ -n "${key}" ]; do
  case "${key}" in
    '' | \#*) continue ;;
  esac
  if [ -z "${value}" ]; then
    continue
  fi
  value=$(printf '%s\n' "${value}" | tr -d '\r')
  case "${key}" in
    *[!A-Z0-9_]*)
      continue
      ;;
  esac
  set -- "--build-arg=${key}=${value}" "$@"
done <<EOF
$(sh "${METADATA_SCRIPT}" get-env "${TARGET}")
EOF
cd "${ROOT_DIR}"
docker buildx build \
  "--platform=${BUILDKIT_PLATFORM}" \
  "--network=${BUILDKIT_NETWORK}" \
  "--output=type=local,dest=${BUILD_OUTPUT_DEST}" \
  "--cache-from=${CACHE_FROM}" \
  "--cache-to=${CACHE_TO}" \
  "$@" \
  "--file=${ROOT_DIR}/${TARGET}/Dockerfile" \
  "${ROOT_DIR}"
