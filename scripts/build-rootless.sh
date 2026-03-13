#!/usr/bin/env sh
set -eu
ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
METADATA_SCRIPT="${ROOT_DIR}/scripts/metadata.sh"
if [ "$#" -lt 1 ]; then
  printf 'Usage: %s <target>\n' "$0" >&2
  exit 1
fi
TARGET="$1"
case "${TARGET}" in
*[!A-Za-z0-9._-]*)
  printf 'Error: Invalid target %s\n' "${TARGET}" >&2
  exit 1
  ;;
esac
if [ ! -f "${ROOT_DIR}/${TARGET}/Dockerfile" ]; then
  printf 'Error: Dockerfile not found in %s\n' "${TARGET}" >&2
  exit 1
fi
if ! command -v buildctl-daemonless.sh >/dev/null 2>&1; then
  printf 'Error: buildctl-daemonless.sh not found in PATH\n' >&2
  exit 1
fi
BUILDKIT_PLATFORM="${BUILDKIT_PLATFORM:-linux/amd64}"
BUILDKIT_CACHE_BACKEND="${BUILDKIT_CACHE_BACKEND:-local}"
BUILDKIT_NETWORK="${BUILDKIT_NETWORK:-default}"
BUILD_OUTPUT_DEST="${BUILD_OUTPUT_DEST:-${ROOT_DIR}/.out/${TARGET}}"
mkdir -p "${BUILD_OUTPUT_DEST}"
case "${BUILDKIT_CACHE_BACKEND}" in
local)
  mkdir -p "${ROOT_DIR}/.cache/${TARGET}"
  CACHE_FROM="type=local,src=${ROOT_DIR}/.cache/${TARGET}"
  CACHE_TO="type=local,dest=${ROOT_DIR}/.cache/${TARGET},mode=max"
  ;;
*)
  printf 'Error: Invalid BUILDKIT_CACHE_BACKEND %s\n' "${BUILDKIT_CACHE_BACKEND}" >&2
  printf '%s\n' 'Allowed values: local' >&2
  exit 1
  ;;
esac
case "${BUILDKIT_NETWORK}" in
host|default) ;;
*)
  printf '%s\n' 'Error: Rootless BuildKit only supports host/default network semantics in this repository' >&2
  exit 1
  ;;
esac
set -- buildctl-daemonless.sh build \
  --frontend dockerfile.v0 \
  --local "context=${ROOT_DIR}" \
  --local "dockerfile=${ROOT_DIR}" \
  --opt "filename=${TARGET}/Dockerfile" \
  --opt "platform=${BUILDKIT_PLATFORM}" \
  --output "type=local,dest=${BUILD_OUTPUT_DEST}" \
  --import-cache "${CACHE_FROM}" \
  --export-cache "${CACHE_TO}"
while IFS='=' read -r key value || [ -n "${key}" ]; do
  case "${key}" in
  '' | \#*) continue ;;
  esac
  if [ -z "${value}" ]; then
    continue
  fi
  value=$(printf '%s\n' "${value}" | tr -d '\r')
  case "${key}" in
  *[!A-Z0-9_]*) continue ;;
  esac
  set -- "$@" --opt "build-arg:${key}=${value}"
done <<EOF2
$(sh "${METADATA_SCRIPT}" get-env "${TARGET}")
EOF2
"$@"
