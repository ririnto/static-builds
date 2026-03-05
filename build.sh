#!/usr/bin/env sh
set -eu
WORKDIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
usage() {
  echo "Usage: $0 <target> [buildx args...]"
  echo ""
  echo "Arguments:"
  echo "  target          Build target (e.g., nginx, haproxy, apache-httpd)"
  echo "  buildx args     Additional arguments passed to docker buildx build"
  echo ""
  echo "Environment variables:"
  echo "  BUILDKIT_PLATFORM     Target platform (default: linux/amd64)"
  echo "  BUILD_OUTPUT_DEST     Output destination (default: out/<target>/ for local)"
  echo ""
  echo "Examples:"
  echo "  $0 nginx"
  echo "  $0 haproxy --progress=plain"
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
if [ ! -d "${WORKDIR}/${TARGET}" ]; then
  echo "Error: Target directory '${TARGET}' not found"
  exit 1
fi
if [ ! -f "${WORKDIR}/${TARGET}/.env" ]; then
  echo "Error: .env file not found in '${TARGET}'"
  exit 1
fi
if [ ! -f "${WORKDIR}/${TARGET}/Dockerfile" ]; then
  echo "Error: Dockerfile not found in '${TARGET}'"
  exit 1
fi
BUILDKIT_PLATFORM="${BUILDKIT_PLATFORM:-linux/amd64}"
BUILDKIT_CACHE_BACKEND="${BUILDKIT_CACHE_BACKEND:-local}"
if [ "${CI:-}" = "true" ] || [ "${GITHUB_ACTIONS:-}" = "true" ]; then
  BUILDKIT_NETWORK="${BUILDKIT_NETWORK:-default}"
else
  BUILDKIT_NETWORK="${BUILDKIT_NETWORK:-host}"
fi
case "${BUILDKIT_CACHE_BACKEND}" in
local)
  mkdir -p "${WORKDIR}/${TARGET}/.cache"
  CACHE_FROM="type=local,src=${WORKDIR}/${TARGET}/.cache"
  CACHE_TO="type=local,dest=${WORKDIR}/${TARGET}/.cache,mode=max"
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
if [ "${CI:-}" = "true" ] || [ "${GITHUB_ACTIONS:-}" = "true" ]; then
  BUILD_OUTPUT_DEST="${WORKDIR}/${TARGET}"
else
  BUILD_OUTPUT_DEST="${BUILD_OUTPUT_DEST:-${WORKDIR}/out/${TARGET}}"
fi
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
done <"${WORKDIR}/${TARGET}/.env"
cd "${WORKDIR}/${TARGET}"
docker buildx build \
  "--platform=${BUILDKIT_PLATFORM}" \
  "--network=${BUILDKIT_NETWORK}" \
  "--output=type=local,dest=${BUILD_OUTPUT_DEST}" \
  "--cache-from=${CACHE_FROM}" \
  "--cache-to=${CACHE_TO}" \
  "$@" \
  "${WORKDIR}/${TARGET}"
